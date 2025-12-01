package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"HealthHub360/util"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Appointment(c *gin.Engine) {
	appointment := c.Group("appointment", authorization.JWTAuth())
	appointment.POST("/create/:doctorId/:nurseId", authorization.Authorize("appointment", "create"), CreateAppointment)
	appointment.PATCH("/update/:appointmentId", authorization.Authorize("appointment", "update"), UpdateAppointment)
	appointment.GET("/fetch/:appointmentId", authorization.Authorize("appointment", "view"), FetchAppointmentByCode)
	appointment.GET("/fetchAll", authorization.Authorize("appointment", "view"), FetchAllAppointments)
	appointment.DELETE("/delete/:appointmentId", authorization.Authorize("appointment", "delete"), DeleteAppointmentByCode)
}

/*
* Bind JSON
* And Pass to the service
 */
func CreateAppointment(c *gin.Context) {
	doctorId := c.Param("doctorId")
	nurseId := c.Param("nurseId")
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	response, err := services.CreateAppointment(c, doctorId, nurseId, data)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(response))
}

func UpdateAppointment(c *gin.Context) {
	appointmentId := c.Param("appointmentId")
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	updated, err := services.UpdateAppointment(c, appointmentId, data)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(updated))
}
func FetchAppointmentByCode(c *gin.Context) {
	appointmentId := c.Param("appointmentId")
	appointment, err := services.FetchAppointmentByCode(c, appointmentId)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(appointment))
}

func FetchAllAppointments(c *gin.Context) {
	appointments, err := services.FetchAllAppointment(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(appointments))
}
func DeleteAppointmentByCode(c *gin.Context) {
	appointmentId := c.Param("appointmentId")
	data, err := services.DeleteAppointmentByCode(c, appointmentId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(data))
}
