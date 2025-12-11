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

// func CreatePatient(c *gin.Context, data map[string]interface{}) (string, error) {
// 	val := ""
// 	err := ValidateUserInput(data)
// 	if err != nil {
// 		log.Println("Error from ValidateUserInput:", err)
// 		return val, err
// 	}

// 	collection, err := FetchCollectionFromRoleDoc(c, data["roleCode"].(string))
// 	if err != nil {
// 		log.Println("Error from fetchRoleDocAndCollection:", err)
// 		return val, err
// 	}
// 	code, createdBy, err := CheckerAndGenerateUserCodes(c, collection, data["email"].(string), data["phoneNo"].(string))
// 	if err != nil {
// 		log.Println("Error from GenerateUserRole", err)
// 		return val, err
// 	}
// 	log.Println(code)
// 	otp, err := GenerateAndHashOTP(data)
// 	if err != nil {
// 		log.Println("Error from GeneraeAndHashOTP:", err)
// 		return val, err
// 	}
// 	log.Println(otp)
// 	err = trimIfExists(data, "gender")
// 	if err != nil {
// 		log.Println("Error from trimIfExists", err)
// 		return val, err
// 	}
// 	err = trimIfExists(data, "admissionDate")
// 	if err != nil {
// 		log.Println("Error from trimIfExists")
// 		return val, err
// 	}
// 	tenantId, err := GetTenantIdFromContext(c)
// 	if err != nil {
// 		log.Println("Error from getTenantIdFromToken", err)
// 		return val, err
// 	}
// 	log.Println("tenantId from context: ", tenantId)
// 	if err = PrepareUser(data, code, createdBy, tenantId); err != nil {
// 		log.Println("Error from prepareUser :", err)
// 		return val, err
// 	}
// 	age, err := CalculateAge(data["dob"].(string))
// 	if err != nil {
// 		log.Println("Error from CalculateAge")
// 		return val, err
// 	}
// 	data["age"] = age
// 	receptionist, err := FetchReceptionistByCode(c, createdBy)
// 	if err != nil {
// 		log.Println("Error from fetchReceptionistByCode: ", err)
// 		return val, err
// 	}
// 	log.Println("Receptionist(createdBy): ", receptionist["createdBy"].(string))
// 	data["hospitalId"] = receptionist["createdBy"].(string)

// 	if _, err := SaveUserToDB(collection, data); err != nil {
// 		log.Println("Error from the saveUserToDB:", err)
// 		return val, err
// 	}
// 	key := util.PatientKey + code
// 	err = redis.SetCache(c, key, data)
// 	if err != nil {
// 		log.Println("Failed caching new patient: ", err)
// 	}
// 	if err := CreateLoginRecord(c, collection, code, data["email"].(string), data["phoneNo"].(string), data["password"].(string)); err != nil {
// 		log.Println("Error from the createLoginRecord", err)
// 		return val, err
// 	}
// 	subject := "Your Patient OTP Verification"
// 	body := fmt.Sprintf("Hello %s,\n\nYour OTP for patient verification is: %s\n\nThank you!", data["name"].(string), otp)

//		err = SendOTPToMail(data["email"].(string), subject, body)
//		if err != nil {
//			log.Println("OTP email failed:", err)
//			return val, errors.New("failed to send OTP email")
//		}
//		log.Println("mail sent successfully")
//		return "created successfully", nil
//	}
func CreatePatient(c *gin.Context, data map[string]interface{}) (string, error) {
	val := ""
	err := ValidateUserInput(data)
	if err != nil {
		log.Println("Error from ValidateUserInput:", err)
		return val, err
	}

	collection, err := FetchCollectionFromRoleDoc(c, data["roleCode"].(string))
	if err != nil {
		log.Println("Error from fetchRoleDocAndCollection:", err)
		return val, err
	}
	code, createdBy, err := CheckerAndGenerateUserCodes(c, collection, data["email"].(string), data["phoneNo"].(string))
	if err != nil {
		log.Println("Error from GenerateUserRole", err)
		return val, err
	}
	log.Println(code)
	otp, err := GenerateAndHashOTP(data)
	if err != nil {
		log.Println("Error from GeneraeAndHashOTP:", err)
		return val, err
	}
	log.Println(otp)
	err = trimIfExists(data, "gender")
	if err != nil {
		log.Println("Error from trimIfExists", err)
		return val, err
	}
	err = trimIfExists(data, "admissionDate")
	if err != nil {
		log.Println("Error from trimIfExists")
		return val, err
	}
	tenantId, err := GetTenantIdFromContext(c)
	if err != nil {
		log.Println("Error from getTenantIdFromToken", err)
		return val, err
	}
	log.Println("tenantId from context: ", tenantId)
	if err = PrepareUser(data, code, createdBy, tenantId); err != nil {
		log.Println("Error from prepareUser :", err)
		return val, err
	}
	age, err := CalculateAge(data["dob"].(string))
	if err != nil {
		log.Println("Error from CalculateAge")
		return val, err
	}

	data["age"] = age

	var guardianConsent []interface{}
	var consentId string

	if age < 18 {

		if err := ValidateGuardianConsent(data, age); err != nil {
			return "", err
		}

		// Generate consentId
		consentId, _ = GenerateEmpCode("CONSENT")
		data["consentId"] = consentId

		raw := data["consent"]
		guardianConsent, _ = raw.([]interface{})

		// Remove from patient before saving
		delete(data, "consent")
	}

	data["age"] = age

	data["age"] = age
	key := util.PatientKey + code
	err = redis.SetCache(c, key, data)
	if err != nil {
		log.Println("Failed caching new patient: ", err)
	}
	if _, err := SaveUserToDB(collection, data); err != nil {
		log.Println("Error from the saveUserToDB:", err)
		return val, err
	}
	if err := CreateLoginRecord(c, collection, code, data["email"].(string), data["phoneNo"].(string), data["password"].(string)); err != nil {
		log.Println("Error from the createLoginRecord", err)
		return val, err
	}
	if age < 18 && consentId != "" {
		consentRecord := bson.M{
			"consentId": consentId,
			"patientId": code,
			"guardians": guardianConsent,
			"version":   1,
			"createdAt": time.Now(),
		}
		if _, err := SaveUserToDB("CONSENT", consentRecord); err != nil {
			log.Println("Error saving consent:", err)
			return val, err
		}
	}

	subject := "Your Patient OTP Verification"
	body := fmt.Sprintf("Hello %s,\n\nYour OTP for patient verification is: %s\n\nThank you!", data["name"].(string), otp)

	err = SendOTPToMail(data["email"].(string), subject, body)
	if err != nil {
		log.Println("OTP email failed:", err)
		return val, errors.New("failed to send OTP email")
	}
	log.Println("mail sent successfully")
	return "created successfully", nil
}

/*
Here the Validation of the patient will happenn(CONSENT) Validation is done here
*/
func ValidateGuardianConsent(data map[string]interface{}, age int) error {
	raw, ok := data["consent"]
	if !ok {
		return errors.New("minor patient requires at least one guardian")
	}

	consent, ok := raw.([]interface{})
	if !ok || len(consent) == 0 {
		return errors.New("minor patient requires at least one guardian consent")
	}

	if len(consent) > 2 {
		return errors.New("only up to two guardians are allowed")
	}

	for i, g := range consent {
		guardian, ok := g.(map[string]interface{})
		if !ok {
			return fmt.Errorf("guardian %d is invalid", i+1)
		}

		required := []string{"name", "phoneNo", "govId", "relation", "signature"}
		for _, field := range required {
			val, exists := guardian[field]
			if !exists || val == "" {
				return fmt.Errorf("guardian %d missing field: %s", i+1, field)
			}
		}
	}

	return nil
}

func FetchPatientByCode(c *gin.Context, patientId string) (map[string]interface{}, error) {

	key := util.PatientKey + patientId

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
	coll := db.OpenCollections(patientCollection)
	filter := bson.M{"code": patientId}
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

func UpdatePatientByCode(c *gin.Context, patientId string, data map[string]interface{}) (string, error) {
	val := ""
	receptionistId, err := GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return val, err
	}
	fields := []string{"name", "email", "phoneNo", "dob", "admissionDate", "gender"}
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
	if admissionDateVal, ok := data["admissionDate"]; ok {
		if dateStr, ok := admissionDateVal.(string); ok {
			updatedAdmissionDate, err := NormalizeDate(dateStr)
			if err != nil {
				log.Println("Error from NormalizeDate:", err)
				return val, err
			}
			data["admissionDate"] = updatedAdmissionDate
		}
	}
	coll := patientCollection
	collection := db.OpenCollections(coll)

	filter := bson.M{
		"code": patientId,
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
	key := util.PatientKey + patientId
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old patient cache:", err)
	}

	if err := redis.SetCache(c, key, result); err != nil {
		log.Println("Failed caching updated patient:", err)
	}
	return "Updated Successfully", nil
}

func FetchAllPatients(c *gin.Context) ([]interface{}, error) {
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
	coll := patientCollection
	collection := db.OpenCollections(coll)
	patients, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from findall:", err)
		return nil, err
	}
	log.Println("Patients: ", patients)
	return patients, nil
}

func DeletePatient(c *gin.Context, patientId string) (string, error) {
	receptionistId, err := GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return "", err
	}
	filter := bson.M{
		"code":      patientId,
		"createdBy": receptionistId,
	}
	coll := patientCollection
	collection := db.OpenCollections(coll)
	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne function", err)
		return "", err
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
	key := util.PatientKey + patientId
	redis.DeleteCache(c, key)
	return "Deleted successfully", nil
}
