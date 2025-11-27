package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"HealthHub360/util"

	"github.com/gin-gonic/gin"
)

func Patient(router *gin.Engine) {
	patient := router.Group("/patient", authorization.JWTAuth())
	patient.POST("/create", authorization.Authorize("patient", "create"), CreatePatient)
}

func CreatePatient(c *gin.Context) {
	data := make(map[string]interface{})
	err := c.BindJSON(&data)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	msg, err := services.CreatePatient(c, data)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(msg))
}
