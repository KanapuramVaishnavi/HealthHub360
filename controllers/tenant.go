package controllers

import (
	"HealthHub360/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Tenant(router *gin.Engine) {
	tenant := router.Group("/tenant")
	{
		tenant.POST("/register", RegisterTenant)
	}
}

func RegisterTenant(c *gin.Context) {
	var tenant models.Tenant
	if err := c.ShouldBindJSON(&tenant); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	/*err := services.CreateTenant(tenant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}*/

	c.JSON(http.StatusOK, gin.H{"message": "Tenant registered successfully!"})
}
