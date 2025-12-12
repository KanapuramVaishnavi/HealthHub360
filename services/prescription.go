package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"HealthHub360/util"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func VerifyHasAccess(c *gin.Context, doctorId string, medicalRecordId string) (map[string]interface{}, error) {

	medicalRecord, err := FetchMedicalRecordByCode(c, medicalRecordId)
	if err != nil {
		log.Println("Error from fetchMedicalRecordByCode: ", err)
		return nil, err
	}
	doctorIdVal, exists := medicalRecord["doctorId"]
	if !exists {
		log.Println("doctorId doesnot exists in medicalRecord")
		return nil, errors.New("doctorId doesnot exists in medicalRecord")
	}
	doctorIdFromMedicalRecord, ok := doctorIdVal.(string)
	if !ok {
		log.Println("Type assertion error while fetching doctorId from medicalRecord")
		return nil, errors.New("Type assertion error while fetching doctorId from medicalRecord")
	}
	if doctorId != doctorIdFromMedicalRecord {
		log.Println("User doesnot have access")
		return nil, errors.New("User doesnot have access")
	}
	return medicalRecord, nil
}
func CreatePrescription(c *gin.Context, data map[string]interface{}, medicalRecordId string) (string, error) {

	doctorId, err := GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext(doctorId): ", err)
		return "", err
	}

	medicalRecord, err := VerifyHasAccess(c, doctorId, medicalRecordId)
	if err != nil {
		log.Println("Error from VerifyHasAccess: ", err)
		return "", err
	}
	rawMedicines, ok := data["medicines"].([]interface{})
	if !ok {
		log.Println("Medicines field must be list of interface")
		return "", errors.New("medicines must be an array")
	}
	err = getTrimmedString(data, "diagnosis")
	if err != nil {
		log.Println("Error from getTrimmedString: ", err)
		return "", err
	}
	for _, m := range rawMedicines {
		medicine, ok := m.(map[string]interface{})
		if !ok {
			return "", errors.New("invalid medicine format")
		}

		fields := []string{"medicineId", "instructions", "dosagePerFrequency", "noOfDays"}
		for _, field := range fields {
			err := getTrimmedString(medicine, field)
			if err != nil {
				return "", err
			}
		}
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

	tenantId, err := GetFromContext[string](c, "tenantId")
	if err != nil {
		log.Println("Error from getFromContext(tenantId): ", err)
		return "", err
	}

	coll := prescriptionCollection
	prescriptionCode, err := GenerateEmpCode(coll)
	if err != nil {
		return "", err
	}
	doctor, err := FetchDoctorByCode(c, doctorId)
	if err != nil {
		log.Println("Error from fetchDoctorByCode: ", err)
		return "", err
	}
	data["code"] = prescriptionCode
	data["hospitalId"] = doctor["createdBy"].(string)
	data["tenantId"] = tenantId
	data["createdBy"] = doctorId
	data["updatedBy"] = doctorId
	data["createdAt"] = time.Now()
	data["updatedAt"] = time.Now()

	collection := db.OpenCollections(coll)
	_, err = db.CreateOne(c, collection, data)
	if err != nil {
		return "", err
	}

	medRecDoc := make(map[string]interface{})
	medRecDoc["prescriptionId"] = prescriptionCode
	_, err = UpdateMedicalRecord(c, medicalRecordId, medRecDoc)
	if err != nil {
		log.Println("Error from updateMedicalRecord: ", err)
		return "", err
	}
	updAppointment := bson.M{
		"isProcessing": false,
	}

	_, err = UpdateAppointment(c, medicalRecord["appointmentId"].(string), updAppointment)
	if err != nil {
		log.Println("Update(isProcessing) field for the latestAppointment: ", err)
		return "", err
	}
	key := util.PrescriptionKey + prescriptionCode
	err = redis.SetCache(c, key, data)
	if err != nil {
		log.Println("Error while caching new prescription: ", err)
	}
	return "Created successfully", nil
}

func FetchPrescriptionByCode(c *gin.Context, prescriptionId string) (map[string]interface{}, error) {

	key := util.PrescriptionKey + prescriptionId

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
	coll := db.OpenCollections(prescriptionCollection)
	filter := bson.M{"code": prescriptionId}
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

func FetchAllPresciptions(c *gin.Context) ([]interface{}, error) {
	coll := prescriptionCollection
	collection := db.OpenCollections(coll)
	doctorId, err := GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return nil, err
	}
	filter := bson.M{
		"updatedBy": doctorId,
	}
	prescriptions, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from findAll: ", err)
		return nil, err
	}
	return prescriptions, nil
}

func ValidateUpdatePrescriptionData(data map[string]interface{}, doctorId string) (map[string]interface{}, error) {

	Fields := []string{"instructions", "dosagePerFrequency", "noOfDays"}
	for _, field := range Fields {
		err := trimIfExists(data, field)
		if err != nil {
			log.Println("Error from trimIfExists: ", err)
			return nil, err
		}
	}
	if freq, exists := data["frequency"]; exists {
		f, okay := freq.(map[string]interface{})
		if !okay {
			log.Println("frequency field must be object")
			return nil, errors.New("frequency field must be object")
		}
		fields := []string{"morning", "afternoon", "night"}
		for _, field := range fields {
			if val, ok := f[field]; ok {
				boolean, ok := val.(bool)
				if !ok {
					log.Printf("Frequency %s must be true/false", field)
					return nil, fmt.Errorf("Frequency %s must be true/false", field)
				}
				f[field] = boolean
			}
		}
		data["frequency"] = f
	}
	data["updatedBy"] = doctorId
	data["updatedAt"] = time.Now()
	return data, nil
}
func UpdatePrescription(c *gin.Context, prescriptionId string, medicineId string, data map[string]interface{}) (string, error) {
	doctorId, err := GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return "", err
	}
	err = trimIfExists(data, "diagnosis")
	if err != nil {
		log.Println("Error from trimIfExists: ", err)
		return "", err
	}
	data, err = ValidateUpdatePrescriptionData(data, doctorId)
	if err != nil {
		log.Println("Error from validateUpdatePrescriptionData: ", err)
		return "", err
	}
	coll := prescriptionCollection
	collection := db.OpenCollections(coll)
	filter := bson.M{
		"code":                 prescriptionId,
		"medicines.medicineId": medicineId,
	}
	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return "", err
	}
	docFromPrescriptionVal, ok := result["createdBy"]
	if !ok {
		log.Println("createdBy(doctor) field doesnot exists in prescription")
		return "", errors.New("createdBy(doctor) field doesnot exists in prescription")
	}
	if doctorId != docFromPrescriptionVal.(string) {
		log.Println("This doctor doesnot have access")
		return "", errors.New("This doctor doesnot have access")
	}
	update := bson.M{
		"$set": data,
	}
	updated, err := db.UpdateOne(c, collection, filter, update)
	if err != nil {
		log.Println("Error from updateOne: ", err)
		return "", err
	}
	log.Println("Updated prescription count: ", updated.ModifiedCount)
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return "", err
	}
	key := util.PrescriptionKey + prescriptionId
	err = redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error from deleteCache: ", err)
	}
	err = redis.SetCache(c, key, result)
	if err != nil {
		log.Println("Error from setCache: ", err)
	}
	return "updated successfully", nil
}
