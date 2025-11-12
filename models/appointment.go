package models

import "time"

type Appointment struct {
	AppointmentID string    `json:"appointmentID" bson:"appointmentID"`
	PatientID     string    `json:"patientID" bson:"patientID"`
	DoctorID      string    `json:"doctorID" bson:"doctorID"`
	Slot          time.Time `json:"slot" bson:"slot"`
	Status        string    `json:"status" bson:"status"` //Scheduled,Completed,Cancelled
	IsActive      bool      `json:"isActive" bson:"isActive"`
	CreatedAt     time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt" bson:"updatedAt"`
}
