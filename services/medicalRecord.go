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

func FetchMedicalRecordByCode(c *gin.Context, medicalRecordId string) (map[string]interface{}, error) {
	key := util.MedicalRecordKey + medicalRecordId

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
	coll := db.OpenCollections(medicalRecordCollection)
	filter := bson.M{"code": medicalRecordId}
	result := make(map[string]interface{})

	err = db.FindOne(c, coll, filter, &result)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return nil, err
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

func UpdateMedicalRecordByNurse(c *gin.Context, medicalRecordId string, data map[string]interface{}) error {

	code, err := GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return err
	}
	data["updatedBy"] = code
	data["updatedAt"] = time.Now()
	medicalRecordColl := db.OpenCollections(medicalRecordCollection)
	mFilter := bson.M{
		"code": medicalRecordId,
	}
	medicalRecord := make(map[string]interface{})
	err = db.FindOne(c, medicalRecordColl, mFilter, &medicalRecord)
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
	err = db.FindOne(c, collection, filter, &updatedRecord)
	if err != nil {
		log.Println("Error from findOne after updating", err)
		return err
	}
	key := util.MedicalRecordKey + medicalRecordId
	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old medicalRecord cache:", err)
	}

	if err := redis.SetCache(c, key, result); err != nil {
		log.Println("Failed caching updated medicalRecord:", err)
	}
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
	key := util.MedicalRecordKey + medicalRecordId
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old medicalRecord cache:", err)
	}

	if err := redis.SetCache(c, key, updatedRecord); err != nil {
		log.Println("Failed caching updated medicalRecord:", err)
	}
	return nil
}
func UpdateMedicalRecordByPharmacist(c *gin.Context, medicalRecordId string, data map[string]interface{}) error {
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
	pharmacist, err := FetchPharmacistByCode(c, code)
	if err != nil {
		log.Println("Error from fetchPharmacistByCode: ", err)
		return err
	}
	doctorIdFromPharmacist := pharmacist["createdBy"].(string)
	data["updatedBy"] = code
	data["updatedAt"] = time.Now()
	medicalRecordColl := db.OpenCollections(medicalRecordCollection)
	mFilter := bson.M{
		"code": medicalRecordId,
	}
	medicalRecord := make(map[string]interface{})
	err = db.FindOne(c, medicalRecordColl, mFilter, &medicalRecord)
	if err != nil {
		log.Println("Error while fetching medicalRecord(FindOne)", err)
		return err
	}
	hospitalIdVal, ok := medicalRecord["hospitalId"]
	if !ok {
		log.Println("Error while checking the value is present in it or not")
		return errors.New("Error while checking the the hospitalId exists")
	}
	hospitalId, ok := hospitalIdVal.(string)
	if !ok {
		log.Println("Error during type assertion error(hospitalId)")
		return errors.New("Error type assertion error for doctorId")
	}
	if hospitalId != doctorIdFromPharmacist {
		log.Println("This pharmacist doesnot have access to update the record")
		return errors.New("This pharmacist doesnot have access to update the record")
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
	key := util.MedicalRecordKey + medicalRecordId
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old medicalRecord cache:", err)
	}

	if err := redis.SetCache(c, key, updatedRecord); err != nil {
		log.Println("Failed caching updated medicalRecord:", err)
	}
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
	case pharmacistCollection:
		if err := UpdateMedicalRecordByPharmacist(c, medicalRecordId, data); err != nil {
			return "", err
		}
		log.Println("Updated by pharmacist")
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

func FetchAllMedicalRecords(c *gin.Context) ([]interface{}, error) {
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
		collection := db.OpenCollections(receptionistCollection)
		receptionist := make(map[string]interface{})
		err := db.FindOne(c, collection, bson.M{"code": code}, receptionist)
		if err != nil {
			log.Println("Error from findOne: ", err)
			return nil, err
		}
		filter = bson.M{
			"hospitalId": receptionist["createdBy"].(string),
		}
	} else if ctxCollection == doctorCollection {
		filter = bson.M{
			"doctorId": code,
		}
	} else if ctxCollection == nurseCollection {
		filter = bson.M{
			"nurseId": code,
		}
	} else {
		log.Println("This user doesnot have access")
		return nil, errors.New("This user doesnot have access")
	}
	collection := db.OpenCollections(medicalRecordCollection)
	doc, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from FindAll", err)
		return nil, err
	}
	return doc, nil
}
