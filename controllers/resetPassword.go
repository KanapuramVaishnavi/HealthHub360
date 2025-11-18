package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ResetPassword(router *gin.Engine) {
	router.POST("/reset-password", authorization.JWTAuth(), ResetPasswordHandler)
}
func ResetPasswordHandler(c *gin.Context) {
	var body map[string]interface{}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request body",
		})
		return
	}

	msg, err := services.ResetPassword(c, body)
	if err != nil {
		log.Println("ResetPasswordGeneric error:", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": msg,
	})
}
