package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"errors"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func CreatePrescription(c *gin.Context, data map[string]interface{}) (string, error) {
	medicines, ok := data["medicines"].([]map[string]interface{})
	if !ok {
		log.Println("medicines field doesnot exists in response")
		return "", errors.New("Medicines field doesnot exists in response")
	}
	for _, medicine := range medicines {
		fields := []string{"medicineId", "instructions"}
		for _, field := range fields {
			err := getTrimmedString(data, field)
			if err != nil {
				log.Println("Error from getTrimmedString: ", err)
				return "", err
			}
		}
		intFields := []string{"dosagePerFrequency", "noOfDays"}
		for _, field := range intFields {
			number, ok := data[field].(float64)
			if !ok {
				log.Printf("Field %s not in integer format", field)
				return "", errors.New("Field not in integer format")
			}
			data[field] = int(number)
		}
		frequency, ok := medicine["frequency"].([]map[string]interface{})
		if !ok {
			log.Println("frequency field is not in the medicine")
			return "", errors.New("Frequency field not in medicine")
		}
		freFields := []string{"morning", "afternoon", "night"}
		for _, f := range frequency {
			for _, fre := range freFields {
				freType, ok := f[fre].(bool)
				if !ok {
					log.Printf("Field %s is not in frequency", fre)
					return "", errors.New("Field not in frequency")
				}
				f[fre] = freType
			}

		}
		data["frequency"] = frequency
		doctorId, err := GetFromContext[string](c, "code")
		if err != nil {
			log.Println("Error from getFromContext", err)
			return "", err
		}

		coll := prescriptionCollection
		code, err := GenerateEmpCode(coll)
		if err != nil {
			log.Println("Error from generateEmpCode: ", err)
			return "", err
		}
		data["code"] = code
		data["createdBy"] = doctorId
		data["updatedBy"] = doctorId
		data["createdAt"] = time.Now()
		data["updatedAt"] = time.Now()
		collection := db.OpenCollections(coll)
		inserted, err := db.CreateOne(c, collection, data)
		if err != nil {
			log.Println("Error from createOne: ", err)
			return "", err
		}
		log.Println("Inserted prescription: ", inserted.InsertedID)
		key, err := redis.CreateCacheKey(coll, code)
		if err != nil {
			log.Println("Error from createCacheKey: ", err)
			return "", err
		}
	}
}
