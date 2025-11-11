package middleware

import (
	"net/http"
	"strings"

	authentication "ex.com/Authentication"
	"github.com/gin-gonic/gin"
)

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := authentication.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Attach claims to context
		c.Set("user_id", claims.ID)
		c.Set("email", claims.Email)
		c.Set("name", claims.Name)
		c.Set("tenant_id", claims.TenantID)

		c.Next()
	}
}
