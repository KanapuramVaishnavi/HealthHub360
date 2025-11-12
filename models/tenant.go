package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Tenant struct {
	ID        primitive.ObjectID `json:"id" bson:"id"`
	TenantID  string             `json:"tenantID" bson:"tenantID"`
	Name      string             `json:"name" bson:"name"`
	Mail      string             `json:"mail" bson:"mail"`
	PhoneNo   string             `json:"phoneNo" bson:"phoneNo"`
	Password  string             `json:"password,omitempty" bson:"password"`
	Token     string             `json:"token,omitempty" bson:"token"`
	IsActive  bool               `json:"isActive" bson:"isActive"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	CreatedBy string             `json:"createdBy" bson:"createdBy"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy string             `json:"updatedBy" bson:"updatedBy"`
}
