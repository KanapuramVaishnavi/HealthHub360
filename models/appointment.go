package models

import "time"

type Appointment struct {
	AppointmentID string    `json:"appointmentID" bson:"appointmentID"`
	PatientID     string    `json:"patientID" bson:"patientID"`
	DoctorID      string    `json:"doctorID" bson:"doctorID"`
	Slot          time.Time `json:"slot" bson:"slot"`
	Status        string    `json:"status" bson:"status"` //Scheduled,Completed,Cancelled
	CreatedAt     time.Time `json:"createdAt" bson:"createdAt"`
	CreatedBy     string    `json:"createdBy" bson:"createdBy"`
	UpdatedAt     time.Time `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy     string    `json:"updatedBy" bson:"updatedBy"`
}
