package controllers

import (
	"HealthHub360/services"

	"github.com/gin-gonic/gin"
)

func SuperAdmin(router *gin.Engine) {
	tenant := router.Group("/superAdmin")
	{
		tenant.POST("/register", CreateSuperAdmin)
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
	err = services.CreateSuperAdmin(ctx, user)
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
