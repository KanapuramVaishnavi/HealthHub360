package models

import "time"

type Consent struct {
	ConsentID         string    `json:"consentID" bson:"consentID"`
	ConsentType       string    `json:"consentType" bson:"consentType"` // General,Surgery,DataSharing
	ConsentPermission string    `json:"consentPermission" bson:"consentPermission"`
	RefID             string    `json:"refID" bson:"refID"` // PatientID
	IsActive          bool      `json:"isActive" bson:"isActive"`
	CreatedAt         time.Time `json:"createdAt" bson:"createdAt"`
	CreatedBy         string    `json:"createdBy" bson:"createdBy"`
	UpdatedAt         time.Time `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy         string    `json:"updatedBy" bson:"updatedBy"`
}
