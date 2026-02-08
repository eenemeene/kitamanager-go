package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func setupWaitlistTest(t *testing.T) (*WaitlistService, *models.Organization) {
	t.Helper()
	db := setupTestDB(t)
	db.AutoMigrate(&models.WaitlistEntry{})

	waitlistStore := store.NewWaitlistStore(db)
	svc := NewWaitlistService(waitlistStore)
	org := createTestOrganization(t, db, "Test Org")

	return svc, org
}

func TestWaitlistService_Create(t *testing.T) {
	svc, org := setupWaitlistTest(t)
	ctx := context.Background()

	req := &models.WaitlistEntryCreateRequest{
		ChildFirstName:   "Lina",
		ChildLastName:    "Mueller",
		ChildBirthdate:   time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC),
		GuardianName:     "Anna Mueller",
		GuardianEmail:    "anna@example.com",
		GuardianPhone:    "+49 170 1234567",
		DesiredStartDate: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		CareType:         "ganztag",
	}

	resp, err := svc.Create(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.ChildFirstName != "Lina" {
		t.Errorf("expected ChildFirstName 'Lina', got '%s'", resp.ChildFirstName)
	}
	if resp.Status != models.WaitlistStatusWaiting {
		t.Errorf("expected status 'waiting', got '%s'", resp.Status)
	}
	if resp.OrganizationID != org.ID {
		t.Errorf("expected OrganizationID %d, got %d", org.ID, resp.OrganizationID)
	}
}

func TestWaitlistService_Create_WhitespaceName(t *testing.T) {
	svc, org := setupWaitlistTest(t)
	ctx := context.Background()

	req := &models.WaitlistEntryCreateRequest{
		ChildFirstName:   "   ",
		ChildLastName:    "Mueller",
		ChildBirthdate:   time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC),
		GuardianName:     "Anna Mueller",
		DesiredStartDate: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
	}

	_, err := svc.Create(ctx, org.ID, req)
	if err == nil {
		t.Fatal("expected error for whitespace-only name, got nil")
	}
}

func TestWaitlistService_GetByID(t *testing.T) {
	svc, org := setupWaitlistTest(t)
	ctx := context.Background()

	req := &models.WaitlistEntryCreateRequest{
		ChildFirstName:   "Lina",
		ChildLastName:    "Mueller",
		ChildBirthdate:   time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC),
		GuardianName:     "Anna Mueller",
		DesiredStartDate: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
	}
	created, _ := svc.Create(ctx, org.ID, req)

	found, err := svc.GetByID(ctx, created.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, found.ID)
	}
}

func TestWaitlistService_GetByID_NotFound(t *testing.T) {
	svc, org := setupWaitlistTest(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 999, org.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestWaitlistService_GetByID_WrongOrg(t *testing.T) {
	svc, org := setupWaitlistTest(t)
	ctx := context.Background()

	req := &models.WaitlistEntryCreateRequest{
		ChildFirstName:   "Lina",
		ChildLastName:    "Mueller",
		ChildBirthdate:   time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC),
		GuardianName:     "Anna Mueller",
		DesiredStartDate: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
	}
	created, _ := svc.Create(ctx, org.ID, req)

	// Try to access from wrong org
	_, err := svc.GetByID(ctx, created.ID, 999)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestWaitlistService_List(t *testing.T) {
	svc, org := setupWaitlistTest(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		svc.Create(ctx, org.ID, &models.WaitlistEntryCreateRequest{
			ChildFirstName:   "Child",
			ChildLastName:    "Name",
			ChildBirthdate:   time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC),
			GuardianName:     "Guardian",
			DesiredStartDate: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		})
	}

	entries, total, err := svc.List(ctx, org.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestWaitlistService_ListByStatus(t *testing.T) {
	svc, org := setupWaitlistTest(t)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		svc.Create(ctx, org.ID, &models.WaitlistEntryCreateRequest{
			ChildFirstName:   "Child",
			ChildLastName:    "Waiting",
			ChildBirthdate:   time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC),
			GuardianName:     "Guardian",
			DesiredStartDate: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		})
	}

	entries, total, err := svc.ListByStatus(ctx, org.ID, models.WaitlistStatusWaiting, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestWaitlistService_ListByStatus_InvalidStatus(t *testing.T) {
	svc, org := setupWaitlistTest(t)
	ctx := context.Background()

	_, _, err := svc.ListByStatus(ctx, org.ID, "invalid", 10, 0)
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
}

func TestWaitlistService_Update(t *testing.T) {
	svc, org := setupWaitlistTest(t)
	ctx := context.Background()

	created, _ := svc.Create(ctx, org.ID, &models.WaitlistEntryCreateRequest{
		ChildFirstName:   "Lina",
		ChildLastName:    "Mueller",
		ChildBirthdate:   time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC),
		GuardianName:     "Anna Mueller",
		DesiredStartDate: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
	})

	offeredStatus := models.WaitlistStatusOffered
	newNotes := "Contacted guardian"
	updateReq := &models.WaitlistEntryUpdateRequest{
		Status: &offeredStatus,
		Notes:  &newNotes,
	}

	updated, err := svc.Update(ctx, created.ID, org.ID, updateReq)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Status != models.WaitlistStatusOffered {
		t.Errorf("expected status 'offered', got '%s'", updated.Status)
	}
	if updated.Notes != "Contacted guardian" {
		t.Errorf("expected notes 'Contacted guardian', got '%s'", updated.Notes)
	}
}

func TestWaitlistService_Update_InvalidStatus(t *testing.T) {
	svc, org := setupWaitlistTest(t)
	ctx := context.Background()

	created, _ := svc.Create(ctx, org.ID, &models.WaitlistEntryCreateRequest{
		ChildFirstName:   "Lina",
		ChildLastName:    "Mueller",
		ChildBirthdate:   time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC),
		GuardianName:     "Anna Mueller",
		DesiredStartDate: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
	})

	invalidStatus := "invalid"
	_, err := svc.Update(ctx, created.ID, org.ID, &models.WaitlistEntryUpdateRequest{
		Status: &invalidStatus,
	})
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
}

func TestWaitlistService_Delete(t *testing.T) {
	svc, org := setupWaitlistTest(t)
	ctx := context.Background()

	created, _ := svc.Create(ctx, org.ID, &models.WaitlistEntryCreateRequest{
		ChildFirstName:   "Lina",
		ChildLastName:    "Mueller",
		ChildBirthdate:   time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC),
		GuardianName:     "Anna Mueller",
		DesiredStartDate: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
	})

	err := svc.Delete(ctx, created.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = svc.GetByID(ctx, created.ID, org.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestWaitlistService_Delete_WrongOrg(t *testing.T) {
	svc, org := setupWaitlistTest(t)
	ctx := context.Background()

	created, _ := svc.Create(ctx, org.ID, &models.WaitlistEntryCreateRequest{
		ChildFirstName:   "Lina",
		ChildLastName:    "Mueller",
		ChildBirthdate:   time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC),
		GuardianName:     "Anna Mueller",
		DesiredStartDate: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
	})

	err := svc.Delete(ctx, created.ID, 999)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
}
