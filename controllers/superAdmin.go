package controllers

import (
	"HealthHub360/services"

	"github.com/gin-gonic/gin"
)

func SuperAdmin(router *gin.Engine) {
	tenant := router.Group("/superAdmin")
	{
		tenant.POST("/register", CreateSuperAdmin)
		tenant.POST("/login", LoginSuperAdmin)
	}
}

func CreateSuperAdmin(ctx *gin.Context) {
	var user map[string]interface{}
	err := ctx.BindJSON(&user)
	if err != nil {
		ctx.JSON(400, gin.H{
			"Error": err.Error(),
		})
		return
	}
	err = services.CreateSuperAdmin(user)
	if err != nil {
		ctx.JSON(400, gin.H{
			"Error": err.Error(),
		})
		return
	}
	ctx.JSON(200, gin.H{
		"success": "user created succesfully",
	})
}

func LoginSuperAdmin(c *gin.Context) {
	var data map[string]interface{}
	err := c.BindJSON(&data)
	if err != nil {
		c.JSON(400, gin.H{"Error": err.Error()})
		return
	}
	_, err = services.SuperAdminLogin(c, data)
	if err != nil {
		c.JSON(500, gin.H{
			"Error": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": "Login successful"})
}
