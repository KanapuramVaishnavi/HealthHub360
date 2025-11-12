package config

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"HealthHub360/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ctx context.Context = context.Background()

func extractTokenFromHeader(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header required")
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		return "", fmt.Errorf("invalid authorization format")
	}
	return tokenString, nil
}

func verifyTenantExists(tenantID string) error {

	tenantCollection := OpenCollections("tenants")
	var tenant models.Tenant
	filter := bson.M{"tenantID": tenantID}
	opts := options.FindOne().SetSort(bson.M{"UpdatedAt": -1})
	err := FindOne(ctx, tenantCollection, filter, opts, tenant)
	if err != nil {
		return fmt.Errorf("database error: %v", err)
	}
	if !tenant.IsActive {
		return fmt.Errorf("user not found in database")
	}
	return nil
}

func verifyUserExists(collectionName, id string) error {
	collection := OpenCollections(collectionName)
	filter := bson.M{"id": id}
	var user bson.M
	opts := options.FindOne().SetSort(bson.M{"UpdatedAt": -1})
	err := FindOne(ctx, collection, filter, opts, user)
	if err != nil {
		return fmt.Errorf("database error: %v", err)
	}
	isActive, ok := user["isActive"].(bool)
	if !ok {
		return fmt.Errorf("invalid user data format (isActive missing or not boolean)")
	}
	if !isActive {
		return fmt.Errorf("user is inactive")
	}
	return nil
}

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := extractTokenFromHeader(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		claims, err := ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		if err := verifyTenantExists(claims.TenantID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		if claims.Collection != "tenants" {
			if err := verifyUserExists(claims.Collection, claims.ID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				c.Abort()
				return
			}
		}

		c.Set("user_id", claims.ID)
		c.Set("email", claims.Email)
		c.Set("name", claims.Name)
		c.Set("tenant_id", claims.TenantID)
		c.Set("collection", claims.Collection)

		c.Next()
	}
}
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*") // or specific domain
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
