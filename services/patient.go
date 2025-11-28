package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"errors"
	"fmt"
	"log"

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
