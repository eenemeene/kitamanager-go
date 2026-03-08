package models

import (
	"fmt"
	"testing"
	"time"
)

func intPtr(i int) *int {
	return &i
}

func TestGovernmentFundingProperty_MatchesAge(t *testing.T) {
	tests := []struct {
		name     string
		property GovernmentFundingProperty
		age      int
		want     bool
	}{
		// No age filter (nil values)
		{
			name:     "no age filter - matches any age",
			property: GovernmentFundingProperty{MinAge: nil, MaxAge: nil},
			age:      5,
			want:     true,
		},
		{
			name:     "no age filter - matches age 0",
			property: GovernmentFundingProperty{MinAge: nil, MaxAge: nil},
			age:      0,
			want:     true,
		},

		// Only MinAge set
		{
			name:     "only min age - below min",
			property: GovernmentFundingProperty{MinAge: intPtr(3), MaxAge: nil},
			age:      2,
			want:     false,
		},
		{
			name:     "only min age - at min (inclusive)",
			property: GovernmentFundingProperty{MinAge: intPtr(3), MaxAge: nil},
			age:      3,
			want:     true,
		},
		{
			name:     "only min age - above min",
			property: GovernmentFundingProperty{MinAge: intPtr(3), MaxAge: nil},
			age:      10,
			want:     true,
		},

		// Only MaxAge set
		{
			name:     "only max age - below max",
			property: GovernmentFundingProperty{MinAge: nil, MaxAge: intPtr(6)},
			age:      3,
			want:     true,
		},
		{
			name:     "only max age - at max (inclusive)",
			property: GovernmentFundingProperty{MinAge: nil, MaxAge: intPtr(6)},
			age:      6,
			want:     true,
		},
		{
			name:     "only max age - above max",
			property: GovernmentFundingProperty{MinAge: nil, MaxAge: intPtr(6)},
			age:      7,
			want:     false,
		},

		// Both MinAge and MaxAge set - inclusive range tests
		{
			name:     "range [0,2] - below range",
			property: GovernmentFundingProperty{MinAge: intPtr(0), MaxAge: intPtr(2)},
			age:      -1, // edge case: negative age
			want:     false,
		},
		{
			name:     "range [0,2] - at min (inclusive)",
			property: GovernmentFundingProperty{MinAge: intPtr(0), MaxAge: intPtr(2)},
			age:      0,
			want:     true,
		},
		{
			name:     "range [0,2] - in middle",
			property: GovernmentFundingProperty{MinAge: intPtr(0), MaxAge: intPtr(2)},
			age:      1,
			want:     true,
		},
		{
			name:     "range [0,2] - at max (inclusive)",
			property: GovernmentFundingProperty{MinAge: intPtr(0), MaxAge: intPtr(2)},
			age:      2,
			want:     true,
		},
		{
			name:     "range [0,2] - above range",
			property: GovernmentFundingProperty{MinAge: intPtr(0), MaxAge: intPtr(2)},
			age:      3,
			want:     false,
		},

		// Single age range [2,2] - only age 2
		{
			name:     "range [2,2] - below",
			property: GovernmentFundingProperty{MinAge: intPtr(2), MaxAge: intPtr(2)},
			age:      1,
			want:     false,
		},
		{
			name:     "range [2,2] - exact match",
			property: GovernmentFundingProperty{MinAge: intPtr(2), MaxAge: intPtr(2)},
			age:      2,
			want:     true,
		},
		{
			name:     "range [2,2] - above",
			property: GovernmentFundingProperty{MinAge: intPtr(2), MaxAge: intPtr(2)},
			age:      3,
			want:     false,
		},

		// Larger range [3,6]
		{
			name:     "range [3,6] - at min",
			property: GovernmentFundingProperty{MinAge: intPtr(3), MaxAge: intPtr(6)},
			age:      3,
			want:     true,
		},
		{
			name:     "range [3,6] - in middle",
			property: GovernmentFundingProperty{MinAge: intPtr(3), MaxAge: intPtr(6)},
			age:      4,
			want:     true,
		},
		{
			name:     "range [3,6] - at max",
			property: GovernmentFundingProperty{MinAge: intPtr(3), MaxAge: intPtr(6)},
			age:      6,
			want:     true,
		},
		{
			name:     "range [3,6] - just below",
			property: GovernmentFundingProperty{MinAge: intPtr(3), MaxAge: intPtr(6)},
			age:      2,
			want:     false,
		},
		{
			name:     "range [3,6] - just above",
			property: GovernmentFundingProperty{MinAge: intPtr(3), MaxAge: intPtr(6)},
			age:      7,
			want:     false,
		},

		// Berlin funding typical ranges
		{
			name:     "Berlin range [0,1] - age 0 (infant)",
			property: GovernmentFundingProperty{MinAge: intPtr(0), MaxAge: intPtr(1)},
			age:      0,
			want:     true,
		},
		{
			name:     "Berlin range [0,1] - age 1 (toddler)",
			property: GovernmentFundingProperty{MinAge: intPtr(0), MaxAge: intPtr(1)},
			age:      1,
			want:     true,
		},
		{
			name:     "Berlin range [0,1] - age 2 (too old)",
			property: GovernmentFundingProperty{MinAge: intPtr(0), MaxAge: intPtr(1)},
			age:      2,
			want:     false,
		},
		{
			name:     "Berlin range [0,8] - all daycare ages",
			property: GovernmentFundingProperty{MinAge: intPtr(0), MaxAge: intPtr(8)},
			age:      5,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.property.MatchesAge(tt.age); got != tt.want {
				t.Errorf("MatchesAge(%d) = %v, want %v (MinAge=%v, MaxAge=%v)",
					tt.age, got, tt.want,
					formatPtr(tt.property.MinAge), formatPtr(tt.property.MaxAge))
			}
		})
	}
}

func formatPtr(p *int) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", *p)
}

func TestGovernmentFunding_ToResponse(t *testing.T) {
	now := time.Now()
	f := GovernmentFunding{
		ID:        1,
		Name:      "Berlin Kita Funding",
		State:     "berlin",
		CreatedAt: now,
		UpdatedAt: now,
	}

	resp := f.ToResponse()

	if resp.ID != 1 {
		t.Errorf("ID = %d, want 1", resp.ID)
	}
	if resp.Name != "Berlin Kita Funding" {
		t.Errorf("Name = %q, want %q", resp.Name, "Berlin Kita Funding")
	}
	if resp.State != "berlin" {
		t.Errorf("State = %q, want %q", resp.State, "berlin")
	}
}

func TestGovernmentFundingPeriod_ToResponse(t *testing.T) {
	now := time.Now()
	to := now.AddDate(1, 0, 0)

	period := GovernmentFundingPeriod{
		ID:                  10,
		GovernmentFundingID: 1,
		Period:              Period{From: now, To: &to},
		FullTimeWeeklyHours: 39.0,
		Comment:             "2024/2025",
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	resp := period.ToResponse()

	if resp.ID != 10 {
		t.Errorf("ID = %d, want 10", resp.ID)
	}
	if resp.GovernmentFundingID != 1 {
		t.Errorf("GovernmentFundingID = %d, want 1", resp.GovernmentFundingID)
	}
	if resp.FullTimeWeeklyHours != 39.0 {
		t.Errorf("FullTimeWeeklyHours = %f, want 39.0", resp.FullTimeWeeklyHours)
	}
	if resp.Comment != "2024/2025" {
		t.Errorf("Comment = %q, want %q", resp.Comment, "2024/2025")
	}
	if resp.To == nil {
		t.Fatal("To = nil, want non-nil")
	}
}

func TestGovernmentFundingPeriod_ToResponse_NilTo(t *testing.T) {
	period := GovernmentFundingPeriod{
		ID:                  10,
		GovernmentFundingID: 1,
		Period:              Period{From: time.Now()},
		FullTimeWeeklyHours: 39.0,
	}

	resp := period.ToResponse()

	if resp.To != nil {
		t.Errorf("To = %v, want nil", resp.To)
	}
}

func TestGovernmentFundingProperty_ToResponse(t *testing.T) {
	now := time.Now()
	minAge := 0
	maxAge := 2

	prop := GovernmentFundingProperty{
		ID:                  100,
		PeriodID:            10,
		Key:                 "care_type",
		Value:               "ganztag",
		Label:               "Ganztag",
		Payment:             166847,
		Requirement:         0.261,
		MinAge:              &minAge,
		MaxAge:              &maxAge,
		Comment:             "Full-day care U3",
		ApplyToAllContracts: false,
		CreatedAt:           now,
	}

	resp := prop.ToResponse()

	if resp.ID != 100 {
		t.Errorf("ID = %d, want 100", resp.ID)
	}
	if resp.PeriodID != 10 {
		t.Errorf("PeriodID = %d, want 10", resp.PeriodID)
	}
	if resp.Key != "care_type" {
		t.Errorf("Key = %q, want %q", resp.Key, "care_type")
	}
	if resp.Value != "ganztag" {
		t.Errorf("Value = %q, want %q", resp.Value, "ganztag")
	}
	if resp.Label != "Ganztag" {
		t.Errorf("Label = %q, want %q", resp.Label, "Ganztag")
	}
	if resp.Payment != 166847 {
		t.Errorf("Payment = %d, want 166847", resp.Payment)
	}
	if resp.Requirement != 0.261 {
		t.Errorf("Requirement = %f, want 0.261", resp.Requirement)
	}
	if resp.MinAge == nil || *resp.MinAge != 0 {
		t.Errorf("MinAge = %v, want 0", resp.MinAge)
	}
	if resp.MaxAge == nil || *resp.MaxAge != 2 {
		t.Errorf("MaxAge = %v, want 2", resp.MaxAge)
	}
	if resp.Comment != "Full-day care U3" {
		t.Errorf("Comment = %q, want %q", resp.Comment, "Full-day care U3")
	}
	if resp.ApplyToAllContracts != false {
		t.Errorf("ApplyToAllContracts = %v, want false", resp.ApplyToAllContracts)
	}
}

func TestGovernmentFundingProperty_ToResponse_NilAgeFields(t *testing.T) {
	prop := GovernmentFundingProperty{
		ID:       1,
		PeriodID: 10,
		Key:      "supplements",
		Value:    "ndh",
		Label:    "NDH",
		Payment:  5000,
	}

	resp := prop.ToResponse()

	if resp.MinAge != nil {
		t.Errorf("MinAge = %v, want nil", resp.MinAge)
	}
	if resp.MaxAge != nil {
		t.Errorf("MaxAge = %v, want nil", resp.MaxAge)
	}
}

func TestGovernmentFunding_TableName(t *testing.T) {
	if got := (GovernmentFunding{}).TableName(); got != "government_fundings" {
		t.Errorf("TableName() = %q, want %q", got, "government_fundings")
	}
}

func TestGovernmentFundingPeriod_TableName(t *testing.T) {
	if got := (GovernmentFundingPeriod{}).TableName(); got != "government_funding_periods" {
		t.Errorf("TableName() = %q, want %q", got, "government_funding_periods")
	}
}

func TestGovernmentFundingProperty_TableName(t *testing.T) {
	if got := (GovernmentFundingProperty{}).TableName(); got != "government_funding_properties" {
		t.Errorf("TableName() = %q, want %q", got, "government_funding_properties")
	}
}
