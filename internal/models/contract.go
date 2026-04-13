package models

import (
	"slices"
	"time"
)

// ContractProperties represents the JSON properties for a contract.
// Keys are property categories (e.g., "care_type", "supplements").
// Values can be strings (scalar) or []string (array).
type ContractProperties map[string]any

// BaseContract contains fields shared by all contract types.
// This is embedded by ChildContract and EmployeeContract.
type BaseContract struct {
	Period
	SectionID uint     `gorm:"not null;index" json:"section_id"`
	Section   *Section `gorm:"foreignKey:SectionID" json:"section,omitempty"`
	// Properties stores flexible key-value data as JSON.
	// For children: {"care_type": "ganztag", "supplements": ["ndh", "mss"]}
	// For employees: {"benefits": ["christmas_bonus"], "employer_type": "normal"}
	Properties ContractProperties `gorm:"serializer:json" json:"properties,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// GetScalarProperty returns a scalar (string) property value.
// Returns empty string if not found or wrong type.
func (p ContractProperties) GetScalarProperty(key string) string {
	if p == nil {
		return ""
	}
	if val, ok := p[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// GetArrayProperty returns an array property value.
// Returns nil if not found or wrong type.
func (p ContractProperties) GetArrayProperty(key string) []string {
	if p == nil {
		return nil
	}
	val, ok := p[key]
	if !ok {
		return nil
	}

	// Handle []string directly
	if arr, ok := val.([]string); ok {
		return arr
	}

	// Handle []any (from JSON unmarshaling)
	if arr, ok := val.([]any); ok {
		result := make([]string, 0, len(arr))
		for _, v := range arr {
			if str, ok := v.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}

	return nil
}

// HasValue checks if a property has a specific value under a specific key.
// Works for both scalar (exact match) and array (contains) properties.
func (p ContractProperties) HasValue(key, value string) bool {
	if p == nil {
		return false
	}

	val, ok := p[key]
	if !ok {
		return false
	}

	return p.valueMatches(val, value)
}

// ContainsValue checks if any property contains the specified value.
// This allows flexible storage where attributes can be stored as {"attrName": "attrName"}.
func (p ContractProperties) ContainsValue(value string) bool {
	if p == nil {
		return false
	}

	for _, val := range p {
		if p.valueMatches(val, value) {
			return true
		}
	}
	return false
}

// valueMatches checks if a property value matches the target.
func (p ContractProperties) valueMatches(val any, target string) bool {
	// Check scalar match
	if str, ok := val.(string); ok {
		return str == target
	}

	// Check array contains
	if arr, ok := val.([]string); ok {
		return slices.Contains(arr, target)
	}

	// Handle []any (from JSON unmarshaling)
	if arr, ok := val.([]any); ok {
		for _, v := range arr {
			if str, ok := v.(string); ok && str == target {
				return true
			}
		}
	}

	return false
}

// MergeDefaults merges default properties into the contract properties.
// Properties already set on the contract take precedence — defaults never overwrite.
// This is used to auto-apply funding properties marked with ApplyToAllContracts.
func (p ContractProperties) MergeDefaults(defaults ContractProperties) ContractProperties {
	if len(defaults) == 0 {
		return p
	}
	merged := p
	if merged == nil {
		merged = make(ContractProperties, len(defaults))
	}
	for key, val := range defaults {
		if _, exists := merged[key]; !exists {
			merged[key] = val
		}
	}
	return merged
}

// GetAllValues returns all values for a property as a string slice.
// For scalar properties, returns a slice with one element.
// For array properties, returns all elements.
func (p ContractProperties) GetAllValues(key string) []string {
	if p == nil {
		return nil
	}

	val, ok := p[key]
	if !ok {
		return nil
	}

	// Scalar - return single-element slice
	if str, ok := val.(string); ok {
		return []string{str}
	}

	// Array
	return p.GetArrayProperty(key)
}
