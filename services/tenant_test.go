package services

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCreateTenant_InvalidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(nil)

	data := map[string]interface{}{}
	err := CreateTenant(c, data)

	assert.Error(t, err)
}

func TestCreateTenant_ValidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(nil)

	data := map[string]interface{}{
		"name":     "Vaishnavi",
		"email":    "vaishnavi@test.com",
		"phoneNo":  "9123456789",
		"roleCode": "R0002",
		"dob":      "24-12-2025",
	}
	err := CreateTenant(c, data)

	assert.NoError(t, err)
}
