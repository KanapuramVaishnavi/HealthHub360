package routes

import (
	"HealthHub360/controllers"

	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {

	//public

	//privateroutes
	controllers.Role(r)
	controllers.Auth(r)
	controllers.SuperAdmin(r)
	controllers.Tenant(r)
	controllers.Hospital(r)
	controllers.Doctor(r)
	controllers.Patient(r)
	controllers.Receptionist(r)
	controllers.MedicalRecord(r)
	controllers.Nurse(r)
	controllers.Medicines(r)
	controllers.Appointment(r)
	controllers.Pharmacist(r)
	controllers.Prescription(r)
	controllers.TestReport(r)
	controllers.Test(r)
}
