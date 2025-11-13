package controllers

import (
	"HealthHub360/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Tenant(router *gin.Engine) {
	tenant := router.Group("/tenant")
	{
		tenant.POST("/create", CreateTenant)
	}
}

func CreateTenant(c *gin.Context) {
	var tenant models.Tenant
	if err := c.ShouldBindJSON(&tenant); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tenant registered successfully!"})
}
