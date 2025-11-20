package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

/*
CreateTenant handles creating a Tenant user.
It validates email/phone, generates employee code, fetches roleCode,
prepares the data, and inserts the record into MongoDB.
*/
func CreateTenant(c *gin.Context, data map[string]interface{}) error {

	if err := ValidateUserInput(data); err != nil {
		log.Println("Error from validateUserInput:", err)
		return err
	}
	collection, err := FetchCollectionFromRoleDoc(c, data["roleCode"].(string))
	if err != nil {
		log.Println("Error from fetchRoleDocAndCollection:", err)
		return err
	}
	code, CreatedBy, err := CheckerAndGenerateUserCodes(c, collection, data["email"].(string), data["phoneNo"].(string))
	if err != nil {
		log.Println("Error from GenerateUserRole", err)
		return err
	}
	otp, err := GenerateAndHashOTP(data)
	if err != nil {
		log.Println("Error from GenerateAndHashOTP", err)
		return err
	}
	log.Println("otp:", otp)

	if err := PrepareUser(data, code, CreatedBy); err != nil {
		log.Println("Error from PrepareUser", err)
		return err
	}
	if err := CacheUserInRedis(c, code, data, collection); err != nil {
		log.Println("Error from the CacheUserInRedis", err)
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

	subject := "Your Tenant OTP Verification"
	body := fmt.Sprintf("Hello %s,\n\nYour OTP for Tenant verification is: %s\n\nThank you!", data["name"].(string), otp)

	err = SendOTPToMail(data["email"].(string), subject, body)
	if err != nil {
		log.Println("OTP email failed:", err)
		return errors.New("failed to send OTP email")
	}
	log.Println("mail sent successfully")
	return nil
}

/*
It returns an array of documnets
where it matches with the filter given with it and perform
the Find all Function
*/
func FetchAllTenants(c *gin.Context) ([]interface{}, error) {
	collection := db.OpenCollections("TENANT")
	results, err := db.FindAll(c, collection, nil, nil)
	if err != nil {
		return []interface{}{}, err
	}
	log.Println("tenats are", results)
	return results, nil
}

/*
UpdateTenantByCode updates an existing Tenant document using its unique tenant code.

Workflow:
1. Validate tenant code input
2. Check if tenant exists
3. Parse updateData and prepare update fields
4. Normalize DOB if provided
5. Apply updates to MongoDB
6. Refresh cache (delete old → write new)
7. Return updated tenant document
*/
func UpdateTenantByCode(c *gin.Context, code string, updateData map[string]interface{}) (map[string]interface{}, error) {

	if strings.TrimSpace(code) == "" {
		return nil, errors.New("tenant code required")
	}

	_, err := fetchExistingTenant(code)
	if err != nil {
		return nil, err
	}

	updateFields, err := parseTenantUpdateFields(c, updateData)
	if err != nil {
		return nil, err
	}

	err = updateTenantInDB(code, updateFields)
	if err != nil {
		return nil, err
	}

	updatedTenant, err := fetchExistingTenant(code)
	if err != nil {
		return nil, err
	}

	refreshTenantCache(c, code, updatedTenant)

	return updatedTenant, nil
}

/*
fetchExistingTenant retrieves a tenant document by code from MongoDB.
Returns error if tenant not found.
*/
func fetchExistingTenant(code string) (map[string]interface{}, error) {

	collection := db.OpenCollections("TENANT")
	filter := bson.M{"code": code}

	var existing map[string]interface{}
	err := db.FindOne(context.Background(), collection, filter, &existing)
	if err != nil {
		return nil, errors.New("tenant not found")
	}

	return existing, nil
}

/*
parseTenantUpdateFields extracts and validates the update fields from updateData.
It also normalizes DOB and sets metadata like updatedAt and updatedBy.
*/
func parseTenantUpdateFields(c *gin.Context, updateData map[string]interface{}) (bson.M, error) {

	update := bson.M{}

	if v, ok := updateData["name"].(string); ok && strings.TrimSpace(v) != "" {
		update["name"] = v
	}

	if v, ok := updateData["email"].(string); ok && strings.TrimSpace(v) != "" {
		update["email"] = v
	}

	if v, ok := updateData["phoneNo"].(string); ok && strings.TrimSpace(v) != "" {
		update["phoneNo"] = v
	}

	if v, ok := updateData["dob"].(string); ok && strings.TrimSpace(v) != "" {
		modDob, err := NormalizeDOB(v)
		if err != nil {
			return nil, errors.New("invalid dob format")
		}
		update["dob"] = modDob
	}

	if len(update) == 0 {
		return nil, errors.New("no valid fields to update")
	}

	update["updatedAt"] = time.Now()
	update["updatedBy"] = c.GetString("code")

	return update, nil
}

/*
updateTenantInDB applies the parsed updates to the tenant document in MongoDB.
*/
func updateTenantInDB(code string, update bson.M) error {

	collection := db.OpenCollections("tenant")
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
func refreshTenantCache(c *gin.Context, code string, data map[string]interface{}) {

	key, err := redis.CreateCacheKey("tenant", code)
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

/*
It deletes the document which matches the code given in the
tenant where it used delete one function
*/
func DeleteTenantByCode(c *gin.Context, code string) error {
	key, err := redis.CreateCacheKey("tenant", code)
	if err != nil {
		log.Println("error from cache(create key) while creating role")
		return errors.New("Error from cache create Key")
	}
	if code == "" {
		return errors.New("tenant code required")
	}
	collection := db.OpenCollections("tenant")
	filter := bson.M{"code": code}
	delres, err := db.DeleteOne(c, collection, filter)
	if err != nil {
		return err
	}
	err = redis.DeleteCache(c, key)
	if err != nil {
		return err
	}
	log.Println(delres.DeletedCount)
	return nil
}
