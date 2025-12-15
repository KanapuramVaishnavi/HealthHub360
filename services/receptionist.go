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
* Validate user inputs first
* Fetch collection name from the roleCode given
* Check the fields and Generate a code and then createdBy
* Fetch tenantId from the context
* Include tenantId and generate otp and hash the otp
* Combine all the remaining data and prepare it
* Save to db and cache
* Send mail
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
* isSuperAdmin,tenantId,collection and code from context
* Pass those fields and key fetch from cache
* If exists,check who can access(superAdmin,tenantAdmin,hospitalAdmin)
* If not found go to db search for the document
* Search the doument, check who can access receptionist
* If comparision works then return the receptionist
 */
func FetchReceptionistByCode(c *gin.Context, receptionistId string) (map[string]interface{}, error) {

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
	key := util.ReceptionistKey + receptionistId

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
		"code": receptionistId,
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
		log.Println("Error from setCache: ", err)
	}

	return result, nil
}

/*
* Make a filter
* According to the user,the filter condition changes
* Search for listOfReceptionist
* Return them
 */
func FetchAllReceptionist(c *gin.Context) ([]interface{}, error) {
	code := c.GetString("code")
	log.Println("code from context: ", code)
	ctxCollection := c.GetString("collection")
	log.Println("collection from context: ", ctxCollection)
	isSuperAdmin := c.GetBool("isSuperAdmin")
	log.Println("isSuperAdmin from context: ", isSuperAdmin)

	filter := make(map[string]interface{})
	if isSuperAdmin {
		filter = bson.M{}
	} else if ctxCollection == TenantCollection {
		filter = bson.M{
			"tenantId": code,
		}
	} else if ctxCollection == hospitalCollection {
		filter = bson.M{
			"createdBy": code,
		}
	} else {
		log.Println("This user doesnot have access")
		return nil, errors.New("This user doesnot have access")
	}
	collection := db.OpenCollections(receptionistCollection)
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
* Update based on the search filters and update fields
* Fetch updated document
* Delete from cache, set in Cache
 */
func UpdateReceptionist(c *gin.Context, data map[string]interface{}, receptionistId string) (string, error) {
	fields := []string{"name", "email", "phoneNo"}
	for _, f := range fields {
		if err := trimIfExists(data, f); err != nil {
			log.Println("Error from ")
			return "", err
		}
	}
	if err := handleDOB(data); err != nil {
		return "", err
	}

	collection := db.OpenCollections(receptionistCollection)
	err := CheckForEmailAndPhoneNo(c, collection, data)
	if err != nil {
		log.Println("Error from checkForEmailAndPhoneNo: ", err)
		return "", err
	}

	code := c.GetString("code")
	updateFilter := BuildUpdateFilter(data, code)
	filter := bson.M{
		"code": receptionistId,
	}
	receptionist := make(map[string]interface{})

	err = db.FindOne(c, collection, filter, &receptionist)
	if err != nil {
		log.Println("Error from the findOne function: ", err)
		return "", err
	}
	log.Println("Receptionist: ", receptionist)

	hospitalId := receptionist["createdBy"].(string)
	log.Println("hospitalId: ", hospitalId)
	log.Println("code: ", code)
	if code != hospitalId {
		log.Println("This hospitalAdmin does not have access to update")
		return "", errors.New("This hospitalAdmin doesnot have access to update")
	}
	res, err := db.UpdateOne(c, collection, filter, updateFilter)
	if err != nil {
		log.Println("Error from updateOne:", err)
		return "", err
	}

	log.Println(res.ModifiedCount)

	result := make(map[string]interface{})
	key := util.ReceptionistKey + receptionistId
	err = db.FindOne(c, collection, filter, result)
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old receptionist from cache:", err)
	}

	if err := redis.SetCache(c, key, result); err != nil {
		log.Println("Failed caching updated receptionist:", err)
	}

	return "Updated Successfully", nil
}

/*
* Build filter to search based on receptionistId
* If found with the field createdBy from the result document found from document found
* Compare code from context and createdBy, if it works well go for the delete
* If not no another hospital admin can have access to delete it
 */
func DeleteReceptionist(c *gin.Context, receptionistId string) (string, error) {
	collection := db.OpenCollections(receptionistCollection)
	hospitalCodeRaw, ok := c.Get("code")
	if !ok {
		log.Println("Unable to fetch code from the context")
		return "", errors.New("Error unable to fetch code from the context")
	}
	hospitalId, ok := hospitalCodeRaw.(string)
	if !ok {
		return "", errors.New("Unable to get hospitalCode from the context")
	}

	filter := bson.M{
		"code": receptionistId,
	}
	result := make(map[string]interface{})
	err := db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from the findOne function: ", err)
		return "", err
	}
	val := result["createdBy"].(string)
	if val != hospitalId {
		log.Println("This hospital admin doesnot have access")
		return "", errors.New("This hospital admin doesnot have access")
	}
	deleted, err := db.DeleteOne(c, collection, filter)
	if err != nil {
		log.Println("Error from deleteOne: ", err)
		return "", err
	}
	log.Println("deleted: ", deleted)
	key := util.ReceptionistKey + receptionistId
	err = redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error from deleteCache: ", err)
	}
	msg := fmt.Sprintf("The doctor %s deleted", receptionistId)
	return msg, nil
}
