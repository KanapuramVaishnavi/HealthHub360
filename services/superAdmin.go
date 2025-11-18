package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"HealthHub360/models"
	"HealthHub360/util"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

/*
* Insert into the loginRecord
 */
func CreateLoginRecord(ctx context.Context, role string, code string, email string, phone string, password string) error {

	loginCollection := db.OpenCollections("login")
	filter := bson.M{
		"$or": []bson.M{
			{"code": code},
			{"email": email},
			{"phoneNo": phone},
		},
	}

	var existing models.Login
	err := db.FindOne(ctx, loginCollection, filter, &existing)
	if err == nil {
		return fmt.Errorf("login already exists with same code, email or phone")
	}

	if err.Error() == util.ERR_NO_DOC_FOUND {
		login := models.Login{
			Code:       code,
			Collection: role,
			Email:      email,
			PhoneNo:    phone,
			Password:   password,
		}

		_, err = db.CreateOne(ctx, loginCollection, login)
		if err != nil {
			return fmt.Errorf("failed to create login record: %v", err)
		}
		return nil
	}

	return fmt.Errorf("findOne error: %v", err)
}

/*
PrepareSuperAdmin formats and validates SuperAdmin data.
Normalizes the DOB, sets default fields, and populates metadata like timestamps.
Used before inserting the record in the database.
*/
func PrepareSuperAdmin(input map[string]interface{}, name string, email string, code string, roleCode string) error {

	dob, ok := input["dob"].(string)
	if !ok {
		return errors.New("DOB not provided")
	}
	modifiedDob, err := NormalizeDOB(dob)
	if err != nil {
		return err
	}

	input["dob"] = modifiedDob
	input["_id"] = primitive.NewObjectID()
	input["code"] = code
	input["roleCode"] = roleCode
	input["token"] = ""
	input["reset"] = true
	input["isActive"] = true
	input["createdAt"] = time.Now()
	input["updatedAt"] = time.Now()

	return nil
}

/*
CreateSuperAdmin handles creating a SuperAdmin user.
It validates email/phone, generates employee code, fetches roleCode,
prepares the data, and inserts the record into MongoDB.
*/
func CreateSuperAdmin(c *gin.Context, input map[string]interface{}) error {

	name, _ := input["name"].(string)
	email, _ := input["email"].(string)
	phoneNo, _ := input["phoneNo"].(string)
	role := "superAdmin"
	err := Checker(email, phoneNo, role, "")
	if err != nil {
		return err
	}
	collection := db.OpenCollections("superAdmin")

	docs, err := db.FindAll(ctx, collection, bson.M{}, nil)
	if err != nil {
		return err
	}

	if len(docs) >= 1 {
		return errors.New("SuperAdmin already exists")
	}

	code, err := GenerateEmpCode(role)
	if err != nil {
		return err
	}

	roleCollection := db.OpenCollections("role")
	log.Println(roleCollection)
	roleDoc := make(map[string]interface{})
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
	if err := PrepareSuperAdmin(input, name, email, code, roleCode); err != nil {
		return err
	}
	otp := GenerateOTP()
	log.Println("otp:", otp)
	hashedOTP, hashErr := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if hashErr != nil {
		return fmt.Errorf("failed to hash OTP: %v", hashErr)
	}
	log.Println(string(hashedOTP))

	input["password"] = string(hashedOTP)
	superadmin := db.OpenCollections(role)
	res, err := db.CreateOne(context.Background(), superadmin, input)
	if err != nil {
		return err
	}
	log.Println(res.InsertedID)
	err = CreateLoginRecord(c, role, code, email, phoneNo, string(hashedOTP))
	if err != nil {
		log.Println("Error from the createLoginRecord")
		return err
	}

	key, err := redis.CreateCacheKey("SuperAdmin", code)
	if err != nil {
		log.Println("Error from the create Key cache in create superAdmin", err)
		return errors.New("error from create cache key")
	}

	err = redis.SetCache(c, key, input)
	subject := "Your SuperAdmin OTP Verification"
	body := fmt.Sprintf("Hello %s,\n\nYour OTP for SuperAdmin verification is: %s\n\nThank you!", name, otp)

	err = SendOTPToMail(email, subject, body)
	if err != nil {
		log.Println("OTP email failed:", err)
		return errors.New("failed to send OTP email")
	}
	log.Println("mail sent successfully")
	return nil
}
