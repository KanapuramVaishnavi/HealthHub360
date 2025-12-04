package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"HealthHub360/util"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

/*
* Validate inputs
* Fetch collection from roleDoc
* Check if user with same email or phoneNo exists
* GenerateOtp and bcrypt it
* Fill the extra fields to insert into the input fields
* Set in cache as well as in db
* Create login record in login collection
* Send otp to the provided mail
 */
func CreateHospital(c *gin.Context, data map[string]interface{}) error {
	if err := ValidateUserInput(data); err != nil {
		log.Println("Error from ValidateUserInput", err)
		return err
	}
	collection, err := FetchCollectionFromRoleDoc(c, data["roleCode"].(string))
	if err != nil {
		log.Println("Error from FetchRoleDocAndCollection:", err)
		return err
	}
	code, createdBy, err := CheckerAndGenerateUserCodes(c, collection, data["email"].(string), data["phoneNo"].(string))
	if err != nil {
		log.Println("Error from GenerateUserCodes:", err)
		return err
	}
	log.Println(code)
	otp, err := GenerateAndHashOTP(data)
	if err != nil {
		log.Println("Error from GeneraeAndHashOTP:", err)
		return err
	}
	tenantId, err := GetTenantIdFromContext(c)
	if err != nil {
		log.Println("Error from getTenantIdFromToken: ", err)
		return err
	}
	log.Println("tenantId from context: ", tenantId)

	if err = PrepareUser(data, code, createdBy, tenantId); err != nil {
		log.Println("Error from prepareUser :", err)
		return err
	}
	key := util.HospitalKey + code
	err = redis.SetCache(c, key, data)
	if err != nil {
		log.Println("Error from SetCache:", err)
		return errors.New("Error from setCache")
	}
	if _, err := SaveUserToDB(collection, data); err != nil {
		log.Println("Error from the saveUserToDB:", err)
		return err
	}
	if err := CreateLoginRecord(c, collection, code, data["email"].(string), data["phoneNo"].(string), data["password"].(string)); err != nil {
		log.Println("Error from the createLoginRecord", err)
		return err
	}
	subject := "Your Hospital OTP Verification"
	body := fmt.Sprintf("Hello %s,\n\nYour OTP for hospital verification is: %s\n\nThank you!", data["name"].(string), otp)

	err = SendOTPToMail(data["email"].(string), subject, body)
	if err != nil {
		log.Println("OTP email failed:", err)
		return errors.New("failed to send OTP email")
	}
	log.Println("mail sent successfully")
	return nil
}

/*
* Trim fields if they exists and fix them into the input data
 */
func trimIfExists(data map[string]interface{}, key string) error {
	if _, exists := data[key]; exists {
		err := getTrimmedString(data, key)
		if err != nil {
			log.Printf("Error trimming %s: %v", key, err)
			return err
		}
	}
	return nil
}

/*
* If DOB field exists then trim and normalize it
* Insert into the input field
 */
func handleDOB(data map[string]interface{}) error {
	raw, exists := data["dob"]
	if !exists {
		return nil
	}

	dobStr, ok := raw.(string)
	if !ok {
		return errors.New("dob must be a string")
	}

	if err := getTrimmedString(data, "dob"); err != nil {
		return err
	}

	normalized, err := NormalizeDate(dobStr)
	if err != nil {
		return err
	}

	data["dob"] = normalized
	return nil
}

/*
* Include all fields provided and extra field to modify into the input data provided
* Make it as update filter
 */
func BuildUpdateFilter(data map[string]interface{}, createdBy string) map[string]interface{} {
	// data["createdBy"] = createdBy
	data["updatedBy"] = createdBy
	data["updatedAt"] = time.Now()
	updateFilter := bson.M{"$set": data}
	return updateFilter
}

/*
* If fields provided,trim them and append to the input data
* Get the code from claims which is createdBy field
* Update based on the update and search filters
 */
func UpdateHospital(c *gin.Context, data map[string]interface{}, code string) error {
	fields := []string{"name", "email", "phoneNo"}
	for _, f := range fields {
		if err := trimIfExists(data, f); err != nil {
			log.Println("Error from trimIfExists")
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
		"code": code,
	}
	collection := db.OpenCollections(hospitalCollection)
	value := make(map[string]interface{})
	err := db.FindOne(c, collection, filter, value)
	if err != nil {
		log.Println("Error from the findOne function", err)
		return err
	}
	log.Println(value)
	val := value["createdBy"].(string)
	if val != createdBy {
		return errors.New("This tenant doesnot have access")
	}
	res, err := db.UpdateOne(c, collection, filter, updateFilter)
	if err != nil {
		log.Println("Error from updateOne:", err)
		return err
	}

	log.Println(res.UpsertedCount)

	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return err
	}
	key := util.HospitalKey + code
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old tenant cache:", err)
	}

	// Set new cache entry
	if err := redis.SetCache(c, key, result); err != nil {
		log.Println("Failed caching updated tenant:", err)
	}

	return nil
}

/*
* Get code from params
* Fetch from db
 */
func FetchHospitalByCode(c *gin.Context, code string) (map[string]interface{}, error) {
	coll := hospitalCollection

	key := util.HospitalKey + code
	log.Println("Cache key: ", key)
	isSuperAdmin, err := GetFromContext[bool](c, "isSuperAdmin")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
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
	tenantIdCache, ok := cached["tenantId"].(string)
	if !ok {
		fmt.Println("createdBy not found or invalid")
	}
	if !isSuperAdmin {
		if tenantIdCache != tenantId {
			log.Println("Error from the tenant which is tenant doesnot have access")
		}
	}
	if err == nil && exists {
		log.Println("From cache")
		return cached, nil
	}
	result := make(map[string]interface{})
	filter := bson.M{
		"code": code,
	}
	collection := db.OpenCollections(coll)
	log.Println("Filter: ", filter)
	err = db.FindOne(c, collection, filter, &result)
	if err != nil {
		log.Println("Error from the FindOne function,err")
		return nil, err
	}
	if !isSuperAdmin {
		tenantIdFromColl := result["tenantId"].(string)
		if tenantIdFromColl != tenantId {
			log.Println("This tenant does not have access to fetch")
			return nil, errors.New("This tenant does not have access to fetch")
		}
	}

	err = redis.SetCache(c, key, result)
	if err != nil {
		log.Println("Error from the setCache:", err)
		return nil, err
	}

	return result, nil
}

func FetchAllHospital(c *gin.Context) ([]interface{}, error) {
	collection := db.OpenCollections(hospitalCollection)
	tenantCode, ok := c.Get("code")
	if !ok {
		return nil, errors.New("unable to fetch code from context")
	}
	filter := bson.M{
		"createdBy": tenantCode,
	}
	doc, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from FindAll", err)
		return nil, err
	}
	return doc, nil
}

func DeleteHospitalByCode(c *gin.Context, code string) (string, error) {
	collection := db.OpenCollections(hospitalCollection)
	tenantCode, ok := c.Get("code")
	if !ok {
		return "", errors.New("unable to fetch code from context")
	}
	filter := bson.M{
		"code":      code,
		"createdBy": tenantCode,
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
	key := util.HospitalKey + code
	err = redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error from deleteCache:", err)
		return "", err
	}
	msg := fmt.Sprintf("User %s deleted successfuly ", code)
	return msg, nil
}
