package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"HealthHub360/util"
	"log"

	"github.com/gin-gonic/gin"
)

func TestReport(router *gin.Engine) {
	test := router.Group("/testReport", authorization.JWTAuth())
	test.POST("/create/:patientId", CreateTestReport)
}

/*
* Extract code and tenantId from the context
* Pass the code and tenantId to the services
 */
func CreateTestReport(c *gin.Context) {
	patientId := c.Param("patientId")
	response, err := services.CreateTestReport(c, patientId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	log.Println(response)
	c.JSON(200, util.SuccessResponse(response))
}
