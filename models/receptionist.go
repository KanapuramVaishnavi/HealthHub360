package models

import "time"

type Receptionist struct {
	ReceptionistID string    `json:"receptionistID" bson:"receptionistID"`
	Name           string    `json:"name" bson:"name"`
	Mail           string    `json:"mail" bson:"mail"`
	PhoneNo        string    `json:"phoneNo" bson:"phoneNo"`
	Password       string    `json:"password,omitempty" bson:"password,omitempty"`
	Token          string    `json:"token,omitempty" bson:"token,omitempty"`
	IsActive       bool      `json:"isActive" bson:"isActive"`
	CreatedAt      time.Time `json:"createdAt" bson:"createdAt"`
	CreatedBy      string    `json:"createdBy" bson:"createdBy"`
	UpdatedAt      time.Time `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy      string    `json:"updatedBy" bson:"updatedBy"`
}
