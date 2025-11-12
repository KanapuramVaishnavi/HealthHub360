package models

type Medicine struct {
	MedicineName string `json:"medicineName" bson:"medicineName"`
	DrugType     string `json:"drugType" bson:"drugType"`
	Dosage       string `json:"dosage" bson:"dosage"`
	NoOfStrips   int    `json:"noOfStrips" bson:"noOfStrips"`
	Required     bool   `json:"required" bson:"required"`
	IsActive     bool   `json:"isActive" bson:"isActive"`
}
