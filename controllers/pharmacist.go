package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"HealthHub360/util"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Pharmacist(router *gin.Engine) {
	pharma := router.Group("/pharmacist")
	{
		pharma.POST("/create", authorization.Authorize("pharmacist", "create"), CreatePharmacist)
		pharma.GET("/fetch/:code", authorization.Authorize("pharmacist", "view"), FetchPharmacistByCode)
		pharma.GET("/fetchAll/:tenantid", authorization.Authorize("pharmacist", "view"), FetchAllPharmacist)
	}
}

func CreatePharmacist(ctx *gin.Context) {
	var body map[string]interface{}
	err := ctx.BindJSON(&body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	err = services.CreatePharmacist(ctx, body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	ctx.JSON(200, util.SuccessResponse("Created successfully"))

}
func FetchPharmacistByCode(c *gin.Context) {
	code := c.Param("code")
	data, err := services.FetchPharmacistByCode(c, code)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(data))
}

func FetchAllPharmacist(c *gin.Context) {
	tenantId := c.Param("tenantId")
	doc, err := services.FetchAllPharmacist(c, tenantId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(doc))
}

// func DeletePharmacist(c *gin.Context) {

// }
// \
