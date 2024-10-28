package types

import (
	"encoding/json"
	"fmt"

	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/crypto"
	"github.com/scripttoken/script/core"
)

type LicenseWrapper struct {
	core.License
}

// LicenseJSON struct represents the JSON format of the License.
type LicenseJSON struct {
	Issuer    common.Address    `json:"issuer"`    // Issuer's address
	Licensee  common.Address    `json:"licensee"`  // Licensee's address
	From      *common.JSONBig    `json:"from"`      // Start time (unix timestamp)
	To        *common.JSONBig    `json:"to"`        // End time (unix timestamp)
	Items     []string          `json:"items"`     // Items covered by the license
	Signature *crypto.Signature `json:"signature"` // Signature of the license
}

// NewLicenseJSON creates a new LicenseJSON from a License.
func NewLicenseJSON(l core.License) LicenseJSON {
	return LicenseJSON{
		Issuer:    l.Issuer,
		Licensee:  l.Licensee,
		From:      (*common.JSONBig)(l.From),
		To:        (*common.JSONBig)(l.To),
		Items:     l.Items,
		Signature: l.Signature,
	}
}

// License returns a License from LicenseJSON.
func (l LicenseJSON) License() core.License {
	return core.License{
		Issuer:    l.Issuer,
		Licensee:  l.Licensee,
		From:      l.From.ToInt(),
		To:        l.To.ToInt(),
		Items:     l.Items,
		Signature: l.Signature,
	}
}

// MarshalJSON marshals the License to JSON format.
func (lw LicenseWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewLicenseJSON(lw.License))
}

// UnmarshalJSON unmarshals the License from JSON format.
func (lw *LicenseWrapper) UnmarshalJSON(data []byte) error {
	var lJSON LicenseJSON
	if err := json.Unmarshal(data, &lJSON); err != nil {
		return err
	}
	lw.License = lJSON.License()
	return nil
}

// String method for displaying License information.
func (lw LicenseWrapper) String() string {
	return fmt.Sprintf("License{Issuer: %v, Licensee: %v, From: %v, To: %v, Items: %v, Signature: %v}",
		lw.License.Issuer, lw.License.Licensee, lw.License.From, lw.License.To, lw.License.Items, lw.License.Signature)
}
