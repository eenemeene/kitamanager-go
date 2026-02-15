package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestChildService_List(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	createTestChild(t, db, "John", "Doe", org.ID)
	createTestChild(t, db, "Jane", "Doe", org.ID)

	children, total, err := svc.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(children) != 2 {
		t.Errorf("expected 2 children, got %d", len(children))
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestChildService_GetByID(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	found, err := svc.GetByID(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.ID != child.ID {
		t.Errorf("ID = %d, want %d", found.ID, child.ID)
	}
	if found.FirstName != "John" {
		t.Errorf("FirstName = %v, want John", found.FirstName)
	}
}

func TestChildService_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	_, err := svc.GetByID(ctx, 999, org.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// SECURITY TEST: Cross-organization access attempt
func TestChildService_GetByID_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	child := createTestChild(t, db, "John", "Doe", org1.ID)

	// Try to access child from wrong organization - should return not found
	_, err := svc.GetByID(ctx, child.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error when accessing child from wrong org, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound (not forbidden - security), got %v", err)
	}
}

func TestChildService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.ChildCreateRequest{
		FirstName: "John",
		LastName:  "Doe",
		Gender:    "male",
		Birthdate: "2020-05-15",
	}

	child, err := svc.Create(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if child.ID == 0 {
		t.Error("expected ID to be set")
	}
	if child.FirstName != "John" {
		t.Errorf("FirstName = %v, want John", child.FirstName)
	}
	if child.LastName != "Doe" {
		t.Errorf("LastName = %v, want Doe", child.LastName)
	}
	if child.OrganizationID != org.ID {
		t.Errorf("OrganizationID = %d, want %d", child.OrganizationID, org.ID)
	}
}

func TestChildService_Create_WhitespaceOnlyNames(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	tests := []struct {
		name string
		req  *models.ChildCreateRequest
	}{
		{"empty first name", &models.ChildCreateRequest{FirstName: "", LastName: "Doe", Birthdate: "2020-01-01"}},
		{"whitespace first name", &models.ChildCreateRequest{FirstName: "   ", LastName: "Doe", Birthdate: "2020-01-01"}},
		{"empty last name", &models.ChildCreateRequest{FirstName: "John", LastName: "", Birthdate: "2020-01-01"}},
		{"whitespace last name", &models.ChildCreateRequest{FirstName: "John", LastName: "   ", Birthdate: "2020-01-01"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(ctx, org.ID, tt.req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !errors.Is(err, apperror.ErrBadRequest) {
				t.Errorf("expected ErrBadRequest, got %v", err)
			}
		})
	}
}

func TestChildService_Create_TrimmedNames(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.ChildCreateRequest{
		FirstName: "  John  ",
		LastName:  "  Doe  ",
		Gender:    "male",
		Birthdate: "2020-05-15",
	}

	child, err := svc.Create(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if child.FirstName != "John" {
		t.Errorf("FirstName = %v, want 'John' (trimmed)", child.FirstName)
	}
	if child.LastName != "Doe" {
		t.Errorf("LastName = %v, want 'Doe' (trimmed)", child.LastName)
	}
}

func TestChildService_Create_FutureBirthdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.ChildCreateRequest{
		FirstName: "John",
		LastName:  "Doe",
		Birthdate: time.Now().AddDate(1, 0, 0).Format("2006-01-02"), // 1 year in future
	}

	_, err := svc.Create(ctx, org.ID, req)
	if err == nil {
		t.Fatal("expected error for future birthdate, got nil")
	}

	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildService_Update(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	newFirstName := "Jane"
	req := &models.ChildUpdateRequest{
		FirstName: &newFirstName,
	}

	updated, err := svc.Update(ctx, child.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.FirstName != "Jane" {
		t.Errorf("FirstName = %v, want Jane", updated.FirstName)
	}
	if updated.LastName != "Doe" {
		t.Errorf("LastName should not change, got %v", updated.LastName)
	}
}

func TestChildService_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	newName := "Jane"
	req := &models.ChildUpdateRequest{
		FirstName: &newName,
	}

	_, err := svc.Update(ctx, 999, org.ID, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// SECURITY TEST: Cross-organization update attempt
func TestChildService_Update_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	child := createTestChild(t, db, "John", "Doe", org1.ID)

	newName := "Hacked"
	req := &models.ChildUpdateRequest{
		FirstName: &newName,
	}

	// Try to update child from wrong organization
	_, err := svc.Update(ctx, child.ID, org2.ID, req)
	if err == nil {
		t.Fatal("expected error when updating child from wrong org, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound (not forbidden - security), got %v", err)
	}

	// Verify child was not actually updated
	found, _ := svc.GetByID(ctx, child.ID, org1.ID)
	if found.FirstName != "John" {
		t.Errorf("child was modified despite wrong org, FirstName = %v", found.FirstName)
	}
}

func TestChildService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	err := svc.Delete(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's deleted
	_, err = svc.GetByID(ctx, child.ID, org.ID)
	if err == nil {
		t.Error("expected child to be deleted")
	}
}

// SECURITY TEST: Cross-organization delete attempt
func TestChildService_Delete_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	child := createTestChild(t, db, "John", "Doe", org1.ID)

	// Try to delete child from wrong organization
	err := svc.Delete(ctx, child.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error when deleting child from wrong org, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound (not forbidden - security), got %v", err)
	}

	// Verify child still exists
	found, err := svc.GetByID(ctx, child.ID, org1.ID)
	if err != nil {
		t.Errorf("child was deleted despite wrong org: %v", err)
	}
	if found.FirstName != "John" {
		t.Error("child data was corrupted")
	}
}

func TestChildService_CreateContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	req := &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       from,
		To:         &to,
		Properties: models.ContractProperties{"care_type": "ganztag", "supplements": []string{"ndh"}},
	}

	contract, err := svc.CreateContract(ctx, child.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if contract.ID == 0 {
		t.Error("expected ID to be set")
	}
	if contract.ChildID != child.ID {
		t.Errorf("ChildID = %d, want %d", contract.ChildID, child.ID)
	}
	if contract.Properties["care_type"] != "ganztag" {
		t.Errorf("Properties = %v, want care_type=ganztag", contract.Properties)
	}
}

func TestChildService_CreateContract_ChildNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
	}

	_, err := svc.CreateContract(ctx, 999, org.ID, req)
	if err == nil {
		t.Fatal("expected error for non-existent child, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// SECURITY TEST: Cross-organization contract creation attempt
func TestChildService_CreateContract_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	child := createTestChild(t, db, "John", "Doe", org1.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
	}

	// Try to create contract via wrong organization
	_, err := svc.CreateContract(ctx, child.ID, org2.ID, req)
	if err == nil {
		t.Fatal("expected error when creating contract from wrong org, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound (not forbidden - security), got %v", err)
	}
}

func TestChildService_CreateContract_InvalidPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	from := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // Before from

	req := &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
		To:        &to,
	}

	_, err := svc.CreateContract(ctx, child.ID, org.ID, req)
	if err == nil {
		t.Fatal("expected error for invalid period (to before from), got nil")
	}

	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildService_CreateContract_OverlappingContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	// Create first contract
	from1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req1 := &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       from1,
		To:         &to1,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	}
	_, err := svc.CreateContract(ctx, child.ID, org.ID, req1)
	if err != nil {
		t.Fatalf("first contract: expected no error, got %v", err)
	}

	// Try to create overlapping contract
	from2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC) // Overlaps with first
	to2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	req2 := &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       from2,
		To:         &to2,
		Properties: models.ContractProperties{"care_type": "halbtag"},
	}

	_, err = svc.CreateContract(ctx, child.ID, org.ID, req2)
	if err == nil {
		t.Fatal("expected error for overlapping contract, got nil")
	}

	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestChildService_CreateContract_OngoingContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// No 'to' date means ongoing contract
	req := &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
		To:        nil,
	}

	contract, err := svc.CreateContract(ctx, child.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if contract.To != nil {
		t.Errorf("To = %v, want nil (ongoing)", contract.To)
	}
}

func TestChildService_CreateContract_BeforeBirthdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)
	// Child birthdate is set by createTestChild; update it to a known date
	child.Birthdate = time.Date(2022, 6, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	// Contract start date before birthdate should fail
	fromBeforeBirth := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      fromBeforeBirth,
	})
	if err == nil {
		t.Fatal("expected error for contract start before birthdate, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}

	// Contract start date on birthdate should succeed
	fromOnBirth := time.Date(2022, 6, 15, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      fromOnBirth,
	})
	if err != nil {
		t.Fatalf("expected no error for contract on birthdate, got %v", err)
	}
	if contract == nil {
		t.Fatal("expected contract, got nil")
	}
}

func TestChildService_CreateContract_SectionNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 99999,
		From:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error for non-existent section, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildService_CreateContract_SectionFromWrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	child := createTestChild(t, db, "John", "Doe", org1.ID)

	// Get org2's default section
	var org2Section models.Section
	db.Where("organization_id = ?", org2.ID).First(&org2Section)

	_, err := svc.CreateContract(ctx, child.ID, org1.ID, &models.ChildContractCreateRequest{
		SectionID: org2Section.ID,
		From:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error for section from wrong org, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildService_ListContracts(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	// Create two contracts
	from1 := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2022, 12, 31, 0, 0, 0, 0, time.UTC)
	req1 := &models.ChildContractCreateRequest{SectionID: 1, From: from1, To: &to1, Properties: models.ContractProperties{"care_type": "halbtag"}}
	_, _ = svc.CreateContract(ctx, child.ID, org.ID, req1)

	from2 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req2 := &models.ChildContractCreateRequest{SectionID: 1, From: from2, Properties: models.ContractProperties{"care_type": "ganztag"}}
	_, _ = svc.CreateContract(ctx, child.ID, org.ID, req2)

	contracts, _, err := svc.ListContracts(ctx, child.ID, org.ID, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(contracts) != 2 {
		t.Errorf("expected 2 contracts, got %d", len(contracts))
	}
}

func TestChildService_ListContracts_ChildNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	_, _, err := svc.ListContracts(ctx, 999, org.ID, 100, 0)
	if err == nil {
		t.Fatal("expected error for non-existent child, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// SECURITY TEST: Cross-organization list contracts attempt
func TestChildService_ListContracts_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	child := createTestChild(t, db, "John", "Doe", org1.ID)

	// Create a contract for child in org1
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildContractCreateRequest{SectionID: 1, From: from, Properties: models.ContractProperties{"care_type": "ganztag"}}
	_, _ = svc.CreateContract(ctx, child.ID, org1.ID, req)

	// Try to list contracts from wrong organization
	_, _, err := svc.ListContracts(ctx, child.ID, org2.ID, 100, 0)
	if err == nil {
		t.Fatal("expected error when listing contracts from wrong org, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound (not forbidden - security), got %v", err)
	}
}

func TestChildService_ListByOrganizationAndSection_ActiveOn(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	// Child with active contract
	childActive := createTestChild(t, db, "Active", "Child", org.ID)
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, childActive.ID, org.ID, &models.ChildContractCreateRequest{SectionID: 1, From: from})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Child with expired contract
	childExpired := createTestChild(t, db, "Expired", "Child", org.ID)
	fromExpired := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	toExpired := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateContract(ctx, childExpired.ID, org.ID, &models.ChildContractCreateRequest{SectionID: 1, From: fromExpired, To: &toExpired})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Child with no contract
	createTestChild(t, db, "NoContract", "Child", org.ID)

	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// With activeOn filter: only the active child should be returned
	children, total, err := svc.ListByOrganizationAndSection(ctx, org.ID, models.ChildListFilter{ActiveOn: &refDate}, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(children) != 1 {
		t.Errorf("expected 1 child with active_on filter, got %d", len(children))
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(children) == 1 && children[0].FirstName != "Active" {
		t.Errorf("expected Active child, got %s", children[0].FirstName)
	}

	// Without activeOn filter: all 3 children should be returned
	allChildren, allTotal, err := svc.ListByOrganizationAndSection(ctx, org.ID, models.ChildListFilter{}, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(allChildren) != 3 {
		t.Errorf("expected 3 children without filter, got %d", len(allChildren))
	}
	if allTotal != 3 {
		t.Errorf("expected total 3, got %d", allTotal)
	}
}

func TestChildService_ListByOrganization(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	createTestChild(t, db, "John", "Doe", org1.ID)
	createTestChild(t, db, "Jane", "Doe", org1.ID)
	createTestChild(t, db, "Bob", "Smith", org2.ID)

	children, total, err := svc.ListByOrganization(ctx, org1.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(children) != 2 {
		t.Errorf("expected 2 children in org1, got %d", len(children))
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

// SECURITY TEST: Verify ListByOrganization doesn't leak data from other orgs
func TestChildService_ListByOrganization_IsolatesData(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create children in both orgs
	createTestChild(t, db, "John", "Doe", org1.ID)
	createTestChild(t, db, "Secret", "Child", org2.ID)

	// List children in org1
	children, _, err := svc.ListByOrganization(ctx, org1.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify no children from org2 are returned
	for _, child := range children {
		if child.OrganizationID != org1.ID {
			t.Errorf("got child from wrong org: %d (expected %d)", child.OrganizationID, org1.ID)
		}
		if child.FirstName == "Secret" {
			t.Error("data leaked from other organization")
		}
	}
}

// SECURITY TEST: DeleteContract cross-org
func TestChildService_DeleteContract_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	child := createTestChild(t, db, "John", "Doe", org1.ID)

	// Create a contract
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildContractCreateRequest{SectionID: 1, From: from, Properties: models.ContractProperties{"care_type": "ganztag"}}
	contract, _ := svc.CreateContract(ctx, child.ID, org1.ID, req)

	// Try to delete contract from wrong organization
	err := svc.DeleteContract(ctx, contract.ID, child.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error when deleting contract from wrong org, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound (not forbidden - security), got %v", err)
	}

	// Verify contract still exists
	contracts, _, _ := svc.ListContracts(ctx, child.ID, org1.ID, 100, 0)
	if len(contracts) != 1 {
		t.Error("contract was deleted despite wrong org")
	}
}

// SECURITY TEST: GetCurrentRecord cross-org
func TestChildService_GetCurrentRecord_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	child := createTestChild(t, db, "John", "Doe", org1.ID)

	// Create an ongoing contract
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildContractCreateRequest{SectionID: 1, From: from, Properties: models.ContractProperties{"care_type": "ganztag"}}
	_, _ = svc.CreateContract(ctx, child.ID, org1.ID, req)

	// Try to get current contract from wrong organization
	_, err := svc.GetCurrentRecord(ctx, child.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error when getting current contract from wrong org, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound (not forbidden - security), got %v", err)
	}
}

// =========================================
// Funding Calculation Tests
// =========================================

func TestChildService_CalculateFunding_BasicCalculation(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	// Create org with government funding
	org := createTestOrganization(t, db, "Test Org")
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")

	// Assign funding to org
	// Funding is now automatically looked up by org.State ("berlin")

	// Create funding period covering our test date
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)

	// Create properties with age filter (ages 3-6)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 100000, 3, 7) // 1000.00 EUR
	createTestFundingProperty(t, db, period.ID, "supplements", "ndh", 50000, 3, 7)    // 500.00 EUR

	// Create child (born 2022-01-15, age 3 on 2025-01-27)
	child := createTestChild(t, db, "Max", "Mustermann", org.ID)
	child.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	// Create contract with attributes
	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag", "supplements": []string{"ndh"}},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Calculate funding
	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := svc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify weekly hours basis is set from funding period (not pay plan)
	if result.WeeklyHoursBasis != 39.0 {
		t.Errorf("WeeklyHoursBasis = %f, want 39.0", result.WeeklyHoursBasis)
	}

	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}

	cf := result.Children[0]
	if cf.ChildID != child.ID {
		t.Errorf("ChildID = %d, want %d", cf.ChildID, child.ID)
	}
	if cf.Funding != 150000 { // 1000.00 + 500.00 = 1500.00 EUR = 150000 cents
		t.Errorf("Funding = %d, want 150000 (cents)", cf.Funding)
	}
	if cf.Requirement != 0.2 { // 0.1 + 0.1 = 0.2 (two matched properties with Requirement=0.1 each)
		t.Errorf("Requirement = %f, want 0.2", cf.Requirement)
	}
	if len(cf.MatchedProperties) != 2 {
		t.Errorf("MatchedProperties = %v, want 2 items", cf.MatchedProperties)
	}
	if len(cf.UnmatchedProperties) != 0 {
		t.Errorf("UnmatchedProperties = %v, want 0 items", cf.UnmatchedProperties)
	}
}

func TestChildService_CalculateFunding_NoFundingAssigned(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	// Create org WITHOUT government funding
	org := createTestOrganization(t, db, "Test Org")

	// Create child with contract
	child := createTestChild(t, db, "Max", "Mustermann", org.ID)
	child.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag", "supplements": []string{"ndh"}},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Calculate funding
	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := svc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}

	cf := result.Children[0]
	if cf.Funding != 0 {
		t.Errorf("Funding = %d, want 0 (no funding assigned)", cf.Funding)
	}
	if len(cf.UnmatchedProperties) != 2 {
		t.Errorf("UnmatchedProperties = %v, want 2 items", cf.UnmatchedProperties)
	}
}

func TestChildService_CalculateFunding_NoMatchingPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	// Create org with funding
	org := createTestOrganization(t, db, "Test Org")
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	// Funding is now automatically looked up by org.State ("berlin")

	// Create period that doesn't cover our test date
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &to, 39.0)

	// Create child with contract
	child := createTestChild(t, db, "Max", "Mustermann", org.ID)
	child.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})

	// Calculate funding for date outside period
	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := svc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cf := result.Children[0]
	if cf.Funding != 0 {
		t.Errorf("Funding = %d, want 0 (no matching period)", cf.Funding)
	}
}

func TestChildService_CalculateFunding_NoMatchingAgeProperty(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	// Create org with funding
	org := createTestOrganization(t, db, "Test Org")
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	// Funding is now automatically looked up by org.State ("berlin")

	// Create period
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)

	// Create property for ages 0-2 only
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 100000, 0, 2)

	// Create child age 3 (doesn't match 0-2 property)
	child := createTestChild(t, db, "Max", "Mustermann", org.ID)
	child.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC) // Age 3 on 2025-01-27
	db.Save(child)

	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})

	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := svc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cf := result.Children[0]
	if cf.Funding != 0 {
		t.Errorf("Funding = %d, want 0 (no matching age property)", cf.Funding)
	}
	if len(cf.UnmatchedProperties) != 1 {
		t.Errorf("UnmatchedProperties = %v, want 1 item (care_type:ganztag)", cf.UnmatchedProperties)
	}
}

func TestChildService_CalculateFunding_PartialAttributeMatch(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	// Create org with funding
	org := createTestOrganization(t, db, "Test Org")
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	// Funding is now automatically looked up by org.State ("berlin")

	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 100000, 3, 7)
	// "unknown_key" property does NOT exist

	child := createTestChild(t, db, "Max", "Mustermann", org.ID)
	child.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag", "unknown_key": "xyz"}, // unknown_key doesn't match any funding property
	})

	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := svc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cf := result.Children[0]
	if cf.Funding != 100000 {
		t.Errorf("Funding = %d, want 100000", cf.Funding)
	}
	if len(cf.MatchedProperties) != 1 {
		t.Errorf("MatchedProperties = %v, want 1 item", cf.MatchedProperties)
	}
	if len(cf.UnmatchedProperties) != 1 {
		t.Errorf("UnmatchedProperties = %v, want 1 item", cf.UnmatchedProperties)
	}
}

func TestChildService_CalculateFunding_SingleProperty(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	// Funding is now automatically looked up by org.State ("berlin")

	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 100000, 3, 7)

	child := createTestChild(t, db, "Max", "Mustermann", org.ID)
	child.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})

	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := svc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cf := result.Children[0]
	if cf.Funding != 100000 {
		t.Errorf("Funding = %d, want 100000", cf.Funding)
	}
	if len(cf.MatchedProperties) != 1 {
		t.Errorf("MatchedProperties = %v, want 1 item", cf.MatchedProperties)
	}
}

func TestChildService_CalculateFunding_ChildNoActiveOnDate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	// Funding is now automatically looked up by org.State ("berlin")

	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 100000, 3, 7)

	// Child with active contract
	childActive := createTestChild(t, db, "Active", "Child", org.ID)
	childActive.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(childActive)
	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, childActive.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})

	// Child with NO contract (should not appear in results)
	childNoContract := createTestChild(t, db, "NoContract", "Child", org.ID)
	childNoContract.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(childNoContract)

	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := svc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should only include child with active contract
	if len(result.Children) != 1 {
		t.Errorf("expected 1 child (with active contract), got %d", len(result.Children))
	}
	if result.Children[0].ChildName != "Active Child" {
		t.Errorf("expected Active Child, got %s", result.Children[0].ChildName)
	}
}

// SECURITY TEST: Cross-organization funding calculation
func TestChildService_CalculateFunding_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Both orgs have state="berlin", so they share the same funding
	funding := createTestGovernmentFunding(t, db, "Funding")

	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 100000, 3, 7)

	// Child in org1
	child1 := createTestChild(t, db, "Org1", "Child", org1.ID)
	child1.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child1)
	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, child1.ID, org1.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})

	// Child in org2
	child2 := createTestChild(t, db, "Org2", "Child", org2.ID)
	child2.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child2)
	_, _ = svc.CreateContract(ctx, child2.ID, org2.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})

	// Calculate funding for org1 - should NOT include org2's child
	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := svc.CalculateFunding(ctx, org1.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Children) != 1 {
		t.Errorf("expected 1 child from org1, got %d", len(result.Children))
	}

	for _, cf := range result.Children {
		if cf.ChildName == "Org2 Child" {
			t.Error("SECURITY: org2's child leaked into org1's funding calculation")
		}
	}
}

func TestChildService_CalculateFunding_WeeklyHoursFromFundingPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")

	// Create funding period with different weekly hours (40.0 instead of 39.0)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 40.0)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 100000, 3, 7)

	child := createTestChild(t, db, "Max", "Mustermann", org.ID)
	child.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := svc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.WeeklyHoursBasis != 40.0 {
		t.Errorf("WeeklyHoursBasis = %f, want 40.0 (from funding period)", result.WeeklyHoursBasis)
	}
}

func TestChildService_CalculateFunding_NoMatchingPeriod_WeeklyHoursZero(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")

	// Create period that doesn't cover our test date
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &to, 39.0)

	child := createTestChild(t, db, "Max", "Mustermann", org.ID)
	child.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})

	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := svc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.WeeklyHoursBasis != 0 {
		t.Errorf("WeeklyHoursBasis = %f, want 0 (no matching period)", result.WeeklyHoursBasis)
	}
}

// =========================================
// Age Distribution Tests
// =========================================

func TestChildService_GetAgeDistribution_BasicDistribution(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create children with different ages
	// Child age 2 (born 2023-01-28)
	child1 := createTestChild(t, db, "Child1", "Age2", org.ID)
	child1.Birthdate = time.Date(2023, 1, 28, 0, 0, 0, 0, time.UTC)
	db.Save(child1)

	// Child age 3 (born 2022-01-28)
	child2 := createTestChild(t, db, "Child2", "Age3", org.ID)
	child2.Birthdate = time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC)
	db.Save(child2)

	// Child age 5 (born 2020-01-28)
	child3 := createTestChild(t, db, "Child3", "Age5", org.ID)
	child3.Birthdate = time.Date(2020, 1, 28, 0, 0, 0, 0, time.UTC)
	db.Save(child3)

	// Create contracts for all children
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, childID := range []uint{child1.ID, child2.ID, child3.ID} {
		_, err := svc.CreateContract(ctx, childID, org.ID, &models.ChildContractCreateRequest{
			SectionID: 1,
			From:      from,
		})
		if err != nil {
			t.Fatalf("failed to create contract: %v", err)
		}
	}

	stats, err := svc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", stats.TotalCount)
	}

	if stats.Date != "2025-01-28" {
		t.Errorf("Date = %s, want 2025-01-28", stats.Date)
	}

	// Check distribution: should have 7 buckets (0-6+)
	if len(stats.Distribution) != 7 {
		t.Errorf("expected 7 buckets, got %d", len(stats.Distribution))
	}

	// Verify specific counts
	expected := map[string]int{
		"0":  0,
		"1":  0,
		"2":  1, // child1
		"3":  1, // child2
		"4":  0,
		"5":  1, // child3
		"6+": 0,
	}

	for _, bucket := range stats.Distribution {
		if bucket.Count != expected[bucket.AgeLabel] {
			t.Errorf("bucket %s: expected count %d, got %d", bucket.AgeLabel, expected[bucket.AgeLabel], bucket.Count)
		}
	}
}

func TestChildService_GetAgeDistribution_NoChildren(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	stats, err := svc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0", stats.TotalCount)
	}

	// All buckets should have 0 count
	for _, bucket := range stats.Distribution {
		if bucket.Count != 0 {
			t.Errorf("bucket %s: expected count 0, got %d", bucket.AgeLabel, bucket.Count)
		}
	}
}

func TestChildService_GetAgeDistribution_ChildWithNoContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create child without contract
	child := createTestChild(t, db, "NoContract", "Child", org.ID)
	child.Birthdate = time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	stats, err := svc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0 (child has no contract)", stats.TotalCount)
	}
}

func TestChildService_GetAgeDistribution_ContractExpiredBeforeDate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create child with expired contract
	child := createTestChild(t, db, "Expired", "Child", org.ID)
	child.Birthdate = time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC) // Expired before refDate
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
		To:        &to,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	stats, err := svc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0 (contract expired)", stats.TotalCount)
	}
}

func TestChildService_GetAgeDistribution_ContractStartsAfterDate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create child with future contract
	child := createTestChild(t, db, "Future", "Child", org.ID)
	child.Birthdate = time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	from := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC) // Starts after refDate
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	stats, err := svc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0 (contract starts in future)", stats.TotalCount)
	}
}

func TestChildService_GetAgeDistribution_OldestBucket(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create children ages 6, 7, 8 - should all be in 6+ bucket
	ages := []int{6, 7, 8}
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, age := range ages {
		child := createTestChild(t, db, "Child", "OldBucket", org.ID)
		child.Birthdate = refDate.AddDate(-age, 0, 0)
		db.Save(child)

		_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
			SectionID: 1,
			From:      from,
		})
		if err != nil {
			t.Fatalf("failed to create contract: %v", err)
		}
	}

	stats, err := svc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", stats.TotalCount)
	}

	// Find 6+ bucket
	var sixPlusBucket *models.AgeDistributionBucket
	for i := range stats.Distribution {
		if stats.Distribution[i].AgeLabel == "6+" {
			sixPlusBucket = &stats.Distribution[i]
			break
		}
	}

	if sixPlusBucket == nil {
		t.Fatal("6+ bucket not found")
	}

	if sixPlusBucket.Count != 3 {
		t.Errorf("6+ bucket count = %d, want 3", sixPlusBucket.Count)
	}

	if sixPlusBucket.MaxAge != nil {
		t.Errorf("6+ bucket MaxAge = %v, want nil (open-ended)", sixPlusBucket.MaxAge)
	}
}

func TestChildService_GetAgeDistribution_YoungestBucket(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create child age 0 (born recently)
	child := createTestChild(t, db, "Baby", "Child", org.ID)
	child.Birthdate = time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC) // 7 months old
	db.Save(child)

	from := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	stats, err := svc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Find age 0 bucket
	var zeroBucket *models.AgeDistributionBucket
	for i := range stats.Distribution {
		if stats.Distribution[i].AgeLabel == "0" {
			zeroBucket = &stats.Distribution[i]
			break
		}
	}

	if zeroBucket == nil {
		t.Fatal("age 0 bucket not found")
	}

	if zeroBucket.Count != 1 {
		t.Errorf("age 0 bucket count = %d, want 1", zeroBucket.Count)
	}
}

func TestChildService_GetAgeDistribution_BirthdayEdgeCase(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	// Child born 2022-01-28
	child := createTestChild(t, db, "Birthday", "Child", org.ID)
	child.Birthdate = time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	from := time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
	})

	// Test day before birthday - should be age 2
	dayBefore := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	stats, _ := svc.GetAgeDistribution(ctx, org.ID, dayBefore)

	age2Count := 0
	age3Count := 0
	for _, bucket := range stats.Distribution {
		if bucket.AgeLabel == "2" {
			age2Count = bucket.Count
		}
		if bucket.AgeLabel == "3" {
			age3Count = bucket.Count
		}
	}
	if age2Count != 1 {
		t.Errorf("day before birthday: expected age 2 count = 1, got %d", age2Count)
	}
	if age3Count != 0 {
		t.Errorf("day before birthday: expected age 3 count = 0, got %d", age3Count)
	}

	// Test on birthday - should be age 3
	onBirthday := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)
	stats, _ = svc.GetAgeDistribution(ctx, org.ID, onBirthday)

	for _, bucket := range stats.Distribution {
		if bucket.AgeLabel == "2" {
			age2Count = bucket.Count
		}
		if bucket.AgeLabel == "3" {
			age3Count = bucket.Count
		}
	}
	if age2Count != 0 {
		t.Errorf("on birthday: expected age 2 count = 0, got %d", age2Count)
	}
	if age3Count != 1 {
		t.Errorf("on birthday: expected age 3 count = 1, got %d", age3Count)
	}
}

// SECURITY TEST: Cross-organization isolation
func TestChildService_GetAgeDistribution_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create child in org1 with contract
	child := createTestChild(t, db, "Org1", "Child", org1.ID)
	child.Birthdate = time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, child.ID, org1.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
	})

	// Query stats for org2 - should not include org1's children
	stats, err := svc.GetAgeDistribution(ctx, org2.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 0 {
		t.Errorf("SECURITY: TotalCount = %d, want 0 (child in different org)", stats.TotalCount)
	}
}

func TestChildService_GetAgeDistribution_MultipleChildrenSameAge(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create 5 children all age 3
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		child := createTestChild(t, db, "Child", "Age3", org.ID)
		child.Birthdate = time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC) // Age 3
		db.Save(child)

		_, _ = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
			SectionID: 1,
			From:      from,
		})
	}

	stats, err := svc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 5 {
		t.Errorf("TotalCount = %d, want 5", stats.TotalCount)
	}

	// Find age 3 bucket
	for _, bucket := range stats.Distribution {
		if bucket.AgeLabel == "3" {
			if bucket.Count != 5 {
				t.Errorf("age 3 bucket count = %d, want 5", bucket.Count)
			}
			break
		}
	}
}

func TestChildService_GetAgeDistribution_BucketMetadata(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	stats, err := svc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify bucket metadata
	expectedBuckets := []struct {
		label  string
		minAge int
		maxAge *int
	}{
		{"0", 0, intPtr(0)},
		{"1", 1, intPtr(1)},
		{"2", 2, intPtr(2)},
		{"3", 3, intPtr(3)},
		{"4", 4, intPtr(4)},
		{"5", 5, intPtr(5)},
		{"6+", 6, nil},
	}

	for i, expected := range expectedBuckets {
		bucket := stats.Distribution[i]
		if bucket.AgeLabel != expected.label {
			t.Errorf("bucket %d: AgeLabel = %s, want %s", i, bucket.AgeLabel, expected.label)
		}
		if bucket.MinAge != expected.minAge {
			t.Errorf("bucket %d: MinAge = %d, want %d", i, bucket.MinAge, expected.minAge)
		}
		if expected.maxAge == nil {
			if bucket.MaxAge != nil {
				t.Errorf("bucket %d: MaxAge = %v, want nil", i, bucket.MaxAge)
			}
		} else {
			if bucket.MaxAge == nil || *bucket.MaxAge != *expected.maxAge {
				t.Errorf("bucket %d: MaxAge = %v, want %v", i, bucket.MaxAge, *expected.maxAge)
			}
		}
	}
}

func TestChildService_GetAgeDistribution_HistoricalDate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	// Create child born 2020-01-01
	child := createTestChild(t, db, "Historical", "Child", org.ID)
	child.Birthdate = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	// Contract from 2022-01-01 to 2023-12-31
	from := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
		To:        &to,
	})

	// Query for a date when contract was active
	refDate := time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)
	stats, err := svc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1 (contract active on historical date)", stats.TotalCount)
	}

	// Child should be age 3 on 2023-06-15 (born 2020-01-01)
	for _, bucket := range stats.Distribution {
		if bucket.AgeLabel == "3" {
			if bucket.Count != 1 {
				t.Errorf("age 3 bucket count = %d, want 1", bucket.Count)
			}
			break
		}
	}
}

// =========================================
// GetContractByID Tests
// =========================================

func TestChildService_GetContractByID(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req := &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       from,
		To:         &to,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	}
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, req)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	found, err := svc.GetContractByID(ctx, contract.ID, child.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.ID != contract.ID {
		t.Errorf("ID = %d, want %d", found.ID, contract.ID)
	}
	if found.ChildID != child.ID {
		t.Errorf("ChildID = %d, want %d", found.ChildID, child.ID)
	}
	if found.Properties["care_type"] != "ganztag" {
		t.Errorf("Properties = %v, want care_type=ganztag", found.Properties)
	}
}

func TestChildService_GetContractByID_WrongChild(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child1 := createTestChild(t, db, "John", "Doe", org.ID)
	child2 := createTestChild(t, db, "Jane", "Doe", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, child1.ID, org.ID, &models.ChildContractCreateRequest{SectionID: 1, From: from})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Try to access contract via wrong child
	_, err = svc.GetContractByID(ctx, contract.ID, child2.ID, org.ID)
	if err == nil {
		t.Fatal("expected error when accessing contract via wrong child, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// SECURITY TEST: Cross-organization GetContractByID
func TestChildService_GetContractByID_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	child := createTestChild(t, db, "John", "Doe", org1.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, child.ID, org1.ID, &models.ChildContractCreateRequest{SectionID: 1, From: from})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Try to access contract via wrong organization
	_, err = svc.GetContractByID(ctx, contract.ID, child.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error when accessing contract from wrong org, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestChildService_GetContractByID_NonexistentContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	_, err := svc.GetContractByID(ctx, 999, child.ID, org.ID)
	if err == nil {
		t.Fatal("expected error for non-existent contract, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// =========================================
// GetCurrentRecord Tests
// =========================================

func TestChildService_GetCurrentRecord(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	// Create ongoing contract (no end date)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       from,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	current, err := svc.GetCurrentRecord(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if current.ID != contract.ID {
		t.Errorf("ID = %d, want %d", current.ID, contract.ID)
	}
	if current.ChildID != child.ID {
		t.Errorf("ChildID = %d, want %d", current.ChildID, child.ID)
	}
	if current.Properties["care_type"] != "ganztag" {
		t.Errorf("Properties = %v, want care_type=ganztag", current.Properties)
	}
}

func TestChildService_GetCurrentRecord_NoActiveContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	// Create only an expired contract
	from := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
		To:        &to,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	_, err = svc.GetCurrentRecord(ctx, child.ID, org.ID)
	if err == nil {
		t.Fatal("expected error for no active contract, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// =========================================
// UpdateContract Tests
// =========================================

func TestChildService_UpdateContract_InPlace(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	// Use today's date so the contract qualifies for in-place update
	today := time.Now().UTC().Truncate(24 * time.Hour)
	to := today.AddDate(1, 0, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       today,
		To:         &to,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Update dates and properties — in-place update since contract starts today
	newFrom := today.AddDate(0, 1, 0)
	newTo := today.AddDate(1, 6, 0)
	updateReq := &models.ChildContractUpdateRequest{
		From:       &newFrom,
		To:         &newTo,
		Properties: models.ContractProperties{"care_type": "halbtag"},
	}

	updated, err := svc.UpdateContract(ctx, contract.ID, child.ID, org.ID, updateReq)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should be the same ID (updated in place)
	if updated.ID != contract.ID {
		t.Errorf("ID = %d, want %d (should be updated in place)", updated.ID, contract.ID)
	}
	if !updated.From.Equal(newFrom) {
		t.Errorf("From = %v, want %v", updated.From, newFrom)
	}
	if updated.To == nil || !updated.To.Equal(newTo) {
		t.Errorf("To = %v, want %v", updated.To, newTo)
	}
	if updated.Properties["care_type"] != "halbtag" {
		t.Errorf("Properties = %v, want care_type=halbtag", updated.Properties)
	}
}

func TestChildService_UpdateContract_InvalidPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	// Use today so the contract qualifies for in-place update
	today := time.Now().UTC().Truncate(24 * time.Hour)
	to := today.AddDate(1, 0, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      today,
		To:        &to,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Try to update with 'to' before 'from'
	invalidTo := today.AddDate(-1, 0, 0)
	updateReq := &models.ChildContractUpdateRequest{
		To: &invalidTo,
	}

	_, err = svc.UpdateContract(ctx, contract.ID, child.ID, org.ID, updateReq)
	if err == nil {
		t.Fatal("expected error for invalid period (to before from), got nil")
	}

	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildService_UpdateContract_OverlapConflict(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	// Use future dates so contracts are eligible for in-place update
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Create first contract: today to today+6 months
	from1 := today
	to1 := today.AddDate(0, 6, 0)
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from1,
		To:        &to1,
	})
	if err != nil {
		t.Fatalf("failed to create first contract: %v", err)
	}

	// Create second contract: today+8 months to today+12 months
	from2 := today.AddDate(0, 8, 0)
	to2 := today.AddDate(1, 0, 0)
	contract2, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from2,
		To:        &to2,
	})
	if err != nil {
		t.Fatalf("failed to create second contract: %v", err)
	}

	// Try to update second contract to overlap with first
	overlapFrom := today.AddDate(0, 3, 0)
	updateReq := &models.ChildContractUpdateRequest{
		From: &overlapFrom,
	}

	_, err = svc.UpdateContract(ctx, contract2.ID, child.ID, org.ID, updateReq)
	if err == nil {
		t.Fatal("expected error for overlapping contract, got nil")
	}

	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

// SECURITY TEST: Cross-organization UpdateContract
func TestChildService_UpdateContract_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	child := createTestChild(t, db, "John", "Doe", org1.ID)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	contract, err := svc.CreateContract(ctx, child.ID, org1.ID, &models.ChildContractCreateRequest{SectionID: 1, From: today})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newFrom := today.AddDate(0, 1, 0)
	updateReq := &models.ChildContractUpdateRequest{
		From: &newFrom,
	}

	_, err = svc.UpdateContract(ctx, contract.ID, child.ID, org2.ID, updateReq)
	if err == nil {
		t.Fatal("expected error when updating contract from wrong org, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound (not forbidden - security), got %v", err)
	}
}

func TestChildService_UpdateContract_WrongChild(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child1 := createTestChild(t, db, "John", "Doe", org.ID)
	child2 := createTestChild(t, db, "Jane", "Doe", org.ID)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	contract, err := svc.CreateContract(ctx, child1.ID, org.ID, &models.ChildContractCreateRequest{SectionID: 1, From: today})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newFrom := today.AddDate(0, 1, 0)
	updateReq := &models.ChildContractUpdateRequest{
		From: &newFrom,
	}

	_, err = svc.UpdateContract(ctx, contract.ID, child2.ID, org.ID, updateReq)
	if err == nil {
		t.Fatal("expected error when updating contract via wrong child, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// =========================================
// DeleteContract Tests
// =========================================

func TestChildService_DeleteContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       from,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	err = svc.DeleteContract(ctx, contract.ID, child.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's gone
	contracts, _, err := svc.ListContracts(ctx, child.ID, org.ID, 100, 0)
	if err != nil {
		t.Fatalf("expected no error listing contracts, got %v", err)
	}
	if len(contracts) != 0 {
		t.Errorf("expected 0 contracts after delete, got %d", len(contracts))
	}
}

// =========================================
// Amend-specific UpdateContract Tests
// =========================================

func TestChildService_UpdateContract_AmendChangeSection(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)
	section1 := createTestSection(t, db, "Krippe", org.ID, false)
	section2 := createTestSection(t, db, "Elementar", org.ID, false)

	// Create contract starting in the past (triggers amend mode)
	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  section1.ID,
		From:       past,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Update section → should trigger amend
	updated, err := svc.UpdateContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractUpdateRequest{
		SectionID: &section2.ID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)

	// New contract should have a different ID
	if updated.ID == contract.ID {
		t.Error("expected new contract ID (amend creates new contract)")
	}
	// New contract starts today
	if !updated.From.Truncate(24 * time.Hour).Equal(today) {
		t.Errorf("new contract From = %v, want %v", updated.From, today)
	}
	// New contract has new section
	if updated.SectionID != section2.ID {
		t.Errorf("SectionID = %d, want %d", updated.SectionID, section2.ID)
	}
	// Properties carried over
	if updated.Properties["care_type"] != "ganztag" {
		t.Errorf("Properties should carry over, got %v", updated.Properties)
	}
	// Child ID carried over
	if updated.ChildID != child.ID {
		t.Errorf("ChildID = %d, want %d", updated.ChildID, child.ID)
	}

	// Verify old contract was closed (end = yesterday)
	old, err := svc.GetContractByID(ctx, contract.ID, child.ID, org.ID)
	if err != nil {
		t.Fatalf("failed to get old contract: %v", err)
	}
	if old.To == nil {
		t.Fatal("old contract To should not be nil after amend")
	}
	if !old.To.Truncate(24 * time.Hour).Equal(yesterday) {
		t.Errorf("old contract To = %v, want %v", old.To, yesterday)
	}
}

func TestChildService_UpdateContract_AmendChangeProperties(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  section.ID,
		From:       past,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Update properties → triggers amend
	updated, err := svc.UpdateContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractUpdateRequest{
		Properties: models.ContractProperties{"care_type": "halbtag"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.ID == contract.ID {
		t.Error("expected new contract ID (amend)")
	}
	if updated.Properties["care_type"] != "halbtag" {
		t.Errorf("Properties = %v, want care_type=halbtag", updated.Properties)
	}
	// Section carried over
	if updated.SectionID != section.ID {
		t.Errorf("SectionID should carry over, got %d, want %d", updated.SectionID, section.ID)
	}
}

func TestChildService_UpdateContract_AmendFromIgnored(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: section.ID,
		From:      past,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Try to set From to a specific date — should be ignored in amend mode
	requestedFrom := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	updated, err := svc.UpdateContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractUpdateRequest{
		From: &requestedFrom,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	// From should be today, NOT the requested date
	if !updated.From.Truncate(24 * time.Hour).Equal(today) {
		t.Errorf("From = %v, want today (%v) — From should be ignored in amend mode", updated.From, today)
	}
}

func TestChildService_UpdateContract_AmendToApplied(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: section.ID,
		From:      past,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Set To date in the request
	endDate := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 6, 0)
	updated, err := svc.UpdateContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractUpdateRequest{
		To: &endDate,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.To == nil {
		t.Fatal("To should not be nil")
	}
	if !updated.To.Truncate(24 * time.Hour).Equal(endDate) {
		t.Errorf("To = %v, want %v", updated.To, endDate)
	}
}

func TestChildService_UpdateContract_EndedContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	// Create a contract that has already ended
	past := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	pastEnd := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      past,
		To:        &pastEnd,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Try to update an ended contract → should be rejected
	_, err = svc.UpdateContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractUpdateRequest{
		Properties: models.ContractProperties{"care_type": "halbtag"},
	})
	if err == nil {
		t.Fatal("expected error for ended contract, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildService_UpdateContract_InPlace_FutureContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	// Create a contract starting tomorrow
	tomorrow := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, 1)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: section.ID,
		From:      tomorrow,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Update From date — should be in-place (contract hasn't started yet)
	newFrom := tomorrow.AddDate(0, 0, 7)
	updated, err := svc.UpdateContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractUpdateRequest{
		From: &newFrom,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Same contract ID (in-place)
	if updated.ID != contract.ID {
		t.Errorf("ID = %d, want %d (should be in-place update)", updated.ID, contract.ID)
	}
	if !updated.From.Equal(newFrom) {
		t.Errorf("From = %v, want %v", updated.From, newFrom)
	}
}

func TestChildService_UpdateContract_AmendOverlapConflict(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	past := today.AddDate(0, -3, 0)

	// Create ongoing contract starting in the past (qualifies for amend)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: section.ID,
		From:      past,
	})
	if err != nil {
		t.Fatalf("failed to create first contract: %v", err)
	}

	// Insert a blocking contract directly in DB (bypass overlap validation)
	// This simulates a scenario where another contract exists starting today.
	futureEnd := today.AddDate(1, 0, 0)
	blockingContract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: today, To: &futureEnd},
			SectionID: section.ID,
		},
	}
	if err := db.Create(blockingContract).Error; err != nil {
		t.Fatalf("failed to insert blocking contract: %v", err)
	}

	// Try to amend the first contract:
	// Amend closes old contract (To=yesterday), creates new from=today (ongoing).
	// New contract would overlap with blocking contract (today → future).
	_, err = svc.UpdateContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractUpdateRequest{
		Properties: models.ContractProperties{"care_type": "halbtag"},
	})
	if err == nil {
		t.Fatal("expected overlap conflict error, got nil")
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

// =========================================
// Edge Case Tests: Amend Field Preservation
// =========================================

// Amend with only SectionID change: properties should carry over
func TestChildService_UpdateContract_AmendPreservesProperties(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)
	section1 := createTestSection(t, db, "Krippe", org.ID, false)
	section2 := createTestSection(t, db, "Elementar", org.ID, false)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  section1.ID,
		From:       past,
		Properties: models.ContractProperties{"care_type": "ganztag", "supplements": []interface{}{"ndh", "sprachfoerderung"}},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Update only section — properties should carry over
	updated, err := svc.UpdateContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractUpdateRequest{
		SectionID: &section2.ID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.Properties["care_type"] != "ganztag" {
		t.Errorf("care_type should carry over, got %v", updated.Properties["care_type"])
	}
	if updated.Properties["supplements"] == nil {
		t.Error("supplements should carry over, got nil")
	}
}

// Amend with only Properties change: section should carry over
func TestChildService_UpdateContract_AmendPreservesSection(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  section.ID,
		From:       past,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Update only properties — section should carry over
	updated, err := svc.UpdateContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractUpdateRequest{
		Properties: models.ContractProperties{"care_type": "halbtag"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.SectionID != section.ID {
		t.Errorf("SectionID should carry over, got %d, want %d", updated.SectionID, section.ID)
	}
}

// Amend on ongoing contract (nil To): new contract should also have nil To
func TestChildService_UpdateContract_AmendPreservesOngoingTo(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: section.ID,
		From:      past,
		// No To — ongoing
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Amend: change properties, don't set To in request
	updated, err := svc.UpdateContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractUpdateRequest{
		Properties: models.ContractProperties{"care_type": "halbtag"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// New contract should also be ongoing (nil To)
	if updated.To != nil {
		t.Errorf("new contract To should be nil (ongoing), got %v", updated.To)
	}
}

// Amend on contract with specific To: new contract should carry over To when not in request
func TestChildService_UpdateContract_AmendPreservesToWhenNotInRequest(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	endDate := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 6, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: section.ID,
		From:      past,
		To:        &endDate,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Amend: change properties, don't set To in request
	updated, err := svc.UpdateContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractUpdateRequest{
		Properties: models.ContractProperties{"care_type": "halbtag"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// New contract should have the original To date
	if updated.To == nil {
		t.Fatal("new contract To should not be nil — should carry over from original")
	}
	if !updated.To.Truncate(24 * time.Hour).Equal(endDate) {
		t.Errorf("To = %v, want %v (carried over from original)", updated.To, endDate)
	}
}

// After amend: list contracts shows both old (closed) and new, GetCurrentRecord returns new
func TestChildService_UpdateContract_AmendStateConsistency(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)

	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  section.ID,
		From:       past,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Amend the contract
	newContract, err := svc.UpdateContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractUpdateRequest{
		Properties: models.ContractProperties{"care_type": "halbtag"},
	})
	if err != nil {
		t.Fatalf("amend failed: %v", err)
	}

	// List should show 2 contracts
	contracts, total, err := svc.ListContracts(ctx, child.ID, org.ID, 100, 0)
	if err != nil {
		t.Fatalf("ListContracts failed: %v", err)
	}
	if len(contracts) != 2 {
		t.Fatalf("expected 2 contracts after amend, got %d", len(contracts))
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}

	// Old contract should be closed (To = yesterday)
	oldContract, err := svc.GetContractByID(ctx, contract.ID, child.ID, org.ID)
	if err != nil {
		t.Fatalf("failed to get old contract: %v", err)
	}
	if oldContract.To == nil {
		t.Fatal("old contract To should not be nil")
	}
	if !oldContract.To.Truncate(24 * time.Hour).Equal(yesterday) {
		t.Errorf("old contract To = %v, want %v", oldContract.To, yesterday)
	}

	// GetCurrentRecord should return the new contract
	current, err := svc.GetCurrentRecord(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("GetCurrentRecord failed: %v", err)
	}
	if current.ID != newContract.ID {
		t.Errorf("GetCurrentRecord returned ID %d, want %d (new contract)", current.ID, newContract.ID)
	}
	if current.Properties["care_type"] != "halbtag" {
		t.Errorf("current contract should have updated properties, got %v", current.Properties)
	}
}

// =========================================
// Edge Case Tests: Contract Creation Boundaries
// =========================================

// Adjacent contracts (touching, not overlapping) should succeed
func TestChildService_CreateContract_AdjacentContracts(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	// Contract 1: Jan 1 - Jan 31
	from1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from1,
		To:        &to1,
	})
	if err != nil {
		t.Fatalf("first contract: %v", err)
	}

	// Contract 2: Feb 1 - Feb 28 (day after contract 1 ends — should succeed)
	from2 := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	to2 := time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from2,
		To:        &to2,
	})
	if err != nil {
		t.Fatalf("adjacent contract should succeed, got: %v", err)
	}
}

// Overlapping on single day (inclusive boundaries) should fail
func TestChildService_CreateContract_OverlapOnSameDay(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	// Contract 1: Jan 1 - Jan 31
	from1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from1,
		To:        &to1,
	})
	if err != nil {
		t.Fatalf("first contract: %v", err)
	}

	// Contract 2 starts on Jan 31 (same day as contract 1 ends — should fail, dates are inclusive)
	from2 := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	to2 := time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from2,
		To:        &to2,
	})
	if err == nil {
		t.Fatal("expected overlap error for same-day boundary, got nil")
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

// Create contract in gap between two existing contracts
func TestChildService_CreateContract_InGap(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	// Contract 1: Jan 1 - Mar 31
	from1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from1,
		To:        &to1,
	})
	if err != nil {
		t.Fatalf("first contract: %v", err)
	}

	// Contract 2: Jul 1 - Dec 31
	from2 := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	to2 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from2,
		To:        &to2,
	})
	if err != nil {
		t.Fatalf("second contract: %v", err)
	}

	// Contract 3: Apr 1 - Jun 30 (fills the gap — should succeed)
	from3 := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	to3 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from3,
		To:        &to3,
	})
	if err != nil {
		t.Fatalf("gap contract should succeed, got: %v", err)
	}
}

// =========================================
// Edge Case Tests: Delete
// =========================================

// Delete non-existent contract returns not found
func TestChildService_DeleteContract_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	err := svc.DeleteContract(ctx, 99999, child.ID, org.ID)
	if err == nil {
		t.Fatal("expected error for non-existent contract, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Delete contract belonging to different child
func TestChildService_DeleteContract_WrongChild(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child1 := createTestChild(t, db, "John", "Doe", org.ID)
	child2 := createTestChild(t, db, "Jane", "Doe", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, child1.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Try to delete via wrong child
	err = svc.DeleteContract(ctx, contract.ID, child2.ID, org.ID)
	if err == nil {
		t.Fatal("expected error when deleting via wrong child, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// Verify contract still exists
	contracts, _, _ := svc.ListContracts(ctx, child1.ID, org.ID, 100, 0)
	if len(contracts) != 1 {
		t.Error("contract should still exist")
	}
}

// Same-day contract (From == To) can be created and deleted
func TestChildService_CreateContract_SameDay(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      date,
		To:        &date,
	})
	if err != nil {
		t.Fatalf("same-day contract should succeed, got: %v", err)
	}
	if !contract.From.Equal(date) {
		t.Errorf("From = %v, want %v", contract.From, date)
	}
	if contract.To == nil || !contract.To.Equal(date) {
		t.Errorf("To = %v, want %v", contract.To, date)
	}
}

// =============================================================================
// Contract Properties Distribution Tests
// =============================================================================

func TestChildService_GetContractPropertiesDistribution_BasicScalar(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// 2 children with care_type=ganztag, 1 with care_type=halbtag
	child1 := createTestChild(t, db, "Child1", "A", org.ID)
	createTestChildContract(t, db, child1.ID, from, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	child2 := createTestChild(t, db, "Child2", "B", org.ID)
	createTestChildContract(t, db, child2.ID, from, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	child3 := createTestChild(t, db, "Child3", "C", org.ID)
	createTestChildContract(t, db, child3.ID, from, nil, section.ID, models.ContractProperties{"care_type": "halbtag"})

	stats, err := svc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 3 {
		t.Errorf("TotalChildren = %d, want 3", stats.TotalChildren)
	}

	expected := map[string]int{
		"care_type:ganztag": 2,
		"care_type:halbtag": 1,
	}

	if len(stats.Properties) != len(expected) {
		t.Fatalf("expected %d properties, got %d", len(expected), len(stats.Properties))
	}

	for _, p := range stats.Properties {
		key := p.Key + ":" + p.Value
		if expected[key] != p.Count {
			t.Errorf("property %s: expected count %d, got %d", key, expected[key], p.Count)
		}
	}
}

func TestChildService_GetContractPropertiesDistribution_ArrayProperties(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Child with array property supplements=["ndh","mss"]
	child := createTestChild(t, db, "Child1", "A", org.ID)
	createTestChildContract(t, db, child.ID, from, nil, section.ID, models.ContractProperties{
		"supplements": []string{"ndh", "mss"},
	})

	stats, err := svc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 1 {
		t.Errorf("TotalChildren = %d, want 1", stats.TotalChildren)
	}

	expected := map[string]int{
		"supplements:mss": 1,
		"supplements:ndh": 1,
	}

	if len(stats.Properties) != len(expected) {
		t.Fatalf("expected %d properties, got %d", len(expected), len(stats.Properties))
	}

	for _, p := range stats.Properties {
		key := p.Key + ":" + p.Value
		if expected[key] != p.Count {
			t.Errorf("property %s: expected count %d, got %d", key, expected[key], p.Count)
		}
	}
}

func TestChildService_GetContractPropertiesDistribution_MixedScalarAndArray(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Child 1: scalar care_type + array supplements
	child1 := createTestChild(t, db, "Child1", "A", org.ID)
	createTestChildContract(t, db, child1.ID, from, nil, section.ID, models.ContractProperties{
		"care_type":   "ganztag",
		"supplements": []string{"ndh", "mss"},
	})

	// Child 2: scalar care_type + array supplements (overlapping)
	child2 := createTestChild(t, db, "Child2", "B", org.ID)
	createTestChildContract(t, db, child2.ID, from, nil, section.ID, models.ContractProperties{
		"care_type":   "ganztag",
		"supplements": []string{"ndh"},
	})

	stats, err := svc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 2 {
		t.Errorf("TotalChildren = %d, want 2", stats.TotalChildren)
	}

	expected := map[string]int{
		"care_type:ganztag": 2,
		"supplements:mss":   1,
		"supplements:ndh":   2,
	}

	if len(stats.Properties) != len(expected) {
		t.Fatalf("expected %d properties, got %d", len(expected), len(stats.Properties))
	}

	for _, p := range stats.Properties {
		key := p.Key + ":" + p.Value
		if expected[key] != p.Count {
			t.Errorf("property %s: expected count %d, got %d", key, expected[key], p.Count)
		}
	}
}

func TestChildService_GetContractPropertiesDistribution_NoChildren(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	stats, err := svc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 0 {
		t.Errorf("TotalChildren = %d, want 0", stats.TotalChildren)
	}

	if len(stats.Properties) != 0 {
		t.Errorf("expected 0 properties, got %d", len(stats.Properties))
	}
}

func TestChildService_GetContractPropertiesDistribution_ChildWithNoContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Create child without contract
	createTestChild(t, db, "NoContract", "Child", org.ID)

	stats, err := svc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// FindByOrganizationWithActiveOn only returns children with active contracts
	if stats.TotalChildren != 0 {
		t.Errorf("TotalChildren = %d, want 0 (child has no contract)", stats.TotalChildren)
	}
}

func TestChildService_GetContractPropertiesDistribution_ContractWithNoProperties(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Child with active contract but nil properties
	child := createTestChild(t, db, "NoProps", "Child", org.ID)
	createTestChildContract(t, db, child.ID, from, nil, section.ID, nil)

	stats, err := svc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 1 {
		t.Errorf("TotalChildren = %d, want 1 (child counted even without properties)", stats.TotalChildren)
	}

	if len(stats.Properties) != 0 {
		t.Errorf("expected 0 properties, got %d", len(stats.Properties))
	}
}

func TestChildService_GetContractPropertiesDistribution_ExpiredContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Contract expired before refDate
	child := createTestChild(t, db, "Expired", "Child", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, from, &to, section.ID, models.ContractProperties{"care_type": "ganztag"})

	stats, err := svc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 0 {
		t.Errorf("TotalChildren = %d, want 0 (contract expired)", stats.TotalChildren)
	}
}

func TestChildService_GetContractPropertiesDistribution_FutureContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Contract starts after refDate
	child := createTestChild(t, db, "Future", "Child", org.ID)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, from, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	stats, err := svc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 0 {
		t.Errorf("TotalChildren = %d, want 0 (contract starts in future)", stats.TotalChildren)
	}
}

func TestChildService_GetContractPropertiesDistribution_MultipleChildrenSameValue(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// 5 children all with care_type=ganztag
	for i := 0; i < 5; i++ {
		child := createTestChild(t, db, fmt.Sprintf("Child%d", i), "Same", org.ID)
		createTestChildContract(t, db, child.ID, from, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})
	}

	stats, err := svc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 5 {
		t.Errorf("TotalChildren = %d, want 5", stats.TotalChildren)
	}

	if len(stats.Properties) != 1 {
		t.Fatalf("expected 1 property entry, got %d", len(stats.Properties))
	}

	if stats.Properties[0].Count != 5 {
		t.Errorf("expected count 5, got %d", stats.Properties[0].Count)
	}
}

func TestChildService_GetContractPropertiesDistribution_MultiplePropertiesPerChild(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Child with 3 different property keys
	child := createTestChild(t, db, "Multi", "Props", org.ID)
	createTestChildContract(t, db, child.ID, from, nil, section.ID, models.ContractProperties{
		"care_type":   "ganztag",
		"supplements": []string{"ndh"},
		"lunch":       "yes",
	})

	stats, err := svc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 1 {
		t.Errorf("TotalChildren = %d, want 1", stats.TotalChildren)
	}

	// 3 keys: care_type, lunch, supplements
	if len(stats.Properties) != 3 {
		t.Fatalf("expected 3 property entries, got %d", len(stats.Properties))
	}

	expected := map[string]int{
		"care_type:ganztag": 1,
		"lunch:yes":         1,
		"supplements:ndh":   1,
	}

	for _, p := range stats.Properties {
		key := p.Key + ":" + p.Value
		if expected[key] != p.Count {
			t.Errorf("property %s: expected count %d, got %d", key, expected[key], p.Count)
		}
	}
}

func TestChildService_GetContractPropertiesDistribution_CrossOrgIsolation(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	section1 := getDefaultSection(t, db, org1.ID)
	org2 := createTestOrganization(t, db, "Org 2")

	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create child in org1
	child := createTestChild(t, db, "Child1", "Org1", org1.ID)
	createTestChildContract(t, db, child.ID, from, nil, section1.ID, models.ContractProperties{"care_type": "ganztag"})

	// Query org2 - should not see org1's children
	stats, err := svc.GetContractPropertiesDistribution(ctx, org2.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 0 {
		t.Errorf("TotalChildren = %d, want 0 (child in different org)", stats.TotalChildren)
	}
}

func TestChildService_GetContractPropertiesDistribution_HistoricalDate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)

	// Contract active from 2024-01-01 to 2024-06-30
	child := createTestChild(t, db, "Historical", "Child", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, from, &to, section.ID, models.ContractProperties{"care_type": "ganztag"})

	// Query during active period
	activeDate := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	stats, err := svc.GetContractPropertiesDistribution(ctx, org.ID, activeDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 1 {
		t.Errorf("TotalChildren = %d, want 1 (contract active on historical date)", stats.TotalChildren)
	}

	// Query after contract ended
	afterDate := time.Date(2024, 7, 15, 0, 0, 0, 0, time.UTC)
	stats, err = svc.GetContractPropertiesDistribution(ctx, org.ID, afterDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 0 {
		t.Errorf("TotalChildren = %d, want 0 (contract expired by this date)", stats.TotalChildren)
	}
}

func TestChildService_GetContractPropertiesDistribution_SortedOutput(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create child with properties that should be sorted: z_key before a_key alphabetically
	child := createTestChild(t, db, "Sort", "Test", org.ID)
	createTestChildContract(t, db, child.ID, from, nil, section.ID, models.ContractProperties{
		"z_key": "beta",
		"a_key": "alpha",
	})

	// Another child with z_key=alpha (to have multiple values for z_key)
	child2 := createTestChild(t, db, "Sort2", "Test", org.ID)
	createTestChildContract(t, db, child2.ID, from, nil, section.ID, models.ContractProperties{
		"z_key": "alpha",
	})

	stats, err := svc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Expected order: a_key:alpha, z_key:alpha, z_key:beta
	if len(stats.Properties) != 3 {
		t.Fatalf("expected 3 property entries, got %d", len(stats.Properties))
	}

	expectedOrder := []struct {
		Key   string
		Value string
	}{
		{"a_key", "alpha"},
		{"z_key", "alpha"},
		{"z_key", "beta"},
	}

	for i, exp := range expectedOrder {
		if stats.Properties[i].Key != exp.Key || stats.Properties[i].Value != exp.Value {
			t.Errorf("position %d: expected %s:%s, got %s:%s", i, exp.Key, exp.Value, stats.Properties[i].Key, stats.Properties[i].Value)
		}
	}
}

func TestChildService_GetContractPropertiesDistribution_MultipleActiveContracts(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Create child with two overlapping contracts (via direct DB insert, bypassing overlap validation)
	child := createTestChild(t, db, "Overlap", "Child", org.ID)
	from1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, from1, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})
	from2 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, from2, nil, section.ID, models.ContractProperties{"lunch": "yes"})

	stats, err := svc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Both contracts are active, properties from both should be counted
	expected := map[string]int{
		"care_type:ganztag": 1,
		"lunch:yes":         1,
	}

	for _, p := range stats.Properties {
		key := p.Key + ":" + p.Value
		if expected[key] != p.Count {
			t.Errorf("property %s: expected count %d, got %d", key, expected[key], p.Count)
		}
	}
}

func TestChildService_GetContractPropertiesDistribution_EmptyStringValue(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Child with empty string value
	child := createTestChild(t, db, "Empty", "Value", org.ID)
	createTestChildContract(t, db, child.ID, from, nil, section.ID, models.ContractProperties{"care_type": ""})

	stats, err := svc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 1 {
		t.Errorf("TotalChildren = %d, want 1", stats.TotalChildren)
	}

	if len(stats.Properties) != 1 {
		t.Fatalf("expected 1 property entry, got %d", len(stats.Properties))
	}

	if stats.Properties[0].Key != "care_type" || stats.Properties[0].Value != "" {
		t.Errorf("expected care_type with empty value, got %s:%s", stats.Properties[0].Key, stats.Properties[0].Value)
	}
}
