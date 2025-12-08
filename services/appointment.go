package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"HealthHub360/util"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
* Get code from the context
* Return code
 */
func getReceptionistID(c *gin.Context) (interface{}, error) {
	code, ok := c.Get("code")
	if !ok {
		log.Println("Unable to get receptionist code from context")
		return nil, errors.New("Error unable to get code from context")
	}
	return code, nil
}

/*
* Validate the fields that came from request
 */
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
* Compare both createdBy and give access for the receptionist to view doctor
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

// /*
// * Search for patientId in dataBase
// * Search for appointments fields
// * Search for all the appointments and get maxNum
// * Generate new appointmentCode and return
//  */
// func GenerateAppointmentCode(c context.Context, patientId string) (string, error) {

// 	collection := db.OpenCollections(appointmentCollection)

// 	// Fetch patient document
// 	patient := make(map[string]interface{})
// 	filter := bson.M{
// 		"patientId": patientId,
// 	}
// 	appointmentCode, err := GenerateEmpCode(appointmentCollection)
// 	if err != nil {
// 		log.Println("Error from generateEmpCode: ", err)
// 		return "", err
// 	}
// 	return appp
// 	// 	err := db.FindOne(c, collection, filter, patient)
// 	// 	if err != nil {
// 	// 		log.Println("Error from the findOne function: ", err)
// 	// 		return "", err
// 	// 	}

// 	// 	rawAppt := patient["appointments"]
// 	// 	var apptList []interface{}

// 	// 	switch v := rawAppt.(type) {
// 	// 	case primitive.A:
// 	// 		apptList = []interface{}(v)
// 	// 	case []interface{}:
// 	// 		apptList = v
// 	// 	default:
// 	// 		return "A0001", nil
// 	// 	}

// 	// 	if len(apptList) == 0 {
// 	// 		return "A0001", nil
// 	// 	}

// 	// 	// Convert to []map[string]interface{}
// 	// 	var appointments []map[string]interface{}
// 	// 	for _, a := range apptList {
// 	// 		if m, ok := a.(map[string]interface{}); ok {
// 	// 			appointments = append(appointments, m)
// 	// 		}
// 	// 	}

// 	// 	// Find max Axxxx code
// 	// 	maxNum := 0
// 	// 	re := regexp.MustCompile(`A(\d+)$`)

// 	// 	for _, ap := range appointments {
// 	// 		code, _ := ap["code"].(string)
// 	// 		matches := re.FindStringSubmatch(code)

// 	// 		if len(matches) >= 2 {
// 	// 			num, _ := strconv.Atoi(matches[1])
// 	// 			if num > maxNum {
// 	// 				maxNum = num
// 	// 			}
// 	// 		}
// 	// 	}
// 	// 	log.Println("maxNum: ", maxNum)

// 	// // Generate next code
// 	// newCode := fmt.Sprintf("A%04d", maxNum+1)
// 	// return newCode, nil
// }

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
func createMedicalRecord(c *gin.Context, data map[string]interface{}, doctorId string, hospitalId string, nurseId string, createdBy string, tenantId string) (string, error) {
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
		"tenantId":      tenantId,
		"appointmentId": data["code"],
		"reason":        data["reason"],
		"createdBy":     createdBy,
		"updatedBy":     createdBy,
		"createdAt":     time.Now(),
		"updatedAt":     time.Now(),
	}
	_, err = GenerateAndHashOTP(data)
	if err != nil {
		log.Println("Error from GeneraeAndHashOTP:", err)
		return "", err
	}
	coll := medicalRecordCollection
	collection := db.OpenCollections(medicalRecordCollection)
	key := util.MedicalRecordKey + medicalCode
	err = redis.SetCache(c, key, medicalDoc)
	if err != nil {
		log.Println("Error while caching new medicalRecord : ", err)
	}
	if _, err := SaveUserToDB(coll, data); err != nil {
		log.Println("Error from the saveUserToDB:", err)
		return "", err
	}
	_, err = db.CreateOne(ctx, collection, medicalDoc)
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
		"code":         appCode,
		"date":         dateModified,
		"time":         data["time"],
		"doctorId":     doctorId,
		"nurseId":      nurseId,
		"hospitalId":   hospitalId,
		"medicalId":    medicalCode,
		"tenantId":     data["tenantId"],
		"reason":       data["reason"],
		"symptoms":     data["symptoms"],
		"isProcessing": true,
		"createdBy":    createdBy,
		"createdAt":    time.Now(),
		"updateBy":     createdBy,
		"updatedAt":    time.Now(),
	}
}

/*
* Get appointments from the patient
* Update appointmnets with new appointmentId
* Refresh the cache
 */
func PatientUpdate(c *gin.Context, data map[string]interface{}, appCode, patientId string) error {
	patCollection := db.OpenCollections(patientCollection)
	patientFilter := bson.M{
		"code": patientId,
	}
	patient := make(map[string]interface{})
	findOneErr := db.FindOne(c, patCollection, patientFilter, patient)
	if findOneErr != nil {
		log.Println("Error from FindOne function: ", findOneErr)
		return findOneErr
	}
	log.Println("Patient: ", patient)
	log.Printf("patient appointments type %T", patient["appointments"])
	var appointments []string
	raw, exists := patient["appointments"]
	if !exists || raw == nil {
		appointments = []string{}
	} else {
		val, ok := raw.(primitive.A)
		log.Println("val: ", val)
		if !ok {
			log.Println("Unable to fetch appointments")
			return errors.New("Unable to fetch appointments")
		}
		var appointments []string
		for _, a := range val {
			if str, ok := a.(string); ok {
				appointments = append(appointments, str)
			} else {
				log.Println("Non-string value in appointments:", a)
			}
		}
	}

	appointments = append(appointments, appCode)
	patientUpdate := bson.M{
		"$set": bson.M{
			"appointments": appointments,
		},
	}

	log.Println("Appointments:", appointments)
	updated, err := db.UpdateOne(c, patCollection, patientFilter, patientUpdate)
	if err != nil {
		log.Println("Error from UpdateOne: ", err)
		return err
	}
	log.Println("Updated: ", updated.ModifiedCount)
	updPatient := make(map[string]interface{})
	updatedFindOneErr := db.FindOne(c, patCollection, patientFilter, updPatient)
	if updatedFindOneErr != nil {
		log.Println("Error from FindOne function: ", findOneErr)
		return findOneErr
	}
	key := util.PatientKey + patientId
	err = redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error while deleting patient from cache: ", err)
	}
	err = redis.SetCache(c, key, updPatient)
	if err != nil {
		log.Println("Error while caching updated patient: ", err)
	}
	return nil
}

/*
* GetReceptionistID from context
* Validate the input fields
* Normalize the date
* Check whether receptionist have access to create appointment for the doctorId
* DoctorAvailability check for weeklyOff and weekend and get slots
* Check the slot and book slot and update several fields
* Build appointment
* Update patient by appointment
 */

func CreateAppointment(c *gin.Context, doctorId string, nurseId string, data map[string]interface{}) (string, error) {

	receptionistId, err := getReceptionistID(c)
	if err != nil {
		log.Println("Error from getReceptionistID: ", err)
		return "", err
	}
	log.Println("CreatedBy:", receptionistId.(string))

	if err := validateAppointmentInput(data); err != nil {
		log.Println("Error from validateAppointmentInput: ", err)
		return "", err
	}

	dateModified, err := NormalizeDate(data["date"].(string))
	if err != nil {
		log.Println("Error from NormalizeDate: ", err)
		return "", err
	}

	doctor, err := CheckForPrivileges(c, receptionistId.(string), doctorId)
	if err != nil {
		log.Println("Error from FetchReceptionist: ", err)
		return "", err
	}
	slotColl := db.OpenCollections(doctorTimeSlotCollection)
	docSlotFilter := bson.M{
		"doctorId":   doctorId,
		"hospitalId": doctor["createdBy"].(string),
		"date":       dateModified,
	}
	log.Println("filter: ", docSlotFilter)
	doc, err := fetchDoctorSlot(c, slotColl, docSlotFilter)
	if err != nil {
		log.Println("Error from fetchDoctorSlot:", err)
		return "", err
	}
	appCode, err := GenerateEmpCode(appointmentCollection)
	if err != nil {
		log.Println("Error from generateEmpCode: ", err)
		return "", err
	}

	timeGiven := data["time"].(string)
	if err := checkAndBookSlot(c, slotColl, doc, timeGiven, data["patientId"].(string)); err != nil {
		log.Println("Error fron checAndBookSlot: ", err)
		return "", err
	}
	data["code"] = appCode
	tenantId, err := GetTenantIdFromContext(c)
	if err != nil {
		log.Println("Error from getTenantIfFromToken", err)
		return "", err
	}
	data["tenantId"] = tenantId
	medicalCode, err := createMedicalRecord(c, data, doctorId, doctor["createdBy"].(string), nurseId, receptionistId.(string), tenantId)
	if err != nil {
		return "", err
	}

	hospitalId := doctor["createdBy"].(string)
	newApp := buildAppointment(data, doctorId, hospitalId, nurseId, appCode, medicalCode, receptionistId.(string), dateModified)
	patientErr := PatientUpdate(c, data, appCode, data["patientId"].(string))
	if patientErr != nil {
		log.Println("Error from patientUpdate: ", patientErr)
		return "", patientErr
	}

	collection := db.OpenCollections(appointmentCollection)
	inserted, err := db.CreateOne(c, collection, newApp)
	if err != nil {
		log.Println("Error from createOne: ", err)
		return "", err
	}
	log.Println("inserted: ", inserted.InsertedID)
	key := util.AppointmentKey + appCode
	cacheErr := redis.SetCache(c, key, newApp)
	if cacheErr != nil {
		log.Println("Error from setCache : ", cacheErr)
		return "", cacheErr
	}
	return "created Successfully", nil
}

func canAccess(collFromContext string, userData, record map[string]interface{}, tenantId string, code string, isSuperAdmin bool) error {
	log.Println("record: ", record)

	if isSuperAdmin {
		return nil
	}

	if collFromContext == TenantCollection {
		if record["tenantId"].(string) != tenantId {
			return errors.New("tenant does not have access")
		}
		return nil
	}

	if collFromContext == hospitalCollection {
		if record["hospitalId"].(string) != code {
			return errors.New("hospital admin does not have access")
		}
		return nil
	}

	if userData["createdBy"].(string) != record["hospitalId"].(string) {
		return errors.New("user does not have access")
	}

	return nil
}
func checkCacheAccess(c *gin.Context, key string, collFromContext string, userData map[string]interface{},
	tenantId, code string, isSuperAdmin bool) (map[string]interface{}, bool, error) {

	cached := make(map[string]interface{})
	exists, err := redis.GetCache(c, key, &cached)
	if err != nil || !exists {
		return nil, false, nil
	}

	if err := canAccess(collFromContext, userData, cached, tenantId, code, isSuperAdmin); err != nil {
		return nil, true, err
	}

	return cached, true, nil
}
func fetchFromDB(c *gin.Context, appointmentId string, key string, collFromContext string, userData map[string]interface{}, tenantId, code string, isSuperAdmin bool) (map[string]interface{}, error) {

	coll := db.OpenCollections(appointmentCollection)

	result := make(map[string]interface{})
	filter := bson.M{"code": appointmentId}

	err := db.FindOne(c, coll, filter, result)
	if err != nil {
		return nil, errors.New("record not found")
	}

	if err := canAccess(collFromContext, userData, result, tenantId, code, isSuperAdmin); err != nil {
		return nil, err
	}

	_ = redis.SetCache(c, key, result)

	return result, nil
}

/*
* Get appointmentId from the services
* Get tenantId,code,collection,isSuperAdmin from the context
* Fetch that user
 */
func FetchAppointmentByCode(c *gin.Context, appointmentId string) (map[string]interface{}, error) {

	key := util.AppointmentKey + appointmentId

	tenantId := c.GetString("tenantId")
	code := c.GetString("code")
	collFromContext := c.GetString("collection")
	isSuperAdmin := c.GetBool("isSuperAdmin")

	collectionFromContext := db.OpenCollections(collFromContext)
	userData := make(map[string]interface{})
	err := db.FindOne(c, collectionFromContext, bson.M{"code": code}, userData)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return nil, err
	}

	if cached, exists, err := checkCacheAccess(
		c, key, collFromContext, userData, tenantId, code, isSuperAdmin,
	); exists {
		return cached, err
	}

	return fetchFromDB(c, appointmentId, key, collFromContext, userData, tenantId, code, isSuperAdmin)
}

// func FetchAppointmentByCode(c *gin.Context, appointmentId string) (map[string]interface{}, error) {

// 	coll := appointmentCollection
// 	key, err := redis.CreateCacheKey(coll, appointmentId)
// 	if err != nil {
// 		log.Println("Error creating cache key:", err)
// 		return nil, err
// 	}
// 	tenantId, ok := c.Get("tenantId")
// 	if !ok {
// 		log.Println("Unable to get tenantId from context")
// 		return nil, errors.New("Unable to get tenantId from context")
// 	}
// 	isSuperAdmin, ok := c.Get("isSuperAdmin")
// 	if !ok {
// 		log.Println("unable to get isSuperAdmin")
// 		return nil, errors.New("Unable to get isSuperAdmin")
// 	}
// 	code, ok := c.Get("code")
// 	if !ok {
// 		log.Println("Unable to get code from context")
// 		return nil, errors.New("Unable to get code from context")
// 	}
// 	collFromContext, ok := c.Get("collection")
// 	if !ok {
// 		log.Println("Unable to get collection from context")
// 		return nil, errors.New("Unable to get collection from context")
// 	}
// 	collectionFromContext := db.OpenCollections(collFromContext.(string))
// 	filter := bson.M{
// 		"code": code.(string),
// 	}
// 	userData := make(map[string]interface{})
// 	findErr := db.FindOne(c, collectionFromContext, filter, userData)
// 	if findErr != nil {
// 		log.Println("error from findOne: ", findErr)
// 		return nil, findErr
// 	}
// 	cached := make(map[string]interface{})
// 	exists, err := redis.GetCache(c, key, &cached)
// 	if err == nil && exists {
// 		tenantIdFromCache, ok := cached["tenantId"].(string)
// 		if !ok {
// 			return nil, errors.New("cached doctor missing tenantId")
// 		}
// 		if isSuperAdmin.(bool) {
// 			return cached, nil
// 		}
// 		if collFromContext.(string) == tenantCollection {
// 			if tenantId.(string) != tenantIdFromCache {
// 				log.Println("This user with this tenantId not allowed to fetch this doctor")
// 				return nil, errors.New("This user with this tenantId not allowed to fetch this doctor")
// 			}
// 			return cached, nil
// 		}

// 		if collFromContext.(string) == hospitalCollection {
// 			if code != cached["hospitalId"].(string) {
// 				log.Println("This hospital admin have access")
// 				return nil, errors.New("This hospital amdin doesnot have access")
// 			}

// 			return cached, nil
// 		}

// 		if userData["createdBy"].(string) != cached["hospitalId"].(string) {
// 			log.Println("This user doesnot have access")
// 			return nil, errors.New("This user doesnot have access")
// 		}
// 		return cached, nil

// 	}

// 	result := make(map[string]interface{})
// 	collection := db.OpenCollections(coll)
// 	filter = bson.M{
// 		"code": appointmentId,
// 	}
// 	err = db.FindOne(c, collection, filter, result)

// 	if err != nil {
// 		log.Println("Error from findOne function")
// 		return nil, errors.New("Error from the findOne function:")
// 	}
// 	if isSuperAdmin.(bool) {
// 		_ = redis.SetCache(c, key, result)
// 		return result, nil
// 	}
// 	if collFromContext.(string) == tenantCollection {
// 		value := result["tenantId"].(string)
// 		if value != tenantId {
// 			return nil, errors.New("Thistenant doesnot have access")
// 		}
// 		_ = redis.SetCache(c, key, result)
// 		return result, nil
// 	}

// 	hospitalId := result["hospitalId"].(string)
// 	if collFromContext.(string) == hospitalCollection {

// 		if hospitalId != code.(string) {
// 			return nil, errors.New("This hospital doesnot have access")
// 		}
// 		_ = redis.SetCache(c, key, result)

// 		return result, nil
// 	}
// 	if userData["createdBy"].(string) != hospitalId {
// 		return nil, errors.New("This user doesnot have access")
// 	}

// 	_ = redis.SetCache(c, key, result)

// 	return result, nil
// }

// "diagnosis": "Viral fever",
// "medications": ["Paracetamol 500mg", "ORS Solution"],
// "treatmentPlan": ["Rest for 3 days", "Drink plenty of water", "Follow-up after 2 days"],
// "upComingDate": "29-12-2025"
func FetchAllAppointment(c *gin.Context) ([]interface{}, error) {
	collection := db.OpenCollections(appointmentCollection)
	Code, ok := c.Get("code")
	if !ok {
		return nil, errors.New("unable to fetch code from context")
	}
	filter := bson.M{
		"createdBy": Code,
	}
	doc, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from FindAll", err)
		return nil, err
	}
	return doc, nil
}

func DeleteAppointmentByCode(c *gin.Context, code string) (string, error) {
	collection := db.OpenCollections(appointmentCollection)
	ReceptionestCode, ok := c.Get("code")
	if !ok {
		return "", errors.New("unable to fetch code from context")
	}
	filter := bson.M{
		"createdBy": ReceptionestCode,
	}
	log.Println(filter)
	result := make(map[string]interface{})
	err := db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from the findOne function:", err)
		return "", err

	}
	_, err = db.DeleteOne(c, collection, filter)
	if err != nil {
		log.Println("Error from the deleteOne function: ", err)
		return "", err
	}
	key := util.AppointmentKey + code
	err = redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error from deleteCache:", err)
		return "", err
	}
	msg := fmt.Sprintf("User %s deleted successfuly ", code)
	return msg, nil
}

func UpdateAppointment(c *gin.Context, appointmentId string, data map[string]interface{}) (string, error) {
	codeVal, ok := c.Get("code")
	if !ok {
		log.Println("Error while fetching from context. ")
		return "", errors.New("Error while fetching from context")
	}
	code, ok := codeVal.(string)
	if !ok {
		log.Println("Error for type assertion error to get collection. ")
		return "", errors.New("Error while type assertion to get collection")
	}
	data["updatedBy"] = code
	data["updatedAt"] = time.Now()
	appColl := db.OpenCollections(appointmentCollection)
	Filter := bson.M{
		"code": appointmentId,
	}
	appointment := make(map[string]interface{})
	err := db.FindOne(c, appColl, Filter, &appointment)
	if err != nil {
		log.Println("Error while fetching medicalRecord(FindOne)", err)
		return "", err
	}
	receptionistVal, ok := appointment["createdBy"]
	if !ok {
		log.Println("Error while checking the value is present in it or not")
		return "", errors.New("Error while checking the the doctorId exists")
	}
	receptionist, ok := receptionistVal.(string)
	if !ok {
		log.Println("Error during type assertion error")
		return "", errors.New("Error type assertion error for doctorId")
	}
	if receptionist != code {
		log.Println("This receptionist doesnot have access to update the appointment")
		return "", errors.New("This receptionist doesnot have access to update the appointment")
	}
	collection := db.OpenCollections(appointmentCollection)
	filter := bson.M{
		"code": appointmentId,
	}
	update := bson.M{
		"$set": data,
	}
	updated, err := db.UpdateOne(c, collection, filter, update)
	if err != nil {
		log.Println("Error while updating medicalRecord by doctor:", err)
		return "", err
	}
	log.Println("Updated: ", updated.ModifiedCount)
	updatedAppointment := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, updatedAppointment)
	if err != nil {
		log.Println("Error from findOne after updating", err)
		return "", err
	}

	key := util.ReceptionistKey + code
	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return "", err
	}
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old appointment cache:", err)
	}

	if err := redis.SetCache(c, key, result); err != nil {
		log.Println("Failed caching updated appoitment:", err)
	}
	return "updated", nil
}
