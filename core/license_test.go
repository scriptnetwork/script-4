package core 

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/scripttoken/script/crypto"
	"github.com/scripttoken/script/crypto/bls"
)

func setupTestLicenses() {

	license := License{
		Issuer:    ,
		Licensee:  ,
		From:      big.NewInt(1696128000), // Example Unix timestamp
		To:        big.NewInt(1698720000), // Example Unix timestamp
		Signature: ,
		Items:     []string{"VN"},
	}

	licenseMap = make(map[crypto.PublicKey]License)
	licenseMap[license.Licensee] = license
}

func TestReadFile(t *testing.T) {
	tempFile, err := os.CreateTemp("", "licenses.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	licenses := []License{
		{
			Issuer:    /* Your issuer public key */,
			Licensee:  /* Your licensee public key */,
			From:      big.NewInt(1696128000),
			To:        big.NewInt(1698720000),
			Signature: /* Your BLS signature */,
			Items:     []string{"VN"},
		},
	}

	data, _ := json.Marshal(licenses)
	if _, err := tempFile.Write(data); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	tempFile.Close()
	err = ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if len(licenseMap) == 0 {
		t.Fatal("LicenseMap should not be empty after reading the file")
	}
}

func TestValidateLicense_Valid(t *testing.T) {

	setupTestLicenses()
	var validLicensee crypto.PublicKey

	err := ValidateLicense(validLicensee)
	if err != nil {
		t.Fatalf("expected license to be valid, got error: %v", err)
	}
}

func TestValidateLicense_Invalid(t *testing.T) {
	setupTestLicenses()

	var invalidLicensee crypto.PublicKey

	err := ValidateLicense(invalidLicensee)
	if err == nil {
		t.Fatal("expected error for non-existent license, got none")
	}
}

func TestValidateLicense_Expired(t *testing.T) {
	license := License{
		Issuer:    /* Your issuer public key */,
		Licensee:  /* Your licensee public key */,
		From:      big.NewInt(1696128000), // Set From to a past timestamp
		To:        big.NewInt(1696128001), // Set To to just after From
		Signature: /* Your BLS signature */,
		Items:     []string{"VN"},
	}

	licenseMap = make(map[crypto.PublicKey]License)
	licenseMap[license.Licensee] = license

	err := ValidateLicense(license.Licensee)
	if err == nil {
		t.Fatal("expected error for expired license, got none")
	}

	if err.Error() != "current time is outside the valid license period" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

