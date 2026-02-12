package service

import (
	"context"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestDecideSectionTransfer(t *testing.T) {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	tests := []struct {
		name         string
		contractFrom time.Time
		want         transferAction
	}{
		{
			name:         "contract started today → update",
			contractFrom: today,
			want:         transferUpdate,
		},
		{
			name:         "contract started yesterday → replace",
			contractFrom: today.AddDate(0, 0, -1),
			want:         transferReplace,
		},
		{
			name:         "contract started a week ago → replace",
			contractFrom: today.AddDate(0, 0, -7),
			want:         transferReplace,
		},
		{
			name:         "contract started a year ago → replace",
			contractFrom: today.AddDate(-1, 0, 0),
			want:         transferReplace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decideSectionTransfer(tt.contractFrom)
			if got != tt.want {
				t.Errorf("decideSectionTransfer(%v) = %v, want %v", tt.contractFrom, got, tt.want)
			}
		})
	}
}

func TestSameSectionID(t *testing.T) {
	id1 := uint(1)
	id2 := uint(2)
	id1b := uint(1)

	tests := []struct {
		name string
		a    *uint
		b    *uint
		want bool
	}{
		{"both nil", nil, nil, true},
		{"first nil", nil, &id1, false},
		{"second nil", &id1, nil, false},
		{"same value", &id1, &id1b, true},
		{"different values", &id1, &id2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sameSectionID(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("sameSectionID(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestChildService_Update_SectionChange_CreatesNewContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	sectionA := createTestSection(t, db, "Section A", org.ID, false)
	sectionB := createTestSection(t, db, "Section B", org.ID, false)

	child := createTestChildInSection(t, db, "Alice", "Smith", org.ID, sectionA.ID)

	// Create an active contract that started in the past (so it triggers replace, not update)
	pastDate := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -1, 0)
	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: pastDate, To: nil},
			SectionID:  &sectionA.ID,
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Move child to Section B
	newSectionID := sectionB.ID
	_, err := svc.Update(ctx, child.ID, org.ID, &models.ChildUpdateRequest{
		SectionID: &newSectionID,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify: old contract should be closed (has end date)
	var oldContract models.ChildContract
	if err := db.First(&oldContract, contract.ID).Error; err != nil {
		t.Fatalf("failed to find old contract: %v", err)
	}
	if oldContract.To == nil {
		t.Error("old contract should have an end date")
	}

	// Verify: new contract should exist with new section
	var contracts []models.ChildContract
	if err := db.Where("child_id = ?", child.ID).Order("from_date DESC").Find(&contracts).Error; err != nil {
		t.Fatalf("failed to find contracts: %v", err)
	}
	if len(contracts) != 2 {
		t.Fatalf("expected 2 contracts, got %d", len(contracts))
	}

	newContract := contracts[0] // newest first
	if newContract.SectionID == nil || *newContract.SectionID != sectionB.ID {
		t.Errorf("new contract section_id = %v, want %d", newContract.SectionID, sectionB.ID)
	}
	if newContract.To != nil {
		t.Error("new contract should be open-ended")
	}
	// Verify properties are copied
	if newContract.Properties.GetScalarProperty("care_type") != "ganztag" {
		t.Error("new contract should preserve properties from old contract")
	}
}

func TestChildService_Update_SectionChange_SameDayUpdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	sectionA := createTestSection(t, db, "Section A", org.ID, false)
	sectionB := createTestSection(t, db, "Section B", org.ID, false)

	child := createTestChildInSection(t, db, "Bob", "Jones", org.ID, sectionA.ID)

	// Create a contract that started today
	today := time.Now().UTC().Truncate(24 * time.Hour)
	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: today, To: nil},
			SectionID: &sectionA.ID,
		},
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Move child to Section B (same day)
	newSectionID := sectionB.ID
	_, err := svc.Update(ctx, child.ID, org.ID, &models.ChildUpdateRequest{
		SectionID: &newSectionID,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify: only 1 contract exists (updated in place)
	var contracts []models.ChildContract
	if err := db.Where("child_id = ?", child.ID).Find(&contracts).Error; err != nil {
		t.Fatalf("failed to find contracts: %v", err)
	}
	if len(contracts) != 1 {
		t.Fatalf("expected 1 contract (same-day update), got %d", len(contracts))
	}
	if contracts[0].SectionID == nil || *contracts[0].SectionID != sectionB.ID {
		t.Errorf("contract section_id = %v, want %d", contracts[0].SectionID, sectionB.ID)
	}
}

func TestChildService_Update_SectionChange_NoContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	sectionA := createTestSection(t, db, "Section A", org.ID, false)
	sectionB := createTestSection(t, db, "Section B", org.ID, false)

	child := createTestChildInSection(t, db, "Charlie", "Brown", org.ID, sectionA.ID)
	// No contract created

	// Move child to Section B — should succeed without error
	newSectionID := sectionB.ID
	resp, err := svc.Update(ctx, child.ID, org.ID, &models.ChildUpdateRequest{
		SectionID: &newSectionID,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if resp.SectionID == nil || *resp.SectionID != sectionB.ID {
		t.Errorf("child section_id = %v, want %d", resp.SectionID, sectionB.ID)
	}

	// Verify no contracts were created
	var count int64
	db.Model(&models.ChildContract{}).Where("child_id = ?", child.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 contracts, got %d", count)
	}
}

func TestChildService_Update_SameSection_NoTransition(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	sectionA := createTestSection(t, db, "Section A", org.ID, false)

	child := createTestChildInSection(t, db, "Diana", "Prince", org.ID, sectionA.ID)

	pastDate := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -1, 0)
	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: pastDate, To: nil},
			SectionID: &sectionA.ID,
		},
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// "Update" to same section — no transition should happen
	sameSectionID := sectionA.ID
	_, err := svc.Update(ctx, child.ID, org.ID, &models.ChildUpdateRequest{
		SectionID: &sameSectionID,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify: still only 1 contract, unchanged
	var contracts []models.ChildContract
	db.Where("child_id = ?", child.ID).Find(&contracts)
	if len(contracts) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(contracts))
	}
	if contracts[0].To != nil {
		t.Error("contract should still be open-ended")
	}
}

func TestChildService_CreateContract_AutoSetsSection(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSection(t, db, "Krippe", org.ID, false)

	child := createTestChildInSection(t, db, "Eve", "Adams", org.ID, section.ID)

	// Create a contract without specifying section_id
	resp, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateContract failed: %v", err)
	}

	// Verify the contract inherited the child's section
	if resp.SectionID == nil || *resp.SectionID != section.ID {
		t.Errorf("contract section_id = %v, want %d", resp.SectionID, section.ID)
	}
}

func TestEmployeeService_Update_SectionChange_CreatesNewContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	sectionA := createTestSection(t, db, "Section A", org.ID, false)
	sectionB := createTestSection(t, db, "Section B", org.ID, false)
	payPlan := createTestPayPlan(t, db, "TVöD", org.ID)

	// Create employee in section A
	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			SectionID:      &sectionA.ID,
			FirstName:      "Max",
			LastName:       "Mustermann",
			Gender:         "male",
			Birthdate:      time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	if err := db.Create(employee).Error; err != nil {
		t.Fatalf("failed to create employee: %v", err)
	}

	// Create an active contract that started in the past
	pastDate := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -1, 0)
	contract := &models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: pastDate, To: nil},
			SectionID:  &sectionA.ID,
			Properties: models.ContractProperties{"employer_type": "normal"},
		},
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          3,
		WeeklyHours:   39,
		PayPlanID:     payPlan.ID,
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Move employee to Section B
	newSectionID := sectionB.ID
	_, err := svc.Update(ctx, employee.ID, org.ID, &models.EmployeeUpdateRequest{
		SectionID: &newSectionID,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify: old contract should be closed
	var oldContract models.EmployeeContract
	if err := db.First(&oldContract, contract.ID).Error; err != nil {
		t.Fatalf("failed to find old contract: %v", err)
	}
	if oldContract.To == nil {
		t.Error("old contract should have an end date")
	}

	// Verify: new contract exists with correct fields
	var contracts []models.EmployeeContract
	db.Where("employee_id = ?", employee.ID).Order("from_date DESC").Find(&contracts)
	if len(contracts) != 2 {
		t.Fatalf("expected 2 contracts, got %d", len(contracts))
	}

	newContract := contracts[0]
	if newContract.SectionID == nil || *newContract.SectionID != sectionB.ID {
		t.Errorf("new contract section_id = %v, want %d", newContract.SectionID, sectionB.ID)
	}
	// Verify employee-specific fields are copied
	if newContract.StaffCategory != "qualified" {
		t.Errorf("StaffCategory = %q, want %q", newContract.StaffCategory, "qualified")
	}
	if newContract.Grade != "S8a" {
		t.Errorf("Grade = %q, want %q", newContract.Grade, "S8a")
	}
	if newContract.Step != 3 {
		t.Errorf("Step = %d, want 3", newContract.Step)
	}
	if newContract.WeeklyHours != 39 {
		t.Errorf("WeeklyHours = %f, want 39", newContract.WeeklyHours)
	}
	if newContract.PayPlanID != payPlan.ID {
		t.Errorf("PayPlanID = %d, want %d", newContract.PayPlanID, payPlan.ID)
	}
	if newContract.Properties.GetScalarProperty("employer_type") != "normal" {
		t.Error("new contract should preserve properties")
	}
}

func TestEmployeeService_Update_SectionChange_SameDayUpdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	sectionA := createTestSection(t, db, "Section A", org.ID, false)
	sectionB := createTestSection(t, db, "Section B", org.ID, false)
	payPlan := createTestPayPlan(t, db, "TVöD", org.ID)

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			SectionID:      &sectionA.ID,
			FirstName:      "Lisa",
			LastName:       "Schmidt",
			Gender:         "female",
			Birthdate:      time.Date(1985, 3, 20, 0, 0, 0, 0, time.UTC),
		},
	}
	if err := db.Create(employee).Error; err != nil {
		t.Fatalf("failed to create employee: %v", err)
	}

	// Create a contract that started today
	today := time.Now().UTC().Truncate(24 * time.Hour)
	contract := &models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: today, To: nil},
			SectionID: &sectionA.ID,
		},
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          1,
		WeeklyHours:   39,
		PayPlanID:     payPlan.ID,
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Move employee to Section B (same day)
	newSectionID := sectionB.ID
	_, err := svc.Update(ctx, employee.ID, org.ID, &models.EmployeeUpdateRequest{
		SectionID: &newSectionID,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify: only 1 contract, updated in place
	var contracts []models.EmployeeContract
	db.Where("employee_id = ?", employee.ID).Find(&contracts)
	if len(contracts) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(contracts))
	}
	if contracts[0].SectionID == nil || *contracts[0].SectionID != sectionB.ID {
		t.Errorf("contract section_id = %v, want %d", contracts[0].SectionID, sectionB.ID)
	}
}

func TestEmployeeService_CreateContract_AutoSetsSection(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSection(t, db, "Elemente", org.ID, false)
	payPlan := createTestPayPlan(t, db, "TVöD", org.ID)

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			SectionID:      &section.ID,
			FirstName:      "Hans",
			LastName:       "Mueller",
			Gender:         "male",
			Birthdate:      time.Date(1988, 7, 10, 0, 0, 0, 0, time.UTC),
		},
	}
	if err := db.Create(employee).Error; err != nil {
		t.Fatalf("failed to create employee: %v", err)
	}

	resp, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		From:          time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          1,
		WeeklyHours:   39,
		PayPlanID:     payPlan.ID,
	})
	if err != nil {
		t.Fatalf("CreateContract failed: %v", err)
	}

	if resp.SectionID == nil || *resp.SectionID != section.ID {
		t.Errorf("contract section_id = %v, want %d", resp.SectionID, section.ID)
	}
}
