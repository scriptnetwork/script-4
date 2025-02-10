package core

import (

	//    "time"

	"bytes"
	"sort"
	"sync"

	"github.com/scripttoken/script/common"
	//    log "github.com/sirupsen/logrus"
)

var LicenseIssuer common.Address = common.HexToAddress("0x0000")

/*
type License struct {
	Issuer    common.Address    // Issuer's address
	Licensee  common.Address    // Licensee's address
	From      *big.Int          // Start time (unix timestamp)
	To        *big.Int          // End time (unix timestamp)
	Items     []string          // Items covered by the license
	Signature *crypto.Signature // Signature of the license
}

//var logger = log.WithFields(log.Fields{"prefix": "license"})

// package-level variable to store the license map
//var licenseMap = make(map[common.Address]License)

*/

type AddressSet map[common.Address]struct{}

func (e *AddressSet) Has(addr common.Address) bool {
	if _, exists := (*e)[addr]; exists {
		return true
	}
	return false
}

func (e *AddressSet) Hash() common.Hash {
	var buffer bytes.Buffer
	for addr := range *e {
		buffer.Write(addr.Bytes())
	}
	return common.BytesToHash(buffer.Bytes())
}

func (e AddressSet) Copy() AddressSet {
	newSet := make(AddressSet)
	for addr := range e {
		newSet[addr] = e[addr]
	}
	return newSet
}

func (e *AddressSet) ToSortedSlice() []common.Address {
	slice := make([]common.Address, 0, len(*e))
	for key := range *e {
		slice = append(slice, key)
	}
	sort.Slice(slice, func(i, j int) bool {
		return bytes.Compare(slice[i].Bytes(), slice[j].Bytes()) < 0
	})
	return slice
}

/*
func (e *AddressSet) PubKeys() []*crypto.PublicKey {
	pubKeys := make([]*crypto.PublicKey, 0, len(*e))
	for _, pubKey := range *e {
		pubKeyCopy := pubKey
		pubKeys = append(pubKeys, &pubKeyCopy)
	}
	return pubKeys
}
*/

func NewAddressSet() AddressSet {
	return make(AddressSet)
}

func (this *AddressSet) Add(addr common.Address) {
	(*this)[addr] = struct{}{}
}

func (this *AddressSet) Remove(addr common.Address) {
	delete(*this, addr)
}

func (this *AddressSet) Equals(other *AddressSet) bool {
	if len(*this) != len(*other) {
		return false
	}
	for addr := range *this {
		if !other.Has(addr) {
			return false
		}
	}
	return true
}

var licenses__mx sync.RWMutex

var lightnings AddressSet = make(AddressSet)
var validators AddressSet = make(AddressSet)

func clear() {
	lightnings = make(AddressSet)
	validators = make(AddressSet)
}

func AddValidator(address common.Address) {
	licenses__mx.RLock()
	defer licenses__mx.RUnlock()
	validators.Add(address)
}

func AddLightning(address common.Address) {
	licenses__mx.RLock()
	defer licenses__mx.RUnlock()
	lightnings.Add(address)
}

func RemoveValidator(address common.Address) {
	licenses__mx.RLock()
	defer licenses__mx.RUnlock()
	validators.Remove(address)

}

func RemoveLightning(address common.Address) {
	licenses__mx.RLock()
	defer licenses__mx.RUnlock()
	lightnings.Remove(address)
}

func For_each_lightning(visitor func(common.Address)) {
	licenses__mx.RLock()
	defer licenses__mx.RUnlock()
	for addr := range lightnings {
		visitor(addr)
	}
}

func For_each_validator(visitor func(common.Address)) {
	licenses__mx.RLock()
	defer licenses__mx.RUnlock()
	for addr := range validators {
		visitor(addr)
	}
}

func IsValidator(address common.Address) bool {
	licenses__mx.RLock()
	defer licenses__mx.RUnlock()
	if _, exists := validators[address]; exists {
		return true
	}
	return false
}

func IsLightning(address common.Address) bool {
	licenses__mx.RLock()
	defer licenses__mx.RUnlock()
	if _, exists := lightnings[address]; exists {
		return true
	}
	return false
}

func Has_license_peer(address common.Address) bool {
	licenses__mx.RLock()
	defer licenses__mx.RUnlock()
	if _, exists := lightnings[address]; exists {
		return true
	}
	if _, exists := validators[address]; exists {
		return true
	}
	return false
}

/*

func verify_license(license *License, expected_issuer common.Address) string {
	if license.Issuer != expected_issuer {
		logger.Infof("Invalid License from issuer %v, expected %v", license.Issuer, expected_issuer)
		return ""
	}
	dataToValidate := concatenateLicenseData(*license)
	if !license.Signature.Verify(dataToValidate, expected_issuer) {
		return ""
	}
	x := ""
	for _, item := range license.Items {
		x += item + " "
	}
	return x
}

// read license file
//func ReadLicenses(filename string) (map[common.Address]License, error) {

func read_licenses0() error {
	license_dir := viper.GetString(common.CfgLicenseDir)
	if license_dir == "" {
		return fmt.Errorf("failed license_dir: %v", license_dir)
	}
	license_issuer := viper.GetString(common.CfgLicenseIssuer) //issuer public key
	if license_dir == "" {
		return fmt.Errorf("failed license_issuer: %v", license_issuer)
	}

	licenseFile := viper.GetString(common.CfgLicenseDir + "/license.json")
	file, err := os.Open(licenseFile)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("Failed to read file: %v", err)
	}

	var licenses []License
	err = json.Unmarshal(bytes, &licenses)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal JSON: %v", err)
	}

	//    licenseMap = make(map[common.Address]License) // clear previous map
	//    verifiedLicenseCache = make(map[common.Address]bool) // clear previous cache

	clear()

	expected_issuer := common.HexToAddress(viper.GetString(common.CfgLicenseIssuer))
	logger.Infof("Validating licenses. expected issuer %v", expected_issuer)
	for _, license := range licenses {
		x := verify_license(&license, expected_issuer)
		// Use strings.Contains to check if x contains the unwanted substrings.
		if strings.Contains(x, "LN ") || strings.Contains(x, "LN-L ") {
			lightnings[license.Licensee] = struct{}{}
			logger.Infof("Added LN %v", license.Licensee)
			continue
		}
		if strings.Contains(x, "VN ") {
			validators[license.Licensee] = struct{}{}
			logger.Infof("Added VN %v", license.Licensee)
			continue
		}
		logger.Infof("Ignored entry %v", license)
	}
	logger.Infof("Number of Validators: %v", len(validators))
	logger.Infof("Number of Lightnings: %v", len(lightnings))
	return nil //OK=nullptr KO=char*
}

func Read_licenses() error {
	licenses__mx.Lock()
	defer licenses__mx.Unlock()
	return read_licenses0()
}

func Set_licenses(licenses []License) error {
	licenses__mx.Lock()
	defer licenses__mx.Unlock()
	licenseFile := viper.GetString(common.CfgLicenseDir + "/license.json")
	file, err := os.OpenFile(licenseFile, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open license file: %v", err)
	}
	defer file.Close()
	for _, license := range licenses {
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
	}
	read_licenses0()
	return nil
}
*/

/*
func WriteLicenseFile(license License, filename string) error {
	err := ValidateIncomingLicense(license)
	if err != nil {
		return fmt.Errorf("license validation failed: %v", err)
	}

	if(filename == "") {
		filename = licenseFile
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

	if !license.Signature.Verify(dataToSign, license.Issuer) {
		return fmt.Errorf("invalid license signature")
	}

    if license.Issuer

	return nil
}

// validate license for a public key
func ValidateLicense(licensee common.Address) error {
	// Check cache first
	if verified, exists := verifiedLicenseCache[licensee]; exists {
		if verified {
			return nil // License is already verified
		} else {
			return fmt.Errorf("license is not verified")
		}
	}

	license, exists := licenseMap[licensee]
	if !exists {
		return fmt.Errorf("No license found for the given licensee public key")
	}

	currentTime := big.NewInt(time.Now().Unix())

	if license.From.Cmp(currentTime) > 0 || license.To.Cmp(currentTime) < 0 {
		verifiedLicenseCache[licensee] = false
		return fmt.Errorf("current time is outside the valid license period")
	}

	dataToValidate := concatenateLicenseData(license)

	if !license.Signature.Verify(dataToValidate, license.Issuer) {
		verifiedLicenseCache[licensee] = false
		return fmt.Errorf("invalid license signature")
	}

	// cache the verified status
	verifiedLicenseCache[licensee] = true

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

*/
/*
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
*/
/*
// periodically check and update the cache
func startCacheUpdater(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			updateCache()
		}
	}()
}

func updateCache() {
	currentTime := big.NewInt(time.Now().Unix())
	for licensee, license := range licenseMap {
		if license.From.Cmp(currentTime) > 0 || license.To.Cmp(currentTime) < 0 {
			delete(verifiedLicenseCache, licensee)
		}
	}
}
*/

func init() {
	//startCacheUpdater(1 * time.Hour)
}
