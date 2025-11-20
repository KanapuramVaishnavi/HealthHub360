package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"HealthHub360/util"

	"github.com/gin-gonic/gin"
)

func SuperAdmin(router *gin.Engine) {
	router.POST("/superAdmin/create", CreateSuperAdmin)
	superAdmin := router.Group("/superAdmin", authorization.JWTAuth())
	{
		superAdmin.GET("/fetch", authorization.Authorize("superAdmin", "read"), ReadSuperAdmin)
		superAdmin.DELETE("/delete", authorization.Authorize("superAdmin", "delete"), DeleteSuperAdmin)
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

func ReadSuperAdmin(ctx *gin.Context) {
	user, err := services.ReadSuperAdmin(ctx)
	if err != nil {
		ctx.JSON(400, util.FailedResponse(err))
		return
	}
	ctx.JSON(200, util.SuccessResponse(user))
}

func DeleteSuperAdmin(ctx *gin.Context) {
	err := services.DeleteSuperAdmin(ctx)
	if err != nil {
		ctx.JSON(400, util.FailedResponse(err))
		return
	}
	ctx.JSON(200, util.SuccessResponse("Deleted successfully"))

}
