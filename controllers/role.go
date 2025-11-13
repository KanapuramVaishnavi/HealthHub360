package controllers

import (
	"HealthHub360/role"
	"HealthHub360/services"
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func Role(router *gin.Engine) {
	tenant := router.Group("/role")
	{
		tenant.POST("/create", CreateRole)
	}
}

func PrepareData(roleData map[string]interface{}) role.Role {
	var r role.Role
	if v, ok := roleData["roleName"].(string); ok {
		r.RoleName = v
	}
	collection := "role"
	roleCode, err := services.GenerateEmpCode(collection)
	if err != nil {
		log.Println("Error while generating code", err)
	}
	r.RoleCode = roleCode
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
		r.Privileges = privs
	}
	r.CreatedAt = time.Now()
	r.UpdatedAt = time.Now()
	return r
}

func CreateRole(c *gin.Context) {

	var roleData map[string]interface{}

	if err := c.ShouldBindJSON(&roleData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	data := PrepareData(roleData)
	ctx := context.Background()

	insertedRole, err := services.CreateRole(ctx, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "SuperAdmin Role created successfully",
		"data":    insertedRole,
	})
}
