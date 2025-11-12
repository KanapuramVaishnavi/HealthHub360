package main

import (
	"HealthHub360/config"
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

	config.ConnectDB()
	config.ConnectRedis()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	router := gin.Default()
	router.Use(config.CORSMiddleware())
	routes.Routes(router)
	router.Run(":" + port)
}
