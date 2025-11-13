package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/jwt"
	"HealthHub360/util"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateSuperAdmin(input map[string]interface{}) error {
	name, _ := input["name"].(string)
	Email, _ := input["email"].(string)
	Phone, _ := input["phoneNo"].(string)
	role, _ := input["role"].(string)
	if role == "" {
		role = "superAdmin"
	}

	if Phone == "" || Email == "" {
		return errors.New("missing fields")
	}
	email := NormalizeEmail(Email)
	if email == "" {
		return errors.New(util.EMAIL_NOT_VALID)
	}
	emailsCount, emailError := IsEmailExists(role, Email)
	if emailError != nil {
		return emailError
	}
	if emailsCount == true {
		log.Println("Email Exists triggered")
		return errors.New(util.USER_EXISTING_EMAIL)
	}
	modifiedPhoneNumber := NormalizePhoneNumber(Phone)
	if modifiedPhoneNumber == "" {
		return errors.New(util.PHONENUMBER_NOT_VALID)
	}
	Phone = modifiedPhoneNumber
	check := IsPhoneNumberValid(Phone)
	if check == false {
		return errors.New(util.PHONE_NUMBER_VALIDATION)
	}
	phoneNumbersCount, phoneNumberError := IsPhoneNumberExists(role, Phone)
	if phoneNumberError != nil {
		return phoneNumberError
	}
	if phoneNumbersCount == true {
		log.Println("IsPhone Number Triggered")
		return errors.New(util.USER_EXISTING_PHONE)
	}

	code, err := GenerateEmpCode("superAdmin")
	if err != nil {
		return err
	}

	roleCollection := db.OpenCollections("role")
	var roleDoc bson.M
	filter := bson.M{
		"roleName": "SUPERADMIN",
	}
	err = db.FindOne(context.Background(), roleCollection, filter, roleDoc)
	if err != nil {
		return fmt.Errorf("failed to find SuperAdmin role: %v", err)
	}

	roleCode, ok := roleDoc["roleCode"].(string)
	if !ok {
		return errors.New("invalid roleCode type in role collection")
	}

	input["id"] = primitive.NewObjectID()
	input["code"] = code
	input["roleCode"] = roleCode
	token, err := jwt.GenerateJWT(code, name, email, roleCode, "superAdmin")
	if err != nil {
		log.Println("error while generating superAdmin token", err)
	}
	input["token"] = token
	input["createdAt"] = time.Now()
	input["updatedAt"] = time.Now()
	superadmin := db.OpenCollections(role)
	res, err := db.CreateOne(context.Background(), superadmin, input)
	if err != nil {
		return err
	}
	log.Println(res.InsertedID)
	return nil
}
