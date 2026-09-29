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

func TestChildService_FindAllByOrganization(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)

	// Create several children with contracts.
	for i := range 3 {
		createTestChildWithContract(t, db, fmt.Sprintf("Child%d", i), "Doe", org.ID, section.ID)
	}

	results, err := svc.FindAllByOrganization(ctx, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 children, got %d", len(results))
	}
}

func TestChildService_FindAllByOrganization_IsolatesOrgs(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	section1 := getDefaultSection(t, db, org1.ID)
	section2 := getDefaultSection(t, db, org2.ID)

	createTestChildWithContract(t, db, "Child", "Org1", org1.ID, section1.ID)
	createTestChildWithContract(t, db, "Child", "Org2", org2.ID, section2.ID)

	results, err := svc.FindAllByOrganization(ctx, org1.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 child for org1, got %d", len(results))
	}
}

func TestChildService_FindAllByOrganization_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Empty Org")

	results, err := svc.FindAllByOrganization(ctx, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 children, got %d", len(results))
	}
}

// --- Batch Update Contract Tests ---

// --- school entry date (Zurückstellung) ---

func schoolEntry(t *testing.T, s string) *time.Time {
	t.Helper()
	d, err := time.Parse(models.DateFormat, s)
	if err != nil {
		t.Fatalf("bad test date %q: %v", s, err)
	}
	return &d
}

func TestChildService_Create_StoresSchoolEntryDate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()
	org := createTestOrganization(t, db, "Test Org")

	child, err := svc.Create(ctx, org.ID, &models.ChildCreateRequest{
		FirstName: "Mira", LastName: "Sonnenschein", Gender: "female",
		Birthdate:       "2020-05-15",
		SchoolEntryDate: schoolEntry(t, "2027-08-01"),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if child.SchoolEntryDate == nil {
		t.Fatal("SchoolEntryDate = nil, want 2027-08-01")
	}
	if got := child.SchoolEntryDate.Format(models.DateFormat); got != "2027-08-01" {
		t.Errorf("SchoolEntryDate = %s, want 2027-08-01", got)
	}
}

func TestChildService_Create_DefaultsToNil(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()
	org := createTestOrganization(t, db, "Test Org")

	// Nil is what every existing child has, and it means "compute it".
	child, err := svc.Create(ctx, org.ID, &models.ChildCreateRequest{
		FirstName: "Jonas", LastName: "Sonnenschein", Gender: "male", Birthdate: "2020-05-15",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if child.SchoolEntryDate != nil {
		t.Errorf("SchoolEntryDate = %v, want nil", child.SchoolEntryDate)
	}
}

func TestChildService_Create_RejectsDateBeforeBirthdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()
	org := createTestOrganization(t, db, "Test Org")

	_, err := svc.Create(ctx, org.ID, &models.ChildCreateRequest{
		FirstName: "Mira", LastName: "Sonnenschein", Gender: "female",
		Birthdate:       "2020-05-15",
		SchoolEntryDate: schoolEntry(t, "2019-08-01"),
	})
	if err == nil {
		t.Fatal("expected a rejection for a school entry date before the birthdate")
	}
}

func TestChildService_Update_SetsAndClearsSchoolEntryDate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()
	org := createTestOrganization(t, db, "Test Org")

	child, err := svc.Create(ctx, org.ID, &models.ChildCreateRequest{
		FirstName: "Mira", LastName: "Sonnenschein", Gender: "female", Birthdate: "2020-05-15",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Record a Zurückstellung.
	updated, err := svc.Update(ctx, child.ID, org.ID, &models.ChildUpdateRequest{
		SchoolEntryDate: models.Opt[time.Time]{Set: true, Value: schoolEntry(t, "2027-08-01")},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.SchoolEntryDate == nil || updated.SchoolEntryDate.Format(models.DateFormat) != "2027-08-01" {
		t.Fatalf("SchoolEntryDate = %v, want 2027-08-01", updated.SchoolEntryDate)
	}

	// An unrelated edit must leave it alone -- the field is absent, not null.
	updated, err = svc.Update(ctx, child.ID, org.ID, &models.ChildUpdateRequest{
		FirstName: strPtr("Mirabel"),
	})
	if err != nil {
		t.Fatalf("update name: %v", err)
	}
	if updated.SchoolEntryDate == nil {
		t.Fatal("editing the name cleared SchoolEntryDate; absent must mean untouched")
	}

	// Reverse it: present-and-null is the only way to say "no longer deferred".
	updated, err = svc.Update(ctx, child.ID, org.ID, &models.ChildUpdateRequest{
		SchoolEntryDate: models.Opt[time.Time]{Set: true, Value: nil},
	})
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	if updated.SchoolEntryDate != nil {
		t.Errorf("SchoolEntryDate = %v, want nil after an explicit null", updated.SchoolEntryDate)
	}
}

// ---------------------------------------------------------------------------
// Creating a child together with their first contract
// ---------------------------------------------------------------------------

// A child and their first contract are one act, and the API composes them in
// one transaction so a rejected contract cannot leave a child behind.
//
// Composing it client-side is what these replace. Three call sites did it as
// two requests, and when the second failed -- a deleted section, a contract
// before the birthdate, a dropped connection -- the child stayed: absent from a
// list that filters on an active contract, uncounted by funding, and duplicated
// on every retry, because nothing about a child is unique.

func TestChildService_Create_WithContract_CommitsBoth(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSection(t, db, "Krippe", org.ID, false)

	child, err := svc.Create(ctx, org.ID, &models.ChildCreateRequest{
		FirstName: "Emma",
		LastName:  "Schmidt",
		Gender:    "female",
		Birthdate: "2020-05-15",
		Contract: &models.ChildContractCreateRequest{
			From:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			SectionID: section.ID,
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// The response carries the contract, so the caller need not re-read.
	if len(child.Contracts) != 1 {
		t.Fatalf("expected 1 contract on the response, got %d", len(child.Contracts))
	}
	if child.Contracts[0].SectionID != section.ID {
		t.Errorf("SectionID = %d, want %d", child.Contracts[0].SectionID, section.ID)
	}

	var contracts int64
	db.Model(&models.ChildContract{}).Where("child_id = ?", child.ID).Count(&contracts)
	if contracts != 1 {
		t.Errorf("persisted contracts = %d, want 1", contracts)
	}
}

func TestChildService_Create_WithContract_RollsBackChildWhenContractFails(t *testing.T) {
	// The whole point. Each case rejects the contract for a different reason;
	// none of them may leave a child in the database.
	cases := []struct {
		name     string
		contract func(sectionID uint) *models.ChildContractCreateRequest
	}{
		{
			name: "section belongs to no organization",
			contract: func(uint) *models.ChildContractCreateRequest {
				return &models.ChildContractCreateRequest{
					From:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					SectionID: 99999,
				}
			},
		},
		{
			name: "contract starts before the child was born",
			contract: func(sectionID uint) *models.ChildContractCreateRequest {
				return &models.ChildContractCreateRequest{
					From:      time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC),
					SectionID: sectionID,
				}
			},
		},
		{
			name: "period ends before it starts",
			contract: func(sectionID uint) *models.ChildContractCreateRequest {
				to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				return &models.ChildContractCreateRequest{
					From:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					To:        &to,
					SectionID: sectionID,
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := setupTestDB(t)
			svc := createChildService(db)
			ctx := context.Background()

			org := createTestOrganization(t, db, "Test Org")
			section := createTestSection(t, db, "Krippe", org.ID, false)

			_, err := svc.Create(ctx, org.ID, &models.ChildCreateRequest{
				FirstName: "Emma",
				LastName:  "Schmidt",
				Gender:    "female",
				Birthdate: "2020-05-15",
				Contract:  tc.contract(section.ID),
			})
			if err == nil {
				t.Fatal("expected the create to be rejected")
			}

			var children int64
			db.Model(&models.Child{}).Where("organization_id = ?", org.ID).Count(&children)
			if children != 0 {
				t.Errorf("child rows = %d, want 0 -- the child outlived the rejected contract", children)
			}
		})
	}
}

func TestChildService_Create_WithoutContract_StillWorks(t *testing.T) {
	// The field is optional, and every existing caller omits it.
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	child, err := svc.Create(ctx, org.ID, &models.ChildCreateRequest{
		FirstName: "Emma",
		LastName:  "Schmidt",
		Gender:    "female",
		Birthdate: "2020-05-15",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(child.Contracts) != 0 {
		t.Errorf("expected no contracts, got %d", len(child.Contracts))
	}
}
