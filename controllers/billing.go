package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"HealthHub360/util"

	"github.com/gin-gonic/gin"
)

func Bill(router *gin.Engine) {
	bill := router.Group("/bill")
	bill.POST("/create/:code", authorization.Authorize("bill", "create"), CreateBill)
}
func CreateBill(c *gin.Context) {
	patientId := c.Param("code")
	msg, err := services.CreateBill(c, patientId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(msg))
}
