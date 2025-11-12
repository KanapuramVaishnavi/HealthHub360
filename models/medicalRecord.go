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
	CreatedAt       time.Time `json:"createdAt" bson:"createdAt"`
	CreatedBy       string    `json:"createdBy" bson:"createdBy"`
	UpdatedAt       time.Time `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy       string    `json:"updatedBy" bson:"updatedBy"`
}
