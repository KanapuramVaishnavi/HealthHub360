package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func toInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	default:
		return 0
	}
}
func getString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	default:
		return fmt.Sprintf("%v", x)
	}
}

func FormatedDateAndTime(input interface{}) (string, error) {
	var t time.Time

	switch v := input.(type) {

	case primitive.DateTime:
		t = v.Time()

	case primitive.Timestamp:
		t = time.Unix(int64(v.T), 0)

	case time.Time:
		t = v

	case string:
		parsed, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return "", err
		}
		t = parsed

	default:
		return "", fmt.Errorf("unsupported date type: %T", input)
	}

	dateStr := t.Format("02/01/2006")
	timeStr := t.Format("15:04")
	return dateStr + " " + timeStr, nil
}

func BuildReportData(c *gin.Context, patient map[string]interface{}) (map[string]interface{}, error) {

	patientName := patient["name"].(string)
	age := toInt(patient["age"])
	gender := patient["gender"].(string)
	patientID := patient["code"].(string)
	admissionDate := patient["admissionDate"]

	rawIDs, ok := patient["appointments"].([]interface{})
	if !ok || len(rawIDs) == 0 {
		return nil, errors.New("no appointment IDs found")
	}

	var appointments []map[string]interface{}
	for _, id := range rawIDs {
		appId := id.(string)
		app, err := FetchAppointmentByCode(c, appId)
		if err != nil {
			return nil, fmt.Errorf("failed fetching appointment %s: %v", appId, err)
		}
		appointments = append(appointments, app)
	}

	firstApp := appointments[0]

	hospital, err := FetchHospitalByCode(c, firstApp["hospitalId"].(string))
	if err != nil {
		return nil, err
	}

	primaryDoc, err := FetchDoctorByCode(c, firstApp["doctorId"].(string))
	if err != nil {
		return nil, err
	}

	doctorSeen := map[string]bool{}
	var doctorTable []map[string]interface{}
	var appointmentTable []map[string]interface{}
	var medicationTable []map[string]interface{}

	for _, app := range appointments {

		did := app["doctorId"].(string)

		if !doctorSeen[did] {
			doctorSeen[did] = true
			doc, err := FetchDoctorByCode(c, did)
			if err != nil {
				return nil, err
			}
			doctorTable = append(doctorTable, map[string]interface{}{
				"DoctorName": doc["name"],
				"DoctorID":   doc["code"],
				"Department": doc["department"],
			})
		}

		formatted, err := FormatedDateAndTime(app["createdAt"])
		if err != nil {
			return nil, err
		}
		appointmentTable = append(appointmentTable, map[string]interface{}{
			"AppointmentID":   app["code"],
			"AppointmentDate": formatted,
			"Reason":          app["reason"],
		})

		medical, err := FetchMedicalRecordByCode(c, app["medicalId"].(string))
		if err != nil {
			return nil, fmt.Errorf("failed fetching medical record for appointment %s", app["appointmentId"])
		}
		log.Println("medical: ", medical)
		prescriptionId := getString(medical["prescriptionId"])
		pres := make(map[string]interface{})
		if prescriptionId == "" {

			log.Println("No prescriptionId found for medical record:", medical)
			pres = nil
		} else {
			pres, err = FetchPrescriptionByCode(c, prescriptionId)
			if err != nil {
				log.Println("FetchPrescriptionByCode error:", err)
			}
		}
		log.Println("Hi1")

		originalPrescription, err := BuildPrescriptionData(c, pres)
		if err != nil {
			return nil, err
		}
		medicationTable = append(medicationTable, map[string]interface{}{
			"AppointmentID": app["code"],
			"Medications":   originalPrescription,
		})
	}
	logoBase64, err := ImageToBase64("https://healthhub360.s3.ap-southeast-2.amazonaws.com/smalllogo.jpg")
	if err != nil {
		log.Println("Image load error:", err)
	}
	qrcodeImage, err := ImageToBase64("https://healthhub360.s3.ap-southeast-2.amazonaws.com/qrcode.png")
	if err != nil {
		log.Println("Image load error:", err)
	}
	reportData := map[string]interface{}{
		"HospitalLogo": template.URL(logoBase64),
		"HospitalName": hospital["name"],
		"Barcode":      template.URL(qrcodeImage),

		"PatientName":   patientName,
		"Age":           age,
		"Gender":        gender,
		"PatientID":     patientID,
		"AdmissionDate": admissionDate,

		"HospitalAddress": hospital["address"],
		"HospitalContact": hospital["phoneNo"],
		"HospitalRegID":   hospital["code"],

		"PrimaryDoctorName":       primaryDoc["name"],
		"PrimaryDoctorID":         primaryDoc["code"],
		"PrimaryDoctorDepartment": primaryDoc["department"],

		"Doctors":             doctorTable,
		"AppointmentsTable":   appointmentTable,
		"MedicalRecordsTable": medicationTable,
	}

	return reportData, nil
}
func BuildPrescriptionData(c *gin.Context, prescription map[string]interface{}) ([]map[string]interface{}, error) {
	raw := prescription["medicines"]
	if raw == nil {
		return nil, fmt.Errorf("medicines is nil")
	}

	result := []map[string]interface{}{}

	switch meds := raw.(type) {

	case primitive.A: // <-- this is your real case
		for _, v := range meds {
			med, ok := v.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid medicine entry format")
			}
			result = append(result, flattenMedicine(c, med))
		}

	case []interface{}:
		for _, v := range meds {
			med := v.(map[string]interface{})
			result = append(result, flattenMedicine(c, med))
		}

	default:
		return nil, fmt.Errorf("invalid medicines format: %T", raw)
	}

	return result, nil
}

func flattenMedicine(c *gin.Context, med map[string]interface{}) map[string]interface{} {
	freq, _ := med["frequency"].(map[string]interface{})
	medicineId := med["medicineId"].(string)
	log.Println(medicineId)
	medicine, err := FetchMedicineByCode(c, medicineId)
	log.Println("Medicine is ")
	log.Println(medicine)
	if err != nil {
		log.Println("error from fetch Medicine bY code", err)
	}
	return map[string]interface{}{
		"medicineName":       medicine["name"],
		"dosagePerFrequency": med["dosagePerFrequency"],
		"morning":            freq["morning"],
		"afternoon":          freq["afternoon"],
		"night":              freq["night"],
		"instructions":       med["instructions"],
		"noOfDays":           med["noOfDays"],
	}
}

func GenerateReportToPDF(reportData map[string]interface{}, htmlPath string, pdfPath string) error {

	// Register template functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
	}

	// Parse template with function map
	tmpl, err := template.New("report.html").Funcs(funcMap).ParseFiles("./templates/report.html")
	if err != nil {
		return errors.New("template parse error: " + err.Error())
	}

	// Execute template into buffer
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, reportData); err != nil {
		return errors.New("template execute error: " + err.Error())
	}

	// Write HTML file
	if err := os.WriteFile(htmlPath, buf.Bytes(), 0644); err != nil {
		return errors.New("failed to write HTML: " + err.Error())
	}
	cmd := exec.Command(
		"wkhtmltopdf",
		"--enable-local-file-access",
		"--load-error-handling", "ignore",
		htmlPath,
		pdfPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return errors.New("wkhtmltopdf error: " + stderr.String())
	}

	return nil
}
func GenerateReport(c *gin.Context, code string) ([]string, error) {
	patient, err := FetchPatientByCode(c, code)
	if err != nil {
		return []string{}, err
	}
	log.Println("Hi")
	raw, ok := patient["appointments"]
	if !ok {
		return []string{}, errors.New("appointments field missing")
	}

	arr, ok := raw.(primitive.A)
	if !ok {
		// fallback if driver decoded differently
		if tmp, ok2 := raw.([]interface{}); ok2 {
			arr = tmp
		} else {
			return []string{}, errors.New("invalid appointment format")
		}
	}

	if len(arr) == 0 {
		return []string{}, errors.New("No appointments  ! for this patient")
	}
	patientname := patient["name"].(string)
	var generatedPDFs []string

	reportData, err := BuildReportData(c, patient)
	if err != nil {
		return []string{}, err
	}
	htmlPath := fmt.Sprintf("report_%s_.html", code)
	pdfPath := fmt.Sprintf("%s_report.pdf", patientname)
	err = GenerateReportToPDF(reportData, htmlPath, pdfPath)
	if err != nil {
		return []string{}, err
	}

	generatedPDFs = append(generatedPDFs, pdfPath)

	return generatedPDFs, nil
}

//	func ImageToBase64(path string) (string, error) {
//		data, err := ioutil.ReadFile(path)
//		if err != nil {
//			return "", err
//		}
//		encoded := base64.StdEncoding.EncodeToString(data)
//		return "data:image/jpeg;base64," + encoded, nil
//	}
func ImageToBase64(path string) (string, error) {

	// Check if it's a URL
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {

		// Download from URL
		resp, err := http.Get(path)
		if err != nil {
			return "", fmt.Errorf("failed to fetch image from URL: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return "", fmt.Errorf("non-200 response fetching image: %v", resp.Status)
		}

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed reading image bytes: %v", err)
		}

		encoded := base64.StdEncoding.EncodeToString(data)
		return "data:image/jpeg;base64," + encoded, nil
	}

	// Otherwise treat as local file path
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return "data:image/jpeg;base64," + encoded, nil
}
