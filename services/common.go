package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"HealthHub360/util"
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var tenantCollection *mongo.Collection

func InitCommonCollections() {
	RoleCollection = db.OpenCollections("role")
}

var ctx context.Context = context.Background()

/*
Here It is Used for Generating EmpCode for
unique like tenants,patients like
based on the collection name we differ the prefix.
*/
func GenerateEmpCode(collName string) (string, error) {
	// Define prefix and number width for each collection
	var prefix string
	width := 4 // e.g. T0001 → 4 digits
	var sortField string = "roleCode"
	switch collName {
	case "tenant", "tenants":
		prefix = "T"
	case "patient", "patients":
		prefix = "P"
	case "doctors", "doctor":
		prefix = "D"
	case "superAdmin":
		prefix = "S"
	case "Role", "role":
		prefix = "R"
	default:
		return "", fmt.Errorf("unsupported collection: %s", collName)
	}

	collection := db.OpenCollections(collName)
	// Find last document sorted by code descending
	opts := options.FindOne().SetSort(bson.D{{Key: sortField, Value: -1}})
	var lastDoc bson.M

	err := collection.FindOne(ctx, bson.M{}, opts).Decode(&lastDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Start fresh
			return fmt.Sprintf("%s%0*d", prefix, width, 1), nil
		}
		return "", err
	}

	// Extract last code
	codeVal, ok := lastDoc[sortField].(string)
	if !ok || codeVal == "" {
		return fmt.Sprintf("%s%0*d", prefix, width, 1), nil
	}

	// Extract numeric part (e.g., T0005 → 5)
	re := regexp.MustCompile(`(\d+)$`)
	matches := re.FindStringSubmatch(codeVal)
	if len(matches) < 2 {
		return fmt.Sprintf("%s%0*d", prefix, width, 1), nil
	}

	lastNum, err := strconv.Atoi(matches[1])
	if err != nil {
		return fmt.Sprintf("%s%0*d", prefix, width, 1), nil
	}

	newNum := lastNum + 1
	newCode := fmt.Sprintf("%s%0*d", prefix, width, newNum)
	return newCode, nil
}

func IsPhoneNumberExists(collName string, phone string) (bool, error) {
	collection := db.OpenCollections(collName)
	filter := bson.M{"phone": phone}
	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

/*
Here It Verify Whether the Email is present in The Database.
*/
func IsEmailExists(collName string, email string) (bool, error) {
	collection := db.OpenCollections(collName)
	filter := bson.M{"email": email}
	emailcount, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return false, err
	}
	return emailcount > 0, err
}

/*
function for normalizing the Email
*/
func NormalizeEmail(email string) string {
	loweredEmail := strings.ToLower(email)
	trimmedEmail := strings.TrimSpace(loweredEmail)
	if strings.Contains(trimmedEmail, " ") {
		return ""
	}
	return trimmedEmail
}

/*
function for normalizing the phone Number
*/
func NormalizePhoneNumber(phone string) string {
	trimmedPhone := strings.TrimSpace(phone)
	if strings.Contains(trimmedPhone, " ") {
		return ""
	}
	return trimmedPhone
}

/*
Function For Phone Number Validation
*/
func IsPhoneNumberValid(phone string) bool {
	trimmedPhone := strings.TrimSpace(phone)
	rephone := strings.ReplaceAll(trimmedPhone, " ", "")
	re := regexp.MustCompile(`^(\+91)?[6-9]\d{9}$`)
	check := re.MatchString(rephone)
	return check
}

// /*
//   - UserFetch
//     */
// func UserFetch(ctx *gin.Context) (interface{}, error) {
// 	//interface
// 	id, exists := ctx.Get("user_id")
// 	if !exists {
// 		log.Println("Error while fetching from context")
// 		return nil, errors.New(util.ERROR_WHILE_FETCH_FROM_CONTEXT)
// 	}
// 	//convert to string
// 	idStr, exist := id.(string)
// 	if !exist {
// 		log.Println("Error while converting from mongo collection to string")
// 	}

// 	claimsCollection, exists := ctx.Get("collection")
// 	if !exists {
// 		log.Println("Error while fetching from context")
// 		return nil, errors.New(util.ERROR_WHILE_FETCH_FROM_CONTEXT)
// 	}

// 	collectionStr, exist := claimsCollection.(string)
// 	if !exist {
// 		log.Println("Error while converting from mongo collection to string")
// 	}

// 	var user bson.M
// 	collection := db.OpenCollections(collectionStr)
// 	filter := bson.M{"user_id": idStr}

// 	err := db.FindOne(ctx, collection, filter, user)
// 	if err != nil {
// 		log.Println("Error while finding a document")
// 		return nil, errors.New(util.ERR_NO_DOC_FOUND)
// 	}
// 	return user, nil
// }

/*
* Fetch user by code
* Fetch from cache either exist return true nor false
* If true return user ,if not go to db
* Fetch from db that return error
* If no doc found nor fetching error return error ,if not bind with the varibale
* Set the value into the cache and then return the bind with the variable
 */
func FetchUserByCode(ctx *gin.Context) (interface{}, error) {

	code, exists := ctx.Get("code")
	if !exists {
		log.Println("Error while fetching from context")
		return nil, errors.New(util.ERROR_WHILE_FETCH_FROM_CONTEXT)
	}
	codeStr, exist := code.(string)
	if !exist {
		log.Println("Error while converting from mongo collection to string")
	}

	claimsCollection, exists := ctx.Get("collection")
	if !exists {
		log.Println("Error while fetching from context")
		return nil, errors.New(util.ERROR_WHILE_FETCH_FROM_CONTEXT)
	}
	collectionStr, exist := claimsCollection.(string)
	if !exist {
		log.Println("Error while converting from mongo collection to string")
	}
	collection := db.OpenCollections(collectionStr)

	var user bson.M
	filter := bson.M{"code": codeStr}
	exists, err := redis.GetCache(ctx, codeStr, user)
	if err != nil {
		return nil, err
	}
	if !exists {
		err := db.FindOne(ctx, collection, filter, user)
		if err != nil {
			log.Println("Database fetch failed:", err)
			return nil, err

		}
		err = redis.SetCache(ctx, codeStr, user)
		if err != nil {
			log.Println("Failed to set cache:", err)
			return nil, err
		}
	}
	return user, nil
}
