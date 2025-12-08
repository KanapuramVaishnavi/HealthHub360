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
It will Create pharmacist by making certain validatiosn by generating the code
and fetching tennatid from the hospitaldoc
reespectively .Finally it sent email to the respected person states that validation is completed
*/
func CreatePharmacist(ctx *gin.Context, body map[string]interface{}) error {
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
	key := util.PharamacistKey + code
	err = redis.SetCache(ctx, key, body)
	if err != nil {
		log.Println("Error while caching new pharmacist: ", err)
	}
	if _, err := SaveUserToDB(collection, body); err != nil {
		log.Println("Error from the saveUserToDB:", err)
		return err
	}
	if err := CreateLoginRecord(ctx, collection, code, body["email"].(string), body["phoneNo"].(string), body["password"].(string)); err != nil {
		log.Println("Error from the createLoginRecord", err)
		return err
	}

	subject := "Your Pharmacist OTP Verification"
	mbody := fmt.Sprintf("Hello %s,\n\nYour OTP for Pharmacist verification is: %s\n\nThank you!", body["name"].(string), otp)

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
func FetchPharmacistByCode(c *gin.Context, pharmacistId string) (map[string]interface{}, error) {

	coll := pharmacistCollection
	key := util.PharamacistKey + pharmacistId
	isSuperAdmin, err := IsSuperAdmin(c)
	if err != nil {
		log.Println("Error from isSuperAdmin: ", err)
		return nil, err
	}

	tenantId, err := GetTenantIdFromContext(c)
	if err != nil {
		log.Println("Error from the getTenantIdFromToken:", err)
		return nil, err
	}

	code, err := GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext(code): ", err)
		return nil, err
	}
	ctxCollection, err := GetFromContext[string](c, "collection")
	if err != nil {
		log.Println("Error from getFromContext(collection): ", err)
		return nil, err
	}
	cached := make(map[string]interface{})

	cached, exists, err := FetchByCodeFromCache(c, key, isSuperAdmin, tenantId, code, ctxCollection)
	if err != nil {
		log.Println("Error from FetchByCodeFromCache: ", err)
		return nil, err
	}
	if exists && cached != nil {
		return cached, nil
	}

	result := make(map[string]interface{})
	collection := db.OpenCollections(coll)
	log.Println("Error from getCache:", err)
	filter := bson.M{
		"code": pharmacistId,
	}

	err = db.FindOne(c, collection, filter, &result)
	if err != nil {
		log.Println("Error from findOne function")
		return nil, errors.New("Error from the findOne function:")
	}
	err = HasAccess(isSuperAdmin, ctxCollection, tenantId, code, result)
	if err != nil {
		log.Println("Error from HasAccess: ", err)
		return nil, err
	}

	err = redis.SetCache(c, key, result)
	if err != nil {
		log.Println("Error from setCache")
		return nil, err
	}

	return result, nil
}

/*
It gives the all the pharmacist on the database
*/
func FetchAllPharmacist(c *gin.Context, tenantId string) ([]interface{}, error) {
	collection := db.OpenCollections(pharmacistCollection)
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
func UpdatePharmacist(c *gin.Context, data map[string]interface{}, code string) error {
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
	collection := db.OpenCollections(pharmacistCollection)
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
		log.Println("This Pharmacist does not have access to update")
		return errors.New("This Pharmacist doesnot have access")
	}
	res, err := db.UpdateOne(c, collection, filter, updateFilter)
	if err != nil {
		log.Println("Error from updateOne:", err)
		return err
	}

	log.Println(res.UpsertedCount)

	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	key := util.PharamacistKey + code
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old pharmacist cache:", err)
	}

	if err := redis.SetCache(c, key, data); err != nil {
		log.Println("Failed caching updated pharmacist:", err)
	}

	return nil
}

/*
Delete the Pharmacist from the pharmacist Collection
*/
func DeletePharmacist(c *gin.Context, code string) (string, error) {
	key := util.PharamacistKey + code
	collection := db.OpenCollections(pharmacistCollection)
	hospitalCodeRaw, ok := c.Get("code")
	if !ok {
		log.Println("Unable to fetch code from the context")
		return "", errors.New("Error unable to fetch code from the context")
	}
	hospitalCode, ok := hospitalCodeRaw.(string)
	if !ok {
		return "", errors.New("Unable to get hospitalCode from the context")
	}
	filter := bson.M{
		"code": code,
	}
	result := make(map[string]interface{})
	err := db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from the findOne function: ", err)
		return "", err
	}
	val := result["createdBy"].(string)
	if val != hospitalCode {
		log.Println("This hospital admin doesnot have access")
		return "", errors.New("This hospital admin doesnot have access")
	}
	err = redis.DeleteCache(c, key)
	if err != nil {
		return "", err
	}
	deleted, err := db.DeleteOne(c, collection, filter)
	msg := fmt.Sprintf("The Pharamacist %s deleted and the count is %d", code, deleted)
	return msg, nil
}
