package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"HealthHub360/util"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Medicines(router *gin.Engine) {
	medicines := router.Group("/medicines")
	{
		medicines.POST("/create", authorization.Authorize("medicines", "create"), CreateMedicines)
		medicines.GET("/fetch/:medicineCode", authorization.Authorize("medicines", "view"), FetchMedicineByCode)
		medicines.GET("/fetchAll", authorization.Authorize("medicines", "view"), FetchAllMedicines)
		medicines.PATCH("/update/:medicineCode", authorization.Authorize("medicines", "update"), UpdateMedicines)
		medicines.DELETE("/delete/:medicineCode", authorization.Authorize("medicines", "delete"), DeleteMedicine)
	}
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
	medicineId := c.Param("medicineCode")
	medicine, err := services.FetchMedicineByCode(c, medicineId)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(medicine))
}

func FetchAllMedicines(c *gin.Context) {
	medicines, err := services.FetchAllMedicines(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(medicines))
}
func UpdateMedicines(c *gin.Context) {
	medicineId := c.Param("medicineCode")
	data := make(map[string]interface{})
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
	}
	msg, err := services.UpdateMedicines(c, medicineId, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return

	}
	c.JSON(http.StatusOK, util.SuccessResponse(msg))
}

func DeleteMedicine(c *gin.Context) {
	medicineId := c.Param("medicineCode")
	msg, err := services.DeleteMedicine(c, medicineId)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return

	}
	c.JSON(http.StatusOK, util.SuccessResponse(msg))
}
