package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/jwt"
	"HealthHub360/util"
	"context"
	"errors"
	"fmt"
	"log"
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
	input["createdAt"] = time.Now()
	input["updatedAt"] = time.Now()

	return nil
}

/*
CreateSuperAdmin handles creating a SuperAdmin user.
It validates email/phone, generates employee code, fetches roleCode,
prepares the data, and inserts the record into MongoDB.
*/
func CreateSuperAdmin(input map[string]interface{}) error {
	name, _ := input["name"].(string)
	Email, _ := input["email"].(string)
	Phone, _ := input["phoneNo"].(string)
	role := "superAdmin"
	err := Checker(Email, Phone, role, "")
	if err != nil {
		return err
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
	if err := PrepareSuperAdmin(input, name, Email, code, roleCode); err != nil {
		return err
	}
	otp := GenerateOTP()
	input["password"] = otp
	input["token"] = ""
	superadmin := db.OpenCollections(role)
	res, err := db.CreateOne(context.Background(), superadmin, input)
	if err != nil {
		return err
	}
	log.Println(res.InsertedID)

	subject := "Your SuperAdmin OTP Verification"
	body := fmt.Sprintf("Hello %s,\n\nYour OTP for SuperAdmin verification is: %s\n\nThank you!", name, otp)

	err = SendOTPToMail(Email, subject, body)
	if err != nil {
		log.Println("OTP email failed:", err)
		return errors.New("failed to send OTP email")
	}
	log.Println("mail sent successfully")
	return nil
}

/*
* Check is the emailExists,phoneExists,codeExists or not
* If non of these three exists then throw error
* If any of the field provided and the value is empty or type assertion then throw error
 */
func validateSuperAdminLoginInput(data map[string]interface{}) error {
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
func fetchSuperAdmin(ctx context.Context, filter bson.M) (map[string]interface{}, error) {
	collection := db.OpenCollections("superAdmin")
	result := make(map[string]interface{})

	err := db.FindOne(ctx, collection, filter, &result)
	if err != nil {
		log.Println("Error from the findOne")
		return nil, err
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

	if dbPassword != inputPassword {
		return errors.New("Password mismatch")
	}

	return nil
}

/*
* Pass the token
* And update the document with the token generated
 */
func updateSuperAdminToken(ctx context.Context, roleCode string, token string) error {
	collection := db.OpenCollections("superAdmin")

	filter := bson.M{"roleCode": roleCode}
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
func SuperAdminLogin(c *gin.Context, data map[string]interface{}) (string, error) {

	if err := validateSuperAdminLoginInput(data); err != nil {
		return "", err
	}

	filter := buildSuperAdminFilter(data)

	result, err := fetchSuperAdmin(context.Background(), filter)
	if err != nil {
		log.Println("error from the findOne function:", err)
		return "", err
	}

	dbPassword := result["password"].(string)
	inputPassword := data["password"].(string)

	if err := verifyPassword(dbPassword, inputPassword); err != nil {
		log.Println("Error from the verifyPassword")
		return "", err
	}

	resCode := result["code"].(string)
	resEmail := result["email"].(string)
	roleCode := result["roleCode"].(string)

	token, err := jwt.GenerateJWT(resCode, resEmail, roleCode, "superAdmin")
	if err != nil {
		log.Println("Unable to generate JWT")
		return "", err
	}

	if err := updateSuperAdminToken(context.Background(), roleCode, token); err != nil {
		log.Println("Error while Updating the token")
		return "", err
	}
	return "login successful", nil
}
