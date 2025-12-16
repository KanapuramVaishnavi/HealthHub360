package main

import (
	"HealthHub360/config/authorization"
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"HealthHub360/jobs"
	"HealthHub360/nats"
	"HealthHub360/routes"
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
	redis.ConnectRedis()
	jobs.SeedDoctorLeaves()
	jobs.StartDailyScheduler()
	if err := nats.InitNATS(); err != nil {
		log.Fatal("Failed to init NATS:", err)
	}
	if err := nats.StartAppointmentNotificationSubscriber(); err != nil {
		log.Fatal("Failed to start appointment subscriber:", err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	router := gin.Default()
	router.Use(authorization.CORSMiddleware())
	routes.Routes(router)
	router.Run(":" + port)
}
