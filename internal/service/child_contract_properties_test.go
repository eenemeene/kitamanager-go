package service

import (
	"strings"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// berlinShapedPeriod mirrors the real configuration closely enough for the
// rules to mean what they mean there: several values under one key, a second
// key with its own values, and the universal deduction.
func berlinShapedPeriod(requiredKeys ...string) *models.GovernmentFundingPeriod {
	return &models.GovernmentFundingPeriod{
		ID:           1,
		RequiredKeys: requiredKeys,
		Properties: []models.GovernmentFundingProperty{
			{Key: "care_type", Value: "ganztag"},
			{Key: "care_type", Value: "halbtag"},
			{Key: "care_type", Value: "teilzeit"},
			{Key: "integration", Value: "integration a"},
			{Key: "parent", Value: "meals", ApplyToAllContracts: true},
		},
	}
}

func fieldPaths(t *testing.T, err error) []string {
	t.Helper()
	if err == nil {
		return nil
	}
	// The message carries the paths; asserting on them keeps the test readable
	// without reaching into apperror's internals.
	var paths []string
	for _, p := range []string{"properties.care_type", "properties.integration", "properties.made_up"} {
		if strings.Contains(err.Error(), p) {
			paths = append(paths, p)
		}
	}
	return paths
}

func TestValidateContractProperties(t *testing.T) {
	tests := []struct {
		name    string
		props   models.ContractProperties
		period  *models.GovernmentFundingPeriod
		wantErr bool
		wantIn  string
	}{
		// The four shapes the live API accepted with 201.
		{
			name:    "array value is rejected",
			props:   models.ContractProperties{"care_type": []any{"ganztag", "halbtag"}, "parent": "meals"},
			period:  berlinShapedPeriod("care_type"),
			wantErr: true,
			wantIn:  "properties.care_type",
		},
		{
			name:    "misspelled value is rejected",
			props:   models.ContractProperties{"care_type": "ganztagg", "parent": "meals"},
			period:  berlinShapedPeriod("care_type"),
			wantErr: true,
			wantIn:  "properties.care_type",
		},
		{
			name:    "wrong case is rejected",
			props:   models.ContractProperties{"care_type": "Ganztag", "parent": "meals"},
			period:  berlinShapedPeriod("care_type"),
			wantErr: true,
			wantIn:  "properties.care_type",
		},
		{
			name:    "missing required key is rejected",
			props:   models.ContractProperties{"parent": "meals"},
			period:  berlinShapedPeriod("care_type"),
			wantErr: true,
			wantIn:  "properties.care_type",
		},
		{
			name:    "nested object under a key is rejected",
			props:   models.ContractProperties{"care_type": "ganztag", "made_up": map[string]any{"a": 1}, "parent": "meals"},
			period:  berlinShapedPeriod("care_type"),
			wantErr: true,
			wantIn:  "properties.made_up",
		},

		// Valid shapes must keep working.
		{
			name:   "the ordinary contract passes",
			props:  models.ContractProperties{"care_type": "ganztag", "parent": "meals"},
			period: berlinShapedPeriod("care_type"),
		},
		{
			name:   "supplements alongside the base rate pass",
			props:  models.ContractProperties{"care_type": "halbtag", "integration": "integration a", "parent": "meals"},
			period: berlinShapedPeriod("care_type"),
		},
		{
			name:   "a single-element array is still one value",
			props:  models.ContractProperties{"care_type": []any{"ganztag"}, "parent": "meals"},
			period: berlinShapedPeriod("care_type"),
		},

		// The generalisation: the rule follows the configuration, not the code.
		{
			name:   "a period requiring nothing accepts a contract with no care_type",
			props:  models.ContractProperties{"parent": "meals"},
			period: berlinShapedPeriod(),
		},
		{
			name:    "a period requiring a different key enforces that one instead",
			props:   models.ContractProperties{"care_type": "ganztag", "parent": "meals"},
			period:  berlinShapedPeriod("integration"),
			wantErr: true,
			wantIn:  "properties.integration",
		},
		{
			name:   "no configuration for the date checks nothing",
			props:  models.ContractProperties{"care_type": "anything at all"},
			period: nil,
		},
		{
			name:   "empty properties against no configuration is fine",
			props:  nil,
			period: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContractProperties(tt.props, tt.period, "")
			if tt.wantErr && err == nil {
				t.Fatalf("expected rejection, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected acceptance, got %v", err)
			}
			if tt.wantIn != "" && !strings.Contains(err.Error(), tt.wantIn) {
				t.Errorf("error should name %q so the form can mark the field; got %v", tt.wantIn, err)
			}
		})
	}
}

// A contract with several problems must report all of them, so a caller fixing
// one does not discover the next on the following round trip.
func TestValidateContractProperties_ReportsEveryViolation(t *testing.T) {
	err := validateContractProperties(models.ContractProperties{
		"care_type": []any{"ganztag", "halbtag"},
		"made_up":   map[string]any{"deeply": "nested"},
	}, berlinShapedPeriod("care_type"), "")
	if err == nil {
		t.Fatal("expected rejection")
	}
	paths := fieldPaths(t, err)
	if len(paths) != 2 {
		t.Errorf("expected both care_type and made_up reported, got %v from %v", paths, err)
	}
}

// The message has to be actionable: naming the field is not enough if the
// caller cannot tell what to put there.
func TestValidateContractProperties_MissingKeyNamesTheChoices(t *testing.T) {
	err := validateContractProperties(models.ContractProperties{"parent": "meals"}, berlinShapedPeriod("care_type"), "")
	if err == nil {
		t.Fatal("expected rejection")
	}
	for _, want := range []string{"ganztag", "halbtag", "teilzeit"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should list the permitted values so the caller knows what to send; %q missing from %v", want, err)
		}
	}
}
