package models

import "testing"

func TestIsValidStaffCategory(t *testing.T) {
	t.Parallel()
	validCategories := []string{"qualified", "supplementary", "non_pedagogical"}
	for _, cat := range validCategories {
		if !IsValidStaffCategory(cat) {
			t.Errorf("IsValidStaffCategory(%q) = false, want true", cat)
		}
	}
}

func TestIsValidStaffCategory_Invalid(t *testing.T) {
	t.Parallel()
	invalidCategories := []string{"", "invalid", "teacher", "QUALIFIED", "Qualified", "fachkraft", "ergaenzungskraft", "non-pedagogical", "non pedagogical"}
	for _, cat := range invalidCategories {
		if IsValidStaffCategory(cat) {
			t.Errorf("IsValidStaffCategory(%q) = true, want false", cat)
		}
	}
}

func TestStaffCategory_Constants(t *testing.T) {
	t.Parallel()
	if string(StaffCategoryQualified) != "qualified" {
		t.Errorf("StaffCategoryQualified = %q, want %q", StaffCategoryQualified, "qualified")
	}
	if string(StaffCategorySupplementary) != "supplementary" {
		t.Errorf("StaffCategorySupplementary = %q, want %q", StaffCategorySupplementary, "supplementary")
	}
	if string(StaffCategoryNonPedagogical) != "non_pedagogical" {
		t.Errorf("StaffCategoryNonPedagogical = %q, want %q", StaffCategoryNonPedagogical, "non_pedagogical")
	}
}
