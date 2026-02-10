package store

import (
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func setupChildAttendanceTestDB(t *testing.T) *ChildAttendanceStore {
	t.Helper()
	db := setupTestDB(t)
	db.AutoMigrate(&models.ChildAttendance{})
	return NewChildAttendanceStore(db)
}

func createChildAttendanceTestChild(t *testing.T, store *ChildAttendanceStore, orgID uint) *models.Child {
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

func TestChildAttendanceStore_Create(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildAttendanceTestChild(t, s, org.ID)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	attendance := &models.ChildAttendance{
		ChildID:        child.ID,
		OrganizationID: org.ID,
		Date:           today,
		CheckInTime:    &now,
		Status:         models.ChildAttendanceStatusPresent,
		RecordedBy:     1,
	}

	if err := s.Create(attendance); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if attendance.ID == 0 {
		t.Error("expected ID to be set")
	}
}

func TestChildAttendanceStore_Create_AllStatuses(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")

	statuses := []string{
		models.ChildAttendanceStatusPresent,
		models.ChildAttendanceStatusAbsent,
		models.ChildAttendanceStatusSick,
		models.ChildAttendanceStatusVacation,
	}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			child := createChildAttendanceTestChild(t, s, org.ID)
			attendance := &models.ChildAttendance{
				ChildID:        child.ID,
				OrganizationID: org.ID,
				Date:           time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
				Status:         status,
				RecordedBy:     1,
			}
			if err := s.Create(attendance); err != nil {
				t.Fatalf("failed to create attendance with status %s: %v", status, err)
			}
			found, err := s.FindByID(attendance.ID)
			if err != nil {
				t.Fatalf("failed to find attendance: %v", err)
			}
			if found.Status != status {
				t.Errorf("expected status %s, got %s", status, found.Status)
			}
		})
	}
}

func TestChildAttendanceStore_FindByID(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildAttendanceTestChild(t, s, org.ID)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	attendance := &models.ChildAttendance{
		ChildID:        child.ID,
		OrganizationID: org.ID,
		Date:           today,
		CheckInTime:    &now,
		Status:         models.ChildAttendanceStatusPresent,
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
	if found.Status != models.ChildAttendanceStatusPresent {
		t.Errorf("expected status present, got %s", found.Status)
	}
}

func TestChildAttendanceStore_FindByID_NotFound(t *testing.T) {
	s := setupChildAttendanceTestDB(t)

	_, err := s.FindByID(999)
	if err == nil {
		t.Fatal("expected error for non-existent ID, got nil")
	}
}

func TestChildAttendanceStore_FindByID_PreloadsChild(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildAttendanceTestChild(t, s, org.ID)

	attendance := &models.ChildAttendance{
		ChildID:        child.ID,
		OrganizationID: org.ID,
		Date:           time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		Status:         models.ChildAttendanceStatusPresent,
		RecordedBy:     1,
	}
	s.Create(attendance)

	found, err := s.FindByID(attendance.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found.Child == nil {
		t.Fatal("expected Child to be preloaded")
	}
	if found.Child.FirstName != "Test" {
		t.Errorf("expected child first name 'Test', got '%s'", found.Child.FirstName)
	}
}

func TestChildAttendanceStore_FindByChildAndDate(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildAttendanceTestChild(t, s, org.ID)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	attendance := &models.ChildAttendance{
		ChildID:        child.ID,
		OrganizationID: org.ID,
		Date:           today,
		CheckInTime:    &now,
		Status:         models.ChildAttendanceStatusPresent,
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

func TestChildAttendanceStore_FindByChildAndDate_NotFound(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildAttendanceTestChild(t, s, org.ID)

	_, err := s.FindByChildAndDate(child.ID, time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected error for non-existent record, got nil")
	}
}

func TestChildAttendanceStore_FindByChildAndDate_DifferentDates(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildAttendanceTestChild(t, s, org.ID)

	day1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	s.Create(&models.ChildAttendance{ChildID: child.ID, OrganizationID: org.ID, Date: day1, Status: models.ChildAttendanceStatusPresent, RecordedBy: 1})
	s.Create(&models.ChildAttendance{ChildID: child.ID, OrganizationID: org.ID, Date: day2, Status: models.ChildAttendanceStatusSick, RecordedBy: 1})

	found, _ := s.FindByChildAndDate(child.ID, day1)
	if found.Status != models.ChildAttendanceStatusPresent {
		t.Errorf("expected present for day1, got %s", found.Status)
	}

	found, _ = s.FindByChildAndDate(child.ID, day2)
	if found.Status != models.ChildAttendanceStatusSick {
		t.Errorf("expected sick for day2, got %s", found.Status)
	}
}

func TestChildAttendanceStore_FindByOrganizationAndDate(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child1 := createChildAttendanceTestChild(t, s, org.ID)
	child2 := createChildAttendanceTestChild(t, s, org.ID)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	s.Create(&models.ChildAttendance{ChildID: child1.ID, OrganizationID: org.ID, Date: today, Status: models.ChildAttendanceStatusPresent, RecordedBy: 1, CheckInTime: &now})
	s.Create(&models.ChildAttendance{ChildID: child2.ID, OrganizationID: org.ID, Date: today, Status: models.ChildAttendanceStatusSick, RecordedBy: 1})

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

func TestChildAttendanceStore_FindByOrganizationAndDate_CrossOrgIsolation(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org1 := createTestOrganization(t, s.db, "Org 1")
	org2 := createTestOrganization(t, s.db, "Org 2")
	child1 := createChildAttendanceTestChild(t, s, org1.ID)
	child2 := createChildAttendanceTestChild(t, s, org2.ID)

	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	s.Create(&models.ChildAttendance{ChildID: child1.ID, OrganizationID: org1.ID, Date: today, Status: models.ChildAttendanceStatusPresent, RecordedBy: 1})
	s.Create(&models.ChildAttendance{ChildID: child2.ID, OrganizationID: org2.ID, Date: today, Status: models.ChildAttendanceStatusPresent, RecordedBy: 1})

	records, total, err := s.FindByOrganizationAndDate(org1.ID, today, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1 for org1, got %d", total)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record for org1, got %d", len(records))
	}
}

func TestChildAttendanceStore_FindByOrganizationAndDate_EmptyResult(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")

	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	records, total, err := s.FindByOrganizationAndDate(org.ID, today, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestChildAttendanceStore_FindByOrganizationAndDate_Pagination(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 5; i++ {
		child := createChildAttendanceTestChild(t, s, org.ID)
		s.Create(&models.ChildAttendance{ChildID: child.ID, OrganizationID: org.ID, Date: today, Status: models.ChildAttendanceStatusPresent, RecordedBy: 1})
	}

	// Page 1: limit 2
	records, total, err := s.FindByOrganizationAndDate(org.ID, today, 2, 0)
	if err != nil {
		t.Fatalf("page 1 error: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records on page 1, got %d", len(records))
	}

	// Page 2: limit 2, offset 2
	records, _, err = s.FindByOrganizationAndDate(org.ID, today, 2, 2)
	if err != nil {
		t.Fatalf("page 2 error: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records on page 2, got %d", len(records))
	}

	// Page 3: limit 2, offset 4
	records, _, err = s.FindByOrganizationAndDate(org.ID, today, 2, 4)
	if err != nil {
		t.Fatalf("page 3 error: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record on page 3, got %d", len(records))
	}
}

func TestChildAttendanceStore_FindByChildAndDateRange(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildAttendanceTestChild(t, s, org.ID)

	now := time.Now()
	day1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)

	s.Create(&models.ChildAttendance{ChildID: child.ID, OrganizationID: org.ID, Date: day1, Status: models.ChildAttendanceStatusPresent, RecordedBy: 1, CheckInTime: &now})
	s.Create(&models.ChildAttendance{ChildID: child.ID, OrganizationID: org.ID, Date: day2, Status: models.ChildAttendanceStatusSick, RecordedBy: 1})
	s.Create(&models.ChildAttendance{ChildID: child.ID, OrganizationID: org.ID, Date: day3, Status: models.ChildAttendanceStatusPresent, RecordedBy: 1, CheckInTime: &now})

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

	// Subset: only days 1-2
	records, total, err = s.FindByChildAndDateRange(child.ID, day1, day2, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestChildAttendanceStore_FindByChildAndDateRange_EmptyRange(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildAttendanceTestChild(t, s, org.ID)

	from := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)

	records, total, err := s.FindByChildAndDateRange(child.ID, from, to, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestChildAttendanceStore_FindByChildAndDateRange_Pagination(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildAttendanceTestChild(t, s, org.ID)

	for i := 1; i <= 5; i++ {
		day := time.Date(2025, 1, i, 0, 0, 0, 0, time.UTC)
		s.Create(&models.ChildAttendance{ChildID: child.ID, OrganizationID: org.ID, Date: day, Status: models.ChildAttendanceStatusPresent, RecordedBy: 1})
	}

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)

	records, total, err := s.FindByChildAndDateRange(child.ID, from, to, 3, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records with limit, got %d", len(records))
	}
}

func TestChildAttendanceStore_Update(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildAttendanceTestChild(t, s, org.ID)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	attendance := &models.ChildAttendance{
		ChildID:        child.ID,
		OrganizationID: org.ID,
		Date:           today,
		CheckInTime:    &now,
		Status:         models.ChildAttendanceStatusPresent,
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

func TestChildAttendanceStore_Update_StatusChange(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildAttendanceTestChild(t, s, org.ID)

	attendance := &models.ChildAttendance{
		ChildID:        child.ID,
		OrganizationID: org.ID,
		Date:           time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		Status:         models.ChildAttendanceStatusPresent,
		RecordedBy:     1,
	}
	s.Create(attendance)

	attendance.Status = models.ChildAttendanceStatusSick
	if err := s.Update(attendance); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := s.FindByID(attendance.ID)
	if found.Status != models.ChildAttendanceStatusSick {
		t.Errorf("expected status sick, got %s", found.Status)
	}
}

func TestChildAttendanceStore_Delete(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildAttendanceTestChild(t, s, org.ID)

	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	attendance := &models.ChildAttendance{
		ChildID:        child.ID,
		OrganizationID: org.ID,
		Date:           today,
		Status:         models.ChildAttendanceStatusAbsent,
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

func TestChildAttendanceStore_Delete_NonExistent(t *testing.T) {
	s := setupChildAttendanceTestDB(t)

	// Deleting a non-existent record should not error in GORM (soft-delete or no rows affected)
	err := s.Delete(9999)
	if err != nil {
		t.Fatalf("expected no error deleting non-existent record, got %v", err)
	}
}

func TestChildAttendanceStore_GetDailySummary(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")

	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	now := time.Now()

	// 3 present, 1 absent, 1 sick, 1 vacation
	for i := 0; i < 3; i++ {
		child := createChildAttendanceTestChild(t, s, org.ID)
		s.Create(&models.ChildAttendance{ChildID: child.ID, OrganizationID: org.ID, Date: today, Status: models.ChildAttendanceStatusPresent, RecordedBy: 1, CheckInTime: &now})
	}
	childAbsent := createChildAttendanceTestChild(t, s, org.ID)
	s.Create(&models.ChildAttendance{ChildID: childAbsent.ID, OrganizationID: org.ID, Date: today, Status: models.ChildAttendanceStatusAbsent, RecordedBy: 1})
	childSick := createChildAttendanceTestChild(t, s, org.ID)
	s.Create(&models.ChildAttendance{ChildID: childSick.ID, OrganizationID: org.ID, Date: today, Status: models.ChildAttendanceStatusSick, RecordedBy: 1})
	childVac := createChildAttendanceTestChild(t, s, org.ID)
	s.Create(&models.ChildAttendance{ChildID: childVac.ID, OrganizationID: org.ID, Date: today, Status: models.ChildAttendanceStatusVacation, RecordedBy: 1})

	summary, err := s.GetDailySummary(org.ID, today)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if summary.TotalChildren != 6 {
		t.Errorf("expected 6 total, got %d", summary.TotalChildren)
	}
	if summary.Present != 3 {
		t.Errorf("expected 3 present, got %d", summary.Present)
	}
	if summary.Absent != 1 {
		t.Errorf("expected 1 absent, got %d", summary.Absent)
	}
	if summary.Sick != 1 {
		t.Errorf("expected 1 sick, got %d", summary.Sick)
	}
	if summary.Vacation != 1 {
		t.Errorf("expected 1 vacation, got %d", summary.Vacation)
	}
	if summary.Date != "2025-06-15" {
		t.Errorf("expected date '2025-06-15', got '%s'", summary.Date)
	}
}

func TestChildAttendanceStore_GetDailySummary_EmptyDay(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")

	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	summary, err := s.GetDailySummary(org.ID, today)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if summary.TotalChildren != 0 {
		t.Errorf("expected 0 total, got %d", summary.TotalChildren)
	}
	if summary.Present != 0 {
		t.Errorf("expected 0 present, got %d", summary.Present)
	}
}

func TestChildAttendanceStore_GetDailySummary_CrossOrgIsolation(t *testing.T) {
	s := setupChildAttendanceTestDB(t)
	org1 := createTestOrganization(t, s.db, "Org 1")
	org2 := createTestOrganization(t, s.db, "Org 2")

	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	now := time.Now()

	child1 := createChildAttendanceTestChild(t, s, org1.ID)
	child2 := createChildAttendanceTestChild(t, s, org2.ID)
	child3 := createChildAttendanceTestChild(t, s, org2.ID)

	s.Create(&models.ChildAttendance{ChildID: child1.ID, OrganizationID: org1.ID, Date: today, Status: models.ChildAttendanceStatusPresent, RecordedBy: 1, CheckInTime: &now})
	s.Create(&models.ChildAttendance{ChildID: child2.ID, OrganizationID: org2.ID, Date: today, Status: models.ChildAttendanceStatusPresent, RecordedBy: 1, CheckInTime: &now})
	s.Create(&models.ChildAttendance{ChildID: child3.ID, OrganizationID: org2.ID, Date: today, Status: models.ChildAttendanceStatusSick, RecordedBy: 1})

	summary1, _ := s.GetDailySummary(org1.ID, today)
	if summary1.TotalChildren != 1 {
		t.Errorf("expected 1 total for org1, got %d", summary1.TotalChildren)
	}

	summary2, _ := s.GetDailySummary(org2.ID, today)
	if summary2.TotalChildren != 2 {
		t.Errorf("expected 2 total for org2, got %d", summary2.TotalChildren)
	}
}
