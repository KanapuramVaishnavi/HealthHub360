package models

import "time"

type DoctorAvailability struct {
	DoctorID     string      `json:"doctorID" bson:"doctorID"`
	Availability []time.Time `json:"availability" bson:"availability"`
	IsActive     bool        `json:"isActive" bson:"isActive"`
}
