package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.GET("/main", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"Result": "Sucessufully Created",
		})
	})
	router.Run(":8000")
}
