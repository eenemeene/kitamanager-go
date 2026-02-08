package store

import (
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func setupWaitlistTestDB(t *testing.T) *WaitlistStore {
	t.Helper()
	db := setupTestDB(t)
	db.AutoMigrate(&models.WaitlistEntry{})
	return NewWaitlistStore(db)
}

func createTestWaitlistEntry(t *testing.T, s *WaitlistStore, orgID uint, status string) *models.WaitlistEntry {
	t.Helper()
	entry := &models.WaitlistEntry{
		OrganizationID:   orgID,
		ChildFirstName:   "Lina",
		ChildLastName:    "Mueller",
		ChildBirthdate:   time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC),
		GuardianName:     "Anna Mueller",
		GuardianEmail:    "anna@example.com",
		GuardianPhone:    "+49 170 1234567",
		DesiredStartDate: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		CareType:         "ganztag",
		Status:           status,
		Priority:         0,
	}
	if err := s.Create(entry); err != nil {
		t.Fatalf("failed to create test waitlist entry: %v", err)
	}
	return entry
}

func TestWaitlistStore_Create(t *testing.T) {
	s := setupWaitlistTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")

	entry := &models.WaitlistEntry{
		OrganizationID:   org.ID,
		ChildFirstName:   "Lina",
		ChildLastName:    "Mueller",
		ChildBirthdate:   time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC),
		GuardianName:     "Anna Mueller",
		GuardianEmail:    "anna@example.com",
		DesiredStartDate: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		Status:           models.WaitlistStatusWaiting,
	}

	if err := s.Create(entry); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if entry.ID == 0 {
		t.Error("expected ID to be set")
	}
}

func TestWaitlistStore_FindByID(t *testing.T) {
	s := setupWaitlistTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	entry := createTestWaitlistEntry(t, s, org.ID, models.WaitlistStatusWaiting)

	found, err := s.FindByID(entry.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found.ID != entry.ID {
		t.Errorf("expected ID %d, got %d", entry.ID, found.ID)
	}
	if found.ChildFirstName != "Lina" {
		t.Errorf("expected ChildFirstName 'Lina', got '%s'", found.ChildFirstName)
	}
	if found.GuardianName != "Anna Mueller" {
		t.Errorf("expected GuardianName 'Anna Mueller', got '%s'", found.GuardianName)
	}
}

func TestWaitlistStore_FindByID_NotFound(t *testing.T) {
	s := setupWaitlistTestDB(t)

	_, err := s.FindByID(999)
	if err == nil {
		t.Fatal("expected error for non-existent ID, got nil")
	}
}

func TestWaitlistStore_FindByOrganization(t *testing.T) {
	s := setupWaitlistTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")

	createTestWaitlistEntry(t, s, org.ID, models.WaitlistStatusWaiting)
	createTestWaitlistEntry(t, s, org.ID, models.WaitlistStatusOffered)
	createTestWaitlistEntry(t, s, org.ID, models.WaitlistStatusWaiting)

	entries, total, err := s.FindByOrganization(org.ID, 10, 0)
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

func TestWaitlistStore_FindByOrganizationAndStatus(t *testing.T) {
	s := setupWaitlistTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")

	createTestWaitlistEntry(t, s, org.ID, models.WaitlistStatusWaiting)
	createTestWaitlistEntry(t, s, org.ID, models.WaitlistStatusOffered)
	createTestWaitlistEntry(t, s, org.ID, models.WaitlistStatusWaiting)

	entries, total, err := s.FindByOrganizationAndStatus(org.ID, models.WaitlistStatusWaiting, 10, 0)
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

func TestWaitlistStore_Update(t *testing.T) {
	s := setupWaitlistTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	entry := createTestWaitlistEntry(t, s, org.ID, models.WaitlistStatusWaiting)

	entry.Status = models.WaitlistStatusOffered
	entry.Notes = "Contacted guardian"

	if err := s.Update(entry); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := s.FindByID(entry.ID)
	if found.Status != models.WaitlistStatusOffered {
		t.Errorf("expected status 'offered', got '%s'", found.Status)
	}
	if found.Notes != "Contacted guardian" {
		t.Errorf("expected notes 'Contacted guardian', got '%s'", found.Notes)
	}
}

func TestWaitlistStore_Delete(t *testing.T) {
	s := setupWaitlistTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	entry := createTestWaitlistEntry(t, s, org.ID, models.WaitlistStatusWaiting)

	if err := s.Delete(entry.ID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err := s.FindByID(entry.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestWaitlistStore_CountByOrganizationAndStatus(t *testing.T) {
	s := setupWaitlistTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")

	createTestWaitlistEntry(t, s, org.ID, models.WaitlistStatusWaiting)
	createTestWaitlistEntry(t, s, org.ID, models.WaitlistStatusWaiting)
	createTestWaitlistEntry(t, s, org.ID, models.WaitlistStatusOffered)

	count, err := s.CountByOrganizationAndStatus(org.ID, models.WaitlistStatusWaiting)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

func TestWaitlistStore_OrganizationIsolation(t *testing.T) {
	s := setupWaitlistTestDB(t)
	org1 := createTestOrganization(t, s.db, "Org 1")
	org2 := createTestOrganization(t, s.db, "Org 2")

	createTestWaitlistEntry(t, s, org1.ID, models.WaitlistStatusWaiting)
	createTestWaitlistEntry(t, s, org1.ID, models.WaitlistStatusWaiting)
	createTestWaitlistEntry(t, s, org2.ID, models.WaitlistStatusWaiting)

	entries1, total1, _ := s.FindByOrganization(org1.ID, 10, 0)
	entries2, total2, _ := s.FindByOrganization(org2.ID, 10, 0)

	if total1 != 2 || len(entries1) != 2 {
		t.Errorf("expected 2 entries for org1, got total=%d, len=%d", total1, len(entries1))
	}
	if total2 != 1 || len(entries2) != 1 {
		t.Errorf("expected 1 entry for org2, got total=%d, len=%d", total2, len(entries2))
	}
}

func TestWaitlistStore_Pagination(t *testing.T) {
	s := setupWaitlistTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")

	for i := 0; i < 5; i++ {
		createTestWaitlistEntry(t, s, org.ID, models.WaitlistStatusWaiting)
	}

	// Get first page
	entries, total, err := s.FindByOrganization(org.ID, 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries on first page, got %d", len(entries))
	}

	// Get second page
	entries, _, _ = s.FindByOrganization(org.ID, 2, 2)
	if len(entries) != 2 {
		t.Errorf("expected 2 entries on second page, got %d", len(entries))
	}

	// Get last page
	entries, _, _ = s.FindByOrganization(org.ID, 2, 4)
	if len(entries) != 1 {
		t.Errorf("expected 1 entry on last page, got %d", len(entries))
	}
}
