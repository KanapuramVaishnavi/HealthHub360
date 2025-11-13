package routes

import (
	"HealthHub360/controllers"

	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	controllers.Role(r)
	controllers.Tenant(r)
	controllers.SuperAdmin(r)
}
