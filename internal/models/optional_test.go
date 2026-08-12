package models

import (
	"encoding/json"
	"testing"
	"time"
)

// The whole point of Opt is telling these three states apart, so test them
// through a real struct rather than by calling UnmarshalJSON directly —
// "absent" only exists at the struct level.
func TestOpt_ThreeStates(t *testing.T) {
	type req struct {
		To        Opt[time.Time] `json:"to"`
		SectionID Opt[uint]      `json:"section_id"`
	}

	tests := []struct {
		name      string
		body      string
		wantSet   bool
		wantNull  bool
		wantValue string // formatted date, empty when there is none
	}{
		{"absent", `{}`, false, false, ""},
		{"explicit null", `{"to":null}`, true, true, ""},
		{"value", `{"to":"2026-03-01T00:00:00Z"}`, true, false, "2026-03-01"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var r req
			if err := json.Unmarshal([]byte(tc.body), &r); err != nil {
				t.Fatalf("unmarshal %s: %v", tc.body, err)
			}
			if r.To.Set != tc.wantSet {
				t.Errorf("Set = %v, want %v", r.To.Set, tc.wantSet)
			}
			if r.To.IsNull() != tc.wantNull {
				t.Errorf("IsNull = %v, want %v", r.To.IsNull(), tc.wantNull)
			}
			v, ok := r.To.Get()
			if tc.wantValue == "" {
				if ok {
					t.Errorf("Get returned a value %v, want none", v)
				}
			} else {
				if !ok {
					t.Fatalf("Get returned no value, want %s", tc.wantValue)
				}
				if got := v.Format("2006-01-02"); got != tc.wantValue {
					t.Errorf("value = %s, want %s", got, tc.wantValue)
				}
			}
		})
	}
}

// A field omitted from the payload must stay untouched even when a sibling field
// is present — this is the case that previously destroyed data.
func TestOpt_OmittedSiblingUnaffected(t *testing.T) {
	type req struct {
		To         Opt[time.Time]          `json:"to"`
		Properties Opt[ContractProperties] `json:"properties"`
	}
	var r req
	if err := json.Unmarshal([]byte(`{"to":"2026-06-30T00:00:00Z"}`), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !r.To.Set {
		t.Error("to should be Set")
	}
	if r.Properties.Set {
		t.Error("properties was not in the payload; it must not be Set — otherwise a dates-only edit wipes it")
	}
}

// Maps need to work, since properties is the field whose loss moved real money.
func TestOpt_Properties(t *testing.T) {
	type req struct {
		Properties Opt[ContractProperties] `json:"properties"`
	}

	var withValue req
	if err := json.Unmarshal([]byte(`{"properties":{"care_type":"ganztag","ndh":"ndh"}}`), &withValue); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	props, ok := withValue.Properties.Get()
	if !ok {
		t.Fatal("expected a properties value")
	}
	if props["care_type"] != "ganztag" || props["ndh"] != "ndh" {
		t.Errorf("properties = %v", props)
	}

	// An explicit empty object is a real instruction: clear them.
	var empty req
	if err := json.Unmarshal([]byte(`{"properties":{}}`), &empty); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !empty.Properties.Set {
		t.Error("an explicit {} must count as Set")
	}
	if p, ok := empty.Properties.Get(); !ok || len(p) != 0 {
		t.Errorf("expected an empty map value, got %v ok=%v", p, ok)
	}
}

func TestOpt_Constructors(t *testing.T) {
	set := OptOf(uint(7))
	if v, ok := set.Get(); !ok || v != 7 {
		t.Errorf("OptOf: got %v ok=%v", v, ok)
	}
	if set.IsNull() {
		t.Error("OptOf must not be null")
	}

	null := OptNull[uint]()
	if !null.Set || !null.IsNull() {
		t.Error("OptNull must be Set and null")
	}
}

func TestOpt_MarshalRoundTrip(t *testing.T) {
	type req struct {
		SectionID Opt[uint] `json:"section_id"`
	}
	b, err := json.Marshal(req{SectionID: OptOf(uint(3))})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != `{"section_id":3}` {
		t.Errorf("marshal = %s", b)
	}

	b, err = json.Marshal(req{SectionID: OptNull[uint]()})
	if err != nil {
		t.Fatalf("marshal null: %v", err)
	}
	if string(b) != `{"section_id":null}` {
		t.Errorf("marshal null = %s", b)
	}
}
