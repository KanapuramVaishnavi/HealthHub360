package main

import (
	"HealthHub360/middleware"
	"HealthHub360/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	//CORS Middleware
	router.Use(middleware.CORSMiddleware())
	routes.Routes(router)
	router.Run(":8000")
}
