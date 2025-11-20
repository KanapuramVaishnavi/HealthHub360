package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"HealthHub360/util"

	"github.com/gin-gonic/gin"
)

func Hospital(router *gin.Engine) {
	hospital := router.Group("/hospital", authorization.JWTAuth())
	{
		hospital.POST("/create", authorization.Authorize("hospital", "create"), HospitalCreate)
		hospital.POST("/update/:code", authorization.Authorize("hospital", "update"), UpdateHospital)
	}
}
func HospitalCreate(c *gin.Context) {
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	if err := services.CreateHospital(c, data); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse("created successfully"))
}
func UpdateHospital(c *gin.Context) {
	code := c.Param("code")
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	if err := services.UpdateHospital(c, data, code); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse("created successfully"))
}
