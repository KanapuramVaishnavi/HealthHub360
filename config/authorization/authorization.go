package authorization

import (
	"HealthHub360/config/db"
	"HealthHub360/config/jwt"
	"HealthHub360/models"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
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

	tenantCollection := db.OpenCollections("tenants")
	var tenant models.Tenant
	filter := bson.M{"tenantID": tenantID}
	err := db.FindOne(ctx, tenantCollection, filter, &tenant)
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
func verifyUserExists(collectionName, code string) error {
	collection := db.OpenCollections(collectionName)
	filter := bson.M{"code": code}
	var user bson.M
	err := db.FindOne(ctx, collection, filter, user)
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

		claims, err := jwt.ValidateToken(tokenString)
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
			if err := verifyUserExists(claims.Collection, claims.Code); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				c.Abort()
				return
			}
		}

		c.Set("code", claims.Code)
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
