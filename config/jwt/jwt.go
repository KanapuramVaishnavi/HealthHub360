package jwt

import (
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("your_secret_key")

type JWTClaim struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	RoleCode   string `json:"roleCode"`
	Collection string `json:"collection"`
	jwt.RegisteredClaims
}

/*
Function For Generateing JWT Token where the claims
takes the name,id,email as input and gets stored in the claims
Storage in the  JWT token
*/
func GenerateJWT(code, name, email, roleCode, collectionName string) (string, error) {
	expMinutesStr := os.Getenv("JWT_EXP_MINUTES")
	expMinutes, err := strconv.Atoi(expMinutesStr)
	if err != nil || expMinutes <= 0 {
		expMinutes = 60
	}
	expHours := time.Duration(expMinutes) * time.Minute
	claims := &JWTClaim{
		Code:       code,
		Name:       name,
		Email:      email,
		RoleCode:   roleCode,
		Collection: collectionName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expHours)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

/*
Function For Verifying that the token is Valid or Not
*/
func ValidateToken(tokenString string) (*JWTClaim, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaim{}, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaim)
	if !ok || !token.Valid {
		return nil, err
	}
	return claims, nil
}
