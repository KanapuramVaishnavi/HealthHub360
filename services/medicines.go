package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"errors"
	"log"

	"github.com/gin-gonic/gin"
)

func CreateMedicines(c *gin.Context, data map[string]interface{}) (string, error) {
	fields := []string{"name", "dosage", "expiryDate"}
	for _, value := range fields {
		err := getTrimmedString(data, value)
		if err != nil {
			log.Println("Error from getTrimmedString")
			return "", err
		}
	}
	intFields := []string{"noOfStrips", "tabletsPerStrip"}
	for _, v := range intFields {
		number, ok := data[v].(float64)
		if !ok {
			return "", errors.New(v + " must be a number")
		}
		data[v] = int(number)
	}
	dateStr, err := NormalizeDOB(data["expiryDate"].(string))
	if err != nil {
		log.Println("Error from normalizeDOB: ", err)
		return "", err
	}
	data["expiryDate"] = dateStr
	createdByVal, ok := c.Get("code")
	if !ok {
		log.Println("Unable to get code from context ")
		return "", errors.New("Unable to get code from context")
	}
	createdBy, ok := createdByVal.(string)
	if !ok {
		log.Println("Type assertion error")
		return "", errors.New("Type assertion error")
	}
	data["createdBy"] = createdBy
	code, err := GenerateEmpCode(medicineCollection)
	if err != nil {
		log.Println("Error from generateEmpCode: ", err)
		return "", err
	}
	data["code"] = code
	log.Println("MEDICINE CODE:", code)
	coll := medicineCollection
	collection := db.OpenCollections(coll)
	inserted, err := db.CreateOne(c, collection, data)
	if err != nil {
		log.Println("Error from createOne: ", err)
		return "", err
	}
	log.Println("Inserted: ", inserted.InsertedID)
	key, err := redis.CreateCacheKey(coll, code)
	if err != nil {
		log.Println("Error from createCacheKey: ", err)
		return "", err
	}
	err = redis.SetCache(c, key, data)
	if err != nil {
		log.Println("Error from setCache: ", err)
		return "", err
	}
	return "Successfully created", nil
}
