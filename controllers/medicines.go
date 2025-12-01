package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"HealthHub360/util"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Medicines(router *gin.Engine) {
	medicines := router.Group("/medicines", authorization.JWTAuth())
	medicines.POST("/create", authorization.Authorize("medicines", "create"), CreateMedicines)
	medicines.GET("/fetch/:medicalCode", authorization.Authorize("medicines", "view"), FetchMedicineByCode)
	// medicines.PATCH("/update/:medicalCode", authorization.Authorize("medicines", "update"), UpdateMedicines)
}
func CreateMedicines(c *gin.Context) {
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	msg, err := services.CreateMedicines(c, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(msg))
}

func FetchMedicineByCode(c *gin.Context) {
	medicineId := c.Param("medicalCode")
	medicine, err := services.FetchMedicineByCode(c, medicineId)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(medicine))
}
