package model

import "time"

type Tenant struct {
	TenantID      string    `json:"tenantID" bson:"tenantID"`
	TenantName    string    `json:"tenantName" bson:"tenantName"`
	TenantMail    string    `json:"tenantMail" bson:"tenantMail"`
	TenantPhoneNo string    `json:"tenantPhoneNo" bson:"tenantPhoneNo"`
	Password      string    `json:"password,omitempty" bson:"password"`
	Token         string    `json:"token,omitempty" bson:"token"`
	CreatedAt     time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt" bson:"updatedAt"`
}
