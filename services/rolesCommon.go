package services

import (
	"HealthHub360/config/db"
	"HealthHub360/util"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
* Check if the collection consists of document with the filter
* If any document not fund nor db error throw error
* If not find the document and check with the field if already exists return false
* Return true only when the document not fund
 */
func CheckIfRoleNameExists(c context.Context, roleName string) (bool, error) {
	collectionStr := "role"
	collection := db.OpenCollections(collectionStr)
	filter := bson.M{
		"roleName": roleName,
	}
	result := make(map[string]interface{})

	err := db.FindOne(c, collection, filter, &result)
	if err != nil {
		if err == mongo.ErrNoDocuments || strings.Contains(err.Error(), "no matching document found") {
			return true, nil
		}
		return false, fmt.Errorf("database error: %v", err)
	}

	if value, exists := result["roleName"]; exists && value == roleName {
		log.Printf("Role name '%s' already exists\n", roleName)
		return false, errors.New(util.ROLE_NAME_ALREADY_EXISTS)
	}
	return true, nil
}

/*
* Check if previleges exists or not
* Check if the module data is present or not
* Check if the access length is more than 0 or not
* Return false for the above conditions
* only return true when previleges field is fine
 */
func CheckIfPrivilegesIsEmpty(c context.Context, previleges []map[string]interface{}) (bool, error) {
	for _, p := range previleges {

		val, exists := p["module"]
		if !exists {
			return false, errors.New(util.MODULE_NOT_PROVIDED)
		}

		module, ok := val.(string)
		if !ok || strings.TrimSpace(module) == "" {
			return false, errors.New(util.MODULE_NOT_PROVIDED)
		}

		val, exists = p["access"]
		if !exists {
			return false, errors.New(util.ACCESS_NOT_PROVIDED)
		}

		access, ok := val.([]string)
		if !ok || len(access) == 0 {
			return false, errors.New(util.ACCESS_NOT_PROVIDED)
		}
	}
	return true, nil
}
