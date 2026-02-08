package store

import (
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func setupAttendanceTestDB(t *testing.T) *AttendanceStore {
	t.Helper()
	db := setupTestDB(t)
	db.AutoMigrate(&models.Attendance{})
	return NewAttendanceStore(db)
}

func createAttendanceTestChild(t *testing.T, store *AttendanceStore, orgID uint) *models.Child {
	t.Helper()
	child := &models.Child{
		Person: models.Person{
			FirstName:      "Test",
			LastName:       "Child",
			OrganizationID: orgID,
			Birthdate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	if err := store.db.Create(child).Error; err != nil {
		t.Fatalf("failed to create test child: %v", err)
	}
	return child
}

func TestAttendanceStore_Create(t *testing.T) {
	s := setupAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createAttendanceTestChild(t, s, org.ID)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	attendance := &models.Attendance{
		ChildID:        child.ID,
		OrganizationID: org.ID,
		Date:           today,
		CheckInTime:    &now,
		Status:         models.AttendanceStatusPresent,
		RecordedBy:     1,
	}

	if err := s.Create(attendance); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if attendance.ID == 0 {
		t.Error("expected ID to be set")
	}
}

func TestAttendanceStore_FindByID(t *testing.T) {
	s := setupAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createAttendanceTestChild(t, s, org.ID)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	attendance := &models.Attendance{
		ChildID:        child.ID,
		OrganizationID: org.ID,
		Date:           today,
		CheckInTime:    &now,
		Status:         models.AttendanceStatusPresent,
		RecordedBy:     1,
	}
	s.Create(attendance)

	found, err := s.FindByID(attendance.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found.ID != attendance.ID {
		t.Errorf("expected ID %d, got %d", attendance.ID, found.ID)
	}
	if found.ChildID != child.ID {
		t.Errorf("expected ChildID %d, got %d", child.ID, found.ChildID)
	}
	if found.Status != models.AttendanceStatusPresent {
		t.Errorf("expected status present, got %s", found.Status)
	}
}

func TestAttendanceStore_FindByID_NotFound(t *testing.T) {
	s := setupAttendanceTestDB(t)

	_, err := s.FindByID(999)
	if err == nil {
		t.Fatal("expected error for non-existent ID, got nil")
	}
}

func TestAttendanceStore_FindByChildAndDate(t *testing.T) {
	s := setupAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createAttendanceTestChild(t, s, org.ID)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	attendance := &models.Attendance{
		ChildID:        child.ID,
		OrganizationID: org.ID,
		Date:           today,
		CheckInTime:    &now,
		Status:         models.AttendanceStatusPresent,
		RecordedBy:     1,
	}
	s.Create(attendance)

	found, err := s.FindByChildAndDate(child.ID, today)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found.ChildID != child.ID {
		t.Errorf("expected ChildID %d, got %d", child.ID, found.ChildID)
	}
}

func TestAttendanceStore_FindByOrganizationAndDate(t *testing.T) {
	s := setupAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child1 := createAttendanceTestChild(t, s, org.ID)
	child2 := createAttendanceTestChild(t, s, org.ID)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	s.Create(&models.Attendance{ChildID: child1.ID, OrganizationID: org.ID, Date: today, Status: models.AttendanceStatusPresent, RecordedBy: 1, CheckInTime: &now})
	s.Create(&models.Attendance{ChildID: child2.ID, OrganizationID: org.ID, Date: today, Status: models.AttendanceStatusSick, RecordedBy: 1})

	records, total, err := s.FindByOrganizationAndDate(org.ID, today, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
}

func TestAttendanceStore_FindByChildAndDateRange(t *testing.T) {
	s := setupAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createAttendanceTestChild(t, s, org.ID)

	now := time.Now()
	day1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)

	s.Create(&models.Attendance{ChildID: child.ID, OrganizationID: org.ID, Date: day1, Status: models.AttendanceStatusPresent, RecordedBy: 1, CheckInTime: &now})
	s.Create(&models.Attendance{ChildID: child.ID, OrganizationID: org.ID, Date: day2, Status: models.AttendanceStatusSick, RecordedBy: 1})
	s.Create(&models.Attendance{ChildID: child.ID, OrganizationID: org.ID, Date: day3, Status: models.AttendanceStatusPresent, RecordedBy: 1, CheckInTime: &now})

	records, total, err := s.FindByChildAndDateRange(child.ID, day1, day3, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records, got %d", len(records))
	}

	// Only days 1-2
	records, total, err = s.FindByChildAndDateRange(child.ID, day1, day2, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestAttendanceStore_Update(t *testing.T) {
	s := setupAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createAttendanceTestChild(t, s, org.ID)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	attendance := &models.Attendance{
		ChildID:        child.ID,
		OrganizationID: org.ID,
		Date:           today,
		CheckInTime:    &now,
		Status:         models.AttendanceStatusPresent,
		RecordedBy:     1,
	}
	s.Create(attendance)

	checkOut := time.Now()
	attendance.CheckOutTime = &checkOut
	attendance.Note = "Updated note"

	if err := s.Update(attendance); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := s.FindByID(attendance.ID)
	if found.CheckOutTime == nil {
		t.Error("expected CheckOutTime to be set")
	}
	if found.Note != "Updated note" {
		t.Errorf("expected note 'Updated note', got '%s'", found.Note)
	}
}

func TestAttendanceStore_Delete(t *testing.T) {
	s := setupAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createAttendanceTestChild(t, s, org.ID)

	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	attendance := &models.Attendance{
		ChildID:        child.ID,
		OrganizationID: org.ID,
		Date:           today,
		Status:         models.AttendanceStatusAbsent,
		RecordedBy:     1,
	}
	s.Create(attendance)

	if err := s.Delete(attendance.ID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err := s.FindByID(attendance.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestAttendanceStore_GetDailySummary(t *testing.T) {
	s := setupAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")

	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	now := time.Now()

	// Create 5 children with different statuses
	for i := 0; i < 3; i++ {
		child := createAttendanceTestChild(t, s, org.ID)
		s.Create(&models.Attendance{ChildID: child.ID, OrganizationID: org.ID, Date: today, Status: models.AttendanceStatusPresent, RecordedBy: 1, CheckInTime: &now})
	}
	for i := 0; i < 1; i++ {
		child := createAttendanceTestChild(t, s, org.ID)
		s.Create(&models.Attendance{ChildID: child.ID, OrganizationID: org.ID, Date: today, Status: models.AttendanceStatusSick, RecordedBy: 1})
	}
	for i := 0; i < 1; i++ {
		child := createAttendanceTestChild(t, s, org.ID)
		s.Create(&models.Attendance{ChildID: child.ID, OrganizationID: org.ID, Date: today, Status: models.AttendanceStatusVacation, RecordedBy: 1})
	}

	summary, err := s.GetDailySummary(org.ID, today)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if summary.TotalChildren != 5 {
		t.Errorf("expected 5 total, got %d", summary.TotalChildren)
	}
	if summary.Present != 3 {
		t.Errorf("expected 3 present, got %d", summary.Present)
	}
	if summary.Sick != 1 {
		t.Errorf("expected 1 sick, got %d", summary.Sick)
	}
	if summary.Vacation != 1 {
		t.Errorf("expected 1 vacation, got %d", summary.Vacation)
	}
}
