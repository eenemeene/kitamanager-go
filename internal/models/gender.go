package models

// Gender represents a person's gender
type Gender string

const (
	GenderMale    Gender = "male"
	GenderFemale  Gender = "female"
	GenderDiverse Gender = "diverse"
)

// ValidGenders contains all valid gender values
var ValidGenders = []Gender{GenderMale, GenderFemale, GenderDiverse}

// IsValidGender checks if a gender value is valid
func IsValidGender(g string) bool {
	for _, valid := range ValidGenders {
		if string(valid) == g {
			return true
		}
	}
	return false
}
