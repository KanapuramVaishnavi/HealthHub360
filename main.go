package main

import (
	"HealthHub360/jobs"
	"HealthHub360/routes"
	"log"
	"os"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	db "github.com/KanapuramVaishnavi/Core/config/db"
	redis "github.com/KanapuramVaishnavi/Core/config/redis"

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
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	router := gin.Default()
	router.Use(authorization.CORSMiddleware())
	routes.Routes(router)
	router.Run(":" + port)
}
