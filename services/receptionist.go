package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"HealthHub360/util"
	"errors"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
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
	key := util.ReceptionistKey + code
	err = redis.SetCache(ctx, key, body)
	if err != nil {
		log.Println("Unable to set receptionist in cache: ", err)
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
	tenantId, err := GetFromContext[string](c, "tenantId")
	if err != nil {
		log.Println("Error while fetching tenantId from getFromContext: ", err)
		return nil, err
	}
	isSuperAdmin, err := GetFromContext[bool](c, "isSuperAdmin")
	if err != nil {
		log.Println("Error while fetching isSuperAdmin from getFromContext: ", err)
		return nil, err
	}
	log.Println("tenantId from getFromContext: ", tenantId)
	log.Println("isSuperAdmin from getFromContext: ", isSuperAdmin)
	cached := make(map[string]interface{})
	key := util.ReceptionistKey + code

	exists, err := redis.GetCache(c, key, &cached)

	if err == nil && exists {
		tenantIdFromCache, ok := cached["tenantId"].(string)
		if !ok {
			return nil, errors.New("cached doctor missing tenantId")
		}
		if !isSuperAdmin {
			if tenantId != tenantIdFromCache {
				return nil, errors.New("tenant not allowed to fetch this doctor")
			}
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
	if !isSuperAdmin {
		if value != tenantId {
			return nil, errors.New("This User admin doesnot have access")
		}
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
	key := util.ReceptionistKey + code
	err = db.FindOne(c, collection, filter, result)
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old tenant cache:", err)
	}

	if err := redis.SetCache(c, key, data); err != nil {
		log.Println("Failed caching updated tenant:", err)
	}

	return nil
}
