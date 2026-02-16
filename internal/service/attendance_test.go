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

func setupChildAttendanceTest(t *testing.T) (*ChildAttendanceService, *store.ChildAttendanceStore, *store.ChildStore, *models.Organization, *models.Child) {
	t.Helper()
	db := setupTestDB(t)

	attendanceStore := store.NewChildAttendanceStore(db)
	childStore := store.NewChildStore(db)
	svc := NewChildAttendanceService(attendanceStore, childStore)

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "Emma", "Schmidt", org.ID)

	return svc, attendanceStore, childStore, org, child
}

// ============================================================
// Create tests (present status)
// ============================================================

func TestChildAttendanceService_Create_Present(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
		Note:   "Arrived with father",
	}

	resp, err := svc.Create(ctx, org.ID, child.ID, req, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.ChildID != child.ID {
		t.Errorf("expected ChildID %d, got %d", child.ID, resp.ChildID)
	}
	if resp.Status != models.ChildAttendanceStatusPresent {
		t.Errorf("expected status 'present', got '%s'", resp.Status)
	}
	if resp.CheckInTime == nil {
		t.Error("expected CheckInTime to be set")
	}
	if resp.Note != "Arrived with father" {
		t.Errorf("expected note 'Arrived with father', got '%s'", resp.Note)
	}
	if resp.OrganizationID != org.ID {
		t.Errorf("expected OrganizationID %d, got %d", org.ID, resp.OrganizationID)
	}
}

func TestChildAttendanceService_Create_Present_WithCustomTime(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	customTime := time.Date(2025, 6, 15, 7, 30, 0, 0, time.UTC)
	req := &models.ChildAttendanceCreateRequest{
		Status:      models.ChildAttendanceStatusPresent,
		CheckInTime: &customTime,
	}

	resp, err := svc.Create(ctx, org.ID, child.ID, req, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.CheckInTime == nil {
		t.Fatal("expected CheckInTime to be set")
	}
	if !resp.CheckInTime.Equal(customTime) {
		t.Errorf("expected custom time %v, got %v", customTime, resp.CheckInTime)
	}
}

func TestChildAttendanceService_Create_Present_TrimsNote(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
		Note:   "  spaces around  ",
	}

	resp, err := svc.Create(ctx, org.ID, child.ID, req, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Note != "spaces around" {
		t.Errorf("expected trimmed note 'spaces around', got '%s'", resp.Note)
	}
}

func TestChildAttendanceService_Create_ChildNotFound(t *testing.T) {
	svc, _, _, org, _ := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	_, err := svc.Create(ctx, org.ID, 999, req, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestChildAttendanceService_Create_WrongOrg(t *testing.T) {
	svc, _, _, _, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	_, err := svc.Create(ctx, 999, child.ID, req, 1)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestChildAttendanceService_Create_DuplicateToday(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}

	// First create should succeed
	_, err := svc.Create(ctx, org.ID, child.ID, req, 1)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	// Second create should fail (duplicate)
	_, err = svc.Create(ctx, org.ID, child.ID, req, 1)
	if err == nil {
		t.Fatal("expected error for duplicate, got nil")
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestChildAttendanceService_Create_Present_EmptyNote(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
		Note:   "",
	}
	resp, err := svc.Create(ctx, org.ID, child.ID, req, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Note != "" {
		t.Errorf("expected empty note, got '%s'", resp.Note)
	}
}

func TestChildAttendanceService_Create_Present_ReturnsChildName(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	resp, err := svc.Create(ctx, org.ID, child.ID, req, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.ChildName != "Emma Schmidt" {
		t.Errorf("expected child name 'Emma Schmidt', got '%s'", resp.ChildName)
	}
}

// ============================================================
// Create tests (absent/sick/vacation status)
// ============================================================

func TestChildAttendanceService_Create_Absent(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Date:   "2025-06-15",
		Status: models.ChildAttendanceStatusSick,
		Note:   "Has a cold",
	}

	resp, err := svc.Create(ctx, org.ID, child.ID, req, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Status != models.ChildAttendanceStatusSick {
		t.Errorf("expected status 'sick', got '%s'", resp.Status)
	}
	if resp.Note != "Has a cold" {
		t.Errorf("expected note 'Has a cold', got '%s'", resp.Note)
	}
	if resp.Date != "2025-06-15" {
		t.Errorf("expected date '2025-06-15', got '%s'", resp.Date)
	}
}

func TestChildAttendanceService_Create_AllAbsentStatuses(t *testing.T) {
	statuses := []string{
		models.ChildAttendanceStatusAbsent,
		models.ChildAttendanceStatusSick,
		models.ChildAttendanceStatusVacation,
	}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			svc, _, _, org, child := setupChildAttendanceTest(t)
			ctx := context.Background()

			req := &models.ChildAttendanceCreateRequest{
				Date:   "2025-06-15",
				Status: status,
			}
			resp, err := svc.Create(ctx, org.ID, child.ID, req, 1)
			if err != nil {
				t.Fatalf("expected no error for status %s, got %v", status, err)
			}
			if resp.Status != status {
				t.Errorf("expected status '%s', got '%s'", status, resp.Status)
			}
		})
	}
}

func TestChildAttendanceService_Create_InvalidStatus(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Date:   "2025-06-15",
		Status: "invalid",
	}

	_, err := svc.Create(ctx, org.ID, child.ID, req, 1)
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildAttendanceService_Create_Absent_RequiresDate(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusSick,
	}

	_, err := svc.Create(ctx, org.ID, child.ID, req, 1)
	if err == nil {
		t.Fatal("expected error when date missing for absent status, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildAttendanceService_Create_InvalidDate(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Date:   "invalid-date",
		Status: models.ChildAttendanceStatusSick,
	}

	_, err := svc.Create(ctx, org.ID, child.ID, req, 1)
	if err == nil {
		t.Fatal("expected error for invalid date, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildAttendanceService_Create_DuplicateDate(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Date:   "2025-06-15",
		Status: models.ChildAttendanceStatusSick,
	}

	_, err := svc.Create(ctx, org.ID, child.ID, req, 1)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err = svc.Create(ctx, org.ID, child.ID, req, 1)
	if err == nil {
		t.Fatal("expected conflict error for duplicate date, got nil")
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestChildAttendanceService_Create_Absent_ChildNotFound(t *testing.T) {
	svc, _, _, org, _ := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Date:   "2025-06-15",
		Status: models.ChildAttendanceStatusSick,
	}

	_, err := svc.Create(ctx, org.ID, 999, req, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestChildAttendanceService_Create_Absent_WrongOrg(t *testing.T) {
	svc, _, _, _, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Date:   "2025-06-15",
		Status: models.ChildAttendanceStatusSick,
	}

	_, err := svc.Create(ctx, 999, child.ID, req, 1)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
}

func TestChildAttendanceService_Create_Absent_CheckInTimeIgnored(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	customTime := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	req := &models.ChildAttendanceCreateRequest{
		Date:        "2025-06-15",
		Status:      models.ChildAttendanceStatusSick,
		CheckInTime: &customTime,
	}

	resp, err := svc.Create(ctx, org.ID, child.ID, req, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.CheckInTime != nil {
		t.Error("expected CheckInTime to be nil for absent status")
	}
}

// ============================================================
// GetByID tests
// ============================================================

func TestChildAttendanceService_GetByID(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := svc.Create(ctx, org.ID, child.ID, req, 1)

	resp, err := svc.GetByID(ctx, createResp.ID, org.ID, child.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.ID != createResp.ID {
		t.Errorf("expected ID %d, got %d", createResp.ID, resp.ID)
	}
}

func TestChildAttendanceService_GetByID_WrongOrg(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := svc.Create(ctx, org.ID, child.ID, req, 1)

	_, err := svc.GetByID(ctx, createResp.ID, 999, child.ID)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
}

func TestChildAttendanceService_GetByID_WrongChild(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := svc.Create(ctx, org.ID, child.ID, req, 1)

	_, err := svc.GetByID(ctx, createResp.ID, org.ID, 999)
	if err == nil {
		t.Fatal("expected error for wrong child, got nil")
	}
}

func TestChildAttendanceService_GetByID_NotFound(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 9999, org.ID, child.ID)
	if err == nil {
		t.Fatal("expected error for non-existent record, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================
// Update tests
// ============================================================

func TestChildAttendanceService_Update_Status(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := svc.Create(ctx, org.ID, child.ID, createReq, 1)

	newStatus := models.ChildAttendanceStatusSick
	updateReq := &models.ChildAttendanceUpdateRequest{Status: &newStatus}
	resp, err := svc.Update(ctx, createResp.ID, org.ID, child.ID, updateReq)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Status != models.ChildAttendanceStatusSick {
		t.Errorf("expected status sick, got %s", resp.Status)
	}
}

func TestChildAttendanceService_Update_Note(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := svc.Create(ctx, org.ID, child.ID, createReq, 1)

	newNote := "Updated note"
	updateReq := &models.ChildAttendanceUpdateRequest{Note: &newNote}
	resp, err := svc.Update(ctx, createResp.ID, org.ID, child.ID, updateReq)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Note != "Updated note" {
		t.Errorf("expected note 'Updated note', got '%s'", resp.Note)
	}
}

func TestChildAttendanceService_Update_InvalidStatus(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := svc.Create(ctx, org.ID, child.ID, createReq, 1)

	invalid := "invalid"
	updateReq := &models.ChildAttendanceUpdateRequest{Status: &invalid}
	_, err := svc.Update(ctx, createResp.ID, org.ID, child.ID, updateReq)
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildAttendanceService_Update_WrongChild(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := svc.Create(ctx, org.ID, child.ID, createReq, 1)

	newNote := "Updated"
	updateReq := &models.ChildAttendanceUpdateRequest{Note: &newNote}
	_, err := svc.Update(ctx, createResp.ID, org.ID, 999, updateReq)
	if err == nil {
		t.Fatal("expected error for wrong child, got nil")
	}
}

func TestChildAttendanceService_Update_CheckTimes(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := svc.Create(ctx, org.ID, child.ID, createReq, 1)

	newCheckIn := time.Date(2025, 6, 15, 7, 0, 0, 0, time.UTC)
	newCheckOut := time.Date(2025, 6, 15, 15, 0, 0, 0, time.UTC)
	updateReq := &models.ChildAttendanceUpdateRequest{
		CheckInTime:  &newCheckIn,
		CheckOutTime: &newCheckOut,
	}
	resp, err := svc.Update(ctx, createResp.ID, org.ID, child.ID, updateReq)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !resp.CheckInTime.Equal(newCheckIn) {
		t.Errorf("expected CheckInTime %v, got %v", newCheckIn, resp.CheckInTime)
	}
	if !resp.CheckOutTime.Equal(newCheckOut) {
		t.Errorf("expected CheckOutTime %v, got %v", newCheckOut, resp.CheckOutTime)
	}
}

// ============================================================
// Delete tests
// ============================================================

func TestChildAttendanceService_Delete(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	resp, _ := svc.Create(ctx, org.ID, child.ID, req, 1)

	err := svc.Delete(ctx, resp.ID, org.ID, child.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's gone
	_, err = svc.GetByID(ctx, resp.ID, org.ID, child.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestChildAttendanceService_Delete_WrongOrg(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	resp, _ := svc.Create(ctx, org.ID, child.ID, req, 1)

	err := svc.Delete(ctx, resp.ID, 999, child.ID)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
}

func TestChildAttendanceService_Delete_WrongChild(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	resp, _ := svc.Create(ctx, org.ID, child.ID, req, 1)

	err := svc.Delete(ctx, resp.ID, org.ID, 999)
	if err == nil {
		t.Fatal("expected error for wrong child, got nil")
	}
}

func TestChildAttendanceService_Delete_NotFound(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 9999, org.ID, child.ID)
	if err == nil {
		t.Fatal("expected error for non-existent record, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================
// ListByDate tests
// ============================================================

func TestChildAttendanceService_ListByDate(t *testing.T) {
	svc, _, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	_, _ = svc.Create(ctx, org.ID, child.ID, req, 1)

	records, total, err := svc.ListByDate(ctx, org.ID, today, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
}

func TestChildAttendanceService_ListByDate_EmptyResult(t *testing.T) {
	svc, _, _, org, _ := setupChildAttendanceTest(t)
	ctx := context.Background()

	farFuture := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	records, total, err := svc.ListByDate(ctx, org.ID, farFuture, 10, 0)
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

// ============================================================
// ListByChild tests
// ============================================================

func TestChildAttendanceService_ListByChild(t *testing.T) {
	svc, attendanceStore, _, org, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	day1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	if err := attendanceStore.Create(ctx, &models.ChildAttendance{ChildID: child.ID, OrganizationID: org.ID, Date: day1, Status: models.ChildAttendanceStatusPresent, RecordedBy: 1}); err != nil {
		t.Fatalf("failed to create attendance: %v", err)
	}
	if err := attendanceStore.Create(ctx, &models.ChildAttendance{ChildID: child.ID, OrganizationID: org.ID, Date: day2, Status: models.ChildAttendanceStatusSick, RecordedBy: 1}); err != nil {
		t.Fatalf("failed to create attendance: %v", err)
	}

	records, total, err := svc.ListByChild(ctx, child.ID, org.ID, day1, day2, 10, 0)
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

func TestChildAttendanceService_ListByChild_WrongOrg(t *testing.T) {
	svc, _, _, _, child := setupChildAttendanceTest(t)
	ctx := context.Background()

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	_, _, err := svc.ListByChild(ctx, child.ID, 999, from, to, 10, 0)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
}

func TestChildAttendanceService_ListByChild_ChildNotFound(t *testing.T) {
	svc, _, _, org, _ := setupChildAttendanceTest(t)
	ctx := context.Background()

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	_, _, err := svc.ListByChild(ctx, 999, org.ID, from, to, 10, 0)
	if err == nil {
		t.Fatal("expected error for non-existent child, got nil")
	}
}

// ============================================================
// GetDailySummary tests
// ============================================================

func TestChildAttendanceService_GetDailySummary(t *testing.T) {
	// Set up from scratch with a single DB reference so we don't re-truncate
	// between child creations.
	db := setupTestDB(t)
	attendanceStore := store.NewChildAttendanceStore(db)
	childStore := store.NewChildStore(db)
	svc := NewChildAttendanceService(attendanceStore, childStore)
	_ = childStore

	org := createTestOrganization(t, db, "Test Org")
	child1 := createTestChild(t, db, "C1", "L", org.ID)
	child2 := createTestChild(t, db, "C2", "L", org.ID)
	child3 := createTestChild(t, db, "C3", "L", org.ID)

	ctx := context.Background()
	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	now := time.Now()

	if err := attendanceStore.Create(ctx, &models.ChildAttendance{ChildID: child1.ID, OrganizationID: org.ID, Date: today, Status: models.ChildAttendanceStatusPresent, RecordedBy: 1, CheckInTime: &now}); err != nil {
		t.Fatalf("failed to create attendance: %v", err)
	}
	if err := attendanceStore.Create(ctx, &models.ChildAttendance{ChildID: child2.ID, OrganizationID: org.ID, Date: today, Status: models.ChildAttendanceStatusPresent, RecordedBy: 1, CheckInTime: &now}); err != nil {
		t.Fatalf("failed to create attendance: %v", err)
	}
	if err := attendanceStore.Create(ctx, &models.ChildAttendance{ChildID: child3.ID, OrganizationID: org.ID, Date: today, Status: models.ChildAttendanceStatusSick, RecordedBy: 1}); err != nil {
		t.Fatalf("failed to create attendance: %v", err)
	}

	summary, err := svc.GetDailySummary(ctx, org.ID, today)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if summary.TotalChildren != 3 {
		t.Errorf("expected 3 total, got %d", summary.TotalChildren)
	}
	if summary.Present != 2 {
		t.Errorf("expected 2 present, got %d", summary.Present)
	}
	if summary.Sick != 1 {
		t.Errorf("expected 1 sick, got %d", summary.Sick)
	}
}

func TestChildAttendanceService_GetDailySummary_EmptyDay(t *testing.T) {
	svc, _, _, org, _ := setupChildAttendanceTest(t)
	ctx := context.Background()

	emptyDay := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	summary, err := svc.GetDailySummary(ctx, org.ID, emptyDay)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if summary.TotalChildren != 0 {
		t.Errorf("expected 0 total, got %d", summary.TotalChildren)
	}
}
