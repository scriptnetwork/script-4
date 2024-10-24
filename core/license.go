package core

import (
	"math/big"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
	"github.com/scripttoken/script/crypto/bls"
	"github.com/scripttoken/script/common"
)

type License struct {
	Issuer    common.Address   // Issuer's address
	Licensee  common.Address   // Licensee's address
	From      *big.Int        // Start time (unix timestamp)
	To        *big.Int        // End time (unix timestamp)
	Items     []string        // Items covered by the license
	Signature *bls.Signature   // Signature of the license
}

// package-level variable to store the license map
var licenseMap = make(map[common.Address]License)

// read license file
func ReadFile(filename string) (map[common.Address]License, error) {

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file: %v", err)
	}

	var licenses []License

	err = json.Unmarshal(bytes, &licenses)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal JSON: %v", err)
	}

	licenseMap = make(map[common.Address]License) // clear previous map

	for _, license := range licenses {
		licenseMap[license.Licensee] = license
	}

	return licenseMap, nil
}

func WriteLicenseFile(license License, filename string) error {
	err := ValidateIncomingLicense(license)
	if err != nil {
		return fmt.Errorf("license validation failed: %v", err)
	}

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open license file: %v", err)
	}
	defer file.Close()

	licenseJSON, err := json.Marshal(license)
	if err != nil {
		return fmt.Errorf("failed to marshal license to JSON: %v", err)
	}

	_, err = file.Write(licenseJSON)
	if err != nil {
		return fmt.Errorf("failed to write license to file: %v", err)
	}

	_, err = file.WriteString("\n")
	if err != nil {
		return fmt.Errorf("failed to write newline to file: %v", err)
	}

	return nil
}

func ValidateIncomingLicense(license License) error {
	currentTime := big.NewInt(time.Now().Unix())

	if license.From.Cmp(currentTime) > 0 || license.To.Cmp(currentTime) < 0 {
		return fmt.Errorf("current time is outside the valid license period")
	}

	if !isLicenseForValidatorNode(license.Items) {
		return fmt.Errorf("license items do not include 'VN'")
	}

	dataToSign := concatenateLicenseData(license)

	valid := bls.Verify(license.Issuer, dataToSign, license.Signature)
	if !valid {
		return fmt.Errorf("invalid license signature")
	}

	return nil
}

// validate license for a public key
func ValidateLicense(licensee common.Address) error {
	license, exists := licenseMap[licensee]
	if !exists {
		return fmt.Errorf("No license found for the given licensee public key")
	}

	currentTime := big.NewInt(time.Now().Unix())

	if license.From.Cmp(currentTime) > 0 || license.To.Cmp(currentTime) < 0 {
		return fmt.Errorf("current time is outside the valid license period")
	}

	dataToSign := concatenateLicenseData(license)

	valid := bls.Verify(license.Issuer, dataToSign, license.Signature)
	if !valid {
		return fmt.Errorf("invalid license signature")
	}

	// valid license
	return nil
}

func isLicenseForValidatorNode(items []string) bool {
	for _, item := range items {
		if item == "VN" {
			return true
		}
	}
	return false
}

func concatenateLicenseData(license License) []byte {
	// Convert fields to byte slices or strings
	issuerBytes := []byte(license.Issuer.Hex())               
	licenseeBytes := []byte(license.Licensee.Hex())           
	fromBytes := license.From.Bytes()                        
	toBytes := license.To.Bytes()                             

	// Concatenate the items list (assuming it's strings)
	itemsBytes := []byte{}
	for _, item := range license.Items {
		itemsBytes = append(itemsBytes, []byte(item)...)
	}

	// Concatenate all data into a single byte slice
	concatenatedData := append(issuerBytes, licenseeBytes...)
	concatenatedData = append(concatenatedData, fromBytes...)
	concatenatedData = append(concatenatedData, toBytes...)
	concatenatedData = append(concatenatedData, itemsBytes...)

	return concatenatedData
}

