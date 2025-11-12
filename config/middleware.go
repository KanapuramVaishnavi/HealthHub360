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

/*
here we are extracting the info from header
by trimming the prefix and if the header is valid only
it gets passed other and checks whether the bearer
Token is Invalid
*/
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

/*
Verify tenant exists  checks the collection of tenants
sort according to the decreasing order by the key of
Updated At
*/
func verifyTenantExists(tenantID string) error {

	tenantCollection := OpenCollections("tenants")
	var tenant models.Tenant
	filter := bson.M{"tenantID": tenantID}
	opts := options.FindOne().SetSort(bson.M{"UpdatedAt": -1})
	err := FindOne(ctx, tenantCollection, filter, opts, &tenant)
	if err != nil {
		return fmt.Errorf("database error: %v", err)
	}
	if !tenant.IsActive {
		return fmt.Errorf("user not found in database")
	}
	return nil
}

/*
Verify user exists  checks the collection of user
sort according to the decreasing order by the key of
Updated At
*/
func verifyUserExists(collectionName, id string) error {
	collection := OpenCollections(collectionName)
	filter := bson.M{"id": id}
	var user bson.M
	opts := options.FindOne().SetSort(bson.M{"UpdatedAt": -1})
	err := FindOne(ctx, collection, filter, opts, &user)
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

/*
here
1.first the extraction takes place
2.Validation of token takes place
3.Verify tenant is existing at the tenant level
4.if the collection is other than tenants then it gets verify user exists function will be
passed.
*/
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

/*
Here the cors middleware takes place
*/
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
