package controllers

import (
	"HealthHub360/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Role(router *gin.Engine) {
	tenant := router.Group("/role")
	{
		tenant.POST("/create", CreateRole)
	}
}

/*
* Take the json format
* Pass to prepareData
* Move to services with the parameter context and map[string]interface
 */
func CreateRole(c *gin.Context) {

	var roleData map[string]interface{}

	if err := c.BindJSON(&roleData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	insertedRole, err := services.CreateRole(c, roleData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Role created successfully",
		"data":    insertedRole,
	})
}
