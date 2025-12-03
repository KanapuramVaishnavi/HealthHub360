package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"errors"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// func CreatePrescription(c *gin.Context, data map[string]interface{}) (string, error) {
// 	medicines, ok := data["medicines"].([]map[string]interface{})
// 	if !ok {
// 		log.Println("medicines field doesnot exists in response")
// 		return "", errors.New("Medicines field doesnot exists in response")
// 	}
// 	for _, medicine := range medicines {
// 		fields := []string{"medicineId", "instructions"}
// 		for _, field := range fields {
// 			err := getTrimmedString(data, field)
// 			if err != nil {
// 				log.Println("Error from getTrimmedString: ", err)
// 				return "", err
// 			}
// 		}
// 		intFields := []string{"dosagePerFrequency", "noOfDays"}
// 		for _, field := range intFields {
// 			number, ok := data[field].(float64)
// 			if !ok {
// 				log.Printf("Field %s not in integer format", field)
// 				return "", errors.New("Field not in integer format")
// 			}
// 			data[field] = int(number)
// 		}
// 		frequency, ok := medicine["frequency"].([]map[string]interface{})
// 		if !ok {
// 			log.Println("frequency field is not in the medicine")
// 			return "", errors.New("Frequency field not in medicine")
// 		}
// 		freFields := []string{"morning", "afternoon", "night"}
// 		for _, f := range frequency {
// 			for _, fre := range freFields {
// 				freType, ok := f[fre].(bool)
// 				if !ok {
// 					log.Printf("Field %s is not in frequency", fre)
// 					return "", errors.New("Field not in frequency")
// 				}
// 				f[fre] = freType
// 			}

// 		}
// 		data["frequency"] = frequency
// 		doctorId, err := GetFromContext[string](c, "code")
// 		if err != nil {
// 			log.Println("Error from getFromContext", err)
// 			return "", err
// 		}

// 		coll := prescriptionCollection
// 		code, err := GenerateEmpCode(coll)
// 		if err != nil {
// 			log.Println("Error from generateEmpCode: ", err)
// 			return "", err
// 		}
// 		data["code"] = code
// 		data["createdBy"] = doctorId
// 		data["updatedBy"] = doctorId
// 		data["createdAt"] = time.Now()
// 		data["updatedAt"] = time.Now()
// 		collection := db.OpenCollections(coll)
// 		inserted, err := db.CreateOne(c, collection, data)
// 		if err != nil {
// 			log.Println("Error from createOne: ", err)
// 			return "", err
// 		}
// 		log.Println("Inserted prescription: ", inserted.InsertedID)
// 		key, err := redis.CreateCacheKey(coll, code)
// 		if err != nil {
// 			log.Println("Error from createCacheKey: ", err)
// 			return "", err
// 		}
// 	}
// }
func CreatePrescription(c *gin.Context, data map[string]interface{}) (string, error) {
	// Extract medicines list
	rawMedicines, ok := data["medicines"].([]interface{})
	if !ok {
		log.Println("Medicines field must be list of interface")
		return "", errors.New("medicines must be an array")
	}

	for _, m := range rawMedicines {
		medicine, ok := m.(map[string]interface{})
		if !ok {
			return "", errors.New("invalid medicine format")
		}

		// Trim string fields
		fields := []string{"medicineId", "instructions"}
		for _, field := range fields {
			err := getTrimmedString(medicine, field)
			if err != nil {
				return "", err
			}
		}

		// Integer fields
		intFields := []string{"dosagePerFrequency", "noOfDays"}
		for _, field := range intFields {
			floatValue, ok := medicine[field].(float64)
			if !ok {
				return "", errors.New("integer field invalid: " + field)
			}
			medicine[field] = int(floatValue)
		}

		// Frequency validation
		frequency, ok := medicine["frequency"].(map[string]interface{})
		if !ok {
			return "", errors.New("frequency must be an object")
		}

		boolFields := []string{"morning", "afternoon", "night"}
		for _, bf := range boolFields {
			val, ok := frequency[bf].(bool)
			if !ok {
				return "", errors.New("frequency field missing: " + bf)
			}
			frequency[bf] = val
		}
	}

	// Get doctorId from context
	doctorId, err := GetFromContext[string](c, "code")
	if err != nil {
		return "", err
	}

	// Generate prescription code
	coll := prescriptionCollection
	prescriptionCode, err := GenerateEmpCode(coll)
	if err != nil {
		return "", err
	}

	// Set metadata
	data["code"] = prescriptionCode
	data["createdBy"] = doctorId
	data["updatedBy"] = doctorId
	data["createdAt"] = time.Now()
	data["updatedAt"] = time.Now()

	// Insert into DB
	collection := db.OpenCollections(coll)
	_, err = db.CreateOne(c, collection, data)
	if err != nil {
		return "", err
	}

	// Cache
	key, err := redis.CreateCacheKey(coll, prescriptionCode)
	if err == nil {
		redis.SetCache(c, key, data)
	}

	return prescriptionCode, nil
}
