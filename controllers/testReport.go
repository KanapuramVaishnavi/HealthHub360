package controllers

import (
	"HealthHub360/services"

	"log"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
)

func TestReport(router *gin.Engine) {
	test := router.Group("/testReport")
	test.POST("/create/:patientId", authorization.Authorize("testReport", "create"), CreateTestReport)
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
