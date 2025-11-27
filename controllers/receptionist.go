package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"HealthHub360/util"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Receptionist(router *gin.Engine) {
	recep := router.Group("/receptionist", authorization.JWTAuth())
	{
		recep.POST("/create", authorization.Authorize("receptionist", "create"), CreateReceptionist)
		recep.GET("/fetch/:code/:tenantid", authorization.Authorize("receptionist", "view"), FetchReceptionistByCode)
		recep.GET("/fetchAll/:tenantid", authorization.Authorize("receptionist", "view"), FetchAllReceptionist)
		recep.POST("/appointment/book/:doctorId/:nurseId", authorization.Authorize("appointment", "create"), authorization.Authorize("medicalRecord", "create"), BookAppointment)
	}
}

func CreateReceptionist(ctx *gin.Context) {
	var body map[string]interface{}
	err := ctx.BindJSON(&body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	err = services.CreateReceptionist(ctx, body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	ctx.JSON(200, util.SuccessResponse("Created successfully"))

}
func FetchReceptionistByCode(c *gin.Context) {
	code := c.Param("code")
	data, err := services.FetchReceptionistByCode(c, code)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(data))
}

func FetchAllReceptionist(c *gin.Context) {
	tenantId := c.Param("tenantId")
	doc, err := services.FetchAllReceptionist(c, tenantId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(doc))
}

/*
* Get the doctorId,nurseId from the parama
* Get the data and Pass to the services
 */
func BookAppointment(c *gin.Context) {
	doctorId := c.Param("doctorId")
	nurseId := c.Param("nurseId")
	data := make(map[string]interface{})
	err := c.BindJSON(&data)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	appointment, err := services.BookAppointment(c, doctorId, nurseId, data)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(appointment))
}
