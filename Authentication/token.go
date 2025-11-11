package authentication

import (
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("123456789")

type JWTClaim struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

func GenerateJWT(id, name, email, tenantID string) (string, error) {
	expMinutesStr := os.Getenv("JWT_EXPIRATION_TIME_IN_MINUTES")
	expMinutes, err := strconv.Atoi(expMinutesStr)
	if err != nil || expMinutes <= 0 {
		expMinutes = 60
	}

	expHours := time.Duration(expMinutes) * time.Minute
	claims := &JWTClaim{
		ID:       id,
		Name:     name,
		Email:    email,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expHours)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}
