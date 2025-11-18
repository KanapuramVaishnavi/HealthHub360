package services

import (
	"HealthHub360/config/db"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

/*
* Extract token info like code, and collection
 */
func ExtractTokenInfo(c *gin.Context) (collection string, code string, err error) {
	collectionVal, ok := c.Get("collection")
	if !ok {
		return "", "", errors.New("invalid token: collection missing")
	}
	log.Println("collection from token:", collectionVal)
	collection, ok = collectionVal.(string)
	if !ok || collection == "" {
		return "", "", errors.New("invalid token: collection invalid")
	}

	codeVal, ok := c.Get("code")
	if !ok {
		return "", "", errors.New("invalid token: code missing")
	}

	log.Println("code from token:", codeVal)
	code, ok = codeVal.(string)
	if !ok || code == "" {
		return "", "", errors.New("invalid token: code invalid")
	}

	return collection, code, nil
}

/*
* Validate input field
 */
func ValidatePasswordInput(body map[string]interface{}) (string, string, error) {
	newPasswordRaw, npExists := body["newPassword"]
	confirmPasswordRaw, cpExists := body["confirmPassword"]

	if !npExists || !cpExists {
		return "", "", errors.New("newPassword and confirmPassword are required")
	}

	newPassword, ok := newPasswordRaw.(string)
	if !ok || strings.TrimSpace(newPassword) == "" {
		return "", "", errors.New("invalid newPassword")
	}
	confirmPassword, ok := confirmPasswordRaw.(string)
	if !ok || strings.TrimSpace(confirmPassword) == "" {
		return "", "", errors.New("invalid confirmPassword")
	}

	if newPassword != confirmPassword {
		return "", "", errors.New("newPassword and confirmPassword do not match")
	}

	return newPassword, confirmPassword, nil
}

func UpdatePasswordInCollections(c *gin.Context, collectionName string, code string, hashedPassword string) error {
	filter := bson.M{"code": code}

	coll := db.OpenCollections(collectionName)
	update := bson.M{
		"$set": bson.M{
			"password":  hashedPassword,
			"reset":     false,
			"updatedAt": time.Now(),
		},
	}

	_, err := db.UpdateOne(c, coll, filter, update)
	if err != nil {
		log.Println("Error updating password in main collection:", err)
		return err
	}

	loginColl := db.OpenCollections("login")
	_, err = db.UpdateOne(c, loginColl, filter, bson.M{
		"$set": bson.M{"password": hashedPassword},
	})

	if err != nil {
		log.Println("Warning: failed to update login collection password:", err)
	}

	return nil
}

/*
* Check if length less than 7 return error
*
 */
func validatePasswordRules(password string) error {

	if len(password) < 7 {
		return errors.New("password must be at least 7 characters long")
	}

	// At least one uppercase
	hasUpper := false
	// At least one number
	hasNumber := false
	// At least one special character
	hasSpecial := false

	specialChars := "!@#$%^&*()-_=+[]{}|;:',.<>?/`~"

	for _, ch := range password {
		switch {
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= '0' && ch <= '9':
			hasNumber = true
		case strings.ContainsRune(specialChars, ch):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasNumber {
		return errors.New("password must contain at least one number")
	}
	if !hasSpecial {
		return errors.New("password must contain at least one special character")
	}
	if !(hasUpper && hasNumber && hasSpecial) {
		return errors.New("password must include: one uppercase, one number, and one special character")
	}

	return nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

/*
* Extract token and get code and collection
* Validate the input feilds given
* Validate set of rules for the password
* Hash the password
* Now update the password in the collection with the hashCode
* Update in login collection
 */
func ResetPassword(c *gin.Context, body map[string]interface{}) (string, error) {
	collection, code, err := ExtractTokenInfo(c)
	if err != nil {
		log.Println("Error from extractToken Info from resetPassword")
		return "", err
	}
	newPassword, _, err := ValidatePasswordInput(body)
	if err != nil {
		log.Println("Error from validateToken Info from resetPassword")
		return "", err
	}

	if err := validatePasswordRules(newPassword); err != nil {
		log.Println("error from validate password in validatePasswordRules")
		return "", err
	}

	hashedPassword, err := HashPassword(newPassword)
	if err != nil {
		log.Println("Error from hashedPassword")
		return "", errors.New("failed to hash new password")
	}
	log.Println(hashedPassword)

	if err := UpdatePasswordInCollections(c, collection, code, hashedPassword); err != nil {
		log.Println("Error from UpdatePasswordInCollection")
		return "", errors.New("failed to update password")
	}

	return "Password reset successful", nil
}
