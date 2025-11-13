package services

import (
	"HealthHub360/config/db"
	"HealthHub360/role"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

var RoleCollection *mongo.Collection

func InitCollections() {
	RoleCollection = db.OpenCollections("role")
}
func CreateRole(ctx context.Context, roleData role.Role) (role.Role, error) {
	collection := db.OpenCollections("role")
	code := ctx.Value("code").(string)

	roleData.CreatedBy = code
	roleData.UpdatedBy = code

	roleData.CreatedAt = time.Now()
	roleData.UpdatedAt = time.Now()

	_, err := db.CreateOne(ctx, collection, roleData)
	if err != nil {
		return role.Role{}, fmt.Errorf("failed to insert role: %v", err)
	}

	return roleData, nil
}
