package controllers

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	jwt "github.com/KanapuramVaishnavi/Core/config/jwt"
)

// SetupTestRouter sets up Gin router for testing
func SetupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode) // Test mode avoids unnecessary logs
	router := gin.Default()
	return router
}

func TestCreateTenant_InvalidBody(t *testing.T) {
	router := SetupTestRouter()

	reqBody := []byte(`{ invalid json }`)
	req, _ := http.NewRequest("POST", "/tenant/create", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func generateTestToken() string {

	token, err := jwt.GenerateJWT(
		"S0001",
		"vaishnaviSuperAdmin@test.com",
		"R0001",
		"SUPERADMIN",
		"",
		false,
	)
	if err != nil {
		panic(err)
	}
	return token
}

func TestCreateTenant_ValidBody(t *testing.T) {
	router := SetupTestRouter()

	token := generateTestToken()
	log.Println("token: ", token)
	router.Use(authorization.JWTAuth())
	Tenant(router)
	payload := map[string]interface{}{
		"name":     "Vaishnavi",
		"email":    "vaishnavi@test.com",
		"phoneNo":  "9123456789",
		"roleCode": "R0002",
		"dob":      "23/12/2025",
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/tenant/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
