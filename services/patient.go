package services

import (
	"errors"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

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
		log.Println("Error from trimIfExists")
		return val, err
	}
	if err = PrepareUser(data, code, createdBy); err != nil {
		log.Println("Error from prepareUser :", err)
		return val, err
	}
	age, err := CalculateAge(data["dob"].(string))
	if err != nil {
		log.Println("Error from CalculateAge")
		return val, err
	}
	data["age"] = age
	if err := CacheUserInRedis(c, code, data, collection); err != nil {
		log.Println("Error from CacheUserInRedis: ", err)
		return val, err
	}
	if _, err := SaveUserToDB(collection, data); err != nil {
		log.Println("Error from the saveUserToDB:", err)
		return val, err
	}
	if err := CreateLoginRecord(c, collection, code, data["email"].(string), data["phoneNo"].(string), data["password"].(string)); err != nil {
		log.Println("Error from the createLoginRecord", err)
		return val, err
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
