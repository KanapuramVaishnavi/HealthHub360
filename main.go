package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error In Connecting with Env", err.Error())
		return
	}
	router := gin.Default()
	router.GET("/main", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"Result": "Sucessufully Created",
		})
	})
	router.Run(":8000")
}
