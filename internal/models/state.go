package models

import "strings"

// State represents a German Bundesland
type State string

const (
	StateBerlin State = "berlin"
	// Future states can be added here
)

// ValidStates contains all currently supported states
var ValidStates = []State{StateBerlin}

// IsValidState checks if a state is supported
func IsValidState(s string) bool {
	for _, valid := range ValidStates {
		if string(valid) == s {
			return true
		}
	}
	return false
}

// ValidStatesString returns a comma-separated list of valid states for error messages.
func ValidStatesString() string {
	strs := make([]string, len(ValidStates))
	for i, s := range ValidStates {
		strs[i] = string(s)
	}
	return strings.Join(strs, ", ")
}

// Stichtag returns the school enrollment cutoff date (month, day) for a state.
// Children who turn 6 before this date must start school that year.
func (s State) Stichtag() (month int, day int) {
	switch s {
	case StateBerlin:
		return 9, 30 // September 30th
	default:
		return 9, 30 // Default fallback
	}
}
