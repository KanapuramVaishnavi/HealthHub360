package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Tenant(router *gin.Engine) {
	tenant := router.Group("/tenant", authorization.JWTAuth())
	{
		tenant.POST("/create", authorization.Authorize("tenant", "create"), CreateTenant)
	}
}

func CreateTenant(c *gin.Context) {
	tenant := make(map[string]interface{})
	if err := c.ShouldBindJSON(&tenant); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := services.CreateTenant(c, tenant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"message": "Tenant registered successfully!"})
}
