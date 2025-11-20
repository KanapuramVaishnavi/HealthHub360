package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"HealthHub360/util"
	"log"

	"github.com/gin-gonic/gin"
)

func SuperAdmin(router *gin.Engine) {
	router.POST("/superAdmin/create", CreateSuperAdmin)
	superAdmin := router.Group("/superAdmin", authorization.JWTAuth())
	{
		superAdmin.GET("/fetch", authorization.Authorize("superAdmin", "view"), ReadSuperAdmin)
		superAdmin.PUT("/update", authorization.Authorize("superAdmin", "update"), UpdateSuperAdmin)
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
func UpdateSuperAdmin(ctx *gin.Context) {
	var body map[string]interface{}
	if err := ctx.BindJSON(&body); err != nil {
		ctx.JSON(400, util.FailedResponse(err))
		return
	}
	err := services.UpdateSuperAdmin(ctx, body)
	if err != nil {
		ctx.JSON(400, util.FailedResponse(err))
		return
	}
	log.Println("done done done ")
	ctx.JSON(200, util.SuccessResponse("Updated SUCCESSFULLY"))
}
