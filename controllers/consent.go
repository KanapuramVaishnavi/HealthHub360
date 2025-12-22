package controllers

import (
	"HealthHub360/services"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"
	"github.com/gin-gonic/gin"
)

func Consent(router *gin.Engine) {
	consent := router.Group("/consent")
	consent.POST("/create", authorization.Authorize("consent", "create"), CreateConsent)
}

func CreateConsent(c *gin.Context) {
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	response, err := services.CreateConsent(c, data)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(response))

}
