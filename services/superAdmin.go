package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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
	if err = PrepareUser(input, code, createdBy); err != nil {
		log.Println("Error from prepareUser :", err)
		return err
	}
	if err := CacheUserInRedis(c, code, input, collection); err != nil {
		log.Println("Error from CacheUserInRedis: ", err)
		return err
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
func ReadSuperAdmin(c *gin.Context) ([]interface{}, error) {
	coll := db.OpenCollections(superAdminCollection)
	data, err := db.FindAll(c, coll, bson.M{}, nil)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func UpdateSuperAdmin(c *gin.Context, update map[string]interface{}) error {
	err := ValidateUserInput(update)
	if err != nil {
		log.Println("Error from ValidateUserInput:", err)
		return err
	}
	updateFields, err := parseTenantUpdateFields(c, update)
	if err != nil {
		return err
	}
	doc, err := ReteriveDoc(c)
	if err != nil {
		return err
	}
	code := doc["code"].(string)
	err = updateSuperAdminInDB(code, updateFields)
	if err != nil {
		return err
	}
	updatedDoc, err := ReteriveDoc(c)
	if err != nil {
		return err
	}
	refreshCache(c, superAdminCollection, code, updatedDoc)
	return nil
}
func ReteriveDoc(c *gin.Context) (map[string]interface{}, error) {
	docs, err := ReadSuperAdmin(c)
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, errors.New("no superadmin found")
	}
	doc, ok := docs[0].(map[string]interface{})
	if !ok {
		return nil, errors.New("DOCUMNET NIOT FOUND")
	}
	return doc, nil
}
func DeleteSuperAdmin(c *gin.Context) error {

	raw := os.Getenv("COLLECTIONS")
	parts := strings.Split(raw, ",")

	for _, name := range parts {
		name = strings.TrimSpace(name)

		collection := db.OpenCollections(name)

		res, err := db.DeleteMany(context.Background(), collection, bson.M{})
		if err != nil {
			log.Printf("Delete failed for collection %s: %v", name, err)
			return err
		}

		log.Printf("Deleted %d docs from %s", res.DeletedCount, name)
	}

	return nil
}

/*
updateSuperAdminInDB applies the parsed updates to the SuperAdmin document in MongoDB.
*/
func updateSuperAdminInDB(code string, update bson.M) error {

	collection := db.OpenCollections(superAdminCollection)
	filter := bson.M{"code": code}

	_, err := db.UpdateOne(context.Background(), collection, filter, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("update failed: %v", err)
	}

	return nil
}

/*
refreshTenantCache removes any old cache entry and stores the updated tenant data in Redis.
Cache failures are logged but not returned as errors (non-blocking).
*/
func refreshCache(c *gin.Context, collection string, code string, data map[string]interface{}) {

	key, err := redis.CreateCacheKey(collection, code)
	if err != nil {
		log.Println("Failed creating tenant cache key:", err)
		return
	}

	// Delete old cache entry
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old tenant cache:", err)
	}

	// Set new cache entry
	if err := redis.SetCache(c, key, data); err != nil {
		log.Println("Failed caching updated tenant:", err)
	}
}
