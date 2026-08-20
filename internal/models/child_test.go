package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestChild_ToResponse(t *testing.T) {
	now := time.Now()

	t.Run("child without contracts", func(t *testing.T) {
		child := Child{
			Person: Person{
				ID:             1,
				OrganizationID: 2,
				FirstName:      "Emma",
				LastName:       "Schmidt",
				Gender:         "female",
				Birthdate:      time.Date(2020, 3, 10, 0, 0, 0, 0, time.UTC),
				CreatedAt:      now,
				UpdatedAt:      now,
			},
		}

		resp := child.ToResponse()

		if resp.ID != 1 {
			t.Errorf("ID = %d, want 1", resp.ID)
		}
		if resp.OrganizationID != 2 {
			t.Errorf("OrganizationID = %d, want 2", resp.OrganizationID)
		}
		if resp.FirstName != "Emma" {
			t.Errorf("FirstName = %q, want %q", resp.FirstName, "Emma")
		}
		if resp.LastName != "Schmidt" {
			t.Errorf("LastName = %q, want %q", resp.LastName, "Schmidt")
		}
		if resp.Gender != "female" {
			t.Errorf("Gender = %q, want %q", resp.Gender, "female")
		}
		if resp.Contracts != nil {
			t.Errorf("Contracts = %v, want nil", resp.Contracts)
		}
	})

	t.Run("child with contracts", func(t *testing.T) {
		sectionName := "Krippe"
		to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
		child := Child{
			Person: Person{
				ID:             1,
				OrganizationID: 2,
				FirstName:      "Max",
				LastName:       "Müller",
				Gender:         "male",
				Birthdate:      time.Date(2021, 7, 20, 0, 0, 0, 0, time.UTC),
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			Contracts: []ChildContract{
				{
					ID:      10,
					ChildID: 1,
					BaseContract: BaseContract{
						Period:    Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: &to},
						SectionID: 5,
						Section:   &Section{Name: sectionName},
					},
				},
			},
		}

		resp := child.ToResponse()

		if len(resp.Contracts) != 1 {
			t.Fatalf("len(Contracts) = %d, want 1", len(resp.Contracts))
		}
		if resp.Contracts[0].ID != 10 {
			t.Errorf("Contracts[0].ID = %d, want 10", resp.Contracts[0].ID)
		}
		if resp.Contracts[0].ChildID != 1 {
			t.Errorf("Contracts[0].ChildID = %d, want 1", resp.Contracts[0].ChildID)
		}
		if resp.Contracts[0].SectionName == nil || *resp.Contracts[0].SectionName != sectionName {
			t.Errorf("Contracts[0].SectionName = %v, want %q", resp.Contracts[0].SectionName, sectionName)
		}
	})
}

func TestChildContract_ToResponse(t *testing.T) {
	t.Run("without section", func(t *testing.T) {
		contract := ChildContract{
			ID:      1,
			ChildID: 2,
			BaseContract: BaseContract{
				Period:    Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
				SectionID: 3,
			},
		}

		resp := contract.ToResponse()

		if resp.ID != 1 {
			t.Errorf("ID = %d, want 1", resp.ID)
		}
		if resp.SectionName != nil {
			t.Errorf("SectionName = %v, want nil", resp.SectionName)
		}
	})

	t.Run("with section", func(t *testing.T) {
		contract := ChildContract{
			ID:      1,
			ChildID: 2,
			BaseContract: BaseContract{
				SectionID: 3,
				Section:   &Section{Name: "Krippe"},
			},
		}

		resp := contract.ToResponse()

		if resp.SectionName == nil || *resp.SectionName != "Krippe" {
			t.Errorf("SectionName = %v, want %q", resp.SectionName, "Krippe")
		}
	})
}

func TestChildResponse_FullName(t *testing.T) {
	tests := []struct {
		name      string
		firstName string
		lastName  string
		want      string
	}{
		{"normal name", "Emma", "Schmidt", "Emma Schmidt"},
		{"empty first name", "", "Schmidt", " Schmidt"},
		{"empty last name", "Emma", "", "Emma "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ChildResponse{FirstName: tt.firstName, LastName: tt.lastName}
			if got := r.FullName(); got != tt.want {
				t.Errorf("FullName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChildListFilter_Validate(t *testing.T) {
	t.Run("valid with no filters", func(t *testing.T) {
		f := ChildListFilter{}
		if err := f.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("valid with only active_on", func(t *testing.T) {
		now := time.Now()
		f := ChildListFilter{ActiveOn: &now}
		if err := f.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("valid with only contract_after", func(t *testing.T) {
		now := time.Now()
		f := ChildListFilter{ContractAfter: &now}
		if err := f.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("invalid with both active_on and contract_after", func(t *testing.T) {
		now := time.Now()
		f := ChildListFilter{ActiveOn: &now, ContractAfter: &now}
		if err := f.Validate(); err == nil {
			t.Error("Validate() error = nil, want error for mutually exclusive filters")
		}
	})
}

// TestChildUpdateRequest_SchoolEntryDateWireFormat pins the three states the
// field has to distinguish, and the format it arrives in.
//
// The format half is not academic: time.Time parses RFC3339 only, so a bare
// "2027-08-01" -- which is exactly what an <input type="date"> yields -- is
// rejected. The frontend therefore has to send it through formatDateForApi,
// the same helper contract dates already use. Getting this wrong fails no
// existing test and no E2E; it simply 400s every submit.
func TestChildUpdateRequest_SchoolEntryDateWireFormat(t *testing.T) {
	t.Run("absent means leave it alone", func(t *testing.T) {
		var r ChildUpdateRequest
		if err := json.Unmarshal([]byte(`{"first_name":"Mira"}`), &r); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if r.SchoolEntryDate.Set {
			t.Error("Set = true for an absent field; the update would clear a recorded date")
		}
	})

	t.Run("explicit null means no longer deferred", func(t *testing.T) {
		var r ChildUpdateRequest
		if err := json.Unmarshal([]byte(`{"school_entry_date":null}`), &r); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !r.SchoolEntryDate.Set {
			t.Fatal("Set = false for an explicit null; the deferral could never be reversed")
		}
		if r.SchoolEntryDate.Value != nil {
			t.Errorf("Value = %v, want nil", r.SchoolEntryDate.Value)
		}
	})

	t.Run("RFC3339 carries the date", func(t *testing.T) {
		var r ChildUpdateRequest
		if err := json.Unmarshal([]byte(`{"school_entry_date":"2027-08-01T00:00:00Z"}`), &r); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if r.SchoolEntryDate.Value == nil {
			t.Fatal("Value = nil, want 2027-08-01")
		}
		if got := r.SchoolEntryDate.Value.Format(DateFormat); got != "2027-08-01" {
			t.Errorf("date = %s, want 2027-08-01", got)
		}
	})

	t.Run("a bare date is rejected, so the client must format it", func(t *testing.T) {
		var r ChildUpdateRequest
		err := json.Unmarshal([]byte(`{"school_entry_date":"2027-08-01"}`), &r)
		if err == nil {
			t.Fatal("a bare YYYY-MM-DD was accepted; this test exists because it is not")
		}
	})
}
