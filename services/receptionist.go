package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
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

	tenantId, err := GetTenantIdFromContext(ctx)
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
	tenantId, err := GetTenantIdFromContext(c)
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
func createMedicalRecord(c *gin.Context, data map[string]interface{}, doctorId string, hospitalId string, nurseId string, createdBy string) (string, error) {
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
	if err := CacheUserInRedis(c, medicalCode, data, coll); err != nil {
		log.Println("Error from CacheUserInRedis: ", err)
		return "", err
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
