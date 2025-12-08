package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"github.com/gin-gonic/gin"
)

func BuildBillingData(c *gin.Context, patient map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	patientName := patient["name"].(string)
	patientID := patient["code"].(string)
	patientemail := patient["email"].(string)
	age := toInt(patient["age"])
	gender := patient["gender"].(string)
	admissionDate := getString(patient["admissionDate"])
	phone := getString(patient["phoneNo"])

	rawIDs, ok := patient["appointments"].([]interface{})
	if !ok || len(rawIDs) == 0 {
		return nil, errors.New("no appointment IDs found")
	}

	appointmentID := rawIDs[len(rawIDs)-1].(string)
	appointment, err := FetchAppointmentByCode(c, appointmentID)
	if err != nil {
		return nil, err
	}
	//   appointment["isPr"]

	hospital, err := FetchHospitalByCode(c, appointment["hospitalId"].(string))
	if err != nil {
		return nil, err
	}

	totalTests := 0
	totalMeds := 0

	if testReports, ok := appointment["testReports"].([]interface{}); ok {
		for _, tr := range testReports {
			testID := tr.(string)
			t, err := FetchTestReportByCode(c, testID)
			if err != nil {
				return nil, err
			}
			v, _ := strconv.Atoi(t["price"].(string))
			totalTests += v
		}
	}

	var medications []map[string]interface{}
	if presID, ok := appointment["prescriptionId"].(string); ok && presID != "" {
		pres, err := FetchPrescriptionByCode(c, presID)
		if err != nil {
			return nil, err
		}

		rawMeds := pres["medications"].([]interface{})
		for _, m := range rawMeds {
			med := m.(map[string]interface{})
			medications = append(medications, med)
		}

		pval, _ := strconv.Atoi(pres["price"].(string))
		totalMeds += pval
	}

	grandTotal := totalTests + totalMeds

	logo, _ := ImageToBase64("/home/adityakadambala/Desktop/hh360/HealthHub360/images/smalllogo.jpg")
	qr, _ := ImageToBase64("/home/adityakadambala/Desktop/hh360/HealthHub360/images/qrcode.png")
	link, err := CreateRazorpayPaymentLink(2000, patientName, patientemail, phone)
	if err != nil {
		return nil, err
	}
	url, err := GenerateQRCode(link)
	if err != nil {
		return nil, err
	}
	result["HospitalLogo"] = template.URL(logo)
	result["Barcode"] = template.URL(qr)

	result["HospitalName"] = hospital["name"]
	result["HospitalAddress"] = hospital["address"]
	result["HospitalContact"] = hospital["phoneNo"]

	result["PatientName"] = patientName
	result["PatientID"] = patientID
	result["Age"] = age
	result["Gender"] = gender
	result["Phone"] = phone
	result["AdmissionDate"] = admissionDate

	result["totalTests"] = totalTests
	result["totalMeds"] = totalMeds
	result["grandTotal"] = grandTotal

	result["AccountNumber"] = "41330117270"
	result["IFSC"] = "SBIN0011224"
	result["Bank"] = "STATE BANK OF INDIA"
	result["QRLink"] = template.URL(url)
	return result, nil
}

func GenerateBillingPDF(data map[string]interface{}, htmlPath string, pdfPath string) error {
	// funcMap := template.FuncMap{
	//  "add": func(a, b int) int { return a + b },
	// }
	tmpl, err := template.ParseFiles("./templates/billings.html")
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	if err := os.WriteFile(htmlPath, buf.Bytes(), 0644); err != nil {
		return err
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
		return errors.New(stderr.String())
	}

	return nil
}

/*
The Main Function starts here
*/
func GenerateBillingReport(c *gin.Context, patientCode string) ([]string, error) {
	patient, err := FetchPatientByCode(c, patientCode)
	if err != nil {
		return nil, err
	}

	data, err := BuildBillingData(c, patient)
	if err != nil {
		return nil, err
	}

	name := patient["name"].(string)

	htmlPath := fmt.Sprintf("bill_%s.html", patientCode)
	pdfPath := fmt.Sprintf("%s_bill.pdf", name)

	err = GenerateBillingPDF(data, htmlPath, pdfPath)
	if err != nil {
		return nil, err
	}

	return []string{pdfPath}, nil
}
func CreateRazorpayPaymentLink(amount int, name, email, phone string) (string, error) {

	key := "rzp_test_Rp2NNdbZg1oaD3"
	secret := "gIgHjpFH3Dag1PR97gXyGtOb"

	url := "https://api.razorpay.com/v1/payment_links"

	payload := map[string]interface{}{
		"amount":      amount * 100,
		"currency":    "INR",
		"description": "Hospital Bill Payment",
		"customer": map[string]interface{}{
			"name":    name,
			"email":   email,
			"contact": phone,
		},
		"notify": map[string]bool{
			"sms":   true,
			"email": true,
		},
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.SetBasicAuth(key, secret)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	fmt.Println("Razorpay Response:", result)

	if short, ok := result["short_url"].(string); ok {
		return short, nil
	}

	if errMsg, ok := result["error"].(map[string]interface{}); ok {
		return "", fmt.Errorf("razorpay error: %v", errMsg["description"])
	}

	return "", errors.New("failed to create payment link")
}

func GenerateQRCode(data string) (string, error) {
	qrURL := "https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=" + data

	resp, err := http.Get(qrURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	qrBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	base64QR := base64.StdEncoding.EncodeToString(qrBytes)

	return "data:image/png;base64," + base64QR, nil
}
