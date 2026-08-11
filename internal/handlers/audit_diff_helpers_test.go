package handlers

// Direct unit tests for the two contract-specific diff helpers.
//
// These exist because the HTTP-level tests cannot reach every case: the response
// DTO carries `properties,omitempty`, so an empty map round-trips as nil and the
// nil-vs-empty branch is unreachable from a request. Testing the helper directly
// pins the intended semantics regardless of what the transport can express.

import (
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestRecordPropertiesChange(t *testing.T) {
	populated := models.ContractProperties{"care_type": "ganztag", "ndh": "ndh"}

	tests := []struct {
		name          string
		before, after models.ContractProperties
		wantRecorded  bool
	}{
		{"nil to nil", nil, nil, false},
		{"nil to empty", nil, models.ContractProperties{}, false},
		{"empty to nil", models.ContractProperties{}, nil, false},
		{"empty to empty", models.ContractProperties{}, models.ContractProperties{}, false},
		{"identical populated", populated, models.ContractProperties{"care_type": "ganztag", "ndh": "ndh"}, false},
		{"nil to populated", nil, populated, true},
		{"populated to nil", populated, nil, true},
		{"populated to empty", populated, models.ContractProperties{}, true},
		{"one value differs", populated, models.ContractProperties{"care_type": "halbtag", "ndh": "ndh"}, true},
		{"supplement removed", populated, models.ContractProperties{"care_type": "ganztag"}, true},
		// Array-valued properties: == on these would panic, DeepEqual must not.
		{"equal slice values", models.ContractProperties{"tags": []any{"a", "b"}},
			models.ContractProperties{"tags": []any{"a", "b"}}, false},
		{"differing slice values", models.ContractProperties{"tags": []any{"a", "b"}},
			models.ContractProperties{"tags": []any{"a", "c"}}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := map[string]any{}
			recordPropertiesChange(m, "properties", tc.before, tc.after)
			_, recorded := m["properties"]
			if recorded != tc.wantRecorded {
				t.Errorf("recorded=%v, want %v (before=%v after=%v)", recorded, tc.wantRecorded, tc.before, tc.after)
			}
		})
	}
}

func TestRecordNullableTimeChange(t *testing.T) {
	d1 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	// Same instant, different pointer AND a different location — the case that
	// makes pointer comparison (and ==) wrong.
	sameInstantElsewhere := d1.In(time.FixedZone("X", 3600))

	tests := []struct {
		name          string
		before, after *time.Time
		wantRecorded  bool
		wantOld       any
		wantNew       any
	}{
		{"nil to nil", nil, nil, false, nil, nil},
		{"same instant, different pointers", &d1, &sameInstantElsewhere, false, nil, nil},
		{"nil to date", nil, &d1, true, nil, "2026-03-01T00:00:00Z"},
		{"date to nil", &d1, nil, true, "2026-03-01T00:00:00Z", nil},
		{"date to other date", &d1, &d2, true, "2026-03-01T00:00:00Z", "2026-06-01T00:00:00Z"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := map[string]any{}
			recordNullableTimeChange(m, "to", tc.before, tc.after)
			pair, recorded := m["to"].(map[string]any)
			if recorded != tc.wantRecorded {
				t.Fatalf("recorded=%v, want %v", recorded, tc.wantRecorded)
			}
			if !tc.wantRecorded {
				return
			}
			if pair["old"] != tc.wantOld {
				t.Errorf("old = %v, want %v", pair["old"], tc.wantOld)
			}
			if pair["new"] != tc.wantNew {
				t.Errorf("new = %v, want %v", pair["new"], tc.wantNew)
			}
		})
	}
}
