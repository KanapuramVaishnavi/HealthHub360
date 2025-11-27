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
func CreateNurse(c *gin.Context, data map[string]interface{}) (string, error) {
	val := ""
	err := ValidateUserInput(data)
	if err != nil {
		log.Println("Error from ValidateUserInput:", err)
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

	tenantId, err := GetTenantIdFromToken(c)
	if err != nil {
		log.Println("Error from getTenantIdFromToken", err)
		return val, err
	}
	log.Println("tenantId from context: ", tenantId)

	data["tenantid"] = tenantId
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
	subject := "Your Nurse OTP Verification"
	body := fmt.Sprintf("Hello %s,\n\nYour OTP for Nurse verification is: %s\n\nThank you!", data["name"].(string), otp)

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
func UpdateNurse(c *gin.Context, data map[string]interface{}, code string) error {
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
	collection := db.OpenCollections(nurseCollection)
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
	refreshCache(c, hospitalCollection, code, result)

	return nil
}

/*
It gives the all the nurses on the databse
*/
func FetchAllNurses(c *gin.Context, tenantid string) ([]interface{}, error) {
	collection := db.OpenCollections(nurseCollection)
	filter := bson.M{"tenantid": tenantid}
	log.Println(filter)
	doc, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from FindAll", err)
		return nil, err
	}
	return doc, nil
}

/*
It gives the all the nurses on the specific doctor
*/
func FetchAllNursesofDoctor(c *gin.Context, Docid string) ([]interface{}, error) {
	collection := db.OpenCollections(nurseCollection)
	filter := bson.M{"doctorid": Docid}
	doc, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from FindAll", err)
		return nil, err
	}
	return doc, nil
}

/*
* Create a key to fetch from cache
* Fetch from cache if found then extract tenantId and compare with the input tenantId
* If not found go to db search for the document
* Check whether the tenantId matches with the input tenantId
* If comparision works then return the docs
 */
func FetchNurseByCode(c *gin.Context, code string, tenantId string) (map[string]interface{}, error) {

	coll := nurseCollection
	key, err := redis.CreateCacheKey(coll, code)
	if err != nil {
		log.Println("Error creating cache key:", err)
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
	if err != nil {
		log.Println("Error from getCache:", err)
		filter := bson.M{
			"code": code,
		}
		err := db.FindOne(c, collection, filter, result)

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
	}
	return result, nil
}

/*
Delete Nurse By code where it matchs the code of the given parameters
*/
func DeleteNurseByCode(c *gin.Context, nurseid string) error {
	collection := db.OpenCollections(nurseCollection)
	filter := bson.M{"code": nurseid}
	doc, err := db.DeleteOne(c, collection, filter)
	log.Println(doc.DeletedCount)
	if err != nil {
		log.Println("Error from DeleteOne", err)
		return err
	}
	return nil
}
