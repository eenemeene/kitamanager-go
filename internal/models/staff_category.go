package models

// StaffCategory represents the category of staff for an employee contract.
// Used to distinguish pedagogical from non-pedagogical staff for ratio calculations.
type StaffCategory string

const (
	StaffCategoryQualified      StaffCategory = "qualified"       // DE: Fachkraft - fully qualified pedagogical staff (Erzieher, Sozialpadagoge, etc.)
	StaffCategorySupplementary  StaffCategory = "supplementary"   // DE: Erganzungskraft - supplementary pedagogical staff (Kinderpfleger, Sozialassistent, etc.)
	StaffCategoryNonPedagogical StaffCategory = "non_pedagogical" // DE: Nicht-padagogisch - non-pedagogical staff (kitchen, cleaning, admin, finance)
)

// ValidStaffCategories contains all valid staff category values
var ValidStaffCategories = []StaffCategory{StaffCategoryQualified, StaffCategorySupplementary, StaffCategoryNonPedagogical}

// IsValidStaffCategory checks if a staff category value is valid
func IsValidStaffCategory(s string) bool {
	for _, valid := range ValidStaffCategories {
		if string(valid) == s {
			return true
		}
	}
	return false
}
