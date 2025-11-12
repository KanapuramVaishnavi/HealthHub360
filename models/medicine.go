package models

import "time"

type Medicine struct {
	MedicineName string    `json:"medicineName" bson:"medicineName"`
	DrugType     string    `json:"drugType" bson:"drugType"`
	Dosage       string    `json:"dosage" bson:"dosage"`
	NoOfStrips   int       `json:"noOfStrips" bson:"noOfStrips"`
	Required     bool      `json:"required" bson:"required"`
	IsActive     bool      `json:"isActive" bson:"isActive"`
	CreatedAt    time.Time `json:"createdAt" bson:"createdAt"`
	CreatedBy    string    `json:"createdBy" bson:"createdBy"`
	UpdatedAt    time.Time `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy    string    `json:"updatedBy" bson:"updatedBy"`
}
