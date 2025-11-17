package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"HealthHub360/util"
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/smtp"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

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

/*
Changes the DOB of any form into A Single DOB form
and make it parse and format into our style of DOB
checkes it its done return error if it is any invalid format
*/
func NormalizeDOB(dobStr string) (string, error) {
	formats := []string{
		"2006-01-02",
		"02-01-2006",
		"02/01/2006",
		"2006/01/02",
	}

	dobStr = strings.TrimSpace(dobStr)

	var dob time.Time
	var err error

	for _, format := range formats {
		dob, err = time.Parse(format, dobStr)
		if err == nil {
			return dob.Format("2006-01-02"), nil
		}
	}
	return "", errors.New("invalid DOB format")
}

// /*
//   - UserFetch
//     */
func UserFetch(ctx *gin.Context) (interface{}, error) {
	//interface
	code, exists := ctx.Get("code")
	if !exists {
		log.Println("Error while fetching from context")
		return nil, errors.New(util.ERROR_WHILE_FETCH_FROM_CONTEXT)
	}
	//convert to string
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

	var user bson.M
	collection := db.OpenCollections(collectionStr)
	filter := bson.M{"user_id": codeStr}

	err := db.FindOne(ctx, collection, filter, user)
	if err != nil {
		log.Println("Error while finding a document")
		return nil, errors.New(util.ERR_NO_DOC_FOUND)
	}
	return user, nil
}

/*
* Fetch user by code
* Fetch from cache either exist return true nor false
* If true return user ,if not go to db
* Fetch from db that return error
* If no doc found nor fetching error return error ,if not bind with the varibale
* Set the value into the cache and then return the bind with the variable
 */
func FetchUserByCode(ctx *gin.Context, code string, collectionStr string) (interface{}, error) {

	collection := db.OpenCollections(collectionStr)

	user := make(map[string]interface{})
	filter := bson.M{"code": code}
	exists, err := redis.GetCache(ctx, code, user)
	if err != nil {
		return nil, err
	}
	if !exists {
		err := db.FindOne(ctx, collection, filter, user)
		if err != nil {
			log.Println("Database fetch failed:", err)
			return nil, err

		}
		err = redis.SetCache(ctx, code, user)
		if err != nil {
			log.Println("Failed to set cache:", err)
			return nil, err
		}
	}
	return user, nil
}

/*
* Generate a random otp upto 999999
 */
func GenerateOTP() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

/*
 */
func SendOTPToMail(to, subject, body string) error {
	from := os.Getenv("SMTP_FROM")
	username := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASSWORD")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")

	message := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=\"UTF-8\";\r\n\r\n%s",
		from, to, subject, body,
	))

	auth := smtp.PlainAuth("", username, password, smtpHost)

	return smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, message)
}

func IscodeExists(collName string, code string) (bool, error) {
	collection := db.OpenCollections(collName)
	filter := bson.M{"code": code}
	emailcount, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return false, err
	}
	return emailcount > 0, err
}

/*
Checker validates email and phone number formats.
It also checks the database to ensure both fields do not already exist.
Returns an error if any validation rule fails.
*/
func Checker(Email string, Phone string, role string, code string) error {
	if Phone == "" {
		return errors.New("Missing Phone Field")
	}
	if Email == "" {
		return errors.New("Missing Email Field")
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
	if code != "" {
		codeCount, codeError := IscodeExists(role, code)
		if codeError != nil {
			return phoneNumberError
		}
		if codeCount == true {
			log.Println("IsPhone Number Triggered")
			return errors.New(util.USER_EXISTING_PHONE)
		}
	}
	return nil
}
