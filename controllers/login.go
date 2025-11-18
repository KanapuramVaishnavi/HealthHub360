package controllers

import (
	"HealthHub360/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Login(router *gin.Engine) {

	router.POST("/login", LoginController)
}

func LoginController(c *gin.Context) {
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON body",
		})
		return
	}
	msg, err := services.Login(c, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": msg,
	})
}
