package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
It will Create receptionist by making certain validatiosn by generating the code
and fetching tennatid from the hospitaldoc
reespectively .Finally it sent email to the respected person states that validation is completed
*/
func CreateReceptionist(ctx *gin.Context, body map[string]interface{}) error {
	err := ValidateUserInput(body)
	if err != nil {
		log.Println("Error from ValidateUserInput:", err)
		return err
	}
	collection, err := FetchCollectionFromRoleDoc(ctx, body["roleCode"].(string))
	if err != nil {
		log.Println("Error from fetchRoleDocAndCollection:", err)
		return err
	}
	code, CreatedBy, err := CheckerAndGenerateUserCodes(ctx, collection, body["email"].(string), body["phoneNo"].(string))
	if err != nil {
		log.Println("Error from GenerateUserRole", err)
		return err
	}

	otp, err := GenerateAndHashOTP(body)
	if err != nil {
		log.Println("Error from GenerateAndHashOTP", err)
		return err
	}
	log.Println("otp:", otp)

	tenantId, err := GetTenantIdFromToken(ctx)
	if err != nil {
		log.Println("Error from getTenantIdFromToken", err)
		return err
	}
	log.Println("tenantId from context: ", tenantId)

	if err := PrepareUser(body, code, CreatedBy, tenantId); err != nil {
		log.Println("Error from PrepareUser", err)
		return err
	}
	if err := CacheUserInRedis(ctx, code, body, collection); err != nil {
		log.Println("Error from the CacheUserInRedis", err)
		return err
	}
	if _, err := SaveUserToDB(collection, body); err != nil {
		log.Println("Error from the saveUserToDB:", err)
		return err
	}
	if err := CreateLoginRecord(ctx, collection, code, body["email"].(string), body["phoneNo"].(string), body["password"].(string)); err != nil {
		log.Println("Error from the createLoginRecord", err)
		return err
	}

	subject := "Your Receptionist OTP Verification"
	mbody := fmt.Sprintf("Hello %s,\n\nYour OTP for Receptionest verification is: %s\n\nThank you!", body["name"].(string), otp)

	err = SendOTPToMail(body["email"].(string), subject, mbody)
	if err != nil {
		log.Println("OTP email failed:", err)
		return errors.New("failed to send OTP email")
	}
	log.Println("mail sent successfully")
	return nil
}

/*
* Create a key to fetch from cache
* Fetch from cache if found then extract tenantId and compare with the input tenantId
* If not found go to db search for the document
* Check whether the tenantId matches with the input tenantId
* If comparision works then return the docs
 */
func FetchReceptionistByCode(c *gin.Context, code string) (map[string]interface{}, error) {

	coll := receptionistCollection
	key, err := redis.CreateCacheKey(coll, code)
	if err != nil {
		log.Println("Error creating cache key:", err)
		return nil, err
	}
	tenantId, err := GetTenantIdFromToken(c)
	if err != nil {
		log.Println("Error from the getTenantIdFromToken:", err)
		return nil, err
	}
	cached := make(map[string]interface{})
	exists, err := redis.GetCache(c, key, &cached)

	if err == nil && exists {
		tenantIdFromCache, ok := cached["tenantId"].(string)
		if !ok {
			return nil, errors.New("cached doctor missing tenantId")
		}
		if tenantId != tenantIdFromCache {
			return nil, errors.New("tenant not allowed to fetch this doctor")
		}
		return cached, nil
	}
	result := make(map[string]interface{})
	collection := db.OpenCollections(coll)
	log.Println("Error from getCache:", err)
	filter := bson.M{
		"code": code,
	}
	err = db.FindOne(c, collection, filter, &result)
	if err != nil {
		log.Println("Error from findOne function")
		return nil, errors.New("Error from the findOne function:")
	}
	value := result["tenantId"].(string)
	if value != tenantId {
		return nil, errors.New("This User admin doesnot have access")
	}
	err = redis.SetCache(c, key, result)
	if err != nil {
		log.Println("Error from setCache")
		return nil, err
	}

	return result, nil
}

/*
It gives the all the receptionist on the database
*/
func FetchAllReceptionist(c *gin.Context, tenantId string) ([]interface{}, error) {
	collection := db.OpenCollections(receptionistCollection)
	filter := bson.M{"tenantId": tenantId}
	doc, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from FindAll", err)
		return nil, err
	}
	return doc, nil
}

/*
* If fields provided,trim them and append to the input data
* Get the code from claims which is createdBy field
* Update based on the update and search filters
 */
func UpdateReceptionist(c *gin.Context, data map[string]interface{}, code string) error {
	fields := []string{"name", "email", "phoneNo"}
	for _, f := range fields {
		if err := trimIfExists(data, f); err != nil {
			log.Println("Error from ")
			return err
		}
	}
	if err := handleDOB(data); err != nil {
		return err
	}

	createdBy, ok := c.Get("code")
	if !ok {
		return errors.New("unable to fetch code from context")
	}
	updateFilter := BuildUpdateFilter(data, createdBy.(string))
	filter := bson.M{
		"createdBy": code,
	}
	collection := db.OpenCollections(receptionistCollection)
	value := make(map[string]interface{})
	err := db.FindOne(c, collection, filter, value)
	if err != nil {
		log.Println("Error from the findOne function", err)
		return err
	}
	log.Println(value)
	val := value["createdBy"].(string)
	log.Println(val)
	log.Println(createdBy)
	if val != createdBy {
		log.Println("This Receptionist does not have access to update")
		return errors.New("This Receptionist doesnot have access")
	}
	res, err := db.UpdateOne(c, collection, filter, updateFilter)
	if err != nil {
		log.Println("Error from updateOne:", err)
		return err
	}

	log.Println(res.UpsertedCount)

	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	refreshCache(c, receptionistCollection, code, result)

	return nil
}

/*
* Search for patientId in dataBase
* Search for appointments fields
* Search for all the appointments and get maxNum
* Generate new appointmentCode and return
 */
func GenerateAppointmentCode(c context.Context, patientId string) (string, error) {

	collection := db.OpenCollections(patientCollection)

	// Fetch patient document
	patient := make(map[string]interface{})
	filter := bson.M{
		"code": patientId,
	}
	err := db.FindOne(c, collection, filter, patient)
	if err != nil {
		log.Println("Error from the findOne function: ", err)
		return "", err
	}

	rawAppt := patient["appointments"]
	var apptList []interface{}

	switch v := rawAppt.(type) {
	case primitive.A:
		apptList = []interface{}(v)
	case []interface{}:
		apptList = v
	default:
		return "A0001", nil
	}

	if len(apptList) == 0 {
		return "A0001", nil
	}

	// Convert to []map[string]interface{}
	var appointments []map[string]interface{}
	for _, a := range apptList {
		if m, ok := a.(map[string]interface{}); ok {
			appointments = append(appointments, m)
		}
	}

	// Find max Axxxx code
	maxNum := 0
	re := regexp.MustCompile(`A(\d+)$`)

	for _, ap := range appointments {
		code, _ := ap["code"].(string)
		matches := re.FindStringSubmatch(code)

		if len(matches) >= 2 {
			num, _ := strconv.Atoi(matches[1])
			if num > maxNum {
				maxNum = num
			}
		}
	}
	log.Println("maxNum: ", maxNum)

	// Generate next code
	newCode := fmt.Sprintf("A%04d", maxNum+1)
	return newCode, nil
}
func getReceptionistID(c *gin.Context) (interface{}, error) {
	code, ok := c.Get("code")
	if !ok {
		log.Println("Unable to get receptionist code from context")
		return nil, errors.New("Error unable to get code from context")
	}
	return code, nil
}
func validateAppointmentInput(data map[string]interface{}) error {
	fields := []string{"patientId", "reason", "symptoms", "date", "time"}
	for _, f := range fields {
		if err := getTrimmedString(data, f); err != nil {
			log.Println("Error from getTrimmedString:", err)
			return err
		}
	}
	return nil
}

/*
* Search for receptionist and doctor
* Compare both createdBy give access for the receptionist to view doctor
*
 */
func CheckForPrivileges(c *gin.Context, receptionistId, doctorId string) (map[string]interface{}, error) {

	receptionist, err := FetchReceptionistByCode(c, receptionistId)
	if err != nil {
		log.Println("Error from FetchReceptionistByCode:", err)
		return nil, err
	}
	recepHosCodeVal, ok := receptionist["createdBy"]
	if !ok {
		log.Println("Error getting hospitalCode from Receptionist ")
		return nil, errors.New("Error from getting hospitalCode from receptionist")
	}
	recepHosCode, ok := recepHosCodeVal.(string)
	if !ok {
		log.Println("Type assertion while converting recepHosCode")
		return nil, errors.New("Type assertion converting recepHosCode")
	}

	doctor, err := FetchDoctorByCode(c, doctorId)
	if err != nil {
		log.Println("Error from FetchDoctorByCode: ", err)
		return nil, err
	}
	docHosCodeVal, ok := doctor["createdBy"]
	if !ok {
		log.Println("Error getting hospitalCode from doctor ")
		return nil, errors.New("Error from getting hospitalCode from doctor")
	}
	docHosCode, ok := docHosCodeVal.(string)
	if !ok {
		log.Println("Type assertion while converting docHosCode")
		return nil, errors.New("Type assertion converting docHosCode")
	}

	if recepHosCode != docHosCode {
		log.Println("This receptionist doesnot have access to view this doctor")
		return nil, errors.New("This receptionist doesnot have access for this doctor")
	}
	return doctor, nil
}

/*
* Move to doctorAvailabilitySlots
* Get the filter and search for document
* Validate isWeeklyOff and isLeave fields
* Return the found document
 */
func fetchDoctorSlot(c context.Context, coll *mongo.Collection, filter bson.M) (map[string]interface{}, error) {
	doc := make(map[string]interface{})
	err := db.FindOne(c, coll, filter, doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("No timeslot data found for this doctor on this date")
		}
		return nil, err
	}
	if off, _ := doc["isWeeklyOff"].(bool); off {
		return nil, errors.New("Doctor weekly off — doctor not available")
	}
	if leave, _ := doc["isLeave"].(bool); leave {
		return nil, errors.New("Doctor is on leave — doctor not available")
	}
	return doc, nil
}

/*
* Search for the slots in the given document
* Based on the given time filter it
* Then update several fields if match found
* Update the doctorAvailability slots with the search filter as well as update filter
 */
func checkAndBookSlot(ctx context.Context, slotColl *mongo.Collection, doc map[string]interface{}, timeGiven, patientId string) error {
	slotsList := []map[string]interface{}{}
	switch raw := doc["slots"].(type) {
	case primitive.A: // slot is primitive array
		for _, v := range raw {
			slotsList = append(slotsList, v.(map[string]interface{}))
		}

	case []interface{}: // normal array
		for _, v := range raw {
			slotsList = append(slotsList, v.(map[string]interface{}))
		}

	default:
		return errors.New("Invalid slot type found in DB")
	}

	slotFound := false
	for _, slot := range slotsList {
		if slot["start"].(string) == timeGiven {
			slotFound = true
			if !slot["isAvailable"].(bool) {
				return errors.New("Slot is not available")
			}
			if slot["isBooked"].(bool) {
				return errors.New("Slot already booked")
			}
			break
		}
	}

	if !slotFound {
		return errors.New("Slot does not exist for this doctor")
	}

	update := bson.M{
		"$set": bson.M{
			"slots.$.patientId":   patientId,
			"slots.$.isAvailable": false,
			"slots.$.isBooked":    true,
		},
	}
	filter := bson.M{
		"doctorId":    doc["doctorId"],
		"hospitalId":  doc["hospitalId"],
		"date":        doc["date"],
		"slots.start": timeGiven,
	}
	_, err := db.UpdateOne(ctx, slotColl, filter, update)
	if err != nil {
		log.Println("Error while updating slots availability when match found: ", err)
	}
	return err
}

/*
* Generate medicalRecord code
* Generate new medicalDocument
* Insert new document in the medicalRecord db
 */
func createMedicalRecord(ctx context.Context, data map[string]interface{}, doctorId string, hospitalId string, nurseId string, createdBy string) (string, error) {
	medicalCode, err := GenerateEmpCode(medicalRecordCollection)
	if err != nil {
		log.Println("Error while generating medicalRecord code: ", err)
		return "", err
	}

	medicalDoc := bson.M{
		"code":          medicalCode,
		"doctorId":      doctorId,
		"nurseId":       nurseId,
		"hospitalId":    hospitalId,
		"patientId":     data["patientId"],
		"appointmentId": data["appCode"],
		"reason":        data["reason"],
		"createdBy":     createdBy,
		"updatedBy":     createdBy,
		"createdAt":     time.Now(),
		"updatedAt":     time.Now(),
	}

	coll := db.OpenCollections(medicalRecordCollection)
	_, err = db.CreateOne(ctx, coll, medicalDoc)
	if err != nil {
		log.Println("Error while creating createMedicalRecord: ", err)
		return "", err
	}
	return medicalCode, err
}

/*
* BuildAppointment which is newOne
* Get all the fields
* return the new appointment
 */
func buildAppointment(data map[string]interface{}, doctorId, hospitalId, nurseId, appCode, medicalCode, createdBy, dateModified string) map[string]interface{} {
	return map[string]interface{}{
		"code":       appCode,
		"date":       dateModified,
		"time":       data["time"],
		"doctorId":   doctorId,
		"nurseId":    nurseId,
		"hospitalId": hospitalId,
		"medicalId":  medicalCode,
		"tenantId":   data["tenantId"],
		"reason":     data["reason"],
		"symptoms":   data["symptoms"],
		"createdBy":  createdBy,
		"createdAt":  time.Now(),
		"updateBy":   createdBy,
		"updatedAt":  time.Now(),
	}
}

/*
* Check for the patient with the given patientId
* Check for appointments field and get all appointments
* Add new appointment to the list and Update to the patient
 */
func updatePatientAppointments(ctx context.Context, patientId string, newApp map[string]interface{}) error {
	collection := db.OpenCollections(patientCollection)
	filter := bson.M{"code": patientId}

	patient := make(map[string]interface{})
	if err := db.FindOne(ctx, collection, filter, patient); err != nil {
		return err
	}

	appointments := []map[string]interface{}{}
	if raw, ok := patient["appointments"]; ok {
		switch v := raw.(type) {
		case []map[string]interface{}:
			appointments = v

		case []interface{}:
			for _, item := range v {
				m, ok := item.(map[string]interface{})
				if !ok {
					return errors.New("invalid appointment format")
				}
				appointments = append(appointments, m)
			}

		case primitive.A:
			for _, item := range v {
				m, ok := item.(map[string]interface{})
				if !ok {
					return errors.New("invalid appointment format in primitive.A")
				}
				appointments = append(appointments, m)
			}

		default:
			return errors.New("unsupported appointments format")
		}

	} else {
		appointments = []map[string]interface{}{}
	}

	appointments = append(appointments, newApp)
	update := bson.M{
		"$set": bson.M{
			"appointments": appointments,
		}}
	_, err := db.UpdateOne(ctx, collection, filter, update)
	if err != nil {
		log.Println("Error while updating patientAppointments: ", err)
		return errors.New("Error while updating patientAppointment")
	}
	return err
}

func BookAppointment(c *gin.Context, doctorId string, nurseId string, data map[string]interface{}) (map[string]interface{}, error) {
	receptionistId, err := getReceptionistID(c)
	if err != nil {
		log.Println("Error from getReceptionistID: ", err)
		return nil, err
	}
	log.Println("CreatedBy:", receptionistId.(string))

	if err := validateAppointmentInput(data); err != nil {
		log.Println("Error from validateAppointmentInput: ", err)
		return nil, err
	}

	dateModified, err := NormalizeDOB(data["date"].(string))
	if err != nil {
		log.Println("Error from NormalizeDOB: ", err)
		return nil, err
	}

	doctor, err := CheckForPrivileges(c, receptionistId.(string), doctorId)
	if err != nil {
		log.Println("Error from FetchReceptionist: ", err)
		return nil, err
	}
	slotColl := db.OpenCollections(doctorTimeSlotCollection)
	docSlotFilter := bson.M{
		"doctorId":   doctorId,
		"hospitalId": doctor["createdBy"].(string),
		"date":       dateModified,
	}
	doc, err := fetchDoctorSlot(c, slotColl, docSlotFilter)
	if err != nil {
		log.Println("Error from fetchDoctorSlot:", err)
		return nil, err
	}
	timeGiven := data["time"].(string)
	if err := checkAndBookSlot(c, slotColl, doc, timeGiven, data["patientId"].(string)); err != nil {
		log.Println("Error fron checAndBookSlot: ", err)
		return nil, err
	}
	appCode, err := GenerateAppointmentCode(c, data["patientId"].(string))
	if err != nil {
		log.Println("Error from GenerateAppointment: ", err)
		return nil, err
	}
	data["appCode"] = appCode
	medicalCode, err := createMedicalRecord(c, data, doctorId, doctor["createdBy"].(string), nurseId, receptionistId.(string))
	if err != nil {
		return nil, err
	}
	tenantId, err := GetTenantIdFromToken(c)
	if err != nil {
		log.Println("Error from getTenantIfFromToken", err)
		return nil, err
	}
	data["tenantId"] = tenantId
	hospitalId := doctor["createdBy"].(string)
	newApp := buildAppointment(data, doctorId, hospitalId, nurseId, appCode, medicalCode, receptionistId.(string), dateModified)
	if err := updatePatientAppointments(c, data["patientId"].(string), newApp); err != nil {
		log.Println("Error while updatingPatientAppointment")
		return nil, err
	}
	return newApp, nil
}
