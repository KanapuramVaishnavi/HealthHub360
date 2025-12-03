package services

import (
	"HealthHub360/config/db"
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

// /*
// * Check if previleges exists or not
// * Check if the module data is present or not
// * Check if the access length is more than 0 or not
// * Return false for the above conditions
// * only return true when previleges field is fine
//  */
// func CheckIfPrivilegesIsEmpty(c context.Context, previleges []map[string]interface{}) (bool, error) {
// 	for _, p := range previleges {

// 		val, exists := p["module"]
// 		if !exists {
// 			return false, errors.New(util.MODULE_NOT_PROVIDED)
// 		}

// 		module, ok := val.(string)
// 		if !ok || strings.TrimSpace(module) == "" {
// 			return false, errors.New(util.MODULE_NOT_PROVIDED)
// 		}

// 		val, exists = p["access"]
// 		if !exists {
// 			return false, errors.New(util.ACCESS_NOT_PROVIDED)
// 		}

// 		access, ok := val.([]string)
// 		if !ok || len(access) == 0 {
// 			return false, errors.New(util.ACCESS_NOT_PROVIDED)
// 		}
// 	}
// 	return true, nil
// }

/*
*  Check if the same module present in the array
 */
func CheckDuplicateModules(privileges []map[string]interface{}) error {
	moduleSet := make(map[string]bool)

	for _, p := range privileges {
		module, _ := p["module"].(string)
		moduleClean := strings.TrimSpace(module)

		if moduleClean == "" {
			continue
		}

		if moduleSet[moduleClean] {
			return fmt.Errorf("duplicate module found: %s", moduleClean)
		}

		moduleSet[moduleClean] = true
	}

	return nil
}

/*
* Take map[string]interface
* Do type assertion for each of them
* Do generate the roleCode
* Convert the []interface{} to the []map[string]interface{}
 */
func PrepareRoleData(c *gin.Context, roleData map[string]interface{}) (map[string]interface{}, error) {

	err := getTrimmedString(roleData, "roleName")
	if err != nil {
		log.Println("Error from getTrimmedString:", err)
		return nil, err
	}
	if v, ok := roleData["roleName"].(string); ok {
		roleData["roleName"] = v
	}
	roleName := strings.ToUpper(roleData["roleName"].(string))
	exists, err := CheckIfRoleNameExists(c, roleName)
	if !exists {
		log.Println("Error from the CheckIdRoleNameExists")
		return nil, err
	}
	collection := "role"
	roleCode, err := GenerateEmpCode(collection)
	if err != nil {
		log.Println("Error while generating code", err)
	}
	roleData["roleCode"] = roleCode
	v, ok := roleData["privileges"].([]interface{})
	if !ok {

		log.Println("Privileges field not found")
		return nil, errors.New("Privileges field not found")
	}
	privs := make([]map[string]interface{}, 0)

	for _, item := range v {
		if m, ok := item.(map[string]interface{}); ok {
			if moduleVal, exists := m["module"]; exists {
				m["module"] = module
			}
			if accessRaw, exists := m["access"].([]interface{}); exists {
				accessList := make([]string, 0)
				for _, a := range accessRaw {
					if s, ok := a.(string); ok {
						accessList = append(accessList, s)
					}
				}
				m["access"] = accessList
			}

			privs = append(privs, m)
		}
	}
	roleData["privileges"] = privs

	return roleData, nil
}

/*
* Check is RoleName is empty
* check if any document present in db with same RoleName
* Check if previlege is empty,Check if module is empty,Check if access length is 0
* Set remaining feilds
* CreateOne create the document in the respective db provided
 */
func CreateRole(c *gin.Context, data map[string]interface{}) (map[string]interface{}, error) {

	roleData, err := PrepareRoleData(c, data)
	if err != nil {
		log.Println("Error from prepareRoleData", err)
		return nil, err
	}
	privilegesRaw, ok := roleData["privileges"]
	if !ok {
		log.Println("privileges key missing in request")
		return nil, errors.New(util.PRIVILEGES_REQUIRED)
	}
	privileges, ok := privilegesRaw.([]map[string]interface{})
	if !ok || len(privileges) == 0 {
		log.Println("privileges are empty or invalid")
		return nil, errors.New(util.PRIVILEGES_DATA_REQUIRED)
	}

	// exists, err := CheckIfPrivilegesIsEmpty(c, privileges)
	// if err != nil {
	// 	log.Println("Error from checkIfPrivilegesIsEmpty", err)
	// 	return nil, err
	// }

	if err := CheckDuplicateModules(privileges); err != nil {
		return nil, err
	}

	collection := db.OpenCollections("role")
	roleData["CreatedBy"] = "SYSTEM"
	roleData["UpdatedBy"] = "SYSTEM"
	roleData["CreatedAt"] = time.Now()
	roleData["UpdatedAt"] = time.Now()

	_, err = db.CreateOne(ctx, collection, roleData)
	if err != nil {
		return nil, fmt.Errorf("failed to insert role: %v", err)
	}
	key := util.RoleKey + data["roleCode"].(string)
	err = redis.SetCache(c, key, roleData)
	if err != nil {
		return nil, fmt.Errorf("failed to insert role: %v", err)
	}

	return roleData, nil
}

/*
It returns the roles present in the database
*/
func ReadRoles(ctx *gin.Context) ([]interface{}, error) {
	var results []interface{}
	collection := db.OpenCollections("role")
	results, err := db.FindAll(ctx, collection, nil, nil)
	if err != nil {
		return []interface{}{}, err
	}
	return results, nil
}

/*
parsePrivileges converts []interface{} into []map[string]interface{}
and ensures access list is []string.
*/
func parsePrivileges(raw []interface{}) []map[string]interface{} {
	privs := make([]map[string]interface{}, 0)

	for _, item := range raw {
		if m, ok := item.(map[string]interface{}); ok {

			if module, ok := m["module"].(string); ok {
				m["module"] = module
			}

			if accessRaw, exists := m["access"].([]interface{}); exists {
				accessList := make([]string, 0)
				for _, a := range accessRaw {
					if str, ok := a.(string); ok {
						accessList = append(accessList, str)
					}
				}
				m["access"] = accessList
			}

			privs = append(privs, m)
		}
	}

	return privs
}

/*
parseUpdateFields extracts and normalizes update fields
before applying them to the DB.
*/
func parseUpdateFields(updateData map[string]interface{}) (bson.M, error) {

	update := bson.M{}

	if v, ok := updateData["roleName"].(string); ok && strings.TrimSpace(v) != "" {
		update["roleName"] = strings.ToUpper(v)
	}

	if rawPrivs, ok := updateData["privileges"].([]interface{}); ok {
		update["privileges"] = parsePrivileges(rawPrivs)
	}

	return update, nil
}

/*
updateRoleInDB applies update fields to an existing role document.
*/
func updateRoleInDB(roleCode string, update bson.M) error {
	collection := db.OpenCollections("role")
	_, err := db.UpdateOne(ctx, collection, bson.M{"roleCode": roleCode}, update)
	if err != nil {
		return fmt.Errorf("update failed: %v", err)
	}
	return nil
}

/*
UpdateRole handles updating an existing role:
- Validates updates
- Applies update to DB
- Refreshes Redis cache
*/
func UpdateRole(c *gin.Context, roleCode string, updateData map[string]interface{}) (map[string]interface{}, error) {

	if strings.TrimSpace(roleCode) == "" {
		return nil, errors.New("roleCode cannot be empty")
	}

	updateFields, err := parseUpdateFields(updateData)
	if err != nil {
		return nil, err
	}

	if len(updateFields) == 0 {
		return nil, errors.New("no valid fields to update")
	}

	updateFields["UpdatedAt"] = time.Now()
	updateFields["UpdatedBy"] = "SYSTEM"

	if err := updateRoleInDB(roleCode, bson.M{"$set": updateFields}); err != nil {
		return nil, err
	}

	collection := db.OpenCollections("role")
	var updated map[string]interface{}
	err = db.FindOne(c, collection, bson.M{"roleCode": roleCode}, &updated)
	if err != nil {
		return nil, err
	}
	key := util.TestKey + roleCode
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old tenant cache:", err)
	}

	if err := redis.SetCache(c, key, updated); err != nil {
		log.Println("Failed caching updated tenant:", err)
	}

	return updated, nil
}

/*
FetchRoleById retrieves a role by its roleCode.
Steps:
1. Check Redis cache
2. If missing, fetch from MongoDB
3. Repopulate cache
*/
func FetchRoleById(c *gin.Context, roleCode string) (map[string]interface{}, error) {

	if strings.TrimSpace(roleCode) == "" {
		return nil, errors.New("roleCode cannot be empty")
	}
	key := util.RoleKey + roleCode

	var cached map[string]interface{}
	found, err := redis.GetCache(c, key, &cached)
	if err == nil && found {
		return cached, nil
	}

	collection := db.OpenCollections("role")
	filter := bson.M{"roleCode": roleCode}
	role := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, role)
	if err != nil {
		return nil, errors.New("role not found")
	}

	return role, nil
}

/*
DeleteRole removes a role from DB and clears its cache entry.
*/
func DeleteRole(c *gin.Context, roleCode string) error {

	if strings.TrimSpace(roleCode) == "" {
		return errors.New("roleCode cannot be empty")
	}

	collection := db.OpenCollections("role")

	filter := bson.M{"roleCode": roleCode}

	var existing map[string]interface{}
	err := db.FindOne(c, collection, filter, &existing)
	if err != nil {
		return errors.New("role not found")
	}

	_, err = db.DeleteOne(c, collection, filter)
	if err != nil {
		return err
	}
	key := util.RoleKey + roleCode
	redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error while deleting role: ", err)
	}

	return nil
}
