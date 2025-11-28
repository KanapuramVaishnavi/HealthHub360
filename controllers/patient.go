package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"HealthHub360/util"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Patient(router *gin.Engine) {
	patient := router.Group("/patient", authorization.JWTAuth())
	patient.POST("/create", authorization.Authorize("patient", "create"), CreatePatient)
	patient.GET("/fetch/:patientId", authorization.Authorize("patient", "view"), FetchPatientByCode)
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

func FetchPatientByCode(c *gin.Context) {
	patientId := c.Param("patientId")
	patient, err := services.FetchPatientByCode(c, patientId)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(patient))
}
