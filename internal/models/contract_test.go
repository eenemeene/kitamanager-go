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
