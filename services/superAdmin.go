package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"HealthHub360/util"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

/*
CreateSuperAdmin handles creating a SuperAdmin user.
It validates email/phone, generates employee code, fetches roleCode,
prepares the data, and inserts the record into MongoDB.
*/
func CreateSuperAdmin(c *gin.Context, input map[string]interface{}) error {

	err := ValidateUserInput(input)
	if err != nil {
		log.Println("Error from ValidateUserInput:", err)
		return err
	}

	collection, err := FetchCollectionFromRoleDoc(c, input["roleCode"].(string))
	if err != nil {
		log.Println("Error from FetchRoleDocAndCollection:", err)
		return err
	}
	code, createdBy, err := CheckerAndGenerateUserCodes(c, collection, input["email"].(string), input["phoneNo"].(string))
	if err != nil {
		log.Println("Error from GenerateUserCodes:", err)
		return err
	}

	otp, err := GenerateAndHashOTP(input)
	if err != nil {
		log.Println("Error from GeneraeAndHashOTP:", err)
		return err
	}
	tenantId := ""
	if err = PrepareUser(input, code, createdBy, tenantId); err != nil {
		log.Println("Error from prepareUser :", err)
		return err
	}
	key := util.SuperAdminKey + code
	err = redis.SetCache(c, key, input)
	if err != nil {
		log.Println("Error while caching new superAdmin: ", err)
	}
	if _, err := SaveUserToDB(collection, input); err != nil {
		log.Println("Error from the saveUserToDB:", err)
		return err
	}
	if err := CreateLoginRecord(c, collection, code, input["email"].(string), input["phoneNo"].(string), input["password"].(string)); err != nil {
		log.Println("Error from the createLoginRecord", err)
		return err
	}
	subject := "Your SuperAdmin OTP Verification"
	body := fmt.Sprintf("Hello %s,\n\nYour OTP for SuperAdmin verification is: %s\n\nThank you!", input["name"].(string), otp)

	err = SendOTPToMail(input["email"].(string), subject, body)
	if err != nil {
		log.Println("OTP email failed:", err)
		return errors.New("failed to send OTP email")
	}
	log.Println("mail sent successfully")
	return nil
}

/*
* Fetch superAdmin from cache using id ,if data exists
* If doesnot exists in cache,fetch from dataBase
 */
func FetchSuperAdminByCode(c *gin.Context, superAdminId string) (map[string]interface{}, error) {
	coll := db.OpenCollections(SuperAdminCollection)
	superAdmin := make(map[string]interface{})
	key := util.SuperAdminKey + superAdminId
	cached := make(map[string]interface{})
	exists, err := redis.GetCache(ctx, key, &cached)
	if err == nil && exists {
		log.Println("From cache: ", cached)
		return cached, nil
	}
	filter := bson.M{
		"code": superAdminId,
	}
	err = db.FindOne(c, coll, filter, &superAdmin)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return nil, err
	}
	return superAdmin, nil
}

/*
* Validate input fields
* Check for uniqueness of email and phoneNo
* Fetch superAdmin ,update superAdmin
* Delete and set in Cache
 */
func UpdateSuperAdmin(c *gin.Context, superAdminId string, data map[string]interface{}) error {
	err := ValidateUserInput(data)
	if err != nil {
		log.Println("Error from ValidateUserInput:", err)
		return err
	}

	if v, ok := data["dob"].(string); ok && strings.TrimSpace(v) != "" {
		modDob, err := NormalizeDate(v)
		if err != nil {
			return errors.New("invalid dob format")
		}
		data["dob"] = modDob
	}
	collection := db.OpenCollections(SuperAdminCollection)
	err = CheckForEmailAndPhoneNo(c, collection, data)
	if err != nil {
		log.Println("Error from checkForEmailAndPhoneNo: ", err)
		return err
	}
	filter := bson.M{
		"code": superAdminId,
	}
	result := make(map[string]interface{})
	err = db.FindOne(ctx, collection, filter, &result)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return err
	}
	update := bson.M{
		"$set": data,
	}
	updated, err := db.UpdateOne(c, collection, filter, update)
	if err != nil {
		log.Println("Error from updateOne: ", err)
		return err
	}
	log.Println("Updated: ", updated.ModifiedCount)
	updatedSuperAdmin := make(map[string]interface{})
	err = db.FindOne(ctx, collection, filter, &updatedSuperAdmin)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return err
	}
	key := util.SuperAdminKey + superAdminId
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old tenant cache:", err)
	}

	if err := redis.SetCache(c, key, updatedSuperAdmin); err != nil {
		log.Println("Failed caching updated tenant:", err)
	}
	return nil
}
