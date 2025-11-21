package controllers

import (
	"HealthHub360/services"
	"HealthHub360/util"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Receptionist(router *gin.Engine) {
	router.POST("/receptionist/create", CreateReceptionist)
	// superAdmin := router.Group("/receptionist", authorization.JWTAuth())
	// {
	//  // superAdmin.GET("/fetch", authorization.Authorize("receptionist", "view"), ReadSuperAdmin)
	//  // superAdmin.PUT("/update", authorization.Authorize("receptionist", "update"), UpdateSuperAdmin)
	//  // superAdmin.DELETE("/delete", authorization.Authorize("receptionist", "delete"), DeleteSuperAdmin)
	// }
}

func CreateReceptionist(ctx *gin.Context) {
	var body map[string]interface{}
	err := ctx.BindJSON(&body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	err = services.CreateReceptionist(ctx, body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	ctx.JSON(200, util.SuccessResponse("Created successfully"))

}
