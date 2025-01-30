package core

import "github.com/scripttoken/script/common"

// License represents a license with a holder's address.
type License struct {
	Holder common.Address // Address of the license holder
}

// LicenseSet represents a collection of licenses.
type LicenseSet struct {
	Licenses []License // Slice of licenses
}

// Len returns the number of licenses in the set.
func (ls *LicenseSet) Len() int {
	return len(ls.Licenses)
}

