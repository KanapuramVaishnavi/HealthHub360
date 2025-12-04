package routes

import (
	"HealthHub360/config/authorization"
	"HealthHub360/controllers"

	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {

	//public
	r.POST("/role/create/superAdmin", controllers.CreateRole)
	r.POST("/superAdmin/create", controllers.CreateSuperAdmin)

	//privateroutes
	r.Use(authorization.JWTAuth())
	controllers.Role(r)
	controllers.Auth(r)
	controllers.SuperAdmin(r)
	controllers.Tenant(r)
	controllers.Hospital(r)
	controllers.Doctor(r)
	controllers.Receptionist(r)
	controllers.Nurse(r)
	controllers.Pharmacist(r)
	controllers.Patient(r)
	controllers.MedicalRecord(r)
	controllers.Medicines(r)
	controllers.Appointment(r)
	controllers.Prescription(r)
	controllers.TestReport(r)
	controllers.Test(r)
}
