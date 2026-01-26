package service

import (
	"context"
	"errors"
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
		Birthdate: time.Date(2020, 5, 15, 0, 0, 0, 0, time.UTC),
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
		{"empty first name", &models.ChildCreateRequest{FirstName: "", LastName: "Doe", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}},
		{"whitespace first name", &models.ChildCreateRequest{FirstName: "   ", LastName: "Doe", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}},
		{"empty last name", &models.ChildCreateRequest{FirstName: "John", LastName: "", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}},
		{"whitespace last name", &models.ChildCreateRequest{FirstName: "John", LastName: "   ", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}},
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
		Birthdate: time.Date(2020, 5, 15, 0, 0, 0, 0, time.UTC),
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
		Birthdate: time.Now().AddDate(1, 0, 0), // 1 year in future
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
	group := createTestGroupWithOrg(t, db, "Test Group", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	req := &models.ChildContractCreateRequest{
		From:             from,
		To:               &to,
		CareHoursPerWeek: 40,
		GroupID:          &group.ID,
		MealsIncluded:    true,
		SpecialNeeds:     "None",
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
	if contract.CareHoursPerWeek != 40 {
		t.Errorf("CareHoursPerWeek = %v, want 40", contract.CareHoursPerWeek)
	}
}

func TestChildService_CreateContract_ChildNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildContractCreateRequest{
		From:             from,
		CareHoursPerWeek: 40,
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
		From:             from,
		CareHoursPerWeek: 40,
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

func TestChildService_CreateContract_GroupDifferentOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	child := createTestChild(t, db, "John", "Doe", org1.ID)
	group := createTestGroupWithOrg(t, db, "Test Group", org2.ID) // Different org!

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildContractCreateRequest{
		From:             from,
		CareHoursPerWeek: 40,
		GroupID:          &group.ID,
	}

	_, err := svc.CreateContract(ctx, child.ID, org1.ID, req)
	if err == nil {
		t.Fatal("expected error for group in different org, got nil")
	}

	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
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
		From:             from,
		To:               &to,
		CareHoursPerWeek: 40,
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
		From:             from1,
		To:               &to1,
		CareHoursPerWeek: 40,
	}
	_, err := svc.CreateContract(ctx, child.ID, org.ID, req1)
	if err != nil {
		t.Fatalf("first contract: expected no error, got %v", err)
	}

	// Try to create overlapping contract
	from2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC) // Overlaps with first
	to2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	req2 := &models.ChildContractCreateRequest{
		From:             from2,
		To:               &to2,
		CareHoursPerWeek: 30,
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
		From:             from,
		To:               nil,
		CareHoursPerWeek: 40,
	}

	contract, err := svc.CreateContract(ctx, child.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if contract.To != nil {
		t.Errorf("To = %v, want nil (ongoing)", contract.To)
	}
}

func TestChildService_CreateContract_HTMLSanitization(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "John", "Doe", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildContractCreateRequest{
		From:             from,
		CareHoursPerWeek: 40,
		SpecialNeeds:     "<script>alert('xss')</script>Allergy to peanuts",
	}

	contract, err := svc.CreateContract(ctx, child.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Script tags should be removed
	if contract.SpecialNeeds == "<script>alert('xss')</script>Allergy to peanuts" {
		t.Error("expected HTML to be sanitized")
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
	req1 := &models.ChildContractCreateRequest{From: from1, To: &to1, CareHoursPerWeek: 30}
	_, _ = svc.CreateContract(ctx, child.ID, org.ID, req1)

	from2 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req2 := &models.ChildContractCreateRequest{From: from2, CareHoursPerWeek: 40}
	_, _ = svc.CreateContract(ctx, child.ID, org.ID, req2)

	contracts, err := svc.ListContracts(ctx, child.ID, org.ID)
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

	_, err := svc.ListContracts(ctx, 999, org.ID)
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
	req := &models.ChildContractCreateRequest{From: from, CareHoursPerWeek: 40}
	_, _ = svc.CreateContract(ctx, child.ID, org1.ID, req)

	// Try to list contracts from wrong organization
	_, err := svc.ListContracts(ctx, child.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error when listing contracts from wrong org, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound (not forbidden - security), got %v", err)
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
	req := &models.ChildContractCreateRequest{From: from, CareHoursPerWeek: 40}
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
	contracts, _ := svc.ListContracts(ctx, child.ID, org1.ID)
	if len(contracts) != 1 {
		t.Error("contract was deleted despite wrong org")
	}
}

// SECURITY TEST: GetCurrentContract cross-org
func TestChildService_GetCurrentContract_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	child := createTestChild(t, db, "John", "Doe", org1.ID)

	// Create an ongoing contract
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildContractCreateRequest{From: from, CareHoursPerWeek: 40}
	_, _ = svc.CreateContract(ctx, child.ID, org1.ID, req)

	// Try to get current contract from wrong organization
	_, err := svc.GetCurrentContract(ctx, child.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error when getting current contract from wrong org, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound (not forbidden - security), got %v", err)
	}
}
