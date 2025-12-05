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
* Fetch tenantId from the hospital collection
* Include tenantId and generate otp and hash the otp
* Combine all the remaining data and prepare it
* Save to db and cache
* Send mail
 */
func CreateDoctor(c *gin.Context, data map[string]interface{}) (string, error) {
	val := ""
	err := ValidateUserInput(data)
	if err != nil {
		log.Println("Error from ValidateUserInput:", err)
		return val, err
	}
	err = getTrimmedString(data, "department")
	if err != nil {
		log.Println("Error from the getTrimmedString: ", err)
		return val, err
	}
	collection, err := FetchCollectionFromRoleDoc(c, data["roleCode"].(string))
	if err != nil {
		log.Println("Error from FetchRoleDocAndCollection:", err)
		return val, err
	}
	code, createdBy, err := CheckerAndGenerateUserCodes(c, collection, data["email"].(string), data["phoneNo"].(string))
	if err != nil {
		log.Println("Error from GenerateUserRole", err)
		return val, err
	}
	tenantId, err := GetTenantIdFromContext(c)
	if err != nil {
		log.Println("Error from getTenantIfFromToken: ", err)
		return val, err
	}
	log.Println("tenantId from context: ", tenantId)

	data["tenantId"] = tenantId
	otp, err := GenerateAndHashOTP(data)
	if err != nil {
		log.Println("Error from GeneraeAndHashOTP:", err)
		return val, err
	}
	log.Println(otp)
	if err = PrepareUser(data, code, createdBy, tenantId); err != nil {
		log.Println("Error from prepareUser :", err)
		return val, err
	}
	key := util.DoctorKey + code
	err = redis.SetCache(c, key, data)
	if err != nil {
		log.Println("Error from setCache: ", err)
	}
	if _, err := SaveUserToDB(collection, data); err != nil {
		log.Println("Error from the saveUserToDB:", err)
		return val, err
	}
	if err := CreateLoginRecord(c, collection, code, data["email"].(string), data["phoneNo"].(string), data["password"].(string)); err != nil {
		log.Println("Error from the createLoginRecord", err)
		return val, err
	}
	subject := "Your Hospital OTP Verification"
	body := fmt.Sprintf("Hello %s,\n\nYour OTP for Hospital verification is: %s\n\nThank you!", data["name"].(string), otp)

	err = SendOTPToMail(data["email"].(string), subject, body)
	if err != nil {
		log.Println("OTP email failed:", err)
		return "", errors.New("failed to send OTP email")
	}
	log.Println("mail sent successfully")
	return "created successfully", nil
}

/*
* If fields provided,trim them and append to the input data
* Get the code from claims which is createdBy field
* Update based on the update and search filters
 */
func UpdateDoctor(c *gin.Context, data map[string]interface{}, code string) error {
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

	hospitalCode, ok := c.Get("code")
	if !ok {
		return errors.New("unable to fetch code from context")
	}
	updateFilter := BuildUpdateFilter(data, hospitalCode.(string))
	filter := bson.M{
		"code": code,
	}
	collection := db.OpenCollections(doctorCollection)
	value := make(map[string]interface{})
	err := db.FindOne(c, collection, filter, value)
	if err != nil {
		log.Println("Error from the findOne function", err)
		return err
	}
	log.Println(value)
	val := value["createdBy"].(string)
	log.Println(val)
	log.Println(hospitalCode)
	if val != hospitalCode {
		log.Println("This doctor does not have access to update")
		return errors.New("This doctor doesnot have access")
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
		log.Println("Error from findOne:", err)
		return err
	}
	key := util.DoctorKey + code
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old tenant cache:", err)
	}

	if err := redis.SetCache(c, key, result); err != nil {
		log.Println("Failed caching updated tenant:", err)
	}

	return nil
}

/*
* Create a key to fetch from cache
* Fetch from cache if found then extract tenantId and compare with the input tenantId
* If not found go to db search for the document
* Check whether the tenantId matches with the input tenantId
* If comparision works then return the docs
 */
func FetchDoctorByCode(c *gin.Context, doctorId string) (map[string]interface{}, error) {
	coll := doctorCollection
	key := util.DoctorKey + doctorId
	isSuperAdmin, err := IsSuperAdmin(c)
	if err != nil {
		log.Println("Error from isSuperAdmin: ", err)
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
	filter := bson.M{
		"code": doctorId,
	}
	err = db.FindOne(c, collection, filter, &result)
	if err != nil {
		log.Println("Error from findOne function", err)
		return nil, errors.New("Error from the findOne function:")
	}
	value := result["tenantId"].(string)
	if !isSuperAdmin {
		if value != tenantId {
			return nil, errors.New("This User admin doesnot have access because of tenantId mismatch")
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
* Make a filter
* FindAll from the above filter
 */
func FetchAllDoctors(c *gin.Context, tenantId string) ([]interface{}, error) {
	collection := db.OpenCollections(doctorCollection)
	filter := bson.M{
		"tenantId": tenantId,
	}
	result, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from the findAll function: ", err)
		return nil, err
	}
	return result, nil
}

/*
* Get code from the token
* Compare code with the createdBy from the result document found from filter
* If comparision works well go for the delete
* If not return no another hospital admin can have access to delete it
 */
func DeleteDoctor(c *gin.Context, code string) (string, error) {
	collection := db.OpenCollections(doctorCollection)
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
	deleted, err := db.DeleteOne(c, collection, filter)
	msg := fmt.Sprintf("The doctor %s deleted and the count is %d", code, deleted)
	return msg, nil
}
