package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/jwt"
	"HealthHub360/config/redis"
	"HealthHub360/util"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

/*
* Check is the emailExists,phoneExists,codeExists or not
* If non of these three exists then throw error
* If any of the field provided and the value is empty or type assertion then throw error
 */
func validateLoginInput(data map[string]interface{}) error {
	password, passExists := data["password"]

	if !passExists || strings.TrimSpace(password.(string)) == "" {
		return errors.New(util.PASSWORD_NOT_PROVIDED)
	}

	_, emailExists := data["email"]
	_, phoneExists := data["phoneNo"]
	_, codeExists := data["code"]

	if !emailExists && !phoneExists && !codeExists {
		return errors.New(util.PLEASE_PROVIDE_EMAIL_OR_PHONE_OR_CODE)
	}

	if emailExists {
		if v, ok := data["email"].(string); !ok || strings.TrimSpace(v) == "" {
			return errors.New(util.EMAIL_NOT_PROVIDED)
		}
	}

	if phoneExists {
		if v, ok := data["phoneNo"].(string); !ok || strings.TrimSpace(v) == "" {
			return errors.New(util.PHONE_NUMBER_NOT_PROVIDED)
		}
	}

	if codeExists {
		if v, ok := data["code"].(string); !ok || strings.TrimSpace(v) == "" {
			return errors.New(util.CODE_NOT_PROVIDED)
		}
	}

	return nil
}

/*
* Create Filter to find the document in db
 */
func buildSuperAdminFilter(data map[string]interface{}) bson.M {
	filter := bson.M{}

	if v, ok := data["email"].(string); ok && v != "" {
		filter["email"] = v
	}
	if v, ok := data["phoneNo"].(string); ok && v != "" {
		filter["phoneNo"] = v
	}
	if v, ok := data["code"].(string); ok && v != "" {
		filter["code"] = v
	}

	return filter
}

/*
* Pass the fiter and find which document gets matches with the filter
 */
func FetchUser(ctx context.Context, filter bson.M) (map[string]interface{}, error) {
	collection := db.OpenCollections("login")
	result := make(map[string]interface{})

	err := db.FindOne(ctx, collection, filter, &result)
	if err != nil {
		return nil, errors.New("user not found in login collection")
	}

	return result, nil
}

/*
* Fetch user based on the collection and code
* Using findOne search for it
 */
func FetchUserByRole(ctx context.Context, collectionName string, code string) (map[string]interface{}, error) {
	collection := db.OpenCollections(collectionName)
	result := make(map[string]interface{})

	err := db.FindOne(ctx, collection, bson.M{"code": code}, &result)
	if err != nil {
		return nil, fmt.Errorf("no user found in %s collection", collectionName)
	}
	return result, nil
}

/*
* If match found then compare the input password and then the password found from the filtered document
 */
func verifyPassword(dbPassword string, inputPassword string) error {
	if strings.TrimSpace(dbPassword) == "" {
		return errors.New("stored password missing or invalid")
	}

	err := bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(inputPassword))
	if err != nil {
		return errors.New("Password mismatch")
	}

	return nil
}

/*
* If login fails then set in cache the key count
* Increment the count for the key
* Set key value for 10mins
 */

func IncrementLoginAttempts(code string) (int, error) {
	key := "LOGIN_FAIL:" + code

	attempts, err := redis.Rdb.Incr(context.Background(), key).Result()
	if err != nil {
		log.Println("Unable to increment the key login count")
		return 0, err
	}

	if attempts == 1 {
		redis.Rdb.Expire(context.Background(), key, 10*time.Minute)
	}

	return int(attempts), nil
}

/*
* Pass the token
* And update the document with the token generated
 */
func UpdateUserToken(ctx context.Context, collectionName string, code string, token string) error {
	collection := db.OpenCollections(collectionName)

	filter := bson.M{"code": code}
	update := bson.M{"$set": bson.M{"token": token}}

	_, err := db.UpdateOne(ctx, collection, filter, update)
	return err
}

/*
* Validate super admin inputs first
* Build the filter to find the document
* Fetch superAdmin
* Verify Password
* GenerateJWT
* UpdateToken
 */
func Login(c *gin.Context, data map[string]interface{}) (string, error) {

	if err := validateLoginInput(data); err != nil {
		log.Println("error from validation input for the login")
		return "", err
	}

	filter := buildSuperAdminFilter(data)

	loginDoc, err := FetchUser(context.Background(), filter)
	if err != nil {
		log.Println("error from the fetchUser function:", err)
		return "", err
	}
	inputPassword := data["password"].(string)
	dbPassword := loginDoc["password"].(string)
	fmt.Println("DB Password:", dbPassword)
	fmt.Println("Input Password:", inputPassword)
	collection := loginDoc["collection"].(string)
	code := loginDoc["code"].(string)
	email := loginDoc["email"].(string)

	passErr := verifyPassword(dbPassword, inputPassword)
	if passErr != nil {
		attempts, _ := IncrementLoginAttempts(code)
		if attempts >= 3 {
			// Disable account in MongoDB
			_, _ = db.UpdateOne(context.Background(),
				db.OpenCollections(collection),
				bson.M{"code": code},
				bson.M{"$set": bson.M{"isActive": false}},
			)
			log.Println("Error while updating the collection for isActive field")
			return "", errors.New("account disabled due to 3 invalid attempts")
		}
		log.Println("Error from IncrementLoginattempts")
		return "", errors.New("invalid password")
	}

	userDoc, err := FetchUserByRole(c, collection, code)
	if err != nil {
		log.Println("Error from FetchUserByRole", err)
		return "", err
	}

	roleCode := userDoc["roleCode"].(string)
	token, err := jwt.GenerateJWT(code, email, roleCode, collection)
	if err != nil {
		log.Println("Error while generating the token")
		return "", err
	}

	if err := UpdateUserToken(c, collection, code, token); err != nil {
		log.Println("Error while updating the collection with token field")
		return "", err
	}
	return "login successful", nil
}
