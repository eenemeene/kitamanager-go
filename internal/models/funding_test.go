package models

import (
	"fmt"
	"testing"
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
