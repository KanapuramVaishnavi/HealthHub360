package routes

import (
	"HealthHub360/controllers"

	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	controllers.Role(r)
	controllers.Auth(r)
	controllers.SuperAdmin(r)
	controllers.Tenant(r)
	controllers.Hospital(r)
}
