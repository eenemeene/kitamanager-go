package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

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
	err := svc.DeleteContract(ctx, contract.ID, child.ID, org2.ID, nil)
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
// Nullable field clearing tests (child contracts)
// =========================================

func TestChildService_UpdateContract_ClearNullableTo(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	// Create contract with To set (use future date to trigger in-place update)
	from := time.Date(2050, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2050, 12, 31, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
		To:        &to,
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if contract.To == nil {
		t.Fatal("setup: To should be set")
	}

	// Clear To by sending nil (simulates frontend sending null to make open-ended)
	updated, err := svc.CorrectContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractCorrectRequest{
		To: models.OptNull[time.Time](),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.To != nil {
		t.Errorf("To should be nil after clearing, got %v", updated.To)
	}

	// Verify persistence
	refetched, err := svc.GetContractByID(ctx, contract.ID, child.ID, org.ID)
	if err != nil {
		t.Fatalf("re-fetch failed: %v", err)
	}
	if refetched.To != nil {
		t.Errorf("To should be nil after re-fetch, got %v", refetched.To)
	}
}

func TestChildService_UpdateContract_ClearNullableProperties(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	// Create contract with Properties set (use future date to trigger in-place update)
	from := time.Date(2050, 1, 1, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       from,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if contract.Properties == nil {
		t.Fatal("setup: Properties should be set")
	}

	// Clear Properties by sending nil
	updated, err := svc.CorrectContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractCorrectRequest{
		Properties: models.OptNull[models.ContractProperties](),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Properties != nil {
		t.Errorf("Properties should be nil after clearing, got %v", updated.Properties)
	}

	// Verify persistence
	refetched, err := svc.GetContractByID(ctx, contract.ID, child.ID, org.ID)
	if err != nil {
		t.Fatalf("re-fetch failed: %v", err)
	}
	if refetched.Properties != nil {
		t.Errorf("Properties should be nil after re-fetch, got %v", refetched.Properties)
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
	today := models.Today()
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
	updateReq := &models.ChildContractCorrectRequest{
		From:       models.OptOf(newFrom),
		To:         models.OptOf(newTo),
		Properties: models.OptOf(models.ContractProperties{"care_type": "halbtag"}),
	}

	updated, err := svc.CorrectContract(ctx, contract.ID, child.ID, org.ID, updateReq)
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
	today := models.Today()
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
	updateReq := &models.ChildContractCorrectRequest{
		To: models.OptOf(invalidTo),
	}

	_, err = svc.CorrectContract(ctx, contract.ID, child.ID, org.ID, updateReq)
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
	today := models.Today()

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
	updateReq := &models.ChildContractCorrectRequest{
		From: models.OptOf(overlapFrom),
	}

	_, err = svc.CorrectContract(ctx, contract2.ID, child.ID, org.ID, updateReq)
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

	today := models.Today()
	contract, err := svc.CreateContract(ctx, child.ID, org1.ID, &models.ChildContractCreateRequest{SectionID: 1, From: today})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newFrom := today.AddDate(0, 1, 0)
	updateReq := &models.ChildContractCorrectRequest{
		From: models.OptOf(newFrom),
	}

	_, err = svc.CorrectContract(ctx, contract.ID, child.ID, org2.ID, updateReq)
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

	today := models.Today()
	contract, err := svc.CreateContract(ctx, child1.ID, org.ID, &models.ChildContractCreateRequest{SectionID: 1, From: today})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newFrom := today.AddDate(0, 1, 0)
	updateReq := &models.ChildContractCorrectRequest{
		From: models.OptOf(newFrom),
	}

	_, err = svc.CorrectContract(ctx, contract.ID, child2.ID, org.ID, updateReq)
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

	err = svc.DeleteContract(ctx, contract.ID, child.ID, org.ID, nil)
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
	past := models.Today().AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  section1.ID,
		From:       past,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Update section → should trigger amend
	amendResp, err := svc.AmendContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractAmendRequest{
		EffectiveFrom: models.Today(),
		SectionID:     models.OptOf(section2.ID),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	updated := &amendResp.Created

	today := models.Today()
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

	past := models.Today().AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  section.ID,
		From:       past,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Update properties → triggers amend
	amendResp, err := svc.AmendContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractAmendRequest{
		EffectiveFrom: models.Today(),
		Properties:    models.OptOf(models.ContractProperties{"care_type": "halbtag"}),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	updated := &amendResp.Created

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

func TestChildService_UpdateContract_AmendToApplied(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := models.Today().AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: section.ID,
		From:      past,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Set To date in the request
	endDate := models.Today().AddDate(0, 6, 0)
	amendResp, err := svc.AmendContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractAmendRequest{
		EffectiveFrom: models.Today(),
		To:            models.OptOf(endDate),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	updated := &amendResp.Created

	if updated.To == nil {
		t.Fatal("To should not be nil")
	}
	if !updated.To.Truncate(24 * time.Hour).Equal(endDate) {
		t.Errorf("To = %v, want %v", updated.To, endDate)
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
	tomorrow := models.Today().AddDate(0, 0, 1)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: section.ID,
		From:      tomorrow,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Update From date — should be in-place (contract hasn't started yet)
	newFrom := tomorrow.AddDate(0, 0, 7)
	updated, err := svc.CorrectContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractCorrectRequest{
		From: models.OptOf(newFrom),
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

	today := models.Today()
	past := today.AddDate(0, -3, 0)
	originalEnd := today.AddDate(0, 0, 30) // bounded so we can place a blocker after it

	// Original contract: started in the past (amend mode), ends in the future
	// so it remains updatable. The bounded To leaves room for a non-overlapping
	// blocker that the EXCLUDE constraint will accept at setup time.
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: section.ID,
		From:      past,
		To:        &originalEnd,
	})
	if err != nil {
		t.Fatalf("failed to create first contract: %v", err)
	}

	// Blocker contract: starts the day after the original ends, runs for a
	// few months. It coexists with the original because the ranges are
	// disjoint — required by the EXCLUDE constraint added in migration 000022.
	blockerStart := originalEnd.AddDate(0, 0, 1)
	blockerEnd := blockerStart.AddDate(0, 3, 0)
	if _, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: section.ID,
		From:      blockerStart,
		To:        &blockerEnd,
	}); err != nil {
		t.Fatalf("failed to create blocker contract: %v", err)
	}

	// Amend the original AND extend its To past the blocker. Amend closes
	// the old at yesterday and creates a new contract from today with the
	// extended To — that range now overlaps the blocker, and the amend's
	// overlap check inside amendContractTx must produce 409.
	extendedEnd := blockerStart.AddDate(0, 1, 0) // sits inside the blocker's range
	_, err = svc.AmendContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractAmendRequest{
		EffectiveFrom: models.Today(),
		Properties:    models.OptOf(models.ContractProperties{"care_type": "halbtag"}),
		To:            models.OptOf(extendedEnd),
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

	past := models.Today().AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  section1.ID,
		From:       past,
		Properties: models.ContractProperties{"care_type": "ganztag", "supplements": []any{"ndh", "sprachfoerderung"}},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Update only section — properties should carry over
	amendResp, err := svc.AmendContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractAmendRequest{
		EffectiveFrom: models.Today(),
		SectionID:     models.OptOf(section2.ID),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	updated := &amendResp.Created

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

	past := models.Today().AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  section.ID,
		From:       past,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Update only properties — section should carry over
	amendResp, err := svc.AmendContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractAmendRequest{
		EffectiveFrom: models.Today(),
		Properties:    models.OptOf(models.ContractProperties{"care_type": "halbtag"}),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	updated := &amendResp.Created

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

	past := models.Today().AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: section.ID,
		From:      past,
		// No To — ongoing
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Amend: change properties, don't set To in request
	amendResp, err := svc.AmendContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractAmendRequest{
		EffectiveFrom: models.Today(),
		Properties:    models.OptOf(models.ContractProperties{"care_type": "halbtag"}),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	updated := &amendResp.Created

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

	past := models.Today().AddDate(0, -3, 0)
	endDate := models.Today().AddDate(0, 6, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: section.ID,
		From:      past,
		To:        &endDate,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Amend: change properties, don't set To in request
	amendResp, err := svc.AmendContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractAmendRequest{
		EffectiveFrom: models.Today(),
		Properties:    models.OptOf(models.ContractProperties{"care_type": "halbtag"}),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	updated := &amendResp.Created

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

	past := models.Today().AddDate(0, -3, 0)
	today := models.Today()
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
	amendResp, err := svc.AmendContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractAmendRequest{
		EffectiveFrom: models.Today(),
		Properties:    models.OptOf(models.ContractProperties{"care_type": "halbtag"}),
	})
	if err != nil {
		t.Fatalf("amend failed: %v", err)
	}
	newContract := &amendResp.Created

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

	err := svc.DeleteContract(ctx, 99999, child.ID, org.ID, nil)
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
	err = svc.DeleteContract(ctx, contract.ID, child2.ID, org.ID, nil)
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

// --- Auto-apply (ApplyToAllContracts) integration tests ---

// setupAutoApplyFunding creates a government funding with an auto-apply property
// (parent/meals) and a regular property (care_type/ganztag).
// Returns the funding period so callers can reference dates.
func setupAutoApplyFunding(t *testing.T, db *gorm.DB) *models.GovernmentFundingPeriod {
	t.Helper()
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	period := createTestFundingPeriod(t, db, funding.ID,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)

	// Regular properties (NOT auto-applied). Two care types, because contract
	// properties are now checked against what the configuration declares: a test
	// that changes a contract from one care type to another needs both to exist
	// here, and before validation an undeclared value simply went through and
	// earned nothing.
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 100000, 0, 7)
	createTestFundingProperty(t, db, period.ID, "care_type", "halbtag", 60000, 0, 7)

	// Auto-apply property, plus a second value under the same key. The second
	// one is what makes "an explicit value is not overwritten by the default"
	// testable at all: with only one declared value, an explicit setting and the
	// auto-applied one are the same string and the merge branch under test never
	// runs.
	prop := createTestFundingProperty(t, db, period.ID, "parent", "meals", -2300, 0, 7)
	db.Model(prop).Update("apply_to_all_contracts", true)
	createTestFundingProperty(t, db, period.ID, "parent", "no_meals", 0, 0, 7)

	return period
}

func TestChildService_CreateContract_AutoApplyDefaults(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "Anna", "Schmidt", org.ID)
	setupAutoApplyFunding(t, db)
	section := getDefaultSection(t, db, org.ID)

	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  section.ID,
		From:       from,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// care_type was explicitly set by the user
	if contract.Properties["care_type"] != "ganztag" {
		t.Errorf("care_type = %v, want ganztag", contract.Properties["care_type"])
	}
	// parent/meals should be auto-applied
	if contract.Properties["parent"] != "meals" {
		t.Errorf("parent = %v, want meals (auto-applied)", contract.Properties["parent"])
	}
}

func TestChildService_CreateContract_AutoApplyNoOverwrite(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "Ben", "Müller", org.ID)
	setupAutoApplyFunding(t, db)
	section := getDefaultSection(t, db, org.ID)

	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  section.ID,
		From:       from,
		Properties: models.ContractProperties{"care_type": "ganztag", "parent": "no_meals"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Explicit parent value must NOT be overwritten by auto-apply. The value has
	// to be one the configuration declares -- an invented one is refused now --
	// but it still has to differ from the auto-applied "meals", or the assertion
	// would hold whether or not the merge respected it.
	if contract.Properties["parent"] != "no_meals" {
		t.Errorf("parent = %v, want no_meals (explicit should win over auto-apply)", contract.Properties["parent"])
	}
}

func TestChildService_CreateContract_AutoApplyNilProperties(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "Clara", "Weber", org.ID)
	setupAutoApplyFunding(t, db)
	section := getDefaultSection(t, db, org.ID)

	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  section.ID,
		From:       from,
		Properties: nil, // no properties at all
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Auto-apply should still populate parent/meals even with nil input
	if contract.Properties["parent"] != "meals" {
		t.Errorf("parent = %v, want meals (auto-applied into nil properties)", contract.Properties["parent"])
	}
}

func TestChildService_CreateContract_NoFundingGraceful(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "David", "Fischer", org.ID)
	// No government funding created
	section := getDefaultSection(t, db, org.ID)

	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  section.ID,
		From:       from,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Contract should work fine with just the explicit properties
	if contract.Properties["care_type"] != "ganztag" {
		t.Errorf("care_type = %v, want ganztag", contract.Properties["care_type"])
	}
	if _, ok := contract.Properties["parent"]; ok {
		t.Errorf("parent should not be set when no funding exists, got %v", contract.Properties["parent"])
	}
}

func TestChildService_UpdateContract_InPlace_AutoApply(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "Eva", "Koch", org.ID)
	setupAutoApplyFunding(t, db)
	section := getDefaultSection(t, db, org.ID)

	// Create contract starting in the future (update-in-place mode)
	future := models.Today().AddDate(0, 3, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  section.ID,
		From:       future,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Update in place — change properties but omit parent
	updated, err := svc.CorrectContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractCorrectRequest{
		Properties: models.OptOf(models.ContractProperties{"care_type": "halbtag"}),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Same contract (in-place update)
	if updated.ID != contract.ID {
		t.Errorf("expected same contract ID (in-place), got %d", updated.ID)
	}
	if updated.Properties["care_type"] != "halbtag" {
		t.Errorf("care_type = %v, want halbtag", updated.Properties["care_type"])
	}
	// Auto-apply should fill in parent/meals
	if updated.Properties["parent"] != "meals" {
		t.Errorf("parent = %v, want meals (auto-applied on update)", updated.Properties["parent"])
	}
}

func TestChildService_UpdateContract_Amend_AutoApply(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "Fritz", "Bauer", org.ID)
	setupAutoApplyFunding(t, db)
	section1 := getDefaultSection(t, db, org.ID)
	section2 := createTestSection(t, db, "Elementar", org.ID, false)

	// Create contract starting in the past (triggers amend mode)
	past := models.Today().AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  section1.ID,
		From:       past,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Amend: change section (triggers new contract creation)
	amendResp, err := svc.AmendContract(ctx, contract.ID, child.ID, org.ID, &models.ChildContractAmendRequest{
		EffectiveFrom: models.Today(),
		SectionID:     models.OptOf(section2.ID),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	amended := &amendResp.Created

	// Different contract ID (amend creates new)
	if amended.ID == contract.ID {
		t.Error("expected new contract ID (amend creates new contract)")
	}
	// Properties carried over + auto-apply merged
	if amended.Properties["care_type"] != "ganztag" {
		t.Errorf("care_type = %v, want ganztag (carried over)", amended.Properties["care_type"])
	}
	if amended.Properties["parent"] != "meals" {
		t.Errorf("parent = %v, want meals (auto-applied on amend)", amended.Properties["parent"])
	}
}
