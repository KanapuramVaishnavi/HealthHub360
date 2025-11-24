package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"HealthHub360/util"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

/*
* Take map[string]interface
* Do type assertion for each of them
* Do generate the roleCode
* Convert the []interface{} to the []map[string]interface{}
 */
func PrepareData(roleData map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	if v, ok := roleData["roleName"].(string); ok {
		result["roleName"] = v
	}
	collection := "role"
	roleCode, err := GenerateEmpCode(collection)
	if err != nil {
		log.Println("Error while generating code", err)
	}
	result["roleCode"] = roleCode
	if v, ok := roleData["privileges"].([]interface{}); ok {
		privs := make([]map[string]interface{}, 0)

		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if module, ok := m["module"].(string); ok {
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
		result["privileges"] = privs
	}
	return result
}

/*
* Check is RoleName is empty
* check if any document present in db with same RoleName
* Check if previlege is empty,Check if module is empty,Check if access length is 0
* Set remaining feilds
* CreateOne create the document in the respective db provided
 */
func CreateRole(c *gin.Context, data map[string]interface{}) (map[string]interface{}, error) {

	roleData := PrepareData(data)
	roleNameRaw, ok := roleData["roleName"]
	if !ok {
		log.Println("roleName key missing in request")
		return nil, errors.New(util.ROLE_NAME_KEY_NOT_PROVIDED)
	}

	roleNameValue, ok := roleNameRaw.(string)
	if !ok || strings.TrimSpace(roleNameValue) == "" {
		log.Println("invalid or empty roleName")
		return nil, errors.New(util.ROLE_NAME_NOT_PROVIDED)
	}
	roleName := strings.ToUpper(roleNameValue)
	exists, err := CheckIfRoleNameExists(c, roleName)
	if !exists {
		log.Println("Error from the CheckIdRoleNameExists")
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

	exists, err = CheckIfPrivilegesIsEmpty(c, privileges)
	if err != nil {
		log.Println("Error from checkIfPrivilegesIsEmpty", err)
		return nil, err
	}

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
	key, err := redis.CreateCacheKey("role", roleData["roleCode"].(string))
	if err != nil {
		log.Println("error from cache(create key) while creating role")
		return nil, errors.New("Error from cache create Key")
	}
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
invalidateRoleCache removes a cached document for the given roleCode.
*/
func invalidateRoleCache(c *gin.Context, roleCode string) error {
	key, err := redis.CreateCacheKey("role", roleCode)
	if err != nil {
		return errors.New("error creating cache key")
	}
	return redis.DeleteCache(c, key)
}

/*
cacheRole stores a role document in Redis using its roleCode.
*/
func cacheRole(c *gin.Context, data map[string]interface{}) error {
	key, err := redis.CreateCacheKey("role", data["roleCode"].(string))
	if err != nil {
		return errors.New("error creating cache key")
	}
	return redis.SetCache(c, key, data)
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

	_ = invalidateRoleCache(c, roleCode)
	_ = cacheRole(c, updated)

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

	key, _ := redis.CreateCacheKey("role", roleCode)

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

	_ = cacheRole(c, role)

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

	_ = invalidateRoleCache(c, roleCode)

	return nil
}
