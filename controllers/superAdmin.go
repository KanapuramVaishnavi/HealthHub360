package controllers

import (
	"HealthHub360/services"
	"HealthHub360/util"

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
		ctx.JSON(400, util.FailedResponse(err))
		return
	}
	err = services.CreateSuperAdmin(ctx, user)
	if err != nil {
		ctx.JSON(400, util.FailedResponse(err))
		return
	}
	ctx.JSON(200, util.SuccessResponse("Created successfully"))
}
