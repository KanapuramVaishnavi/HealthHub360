package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"HealthHub360/util"
	"errors"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

/*
* If fields provided,trim them and append to the input data
* Get the code from claims which is createdBy field
* Update based on the search filters and update fields
* Fetch updated document
* Delete from cache, set in Cache
 */
func UpdateGuardianByCode(c *gin.Context, guardianId string, data map[string]interface{}) (string, error) {
	val := ""
	receptionistId, err := GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return val, err
	}
	fields := []string{"name", "dob", "phoneNo", "email", "govtId", "relation"}
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
	coll := GuardianCollection
	collection := db.OpenCollections(coll)
	err = CheckForEmailAndPhoneNo(c, collection, data)
	if err != nil {
		log.Println("Error from checkForEmailAndPhoneNo: ", err)
		return "", err
	}

	filter := bson.M{
		"code": guardianId,
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
	key := util.GuardianKey + guardianId
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old patient cache:", err)
	}

	if err := redis.SetCache(c, key, result); err != nil {
		log.Println("Failed caching updated patient:", err)
	}
	return "Updated Successfully", nil
}

/*
* Get isSuperAdmin,tenantId,collection and code values from context
* Pass those fields and key fetch from cache
* If exists,check who can access(superAdmin,tenantAdmin,hospitalAdmin)
* If not found go to db search for the document
* Search the doument, check who can access guardian
* If comparision works then return the guardian
 */
func FetchGuardianByCode(c *gin.Context, guardianId string) (map[string]interface{}, error) {

	key := util.GuardianKey + guardianId

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

	if cached, exists, err := checkCacheAccess(c, key, collFromContext, userData, tenantId, code, isSuperAdmin); exists {
		return cached, err
	}
	coll := db.OpenCollections(GuardianCollection)
	filter := bson.M{"code": guardianId}
	result := make(map[string]interface{})

	err = db.FindOne(c, coll, filter, &result)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return nil, errors.New("record not found")
	}

	if err := canAccess(userData, result, tenantId, code, collFromContext, isSuperAdmin); err != nil {
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
* Search for listOfGuardians
* Return them
 */
func FetchAllGuardians(c *gin.Context) ([]interface{}, error) {
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
			"hospitalId": code,
		}
	} else if ctxCollection == receptionistCollection {
		filter = bson.M{
			"createdBy": code,
		}
	} else {
		log.Println("This user doesnot have access")
		return nil, errors.New("This user doesnot have access")
	}
	coll := GuardianCollection
	collection := db.OpenCollections(coll)
	guardians, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from findall:", err)
		return nil, err
	}
	log.Println("Guardians : ", guardians)
	return guardians, nil
}

/*
* Build filter to search based on guardianId
* If found with the field createdBy from the result document found from document found
* Compare code from context and createdBy, if it works well go for the delete
* If not, no another receptionist can have access to delete it
 */
func DeleteGuardian(c *gin.Context, guardianId string) (string, error) {
	receptionistId, err := GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return "", err
	}
	filter := bson.M{
		"code": guardianId,
	}
	coll := GuardianCollection
	collection := db.OpenCollections(coll)
	key := util.GuardianKey + guardianId
	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne function", &err)
		return "", err
	}
	if result["createdBy"].(string) != receptionistId {
		log.Println("User doesnot have access")
		return "", errors.New("User doesnot have access")
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
	err = redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error from deletedCache: ", err)
	}
	return "Deleted successfully", nil
}
