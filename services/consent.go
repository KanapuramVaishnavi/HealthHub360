package services

import (
	"errors"
	"log"
	"strconv"
	"time"

	db "github.com/KanapuramVaishnavi/Core/config/db"
	redis "github.com/KanapuramVaishnavi/Core/config/redis"
	common "github.com/KanapuramVaishnavi/Core/coreServices"
	util "github.com/KanapuramVaishnavi/Core/util"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func PrepareConsentData(data map[string]interface{}, code string, consentCode string) map[string]interface{} {
	data["code"] = consentCode
	data["createdBy"] = code
	data["updatedBy"] = code
	data["createdAt"] = time.Now()
	data["updatedAt"] = time.Now()
	return data
}
func CreateConsent(c *gin.Context, data map[string]interface{}, medicalRecordId string) (string, error) {
	code := c.GetString("code")
	ctx_collection := c.GetString("collection")
	collection := db.OpenCollections(util.ConsentCollection)

	medicalRecord, err := FetchMedicalRecordByCode(c, medicalRecordId)
	if err != nil {
		return "", err
	}

	patientId := medicalRecord["patientId"].(string)
	log.Println("patientId: ", patientId)
	patient, err := FetchPatientByCode(c, patientId)
	if err != nil {
		return "", err
	}

	age, _ := strconv.Atoi(patient["age"].(string))
	if age > 18 {

		data["isConsentVerified"] = true
	} else {
		collection := db.OpenCollections(util.ConsentVerificationCollection)
		filter := bson.M{
			"guardianId": code,
		}
		consentVerification := make(map[string]interface{})
		err := db.FindOne(c, collection, filter, consentVerification)
		if err != nil {
			log.Println("Error from findOne: ", err)
			return "", err
		}
		otpFromConsentVerification, ok := consentVerification["otp"].(string)
		if !ok {
			log.Println("Unable to fetch otp from the particular document")
			return "", errors.New(util.UNABLE_TO_FETCH_OTP_FROM_DOCUMENT)
		}
		fields := []string{"password", "patientId"}
		for _, field := range fields {
			err := common.GetTrimmedString(data, field)
			if err != nil {
				log.Println("Error from getTrimmedString: ", err)
				return "", err
			}
		}
		if otpFromConsentVerification != data["password"].(string) {
			log.Println("Incorrect password")
			return "", errors.New(util.INCORRECT_PASSWORD)
		}
		data["isConsentVerified"] = true

	}
	consentCode, err := common.GenerateEmpCode(util.ConsentCollection)
	if err != nil {
		log.Println("Error from generateEmpCode: ", err)
		return "", err
	}
	PrepareConsentData(data, code, consentCode)
	inserted, err := db.CreateOne(c, collection, data)
	if err != nil {
		log.Println("Error from createOne: ", err)
		return "", err
	}
	log.Println("inserted: ", inserted.InsertedID)
	key := util.ConsentKey + consentCode
	err = redis.SetCache(c, key, data)
	if err != nil {
		log.Println("Error from setCache: ", err)
	}
	medicalRecordNew := make(map[string]interface{})
	medicalRecordNew["consentId"] = consentCode
	err = UpdateMedicalRecordByNurse(c, medicalRecordId, medicalRecordNew)
	if err != nil {
		log.Println("Error from updateMedicalRecordByNurse: ", err)
		return "", err
	}
	return "created successfully", nil
}
