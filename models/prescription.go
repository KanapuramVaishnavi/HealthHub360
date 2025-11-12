package models

type Prescription struct {
	PrescriptionID string            `json:"prescriptionID" bson:"prescriptionID"`
	AppointmentID  string            `json:"appointmentID" bson:"appointmentID"`
	PatientID      string            `json:"patientID" bson:"patientID"`
	Medicines      []string          `json:"medicines" bson:"medicines"`
	Dosage         map[string]string `json:"dosage" bson:"dosage"`
	Limit          []string          `json:"limit" bson:"limit"`
	IsActive       bool              `json:"isActive" bson:"isActive"`
}
