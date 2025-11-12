package models

import "time"

type Nurse struct {
	NurseID   string    `json:"nurseID" bson:"nurseID"`
	Name      string    `json:"name" bson:"name"`
	Mail      string    `json:"mail" bson:"mail"`
	PhoneNo   string    `json:"phoneNo" bson:"phoneNo"`
	CreatedBy string    `json:"createdBy" bson:"createdBy"`
	Password  string    `json:"password,omitempty" bson:"password,omitempty"`
	Token     string    `json:"token,omitempty" bson:"token,omitempty"`
	IsActive  bool      `json:"isActive" bson:"isActive"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}
