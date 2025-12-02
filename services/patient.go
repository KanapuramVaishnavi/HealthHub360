package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func CreatePatient(c *gin.Context, data map[string]interface{}) (string, error) {
	val := ""
	err := ValidateUserInput(data)
	if err != nil {
		log.Println("Error from ValidateUserInput:", err)
		return val, err
	}

	collection, err := FetchCollectionFromRoleDoc(c, data["roleCode"].(string))
	if err != nil {
		log.Println("Error from fetchRoleDocAndCollection:", err)
		return val, err
	}
	code, createdBy, err := CheckerAndGenerateUserCodes(c, collection, data["email"].(string), data["phoneNo"].(string))
	if err != nil {
		log.Println("Error from GenerateUserRole", err)
		return val, err
	}
	log.Println(code)
	otp, err := GenerateAndHashOTP(data)
	if err != nil {
		log.Println("Error from GeneraeAndHashOTP:", err)
		return val, err
	}
	log.Println(otp)
	err = trimIfExists(data, "gender")
	if err != nil {
		log.Println("Error from trimIfExists", err)
		return val, err
	}
	err = trimIfExists(data, "admissionDate")
	if err != nil {
		log.Println("Error from trimIfExists")
		return val, err
	}
	tenantId, err := GetTenantIdFromContext(c)
	if err != nil {
		log.Println("Error from getTenantIdFromToken", err)
		return val, err
	}
	log.Println("tenantId from context: ", tenantId)
	if err = PrepareUser(data, code, createdBy, tenantId); err != nil {
		log.Println("Error from prepareUser :", err)
		return val, err
	}
	age, err := CalculateAge(data["dob"].(string))
	if err != nil {
		log.Println("Error from CalculateAge")
		return val, err
	}
	data["age"] = age
	if err := CacheUserInRedis(c, code, data, collection); err != nil {
		log.Println("Error from CacheUserInRedis: ", err)
		return val, err
	}
	if _, err := SaveUserToDB(collection, data); err != nil {
		log.Println("Error from the saveUserToDB:", err)
		return val, err
	}
	if err := CreateLoginRecord(c, collection, code, data["email"].(string), data["phoneNo"].(string), data["password"].(string)); err != nil {
		log.Println("Error from the createLoginRecord", err)
		return val, err
	}
	subject := "Your Patient OTP Verification"
	body := fmt.Sprintf("Hello %s,\n\nYour OTP for patient verification is: %s\n\nThank you!", data["name"].(string), otp)

	err = SendOTPToMail(data["email"].(string), subject, body)
	if err != nil {
		log.Println("OTP email failed:", err)
		return val, errors.New("failed to send OTP email")
	}
	log.Println("mail sent successfully")
	return "created successfully", nil
}

func FetchPatientByCode(c *gin.Context, patientId string) (map[string]interface{}, error) {
	coll := patientCollection
	key, err := redis.CreateCacheKey(coll, patientId)
	if err != nil {
		log.Println("Error from CreateCacheKey: ", err)
		return nil, err
	}

	tenantId, err := GetTenantIdFromContext(c)
	if err != nil {
		log.Println("Error from getTenantIdFromToken ", err)
		return nil, err
	}
	log.Println("tenantId from token: ", tenantId)

	cached := make(map[string]interface{})
	exists, err := redis.GetCache(c, key, &cached)
	log.Println("From cache: ", cached)
	tenantIdCache, ok := cached["tenantId"].(string)
	if !ok {
		fmt.Println("createdBy not found or invalid")
		return nil, errors.New("Unable to get the CreatedBy field from cache")
	}
	if tenantIdCache != tenantId {
		log.Println("Error from the tenant which is tenant doesnot have access")
		return nil, errors.New("This tenant does not have access")
	}
	if err == nil && exists {
		return cached, nil
	}

	collection := db.OpenCollections(patientCollection)
	filter := bson.M{
		"code": patientId,
	}

	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, &result)
	log.Println("From db: ", result)
	if err != nil {
		log.Println("Error from findOne function: ", err)
		return nil, err
	}
	tenantIdFromColl := result["tenantId"].(string)
	if tenantIdFromColl != tenantId {
		log.Println("This tenant does not have access to fetch")
		return nil, errors.New("This tenant does not have access to fetch")
	}
	err = redis.SetCache(c, key, result)
	if err != nil {
		log.Println("Error from SetCache: ", err)
		return nil, err
	}
	return result, nil
}

func UpdatePatientByCode(c *gin.Context, patientId string, data map[string]interface{}) (string, error) {
	val := ""
	receptionistId, err := GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return val, err
	}
	fields := []string{"name", "email", "phoneNo", "dob", "admissionDate", "gender"}
	for _, field := range fields {
		err := trimIfExists(data, field)
		if err != nil {
			log.Println("Error from getTrimmedString: ", err)
			return val, err
		}
	}
	err = handleDOB(data)
	if err != nil {
		log.Println("Error from handleDOB", err)
		return val, err
	}
	if admissionDateVal, ok := data["admissionDate"]; ok {
		if dateStr, ok := admissionDateVal.(string); ok {
			updatedAdmissionDate, err := NormalizeDate(dateStr)
			if err != nil {
				log.Println("Error from NormalizeDate:", err)
				return val, err
			}
			data["admissionDate"] = updatedAdmissionDate
		}
	}
	coll := patientCollection
	key, err := redis.CreateCacheKey(coll, patientId)
	if err != nil {
		log.Println("Error from CreateCacheKey: ", key)
	}
	collection := db.OpenCollections(coll)

	filter := bson.M{
		"code": patientId,
	}
	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return val, err
	}
	createdByVal, ok := result["createdBy"]
	if !ok {
		log.Println("Error whil fetching createdBy from patient")
		return val, errors.New("Error whil fetching createdBy from patient")
	}
	if receptionistId != createdByVal.(string) {
		log.Println("This receptionist doesnot have access")
		return val, errors.New("This recptionist doesnot have access")
	}
	data["updatedBy"] = receptionistId
	data["updatedAt"] = time.Now()
	update := bson.M{
		"$set": data,
	}
	updated, err := db.UpdateOne(c, collection, filter, update)
	if err != nil {
		log.Println("Error from updateOne: ", err)
		return val, err
	}
	log.Println("Updated patient: ", updated.ModifiedCount)
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return val, err
	}
	refreshCache(c, coll, patientId, result)
	return "Updated Successfully", nil
}

func FetchAllPatients(c *gin.Context) ([]interface{}, error) {
	receptionistId, err := GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext", err)
		return nil, err
	}
	filter := bson.M{
		"createdBy": receptionistId,
	}
	coll := patientCollection
	collection := db.OpenCollections(coll)
	patients, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from findall:", err)
		return nil, err
	}
	log.Println("Patients: ", patients)
	return patients, nil
}

func DeletePatient(c *gin.Context, patientId string) (string, error) {
	receptionistId, err := GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return "", err
	}
	filter := bson.M{
		"code":      patientId,
		"createdBy": receptionistId,
	}
	coll := patientCollection
	collection := db.OpenCollections(coll)
	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne function", err)
		return "", err
	}
	deleted, err := db.DeleteOne(c, collection, filter)
	if err != nil {
		log.Println("Error from deleteOne: ", err)
		return "", err
	}
	log.Println("Deleted:", deleted.DeletedCount)
	if deleted.DeletedCount == 0 {
		log.Println("This user doesnot have access")
		return "", errors.New("This user doesnot have access")
	}
	key, err := redis.CreateCacheKey(coll, patientId)
	if err != nil {
		log.Println("Error from createCacheKey", err)
	}
	redis.DeleteCache(c, key)
	return "Deleted successfully", nil
}
