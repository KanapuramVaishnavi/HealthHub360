package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DoctorAvailability struct {
	ID           primitive.ObjectID `json:"id" bson:"id"`
	Code         string             `json:"code" bson:"code"`
	DoctorID     string             `json:"doctorID" bson:"doctorID"`
	Availability []time.Time        `json:"availability" bson:"availability"`
	CreatedAt    time.Time          `json:"createdAt" bson:"createdAt"`
	CreatedBy    string             `json:"createdBy" bson:"createdBy"`
	UpdatedAt    time.Time          `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy    string             `json:"updatedBy" bson:"updatedBy"`
}
