package services

import (
	"HealthHub360/config/db"
	"HealthHub360/util"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

var RoleCollection *mongo.Collection

func InitCollections() {
	RoleCollection = db.OpenCollections("role")
}

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

// func PrepareData(roleData map[string]interface{}) map[string]interface{} {
// 	result := make(map[string]interface{})

// 	if v, ok := roleData["roleName"].(string); ok {
// 		result["roleName"] = strings.ToUpper(v)
// 	}

// 	roleCode, err := GenerateEmpCode("role")
// 	if err != nil {
// 		log.Println("Error while generating code", err)
// 	}
// 	result["roleCode"] = roleCode

// 	privilegesRaw, ok := roleData["privileges"].([]interface{})
// 	if ok && len(privilegesRaw) > 0 {
// 		privs := make([]map[string]interface{}, 0, len(privilegesRaw))
// 		for _, item := range privilegesRaw {
// 			m, ok := item.(map[string]interface{})
// 			if !ok {
// 				continue
// 			}

// 			if accessRaw, exists := m["access"].([]interface{}); exists {
// 				accessList := make([]string, 0, len(accessRaw))
// 				for _, a := range accessRaw {
// 					if s, ok := a.(string); ok {
// 						accessList = append(accessList, s)
// 					}
// 				}
// 				m["access"] = accessList
// 			}

// 			privs = append(privs, m)
// 		}
// 		result["privileges"] = privs
// 	} else {
// 		// Always add an empty slice to avoid nil
// 		result["privileges"] = []map[string]interface{}{}
// 	}

// 	return result
// }

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

	collection := db.OpenCollections("role")
	roleData["CreatedBy"] = "SYSTEM"
	roleData["UpdatedBy"] = "SYSTEM"
	roleData["CreatedAt"] = time.Now()
	roleData["UpdatedAt"] = time.Now()

	_, err = db.CreateOne(ctx, collection, roleData)
	if err != nil {
		return nil, fmt.Errorf("failed to insert role: %v", err)
	}

	return roleData, nil
}

// func CreateRole(c *gin.Context, data map[string]interface{}) (map[string]interface{}, error) {
// 	roleData := PrepareData(data)

// 	roleNameRaw, ok := roleData["roleName"]
// 	if !ok {
// 		return nil, errors.New(util.ROLE_NAME_KEY_NOT_PROVIDED)
// 	}

// 	roleNameValue, ok := roleNameRaw.(string)
// 	if !ok || strings.TrimSpace(roleNameValue) == "" {
// 		return nil, errors.New(util.ROLE_NAME_NOT_PROVIDED)
// 	}
// 	roleName := strings.ToUpper(roleNameValue)

// 	exists, err := CheckIfRoleNameExists(c, roleName)
// 	if !exists {
// 		return nil, err
// 	}

// 	// Privileges
// 	privilegesRaw, ok := roleData["privileges"]
// 	if !ok {
// 		return nil, errors.New(util.PRIVILEGES_REQUIRED)
// 	}

// 	// assert privileges as []map[string]interface{}
// 	privileges, ok := privilegesRaw.([]map[string]interface{})
// 	if !ok || len(privileges) == 0 {
// 		return nil, errors.New(util.PRIVILEGES_DATA_REQUIRED)
// 	}

// 	// Validate privileges
// 	_, err = CheckIfPrivilegesIsEmpty(c, privileges)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Set metadata
// 	roleData["CreatedBy"] = "SYSTEM"
// 	roleData["UpdatedBy"] = "SYSTEM"
// 	roleData["CreatedAt"] = time.Now()
// 	roleData["UpdatedAt"] = time.Now()

// 	collection := db.OpenCollections("role")
// 	_, err = db.CreateOne(ctx, collection, roleData)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to insert role: %v", err)
// 	}

// 	return roleData, nil
// }
