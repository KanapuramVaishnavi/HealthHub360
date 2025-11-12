package services

import (
	"HealthHub360/config"
	"HealthHub360/util"
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var tenantCollection *mongo.Collection = config.OpenCollections("tenant")
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

	switch collName {
	case "tenant", "tenants":
		prefix = "T"
	case "patient", "patients":
		prefix = "P"
	case "doctors", "doctor":
		prefix = "D"
	default:
		return "", fmt.Errorf("unsupported collection: %s", collName)
	}

	collection := config.OpenCollections(collName)
	// Find last document sorted by code descending
	opts := options.FindOne().SetSort(bson.D{{Key: "tenantID", Value: -1}})
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
	codeVal, ok := lastDoc["tenantID"].(string)
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

func IsPhoneNumberExists(phone string) (bool, error) {
	filter := bson.M{"phone": phone}
	count, err := tenantCollection.CountDocuments(context.Background(), filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

/*
Here It Verify Whether the Email is present in The Database.
*/
func IsEmailExists(email string) (bool, error) {
	filter := bson.M{"email": email}
	emailcount, err := tenantCollection.CountDocuments(context.Background(), filter)
	if err != nil {
		return false, err
	}
	return emailcount > 0, err
}

// /*
//   - UserFetch
//     */
func UserFetch(ctx *gin.Context) (interface{}, error) {
	//interface
	id, exists := ctx.Get("user_id")
	if !exists {
		log.Println("Error while fetching from context")
		return nil, errors.New(util.ERROR_WHILE_FETCH_FROM_CONTEXT)
	}
	//convert to string
	idStr, exist := id.(string)
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

	var user bson.M
	collection := config.OpenCollections(collectionStr)
	filter := bson.M{"user_id": idStr}

	err := config.FindOne(ctx, collection, filter, user)
	if err != nil {
		log.Println("Error while finding a document")
		return nil, errors.New(util.ERR_NO_DOC_FOUND)
	}
	return user, nil
}
