package authentication

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("your_secret_key")

type JWTClaim struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

/*
Function For Generateing JWT Token where the claims
takes the name,id,email as input and gets stored in the claims
Storage in the  JWT token
*/
func GenerateJWT(id, name, email, tenantID string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &JWTClaim{
		ID:       id,
		Name:     name,
		Email:    email,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
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
