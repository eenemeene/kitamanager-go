package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeDefaults(t *testing.T) {
	t.Run("nil properties + defaults returns defaults", func(t *testing.T) {
		var p ContractProperties
		defaults := ContractProperties{"parent": "meals"}

		result := p.MergeDefaults(defaults)
		assert.Equal(t, ContractProperties{"parent": "meals"}, result)
	})

	t.Run("existing properties + non-overlapping defaults merges both", func(t *testing.T) {
		p := ContractProperties{"care_type": "ganztag"}
		defaults := ContractProperties{"parent": "meals"}

		result := p.MergeDefaults(defaults)
		assert.Equal(t, "ganztag", result["care_type"])
		assert.Equal(t, "meals", result["parent"])
	})

	t.Run("existing key conflicts with default — existing wins", func(t *testing.T) {
		p := ContractProperties{"parent": "custom_value"}
		defaults := ContractProperties{"parent": "meals"}

		result := p.MergeDefaults(defaults)
		assert.Equal(t, "custom_value", result["parent"])
	})

	t.Run("empty defaults returns original unchanged", func(t *testing.T) {
		p := ContractProperties{"care_type": "ganztag"}

		result := p.MergeDefaults(nil)
		assert.Equal(t, ContractProperties{"care_type": "ganztag"}, result)
	})

	t.Run("nil properties + nil defaults returns nil", func(t *testing.T) {
		var p ContractProperties
		result := p.MergeDefaults(nil)
		assert.Nil(t, result)
	})

	t.Run("empty properties + defaults returns merged", func(t *testing.T) {
		p := ContractProperties{}
		defaults := ContractProperties{"parent": "meals"}

		result := p.MergeDefaults(defaults)
		assert.Equal(t, "meals", result["parent"])
	})
}

func TestContractProperties_GetScalarProperty(t *testing.T) {
	tests := []struct {
		name  string
		props ContractProperties
		key   string
		want  string
	}{
		{"nil properties", nil, "key", ""},
		{"missing key", ContractProperties{"other": "val"}, "key", ""},
		{"scalar value", ContractProperties{"care_type": "ganztag"}, "care_type", "ganztag"},
		{"array value returns empty", ContractProperties{"tags": []string{"a", "b"}}, "tags", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.props.GetScalarProperty(tt.key); got != tt.want {
				t.Errorf("GetScalarProperty(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestContractProperties_GetArrayProperty(t *testing.T) {
	t.Run("nil properties", func(t *testing.T) {
		var p ContractProperties
		if got := p.GetArrayProperty("key"); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("missing key", func(t *testing.T) {
		p := ContractProperties{"other": "val"}
		if got := p.GetArrayProperty("key"); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("string slice", func(t *testing.T) {
		p := ContractProperties{"tags": []string{"ndh", "mss"}}
		got := p.GetArrayProperty("tags")
		if len(got) != 2 || got[0] != "ndh" || got[1] != "mss" {
			t.Errorf("got %v, want [ndh mss]", got)
		}
	})

	t.Run("interface slice from JSON", func(t *testing.T) {
		p := ContractProperties{"tags": []any{"ndh", "mss"}}
		got := p.GetArrayProperty("tags")
		if len(got) != 2 || got[0] != "ndh" || got[1] != "mss" {
			t.Errorf("got %v, want [ndh mss]", got)
		}
	})

	t.Run("scalar value returns nil", func(t *testing.T) {
		p := ContractProperties{"care_type": "ganztag"}
		if got := p.GetArrayProperty("care_type"); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
}

func TestContractProperties_HasValue(t *testing.T) {
	tests := []struct {
		name  string
		props ContractProperties
		key   string
		value string
		want  bool
	}{
		{"nil properties", nil, "k", "v", false},
		{"missing key", ContractProperties{"other": "val"}, "k", "v", false},
		{"scalar match", ContractProperties{"care_type": "ganztag"}, "care_type", "ganztag", true},
		{"scalar no match", ContractProperties{"care_type": "ganztag"}, "care_type", "halbtag", false},
		{"array contains", ContractProperties{"tags": []string{"ndh", "mss"}}, "tags", "mss", true},
		{"array not contains", ContractProperties{"tags": []string{"ndh", "mss"}}, "tags", "xyz", false},
		{"interface array contains", ContractProperties{"tags": []any{"ndh", "mss"}}, "tags", "ndh", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.props.HasValue(tt.key, tt.value); got != tt.want {
				t.Errorf("HasValue(%q, %q) = %v, want %v", tt.key, tt.value, got, tt.want)
			}
		})
	}
}

func TestContractProperties_ContainsValue(t *testing.T) {
	tests := []struct {
		name  string
		props ContractProperties
		value string
		want  bool
	}{
		{"nil properties", nil, "v", false},
		{"scalar match in any key", ContractProperties{"a": "x", "b": "y"}, "y", true},
		{"no match", ContractProperties{"a": "x"}, "z", false},
		{"match in array", ContractProperties{"tags": []string{"ndh", "mss"}}, "mss", true},
		{"match in interface array", ContractProperties{"tags": []any{"ndh"}}, "ndh", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.props.ContainsValue(tt.value); got != tt.want {
				t.Errorf("ContainsValue(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestContractProperties_GetAllValues(t *testing.T) {
	t.Run("nil properties", func(t *testing.T) {
		var p ContractProperties
		if got := p.GetAllValues("key"); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("missing key", func(t *testing.T) {
		p := ContractProperties{"other": "val"}
		if got := p.GetAllValues("key"); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("scalar returns single-element slice", func(t *testing.T) {
		p := ContractProperties{"care_type": "ganztag"}
		got := p.GetAllValues("care_type")
		if len(got) != 1 || got[0] != "ganztag" {
			t.Errorf("got %v, want [ganztag]", got)
		}
	})

	t.Run("array returns all elements", func(t *testing.T) {
		p := ContractProperties{"tags": []string{"ndh", "mss"}}
		got := p.GetAllValues("tags")
		if len(got) != 2 {
			t.Errorf("got %d elements, want 2", len(got))
		}
	})
}
