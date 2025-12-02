package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"HealthHub360/util"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Prescription(router *gin.Engine) {
	prescription := router.Group("/prescription", authorization.JWTAuth())
	prescription.POST("/create", authorization.Authorize("prescription", "create"), CreatePrescription)
}
func CreatePrescription(c *gin.Context) {
	data := make(map[string]interface{})
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	msg, err := services.CreatePrescription(c, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(msg))
}
