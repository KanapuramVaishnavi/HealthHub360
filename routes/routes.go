package routes

import (
	"HealthHub360/controllers"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"

	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {

	//public
	r.POST("/role/create/superAdmin", controllers.CreateRole)
	r.POST("/superAdmin/create", controllers.CreateSuperAdmin)
	controllers.Auth(r)
	//privateroutes
	r.Use(authorization.JWTAuth())
	controllers.SuperAdmin(r)
	controllers.Tenant(r)
	controllers.Hospital(r)
	controllers.Doctor(r)
	controllers.Receptionist(r)
	controllers.Nurse(r)
	controllers.Pharmacist(r)
	controllers.Patient(r)
	controllers.Guardian(r)
	controllers.MedicalRecord(r)
	controllers.Medicines(r)
	controllers.Appointment(r)
	controllers.Prescription(r)
	controllers.TestReport(r)
	controllers.Test(r)
	controllers.Bill(r)
	controllers.Report(r)

	controllers.Role(r)
}
