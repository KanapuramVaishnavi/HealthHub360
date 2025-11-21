package main

import (
	"HealthHub360/config/authorization"
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"HealthHub360/jobs"
	"HealthHub360/routes"
	"HealthHub360/services"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db.ConnectDB()
	services.InitCommonCollections()
	services.InitCollections()
	redis.ConnectRedis()
	jobs.StartDailyScheduler()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	router := gin.Default()
	router.Use(authorization.CORSMiddleware())
	routes.Routes(router)
	router.Run(":" + port)
}
