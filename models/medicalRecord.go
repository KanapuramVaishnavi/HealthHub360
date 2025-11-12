package models

import "time"

type MedicalRecord struct {
	MedicalRecordID string    `json:"medicalRecordID" bson:"medicalRecordID"`
	NurseID         string    `json:"nurseID" bson:"nurseID"`
	AppointmentID   string    `json:"appointmentID" bson:"appointmentID"`
	PatientID       string    `json:"patientID" bson:"patientID"`
	BloodGroup      string    `json:"bloodGroup" bson:"bloodGroup"`
	Weight          float64   `json:"weight" bson:"weight"`
	Bp              string    `json:"bp" bson:"bp"`
	RefID           string    `json:"refID" bson:"refID"`
	Status          string    `json:"status" bson:"status"`
	IsActive        bool      `json:"isActive" bson:"isActive"`
	CreatedAt       time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt" bson:"updatedAt"`
}
