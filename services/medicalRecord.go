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

func FetchMedicalRecordByCode(c *gin.Context, medicalRecordId string) (map[string]interface{}, error) {
	coll := medicalRecordCollection
	key, err := redis.CreateCacheKey(coll, medicalRecordId)
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

	collection := db.OpenCollections(medicalRecordCollection)
	filter := bson.M{
		"code": medicalRecordId,
	}

	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
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

func UpdateMedicalRecordByNurse(c *gin.Context, medicalRecordId string, data map[string]interface{}) error {
	codeVal, ok := c.Get("code")
	if !ok {
		log.Println("Error while fetching collection from context. ")
		return errors.New("Error while fetching collection from context")
	}
	code, ok := codeVal.(string)
	if !ok {
		log.Println("Error for type assertion error to get collection. ")
		return errors.New("Error while type assertion to get collection")
	}
	data["updatedBy"] = code
	data["updatedAt"] = time.Now()
	medicalRecordColl := db.OpenCollections(medicalRecordCollection)
	mFilter := bson.M{
		"code": medicalRecordId,
	}
	medicalRecord := make(map[string]interface{})
	err := db.FindOne(c, medicalRecordColl, mFilter, &medicalRecord)
	if err != nil {
		log.Println("Error while fetching medicalRecord(FindOne)", err)
		return err
	}
	nurseIdVal, ok := medicalRecord["nurseId"]
	if !ok {
		log.Println("Error while checking the value is present in it or not")
		return errors.New("Error while checking the the nurseId exists")
	}
	nurseId, ok := nurseIdVal.(string)
	if !ok {
		log.Println("Error during type assertion error")
		return errors.New("Error type assertion error for nurseId")
	}
	if nurseId != code {
		log.Println("This nurse doesnot have access to updatethe record")
		return errors.New("This nurse doesnot have access to updatethe record")
	}
	collection := db.OpenCollections(medicalRecordCollection)
	filter := bson.M{
		"code": medicalRecordId,
	}
	update := bson.M{
		"$set": data,
	}
	updated, err := db.UpdateOne(c, collection, filter, update)
	if err != nil {
		log.Println("Error while updating medicalRecord by nurse:", err)
		return err
	}
	log.Println("Updated: ", updated.ModifiedCount)
	updatedRecord := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, updatedRecord)
	if err != nil {
		log.Println("Error from findOne after updating", err)
		return err
	}
	refreshCache(c, medicalRecordCollection, medicalRecordId, updatedRecord)
	return nil
}
func UpdateMedicalRecordByDoctor(c *gin.Context, medicalRecordId string, data map[string]interface{}) error {
	codeVal, ok := c.Get("code")
	if !ok {
		log.Println("Error while fetching  from context. ")
		return errors.New("Error while fetching  from context")
	}
	code, ok := codeVal.(string)
	if !ok {
		log.Println("Error for type assertion error to get collection. ")
		return errors.New("Error while type assertion to get collection")
	}
	data["updatedBy"] = code
	data["updatedAt"] = time.Now()
	medicalRecordColl := db.OpenCollections(medicalRecordCollection)
	mFilter := bson.M{
		"code": medicalRecordId,
	}
	medicalRecord := make(map[string]interface{})
	err := db.FindOne(c, medicalRecordColl, mFilter, &medicalRecord)
	if err != nil {
		log.Println("Error while fetching medicalRecord(FindOne)", err)
		return err
	}
	doctorIdVal, ok := medicalRecord["doctorId"]
	if !ok {
		log.Println("Error while checking the value is present in it or not")
		return errors.New("Error while checking the the doctorId exists")
	}
	doctorId, ok := doctorIdVal.(string)
	if !ok {
		log.Println("Error during type assertion error")
		return errors.New("Error type assertion error for doctorId")
	}
	if doctorId != code {
		log.Println("This doctor doesnot have access to update the record")
		return errors.New("This doctor doesnot have access to update the record")
	}
	collection := db.OpenCollections(medicalRecordCollection)
	filter := bson.M{
		"code": medicalRecordId,
	}
	update := bson.M{
		"$set": data,
	}
	updated, err := db.UpdateOne(c, collection, filter, update)
	if err != nil {
		log.Println("Error while updating medicalRecord by doctor:", err)
		return err
	}
	log.Println("Updated: ", updated.ModifiedCount)
	updatedRecord := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, updatedRecord)
	if err != nil {
		log.Println("Error from findOne after updating", err)
		return err
	}
	refreshCache(c, medicalRecordCollection, medicalRecordId, updatedRecord)
	return nil
}

// func UpdateMedicalRecordByReceptionist(c *gin.Context, medicalRecordId string, data map[string]interface{}) error {

// 	fields := []string{"doctorId", "nurseId", "patientId", "reason"}
// 	for _, f := range fields {
// 		if err := trimIfExists(data, f); err != nil {
// 			log.Println("Error from trimIfExists")
// 			return err
// 		}
// 	}
// 	createdByVal, ok := c.Get("code")
// 	if !ok {
// 		log.Println("unable to fetch code from context")
// 		return errors.New("unable to fetch code from context")
// 	}
// 	createdBy, ok := createdByVal.(string)
// 	if !ok {
// 		log.Println("unable to convert into string")
// 		return errors.New("unable to convert into string")
// 	}

//		collection := db.OpenCollections(medicalRecordCollection)
//		data["updatedAt"] = time.Now()
//		data["UpdatedBy"] = createdBy
//		filter := bson.M{
//			"code": medicalRecordId,
//		}
//		update := bson.M{
//			"$set": data,
//		}
//		updated, err := db.UpdateOne(c, collection, filter, update)
//		if err != nil {
//			log.Println("Error while updating medicalRecord by Receptionist:", err)
//			return err
//		}
//		log.Println("Updated: ", updated.ModifiedCount)
//		return nil
//	}
func UpdateMedicalRecord(c *gin.Context, medicalRecordId string, data map[string]interface{}) (string, error) {
	val := ""
	collectionVal, ok := c.Get("collection")
	if !ok {
		log.Println("Error while fetching collection from context. ")
		return val, errors.New("Error while fetching collection from context")
	}
	collection, ok := collectionVal.(string)
	if !ok {
		log.Println("Error for type assertion error to get collection. ")
		return val, errors.New("Error while type assertion to get collection")
	}
	msg := "Updated successfully"
	switch collection {
	case nurseCollection:
		if err := UpdateMedicalRecordByNurse(c, medicalRecordId, data); err != nil {
			return "", err
		}
		log.Println("Updated by nurse")
		return msg, nil

	case doctorCollection:
		if err := UpdateMedicalRecordByDoctor(c, medicalRecordId, data); err != nil {
			return "", err
		}
		log.Println("Updated by doctor")
		return msg, nil
	// case receptionistCollection:
	// 	if err := UpdateMedicalRecordByReceptionist(c, medicalRecordId, data); err != nil {
	// 		return "", err
	// 	}
	// 	log.Println("Updated by receptionist")
	// 	return msg, nil
	default:
		return "", errors.New("Unauthorized role")
	}
}
