package models

import "time"

type DoctorAvailability struct {
	DoctorID     string      `json:"doctorID" bson:"doctorID"`
	Availability []time.Time `json:"availability" bson:"availability"`
	CreatedAt    time.Time   `json:"createdAt" bson:"createdAt"`
	CreatedBy    string      `json:"createdBy" bson:"createdBy"`
	UpdatedAt    time.Time   `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy    string      `json:"updatedBy" bson:"updatedBy"`
}
