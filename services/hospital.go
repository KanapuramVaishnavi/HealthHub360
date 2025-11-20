package services

import (
	"HealthHub360/config/db"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func CreateHospital(c *gin.Context, data map[string]interface{}) error {
	if err := ValidateUserInput(data); err != nil {
		log.Println("Error from ValidateUserInput", err)
		return err
	}
	collection, err := FetchCollectionFromRoleDoc(c, data["roleCode"].(string))
	if err != nil {
		log.Println("Error from FetchRoleDocAndCollection:", err)
		return err
	}
	code, createdBy, err := CheckerAndGenerateUserCodes(c, collection, data["email"].(string), data["phoneNo"].(string))
	if err != nil {
		log.Println("Error from GenerateUserCodes:", err)
		return err
	}
	otp, err := GenerateAndHashOTP(data)
	if err != nil {
		log.Println("Error from GeneraeAndHashOTP:", err)
		return err
	}
	if err = PrepareUser(data, code, createdBy); err != nil {
		log.Println("Error from prepareUser :", err)
		return err
	}
	if err := CacheUserInRedis(c, code, data, collection); err != nil {
		log.Println("Error from CacheUserInRedis: ", err)
		return err
	}
	if _, err := SaveUserToDB(collection, data); err != nil {
		log.Println("Error from the saveUserToDB:", err)
		return err
	}
	if err := CreateLoginRecord(c, collection, code, data["email"].(string), data["phoneNo"].(string), data["password"].(string)); err != nil {
		log.Println("Error from the createLoginRecord", err)
		return err
	}
	subject := "Your Hospital OTP Verification"
	body := fmt.Sprintf("Hello %s,\n\nYour OTP for hospital verification is: %s\n\nThank you!", data["name"].(string), otp)

	err = SendOTPToMail(data["email"].(string), subject, body)
	if err != nil {
		log.Println("OTP email failed:", err)
		return errors.New("failed to send OTP email")
	}
	log.Println("mail sent successfully")
	return nil
}
func BuildUpdateFilter(data map[string]interface{}) map[string]interface{} {
	filter := bson.M{}
	if v, ok := data["name"].(string); ok {
		filter["name"] = v
	}
	if v, ok := data["email"].(string); ok {
		filter["email"] = v
	}
	if v, ok := data["phoneNo"].(string); ok {
		filter["phoneNo"] = v
	}
	if v, ok := data["dob"].(string); ok {
		filter["dob"] = v
	}
	return filter
}
func UpdateHospital(c *gin.Context, data map[string]interface{}, code string) error {
	name, nameExists := data["name"].(string)
	if nameExists {
		err := getTrimmedString(data, name)
		if err != nil {
			log.Println("Error from getTrimmedString", err)
			return err
		}
	}
	email, emailExists := data["email"].(string)
	if emailExists {
		err := getTrimmedString(data, email)
		if err != nil {
			log.Println("Error from getTrimmedString", err)
			return err
		}
	}
	phoneNo, phoneExists := data["phoneNo"].(string)
	if phoneExists {
		err := getTrimmedString(data, phoneNo)
		if err != nil {
			log.Println("Error from getTrimmedString", err)
			return err
		}
	}
	dob, phoneExists := data["dob"].(string)
	if phoneExists {
		err := getTrimmedString(data, dob)
		if err != nil {
			log.Println("Error from getTrimmedString", err)
			return err
		}
	}
	_, err := NormalizeDOB(data["dob"].(string))
	if err != nil {
		log.Println("Error from dobNormalize", err)
		return err
	}
	data["dob"] = dob
	update := BuildUpdateFilter(data)
	createdBy, ok := c.Get("code")
	if !ok {
		return errors.New("unable to fetch code from context")
	}
	collection := db.OpenCollections("HOSPITAL")
	filter := bson.M{
		"code":      code,
		"createdBy": createdBy,
		"updatedBy": createdBy,
		"updatedAt": time.Now(),
	}
	res, err := db.UpdateOne(c, collection, filter, update)
	if err != nil {
		log.Println("Error from updateOne:", err)
		return err
	}
	log.Println(res.UpsertedCount)
	return nil
}
