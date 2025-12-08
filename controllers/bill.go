package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"HealthHub360/util"
	"log"

	"github.com/gin-gonic/gin"
)

func Bill(router *gin.Engine) {
	bill := router.Group("/bill", authorization.JWTAuth())
	bill.POST("/generate/:patientId", CreateBill)
}

/*
* Extract code and tenantId from the context
* Pass the code and tenantId to the services
 */
func CreateBill(c *gin.Context) {
	patientId := c.Param("patientId")
	response, err := services.GenerateBillingReport(c, patientId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	log.Println(response)
	c.JSON(200, util.SuccessResponse(response))
}
