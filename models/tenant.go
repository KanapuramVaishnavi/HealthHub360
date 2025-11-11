package models

import "time"

type Tenant struct {
	TenantID  string    `json:"tenantID" bson:"tenantID"`
	Name      string    `json:"name" bson:"name"`
	Mail      string    `json:"mail" bson:"mail"`
	PhoneNo   string    `json:"phoneNo" bson:"phoneNo"`
	Password  string    `json:"password,omitempty" bson:"password"`
	Token     string    `json:"token,omitempty" bson:"token"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}
