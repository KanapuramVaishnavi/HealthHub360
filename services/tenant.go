package services

import (
	"HealthHub360/config/db"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*
PrepareTenant formats and validates Tenant data.
Normalizes the DOB, sets default fields, and populates metadata like timestamps.
Used before inserting the record in the database.
*/
func PrepareTenant(tenant map[string]interface{}, name string, email string, code string, roleCode string, CreatedBy string) error {

	dob, _ := tenant["dob"].(string)
	modifiedDob, err := NormalizeDOB(dob)
	if err != nil {
		return err
	}

	tenant["dob"] = modifiedDob
	tenant["_id"] = primitive.NewObjectID()
	tenant["code"] = code
	tenant["roleCode"] = roleCode
	tenant["CreatedBy"] = CreatedBy
	tenant["UpdatedBy"] = CreatedBy
	tenant["createdAt"] = time.Now()
	tenant["updatedAt"] = time.Now()

	return nil
}

/*
fetchRoleCode returns the roleCode for a given roleName (TENANT, SUPERADMIN, ,,.etc)
*/
func fetchRoleCode(roleName string) (string, error) {

	roleCollection := db.OpenCollections("role")
	roleDoc := bson.M{}

	err := db.FindOne(
		context.Background(),
		roleCollection,
		bson.M{"roleName": roleName},
		&roleDoc)

	if err != nil {
		return "", fmt.Errorf("failed to find role %s: %v", roleName, err)
	}

	roleCode, ok := roleDoc["roleCode"].(string)
	if !ok {
		return "", errors.New("invalid roleCode type in role collection")
	}

	return roleCode, nil
}

/*
CreateTenant handles creating a Tenant user.
It validates email/phone, generates employee code, fetches roleCode,
prepares the data, and inserts the record into MongoDB.
*/
func CreateTenant(c *gin.Context, tenant map[string]interface{}) error {
	name, nameErr := tenant["name"].(string)
	Email, _ := tenant["email"].(string)
	Phone, _ := tenant["phoneNo"].(string)
	userCode, codeErr := c.Get("code")
	if !codeErr {
		return errors.New("Invalid code")
	}
	CreatedBy := userCode.(string)
	if !nameErr {
		return errors.New("Provide the Name")
	}
	role := "tenant"
	err := Checker(Email, Phone, role, CreatedBy)
	if err != nil {
		return err
	}
	code, err := GenerateEmpCode(role)
	if err != nil {
		return err
	}
	roleName := "TENANT"
	roleCode, err := fetchRoleCode(roleName)
	if err != nil {
		return err
	}
	if err := PrepareTenant(tenant, name, Email, code, roleCode, CreatedBy); err != nil {
		return err
	}
	collection := db.OpenCollections(role)
	res, err := db.CreateOne(context.Background(), collection, tenant)
	if err != nil {
		return err
	}
	log.Println(res.InsertedID)
	return nil
}
