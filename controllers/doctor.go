package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"HealthHub360/util"

	"github.com/gin-gonic/gin"
)

func Doctor(router *gin.Engine) {
	doctor := router.Group("/doctor", authorization.JWTAuth())
	doctor.POST("/create", authorization.Authorize("doctor", "create"), CreateDoctor)
}
func CreateDoctor(c *gin.Context) {
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	response, err := services.CreateDoctor(c, data)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(response))
}
