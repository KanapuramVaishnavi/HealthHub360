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
	input["loginAttempts"] = 0
	input["token"] = ""
	input["reset"] = true
	input["isBlocked"] = false
	input["isActive"] = false
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

	err := getTrimmedString(input, "name")
	if err != nil {
		log.Println("error from getTrimmed string:", err)
		return errors.New(util.NAME_NOT_PROVIDED)
	}
	err = getTrimmedString(input, "email")
	if err != nil {
		log.Println("error from getTrimmed string:", err)
		return errors.New(util.EMAIL_NOT_PROVIDED)
	}
	err = getTrimmedString(input, "phoneNo")
	if err != nil {
		log.Println("error from getTrimmed string:", err)
		return errors.New(util.PHONE_NUMBER_NOT_PROVIDED)
	}
	err = getTrimmedString(input, "dob")
	if err != nil {
		log.Println("error from getTrimmed string:", err)
		return errors.New(util.DOB_NOT_PROVIDED)
	}
	err = getTrimmedString(input, "roleCode")
	if err != nil {
		log.Println("error from getTrimmed string:", err)
		return errors.New(util.ROLE_CODE_KEY_NOT_FOUND)
	}
	roleDoc, err := FetchRoleById(c, input["roleCode"].(string))
	if err != nil {
		log.Println("Error from FetchRoleByID", err)
		return err
	}
	collection := roleDoc["roleName"].(string)
	name := input["name"].(string)
	email := input["email"].(string)
	phoneNo := input["phoneNo"].(string)
	roleCode := input["roleCode"].(string)
	err = Checker(email, phoneNo, collection)
	if err != nil {
		return err
	}
	superCollection := db.OpenCollections(collection)

	docs, err := db.FindAll(ctx, superCollection, bson.M{}, nil)
	if err != nil {
		return err
	}

	if len(docs) >= 1 {
		return errors.New("SuperAdmin already exists")
	}

	code, err := GenerateEmpCode(collection)
	if err != nil {
		return err
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
	expiry := time.Now().Add(10 * time.Minute)
	input["otpExpiry"] = expiry
	res, err := db.CreateOne(context.Background(), superCollection, input)
	if err != nil {
		return err
	}
	log.Println(res.InsertedID)
	err = CreateLoginRecord(c, collection, code, email, phoneNo, string(hashedOTP))
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
