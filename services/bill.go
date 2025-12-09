package services

import (
	"HealthHub360/config/db"
	"HealthHub360/config/redis"
	"HealthHub360/util"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CheckForAccess(c *gin.Context, patient map[string]interface{}) error {
	pharamcistTenantId, err := GetFromContext[string](c, "tenantId")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return err
	}
	tenantIdFromPatientVal, ok := patient["tenantId"]
	if !ok {
		log.Println("Unable to fetch tenantId field from patient")
		return errors.New("Unable to fetch tenantId field from patient")
	}
	tenantIdFromPatient, ok := tenantIdFromPatientVal.(string)
	if !ok {
		log.Println("Type assertion error for tenantId from patient")
		return errors.New("Type assertion error for tenantId from patient")
	}
	if pharamcistTenantId != tenantIdFromPatient {
		log.Println("This pharmacist doesnot have access")
		return errors.New("This pharmacist doesnot have access")
	}
	return nil
}
func GetLatestAppointmentIDFromPatient(patient map[string]interface{}) (string, error) {
	rawApp, ok := patient["appointments"]
	if !ok || rawApp == nil {
		log.Println("Unable to find key Appointments from patient")
		return "", errors.New("Unable to find field appointments from patient")
	}
	log.Printf("appointments type : %T", rawApp)
	app, ok := rawApp.([]interface{})
	if !ok {
		log.Println("Type assertion failed for appointments field")
		return "", errors.New("Type assertion failed for appointments field")
	}

	if len(app) == 0 {
		return "", errors.New("no appointments found")
	}

	appointmentId, ok := app[len(app)-1].(string)
	if !ok {
		return "", errors.New("invalid appointmentId format")
	}
	return appointmentId, nil
}
func FetchTestsFromMedicalRecord(medicalRecord map[string]interface{}) ([]string, error) {
	var tests []string
	rawTests, exists := medicalRecord["testList"]
	if !exists || rawTests == nil {
		log.Println("No testList found in medicalRecord")
		return nil, errors.New("No tests found in medicalRecord")
	}
	log.Printf("The type of rawTests: %T ", rawTests)
	var testList []interface{}

	switch v := rawTests.(type) {
	case []interface{}:
		testList = v
	case primitive.A:
		testList = []interface{}(v)
	default:
		return nil, errors.New("unsupported appointments type")
	}
	for _, t := range testList {
		val, ok := t.(string)
		if !ok {
			log.Println("Unable to fetch test from testList")
			return nil, errors.New("Unable to fetch test from testList")
		}
		tests = append(tests, val)
	}
	return tests, nil
}
func GenerateBillForTests(c *gin.Context, medicalRecord map[string]interface{}) ([]map[string]interface{}, int, error) {

	tests, err := FetchTestsFromMedicalRecord(medicalRecord)
	if err != nil {
		log.Println("Error from fetchTestsFromMedicalRecord: ", err)
		return nil, 0, err
	}
	log.Println("tests: ", tests)
	var billTests []map[string]interface{}
	var incTestPrice int
	for _, t := range tests {
		test, err := FetchTestByCode(c, t)
		if err != nil {
			log.Println("Error from fetchTestByCode: ", err)
			return nil, 0, err
		}
		billTest := make(map[string]interface{})
		priceVal, ok := test["price"].(string)
		if !ok {
			log.Println("Unable to get price from single test")
			return nil, 0, errors.New("Unable to get price from single test")
		}
		price, _ := strconv.Atoi(priceVal)
		billTest["testId"] = t
		billTest["price"] = priceVal
		incTestPrice = incTestPrice + price
		billTests = append(billTests, billTest)
	}
	return billTests, incTestPrice, nil
}
func FetchPrescriptionIdFromMedicalRecord(medicalRecord map[string]interface{}) (string, error) {
	prescriptionIdVal, ok := medicalRecord["prescriptionId"]
	if !ok {
		log.Println("Unable to fetch prescription field from medicalRecord")
		return "", errors.New("Unable to fetch prescription field from medicalRecord")
	}
	prescriptionId, ok := prescriptionIdVal.(string)
	if !ok {
		log.Println("Type assertion error for prescription field")
		return "", errors.New("Type assertion error for prescription field")
	}
	return prescriptionId, nil
}
func FetchFieldsFromMedicine(c *gin.Context, medicineId string) (int, int, int, error) {
	var val int
	medicineFetched, err := FetchMedicineByCode(c, medicineId)
	if err != nil {
		log.Println("Error from fetchMedicineByCode: ", err)
		return val, val, val, err
	}

	pricePerStripVal, ok := medicineFetched["pricePerStrip"].(string)
	if !ok {
		log.Println("Unable to fetch pricePerStrip from medicineFetched")
		return val, val, val, errors.New("Unable to fetch pricePerStrip from medicineFetched")
	}
	pricePerStrip, _ := strconv.Atoi(pricePerStripVal)

	log.Printf("tabletsPerStip %T", medicineFetched["tabletsPerStrip"])
	tabletsPerStripVal, ok := medicineFetched["tabletsPerStrip"]
	if !ok {
		log.Println("unable to fetch tabletsPerStrip")
		return val, val, val, errors.New("Unable to fetch tabletsPerStrip")
	}
	tabletsPerStripInt, ok := tabletsPerStripVal.(string)
	if !ok {
		log.Println("Type assertion from tabletsPerStrip")
		return val, val, val, errors.New("Type assertion from tabletsPerStrip")
	}
	tabletsPerStrip, _ := strconv.Atoi(tabletsPerStripInt)
	log.Println("tabletsPerStrip ", tabletsPerStrip)

	totalNoOfTabletsVal, ok := medicineFetched["totalNoOfTablets"]
	if !ok {
		log.Println("unable to fetch totalNoOfTablets")
		return val, val, val, errors.New("Unable to fetch totalNoOfTablets")
	}
	totalNoOfTabletsInt, ok := totalNoOfTabletsVal.(string)
	if !ok {
		log.Println("Type assertion from totalNoOfTablets")
		return val, val, val, errors.New("Type assertion from totalNoOfTablets")
	}
	totalNoOfTablets, _ := strconv.Atoi(totalNoOfTabletsInt)
	log.Println("totalNoOfTablets: ", totalNoOfTablets)
	return pricePerStrip, tabletsPerStrip, totalNoOfTablets, nil
}
func FetchMedicineFieldsFromPrescription(medicine map[string]interface{}) (string, int, error) {

	medicineId, ok := medicine["medicineId"].(string)
	if !ok {
		log.Println("Unable to fetch medicineId or type assertion")
		return "", 0, errors.New("Unable to fetch medicineId")
	}

	dosagePerFrequencyVal, ok := medicine["dosagePerFrequency"].(string)
	if !ok {
		log.Println("Unable to fetch dosagePerFrequency or type assertion")
		return "", 0, errors.New("Unable to fetch dosagePerFrequency")
	}
	dosagePerFrequency, _ := strconv.Atoi(dosagePerFrequencyVal)

	noOfDaysVal, ok := medicine["noOfDays"].(string)
	if !ok {
		log.Println("Unable to fetch noOfDays")
		return "", 0, errors.New("Unable to fetch noOfDays")
	}
	noOfDays, _ := strconv.Atoi(noOfDaysVal)

	log.Printf("frequncy type %T", medicine["frequency"])
	freq := medicine["frequency"].(map[string]interface{})
	timesPerDay := 0
	if freq["morning"].(bool) {
		timesPerDay++
	}
	if freq["afternoon"].(bool) {
		timesPerDay++
	}
	if freq["night"].(bool) {
		timesPerDay++
	}
	totalTablets := dosagePerFrequency * timesPerDay * noOfDays
	return medicineId, totalTablets, nil
}
func GenerateBillForMedicines(c *gin.Context, medicalRecord map[string]interface{}) ([]map[string]interface{}, int, error) {
	prescriptionId, err := FetchPrescriptionIdFromMedicalRecord(medicalRecord)
	if err != nil {
		log.Println("Error from fetchPrescriptionFromMedicalRecord: ", err)
		return nil, 0, err
	}

	prescription, err := FetchPrescriptionByCode(c, prescriptionId)
	if err != nil {
		log.Println("Error from fetchPrescriptionByCode: ", err)
		return nil, 0, err
	}

	medicineRaw, ok := prescription["medicines"]
	if !ok {
		log.Println("Unable to fetch medicines from medicineRaw: ", err)
		return nil, 0, errors.New("Unable to fetch medicines from medicineRaw")
	}
	var medicines []interface{}
	switch v := medicineRaw.(type) {
	case primitive.A:
		medicines = []interface{}(v)
	case []interface{}:
		medicines = v
	default:
		return nil, 0, errors.New("Invalid medicines type")
	}

	var billMedicines []map[string]interface{}
	var incMedicinePrice int
	for _, m := range medicines {

		medicine, ok := m.(map[string]interface{})
		if !ok {
			log.Println("Unable to fetch medicine from listOfMedicines(prescription)")
			return nil, 0, errors.New("Unable to fetch medicines from listOfMedicines(prescription)")
		}
		log.Println("medicine: ", medicine)

		singleMedicine := make(map[string]interface{})
		medicineId, totalTablets, err := FetchMedicineFieldsFromPrescription(medicine)
		if err != nil {
			log.Println("Error from FetchMedicineFieldsFromPrescription: ", err)
			return nil, 0, err
		}
		log.Println("medicineId: ", medicineId)
		pricePerStrip, tabletsPerStrip, totalNoOfTablets, err := FetchFieldsFromMedicine(c, medicineId)
		if err != nil {
			log.Println("Error from FetchFieldsFromMedicines: ", err)
			return nil, 0, err
		}
		costPerTablet := pricePerStrip / tabletsPerStrip
		singleMedicine["requiredTablets"] = strconv.Itoa(totalTablets)
		singleMedicine["medicineId"] = medicineId
		singleMedicine["costPerTablet"] = strconv.Itoa(costPerTablet)
		remainingTablets := totalNoOfTablets - totalTablets
		log.Println("remainingTablets: ", remainingTablets)

		updateMedicine := make(map[string]interface{})
		if remainingTablets < 0 {
			singleMedicine["isDispensed"] = false
			singleMedicine["pricePerMedicine"] = "0"
			singleMedicine["totalNoOfTablets"] = strconv.Itoa(totalNoOfTablets)
		} else {
			singleMedicine["isDispensed"] = true
			singleMedicine["totalNoOfTablets"] = strconv.Itoa(totalNoOfTablets)

			pricePerMedicineVal := totalTablets * costPerTablet
			pricePerMedicine := strconv.Itoa(pricePerMedicineVal)
			singleMedicine["pricePerMedicine"] = pricePerMedicine

			incMedicinePrice = incMedicinePrice + pricePerMedicineVal

			noOfStrips := remainingTablets / tabletsPerStrip
			updateMedicine["noOfstrips"] = noOfStrips
			updateMedicine["totalNoOfTablets"] = strconv.Itoa(remainingTablets)
		}
		if remainingTablets > 0 {
			_, err = UpdateMedicines(c, medicineId, updateMedicine)
			if err != nil {
				log.Println("Unable to update totalNoOfTablets")
				return nil, 0, errors.New("Unable to update totalNoOfTablets")
			}
		}
		log.Println("single medicine: ", singleMedicine)
		billMedicines = append(billMedicines, singleMedicine)
	}
	return billMedicines, incMedicinePrice, nil
}
func CreateBill(c *gin.Context, patientId string) (string, error) {
	patient, err := FetchPatientByCode(c, patientId)
	if err != nil {
		log.Println("Error from fetchPatientByCode: ", err)
		return "", err
	}
	err = CheckForAccess(c, patient)
	if err != nil {
		log.Println("Error from CheckForAccess: ", err)
		return "", err
	}

	latestApp, err := GetLatestAppointmentIDFromPatient(patient)
	if err != nil {
		log.Println("Error from getLatestAppointmentIDFromPatient: ", err)
		return "", err
	}
	log.Println("latestAppId: ", latestApp)
	appointment, err := FetchAppointmentByCode(c, latestApp)
	if err != nil {
		log.Println("Error from fetchAppointmentByCode: ", err)
		return "", err
	}
	medicalIdVal, exists := appointment["medicalId"]
	if !exists {
		log.Println("Unable to fetch medicalId from appointment")
		return "", errors.New("Unable to fetch medicalId from appointment")
	}
	medicalId, ok := medicalIdVal.(string)
	if !ok {
		log.Println("Type assertion error for medicalId from appointment")
		return "", errors.New("Type assertion error for medicalId from appointment")
	}
	medicalRecord, err := FetchMedicalRecordByCode(c, medicalId)
	if err != nil {
		log.Println("Error from fetchMedicalRecordByCode: ", err)
		return "", err
	}
	// tests, err := FetchTestsFromMedicalRecord(medicalRecord)
	// if err != nil {
	// 	log.Println("Error from fetchTestsFromMedicalRecord: ", err)
	// 	return "", err
	// }
	// log.Println("tests: ", tests)
	// var billTests []map[string]interface{}
	// var incTestPrice int
	// for _, t := range tests {
	// 	test, err := FetchTestByCode(c, t)
	// 	if err != nil {
	// 		log.Println("Error from fetchTestByCode: ", err)
	// 		return "", err
	// 	}
	// 	billTest := make(map[string]interface{})
	// 	priceVal, ok := test["price"].(string)
	// 	if !ok {
	// 		log.Println("Unable to get price from single test")
	// 		return "", errors.New("Unable to get price from single test")
	// 	}
	// 	price, _ := strconv.Atoi(priceVal)
	// 	billTest["testId"] = t
	// 	billTest["price"] = priceVal
	// 	incTestPrice = incTestPrice + price
	// 	billTests = append(billTests, billTest)
	// }
	billTests, incTestPrice, err := GenerateBillForTests(c, medicalRecord)
	if err != nil {
		log.Println("Error from generateBillForTests: ", err)
		return "", err
	}
	billMedicines, incMedicinePrice, err := GenerateBillForMedicines(c, medicalRecord)
	if err != nil {
		log.Println("Error from generateBillFromMedicines: ", err)
		return "", err
	}
	bill := bson.M{}
	bill["medicines"] = billMedicines
	bill["tests"] = billTests
	bill["amountForTests"] = strconv.Itoa(incTestPrice)
	bill["amountForMedicine"] = strconv.Itoa(incMedicinePrice)
	bill["amount"] = strconv.Itoa(incMedicinePrice + incTestPrice)
	code, err := GenerateEmpCode(BillCollection)
	if err != nil {
		log.Println("Error from generateEmpCode: ", err)
		return "", err
	}
	bill["code"] = code
	pharmacistId, err := GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from GetFromContext: ", err)
		return "", err
	}
	bill["createdBy"] = pharmacistId
	bill["updatedBy"] = pharmacistId
	bill["createdAt"] = time.Now()
	bill["updatedAt"] = time.Now()
	collection := db.OpenCollections(BillCollection)
	inserted, err := db.CreateOne(c, collection, bill)
	if err != nil {
		log.Println("Error from createOne: ", err)
		return "", err
	}
	log.Println("inserted: ", inserted.InsertedID)
	key := util.BillKey + code
	err = redis.SetCache(c, key, bill)
	if err != nil {
		log.Println("Error while setting cache")
	}
	return "created successfully", nil
}
