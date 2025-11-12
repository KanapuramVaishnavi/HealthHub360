package models

import "time"

type Billing struct {
	BillingID     string    `json:"billingID" bson:"billingID"`
	AppointmentID string    `json:"appointmentID" bson:"appointmentID"`
	PatientID     string    `json:"patientID" bson:"patientID"`
	ServiceCharge float64   `json:"serviceCharge" bson:"serviceCharge"`
	Amount        float64   `json:"amount" bson:"amount"`
	Status        string    `json:"status" bson:"status"`
	IsActive      bool      `json:"isActive" bson:"isActive"`
	CreatedAt     time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt" bson:"updatedAt"`
}
