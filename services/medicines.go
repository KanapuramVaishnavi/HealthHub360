package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"HealthHub360/util"
	"errors"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateMedicines(c *gin.Context, data map[string]interface{}) (string, error) {
	fields := []string{"name", "dosage", "expiryDate", "noOfStrips", "tabletsPerStrip", "pricePerStrip"}
	for _, value := range fields {
		err := getTrimmedString(data, value)
		if err != nil {
			log.Println("Error from getTrimmedString")
			return "", err
		}
	}
	// intFields := []string{"noOfStrips", "tabletsPerStrip", "pricePerStrip"}
	// for _, v := range intFields {
	// 	number, ok := data[v].(float64)
	// 	if !ok {
	// 		return "", errors.New(v + " must be a number")
	// 	}
	// 	data[v] = int(number)
	// }
	dateStr, err := NormalizeDate(data["expiryDate"].(string))
	if err != nil {
		log.Println("Error from normalizeDate: ", err)
		return "", err
	}
	data["expiryDate"] = dateStr
	pharmacistIdVal, ok := c.Get("code")
	if !ok {
		log.Println("Unable to get code from context ")
		return "", errors.New("Unable to get code from context")
	}
	pharmacistId, ok := pharmacistIdVal.(string)
	if !ok {
		log.Println("Type assertion error")
		return "", errors.New("Type assertion error")
	}
	noOfStripsVal, ok := data["noOfStrips"].(string)
	if !ok {
		log.Println("Unable to get noOfStrips")
		return "", errors.New("Unable to get noOfStrips")
	}
	noOfStrips, _ := strconv.Atoi(noOfStripsVal)
	tabletsPerStripVal, ok := data["tabletsPerStrip"].(string)
	if !ok {
		log.Println("Unable to get noOfStrips")
		return "", errors.New("Unable to get noOfStrips")
	}
	tabletsPerStrip, _ := strconv.Atoi(tabletsPerStripVal)
	data["createdBy"] = pharmacistId
	totalNoOfTabletsVal := noOfStrips * tabletsPerStrip
	totalNoOfTablets := strconv.Itoa(totalNoOfTabletsVal)
	data["totalNoOfTablets"] = totalNoOfTablets
	coll := medicineCollection
	collection := db.OpenCollections(coll)

	filter := bson.M{
		"name": data["name"],
	}
	medicine := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, medicine)
	if !errors.Is(err, mongo.ErrNoDocuments) {
		log.Println("Medicine with same name already exists: ", err)
		return "", errors.New("Medicine with same name already exists")
	}
	code, err := GenerateEmpCode(medicineCollection)
	if err != nil {
		log.Println("Error from generateEmpCode: ", err)
		return "", err
	}
	data["code"] = code
	pharmaColl := pharmacistCollection
	pharmaCollection := db.OpenCollections(pharmaColl)
	pharmacist := make(map[string]interface{})
	pFilter := bson.M{
		"code": pharmacistId,
	}
	err = db.FindOne(c, pharmaCollection, pFilter, pharmacist)
	if err != nil {
		log.Println("Error from findOne function: ", err)
		return "", err
	}
	data["tenantId"] = pharmacist["tenantId"].(string)
	data["hospitalId"] = pharmacist["createdBy"].(string)
	log.Println("MEDICINE CODE:", code)

	inserted, err := db.CreateOne(c, collection, data)
	if err != nil {
		log.Println("Error from createOne: ", err)
		return "", err
	}
	log.Println("Inserted: ", inserted.InsertedID)
	key := util.MedicinesKey + code
	err = redis.SetCache(c, key, data)
	if err != nil {
		log.Println("Error from setCache: ", err)
		return "", err
	}
	return "Successfully created", nil
}
func FetchMedicineByCode(c *gin.Context, medicineId string) (map[string]interface{}, error) {

	key := util.MedicinesKey + medicineId
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

	coll := db.OpenCollections(medicineCollection)
	filter := bson.M{"code": medicineId}
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

func FetchAllMedicines(c *gin.Context) ([]interface{}, error) {
	pharmacistId, err := GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return nil, err
	}
	coll := medicineCollection
	collection := db.OpenCollections(coll)
	filter := bson.M{
		"createdBy": pharmacistId,
	}
	medicines, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from findAll: ", err)
		return nil, err
	}
	if len(medicines) == 0 {
		log.Println("This user does not have access")
		return nil, errors.New("This user does not have access")
	}
	return medicines, nil
}

func UpdateMedicines(c *gin.Context, medicineId string, data map[string]interface{}) (string, error) {
	pharmacistId, err := GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return "", err
	}
	fields := []string{"name", "dosage", "expiryDate"}
	for _, field := range fields {
		err := trimIfExists(data, field)
		if err != nil {

			log.Println("Error from trimIfExists: ", err)
			return "", err
		}
	}
	intFields := []string{"noOfStrips", "tabletsPerStrip"}
	for _, field := range intFields {
		number, ok := data[field].(float64)
		if ok {
			data[field] = int(number)
		}
	}
	coll := medicineCollection
	collection := db.OpenCollections(coll)
	filter := bson.M{
		"code": medicineId,
	}
	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne:", err)
		return "", err
	}
	if pharmacistId != result["createdBy"].(string) {
		log.Println("This pharmacist doesnot have access")
		return "", errors.New("This pharamcist does not have access")
	}
	update := bson.M{
		"$set": data,
	}
	updated, err := db.UpdateOne(c, collection, filter, update)
	if err != nil {
		log.Println("Error from updateOne:", err)
		return "", err
	}
	log.Println("updated medicine count: ", updated.ModifiedCount)
	updatedMedicine := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, updatedMedicine)
	if err != nil {
		log.Println("Error from findOne:", err)
		return "", err
	}
	key := util.MedicinesKey + medicineId
	err = db.FindOne(c, collection, filter, result)
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old medicine cache:", err)
	}

	if err := redis.SetCache(c, key, result); err != nil {
		log.Println("Failed caching updated medicine:", err)
	}
	return "Updated successfully", nil
}

func DeleteMedicine(c *gin.Context, medicineId string) (string, error) {
	pharmacistId, err := GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return "", err
	}
	coll := medicineCollection
	collection := db.OpenCollections(coll)
	filter := bson.M{
		"code":      medicineId,
		"createdBy": pharmacistId,
	}
	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne function: ", err)
		return "", err
	}
	deleted, err := db.DeleteOne(c, collection, filter)
	if err != nil {
		log.Println("Error from deleteOne: ", err)
		return "", err
	}
	log.Println("DeletedCount: ", deleted.DeletedCount)
	if deleted.DeletedCount == 0 {
		log.Println("This pharmacist doesnot have access")
		return "", errors.New("This user doesnot have access")
	}
	return "Deleted successfully", nil
}
