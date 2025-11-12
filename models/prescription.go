package models

import "time"

type Prescription struct {
	PrescriptionID string            `json:"prescriptionID" bson:"prescriptionID"`
	AppointmentID  string            `json:"appointmentID" bson:"appointmentID"`
	PatientID      string            `json:"patientID" bson:"patientID"`
	Medicines      []string          `json:"medicines" bson:"medicines"`
	Dosage         map[string]string `json:"dosage" bson:"dosage"`
	Limit          []string          `json:"limit" bson:"limit"`
	CreatedAt      time.Time         `json:"createdAt" bson:"createdAt"`
	CreatedBy      string            `json:"createdBy" bson:"createdBy"`
	UpdatedAt      time.Time         `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy      string            `json:"updatedBy" bson:"updatedBy"`
}
