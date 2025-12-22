package main

import (
	"HealthHub360/jobs"
	"HealthHub360/routes"
	"log"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	server "github.com/KanapuramVaishnavi/Core/server"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	defaultopts := server.GetDefaultOptions()

	options := server.Options{
		CacheEnabled:     defaultopts.CacheEnabled,
		MongoEnabled:     defaultopts.MongoEnabled,
		WebServerEnabled: defaultopts.WebServerEnabled,
		WebServerPort:    defaultopts.WebServerPort,
		JobsEnabled:      defaultopts.JobsEnabled,
		JobsHandler: func() {
			jobs.SeedDoctorLeaves()
			jobs.StartDailyScheduler()
		},
		WebServerPreHandler: func(r *gin.Engine) {
			r.Use(authorization.CORSMiddleware())
			routes.Routes(r)
		},
	}
	server.Start(options)
}
