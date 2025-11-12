package models

import "time"

type Patient struct {
	PatientID string    `json:"patientID" bson:"patientID"`
	Name      string    `json:"name" bson:"name"`
	Mail      string    `json:"mail" bson:"mail"`
	Phone     string    `json:"phoneNo" bson:"phoneNo"`
	Age       int       `json:"age" bson:"age"`
	Gender    string    `json:"gender" bson:"gender"`
	Password  string    `json:"password,omitempty" bson:"password,omitempty"`
	Token     string    `json:"token,omitempty" bson:"token,omitempty"`
	IsActive  bool      `json:"isActive" bson:"isActive"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	CreatedBy string    `json:"createdBy" bson:"createdBy"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy string    `json:"updatedBy" bson:"updatedBy"`
}
