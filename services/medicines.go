package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"errors"
	"log"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
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

func FetchMedicineByCode(c *gin.Context, medicineId string) (map[string]interface{}, error) {
	coll := medicineCollection
	key, err := redis.CreateCacheKey(coll, medicineId)
	if err != nil {
		log.Println("Error from createCacheKey: ", err)
		return nil, err
	}
	cached := make(map[string]interface{})
	exists, err := redis.GetCache(c, key, &cached)
	createdBy, ok := c.Get("code")
	if !ok {
		log.Println("Error while fetching the code")
	}
	if err == nil && exists {
		if createdBy.(string) != cached["createdBy"].(string) {
			log.Println("Receptionist does not have access")
			return nil, errors.New("This receptionist does not have access")
		}
		return cached, nil
	}
	medicine := make(map[string]interface{})
	collection := db.OpenCollections(coll)
	filter := bson.M{
		"code": medicineId,
	}
	err = db.FindOne(c, collection, filter, medicine)
	if err != nil {
		log.Println("Error from findOne function: ", err)
		return nil, err
	}
	log.Println("MEDICINE :", medicine)
	return medicine, nil
}
