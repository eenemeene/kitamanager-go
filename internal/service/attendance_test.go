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

type attendanceTestCtx struct {
	svc             *ChildAttendanceService
	attendanceStore *store.ChildAttendanceStore
	childStore      *store.ChildStore
	org             *models.Organization
	child           *models.Child
	userID          uint
}

func setupChildAttendanceTest(t *testing.T) *attendanceTestCtx {
	t.Helper()
	db := setupTestDB(t)

	user := createTestUser(t, db, "Recorder", "recorder@test.com", "password")
	attendanceStore := store.NewChildAttendanceStore(db)
	childStore := store.NewChildStore(db)
	svc := NewChildAttendanceService(attendanceStore, childStore)

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "Emma", "Schmidt", org.ID)

	return &attendanceTestCtx{
		svc:             svc,
		attendanceStore: attendanceStore,
		childStore:      childStore,
		org:             org,
		child:           child,
		userID:          user.ID,
	}
}

// ============================================================
// Create tests (present status)
// ============================================================

func TestChildAttendanceService_Create_Present(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
		Note:   "Arrived with father",
	}

	resp, err := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.ChildID != tc.child.ID {
		t.Errorf("expected ChildID %d, got %d", tc.child.ID, resp.ChildID)
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
	if resp.OrganizationID != tc.org.ID {
		t.Errorf("expected OrganizationID %d, got %d", tc.org.ID, resp.OrganizationID)
	}
}

func TestChildAttendanceService_Create_Present_WithCustomTime(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	customTime := time.Date(2025, 6, 15, 7, 30, 0, 0, time.UTC)
	req := &models.ChildAttendanceCreateRequest{
		Status:      models.ChildAttendanceStatusPresent,
		CheckInTime: &customTime,
	}

	resp, err := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)
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
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
		Note:   "  spaces around  ",
	}

	resp, err := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Note != "spaces around" {
		t.Errorf("expected trimmed note 'spaces around', got '%s'", resp.Note)
	}
}

func TestChildAttendanceService_Create_ChildNotFound(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	_, err := tc.svc.Create(ctx, tc.org.ID, 999, req, tc.userID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestChildAttendanceService_Create_WrongOrg(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	_, err := tc.svc.Create(ctx, 999, tc.child.ID, req, tc.userID)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestChildAttendanceService_Create_DuplicateToday(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}

	// First create should succeed
	_, err := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	// Second create should fail (duplicate)
	_, err = tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)
	if err == nil {
		t.Fatal("expected error for duplicate, got nil")
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestChildAttendanceService_Create_Present_EmptyNote(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
		Note:   "",
	}
	resp, err := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Note != "" {
		t.Errorf("expected empty note, got '%s'", resp.Note)
	}
}

func TestChildAttendanceService_Create_Present_ReturnsChildName(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	resp, err := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)
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
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Date:   "2025-06-15",
		Status: models.ChildAttendanceStatusSick,
		Note:   "Has a cold",
	}

	resp, err := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)
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
			tc := setupChildAttendanceTest(t)
			ctx := context.Background()

			req := &models.ChildAttendanceCreateRequest{
				Date:   "2025-06-15",
				Status: status,
			}
			resp, err := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)
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
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Date:   "2025-06-15",
		Status: "invalid",
	}

	_, err := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildAttendanceService_Create_Absent_RequiresDate(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusSick,
	}

	_, err := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)
	if err == nil {
		t.Fatal("expected error when date missing for absent status, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildAttendanceService_Create_InvalidDate(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Date:   "invalid-date",
		Status: models.ChildAttendanceStatusSick,
	}

	_, err := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)
	if err == nil {
		t.Fatal("expected error for invalid date, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildAttendanceService_Create_DuplicateDate(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Date:   "2025-06-15",
		Status: models.ChildAttendanceStatusSick,
	}

	_, err := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err = tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)
	if err == nil {
		t.Fatal("expected conflict error for duplicate date, got nil")
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestChildAttendanceService_Create_Absent_ChildNotFound(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Date:   "2025-06-15",
		Status: models.ChildAttendanceStatusSick,
	}

	_, err := tc.svc.Create(ctx, tc.org.ID, 999, req, tc.userID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestChildAttendanceService_Create_Absent_WrongOrg(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Date:   "2025-06-15",
		Status: models.ChildAttendanceStatusSick,
	}

	_, err := tc.svc.Create(ctx, 999, tc.child.ID, req, tc.userID)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
}

func TestChildAttendanceService_Create_Absent_CheckInTimeIgnored(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	customTime := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	req := &models.ChildAttendanceCreateRequest{
		Date:        "2025-06-15",
		Status:      models.ChildAttendanceStatusSick,
		CheckInTime: &customTime,
	}

	resp, err := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)
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
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)

	resp, err := tc.svc.GetByID(ctx, createResp.ID, tc.org.ID, tc.child.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.ID != createResp.ID {
		t.Errorf("expected ID %d, got %d", createResp.ID, resp.ID)
	}
}

func TestChildAttendanceService_GetByID_WrongOrg(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)

	_, err := tc.svc.GetByID(ctx, createResp.ID, 999, tc.child.ID)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
}

func TestChildAttendanceService_GetByID_WrongChild(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)

	_, err := tc.svc.GetByID(ctx, createResp.ID, tc.org.ID, 999)
	if err == nil {
		t.Fatal("expected error for wrong child, got nil")
	}
}

func TestChildAttendanceService_GetByID_NotFound(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	_, err := tc.svc.GetByID(ctx, 9999, tc.org.ID, tc.child.ID)
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
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Verify the record initially has a check-in time (auto-set by Create for present)
	if createResp.CheckInTime == nil {
		t.Fatal("expected check-in time to be set after create")
	}

	newStatus := models.ChildAttendanceStatusSick
	updateReq := &models.ChildAttendanceUpdateRequest{Status: &newStatus}
	resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, updateReq)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Status != models.ChildAttendanceStatusSick {
		t.Errorf("expected status sick, got %s", resp.Status)
	}
	// Times must be cleared when status changes to non-present
	if resp.CheckInTime != nil {
		t.Error("expected CheckInTime to be nil after changing to sick")
	}
	if resp.CheckOutTime != nil {
		t.Error("expected CheckOutTime to be nil after changing to sick")
	}
}

func TestChildAttendanceService_Update_Note(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	newNote := "Updated note"
	updateReq := &models.ChildAttendanceUpdateRequest{Note: &newNote}
	resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, updateReq)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Note != "Updated note" {
		t.Errorf("expected note 'Updated note', got '%s'", resp.Note)
	}
}

func TestChildAttendanceService_Update_InvalidStatus(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	invalid := "invalid"
	updateReq := &models.ChildAttendanceUpdateRequest{Status: &invalid}
	_, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, updateReq)
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildAttendanceService_Update_WrongChild(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	newNote := "Updated"
	updateReq := &models.ChildAttendanceUpdateRequest{Note: &newNote}
	_, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, 999, updateReq)
	if err == nil {
		t.Fatal("expected error for wrong child, got nil")
	}
}

func TestChildAttendanceService_Update_CheckTimes(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	newCheckIn := time.Date(2025, 6, 15, 7, 0, 0, 0, time.UTC)
	newCheckOut := time.Date(2025, 6, 15, 15, 0, 0, 0, time.UTC)
	updateReq := &models.ChildAttendanceUpdateRequest{
		CheckInTime:  &newCheckIn,
		CheckOutTime: &newCheckOut,
	}
	resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, updateReq)
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
// Update: status change clears times (edge cases)
// ============================================================

func TestChildAttendanceService_Update_StatusToAbsentClearsTimes(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	checkIn := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	createReq := &models.ChildAttendanceCreateRequest{
		Status:      models.ChildAttendanceStatusPresent,
		CheckInTime: &checkIn,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Add a check-out time first
	checkOut := time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC)
	_, _ = tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckOutTime: &checkOut,
	})

	// Change to absent — both times should be cleared
	absent := models.ChildAttendanceStatusAbsent
	resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		Status: &absent,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.CheckInTime != nil {
		t.Error("expected CheckInTime to be nil after changing to absent")
	}
	if resp.CheckOutTime != nil {
		t.Error("expected CheckOutTime to be nil after changing to absent")
	}
}

func TestChildAttendanceService_Update_StatusToVacationClearsTimes(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	checkIn := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	createReq := &models.ChildAttendanceCreateRequest{
		Status:      models.ChildAttendanceStatusPresent,
		CheckInTime: &checkIn,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	vacation := models.ChildAttendanceStatusVacation
	resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		Status: &vacation,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.CheckInTime != nil {
		t.Error("expected CheckInTime to be nil after changing to vacation")
	}
}

func TestChildAttendanceService_Update_StatusToPresentKeepsTimes(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	checkIn := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	checkOut := time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC)
	createReq := &models.ChildAttendanceCreateRequest{
		Status:      models.ChildAttendanceStatusPresent,
		CheckInTime: &checkIn,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Set check-out and change to present (explicitly) — times should be preserved
	present := models.ChildAttendanceStatusPresent
	resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		Status:       &present,
		CheckOutTime: &checkOut,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.CheckInTime == nil || !resp.CheckInTime.Equal(checkIn) {
		t.Errorf("expected CheckInTime to be preserved, got %v", resp.CheckInTime)
	}
	if resp.CheckOutTime == nil || !resp.CheckOutTime.Equal(checkOut) {
		t.Errorf("expected CheckOutTime to be preserved, got %v", resp.CheckOutTime)
	}
}

func TestChildAttendanceService_Update_StatusFromSickToVacationTimesStayNil(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Date:   "2025-06-15",
		Status: models.ChildAttendanceStatusSick,
	}
	createResp, err := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if createResp.CheckInTime != nil {
		t.Fatal("expected no check-in time for sick status")
	}

	// Switch from sick to vacation — times should remain nil
	vacation := models.ChildAttendanceStatusVacation
	resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		Status: &vacation,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Status != models.ChildAttendanceStatusVacation {
		t.Errorf("expected status vacation, got %s", resp.Status)
	}
	if resp.CheckInTime != nil {
		t.Error("expected CheckInTime to remain nil")
	}
	if resp.CheckOutTime != nil {
		t.Error("expected CheckOutTime to remain nil")
	}
}

func TestChildAttendanceService_Update_StatusFromSickToPresent_AutoSetsCheckIn(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Date:   "2025-06-15",
		Status: models.ChildAttendanceStatusSick,
	}
	createResp, err := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if createResp.CheckInTime != nil {
		t.Fatal("expected no check-in time for sick status")
	}

	// Switch from sick to present — check-in time should be auto-set to now
	before := time.Now()
	present := models.ChildAttendanceStatusPresent
	resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		Status: &present,
	})
	after := time.Now()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Status != models.ChildAttendanceStatusPresent {
		t.Errorf("expected status present, got %s", resp.Status)
	}
	if resp.CheckInTime == nil {
		t.Fatal("expected CheckInTime to be auto-set when changing to present")
	}
	if resp.CheckInTime.Before(before) || resp.CheckInTime.After(after) {
		t.Errorf("expected CheckInTime to be around now, got %v", resp.CheckInTime)
	}
	if resp.CheckOutTime != nil {
		t.Error("expected CheckOutTime to remain nil")
	}
}

func TestChildAttendanceService_Update_StatusToPresentWithExplicitTime(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Date:   "2025-06-15",
		Status: models.ChildAttendanceStatusAbsent,
	}
	createResp, err := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Switch to present with an explicit check-in time — should use the provided time, not now
	explicitTime := time.Date(2025, 6, 15, 9, 30, 0, 0, time.UTC)
	present := models.ChildAttendanceStatusPresent
	resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		Status:      &present,
		CheckInTime: &explicitTime,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.CheckInTime == nil {
		t.Fatal("expected CheckInTime to be set")
	}
	if !resp.CheckInTime.Equal(explicitTime) {
		t.Errorf("expected explicit time %v, got %v", explicitTime, resp.CheckInTime)
	}
}

func TestChildAttendanceService_Update_StatusToPresentAlreadyHasTimes(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	checkIn := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	createReq := &models.ChildAttendanceCreateRequest{
		Status:      models.ChildAttendanceStatusPresent,
		CheckInTime: &checkIn,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Explicitly set status to present when already present with times — should preserve existing time
	present := models.ChildAttendanceStatusPresent
	resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		Status: &present,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.CheckInTime == nil || !resp.CheckInTime.Equal(checkIn) {
		t.Errorf("expected existing check-in time %v to be preserved, got %v", checkIn, resp.CheckInTime)
	}
}

func TestChildAttendanceService_Update_TimesAndNonPresentStatusClearsTimesAfterSetting(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Send both times AND a non-present status in the same request.
	// The status change should take precedence and clear the times.
	newCheckIn := time.Date(2025, 6, 15, 7, 0, 0, 0, time.UTC)
	newCheckOut := time.Date(2025, 6, 15, 15, 0, 0, 0, time.UTC)
	sick := models.ChildAttendanceStatusSick
	resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckInTime:  &newCheckIn,
		CheckOutTime: &newCheckOut,
		Status:       &sick,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Status != models.ChildAttendanceStatusSick {
		t.Errorf("expected status sick, got %s", resp.Status)
	}
	// Status change to non-present should clear times even though they were sent in the request
	if resp.CheckInTime != nil {
		t.Error("expected CheckInTime to be nil (non-present status overrides sent times)")
	}
	if resp.CheckOutTime != nil {
		t.Error("expected CheckOutTime to be nil (non-present status overrides sent times)")
	}
}

func TestChildAttendanceService_Update_AllNonPresentStatusesClearTimes(t *testing.T) {
	statuses := []string{
		models.ChildAttendanceStatusAbsent,
		models.ChildAttendanceStatusSick,
		models.ChildAttendanceStatusVacation,
	}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			tc := setupChildAttendanceTest(t)
			ctx := context.Background()

			checkIn := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
			checkOut := time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC)
			createReq := &models.ChildAttendanceCreateRequest{
				Status:      models.ChildAttendanceStatusPresent,
				CheckInTime: &checkIn,
			}
			createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

			// Set check-out time first
			_, _ = tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
				CheckOutTime: &checkOut,
			})

			// Change to non-present status
			s := status
			resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
				Status: &s,
			})
			if err != nil {
				t.Fatalf("expected no error for status %s, got %v", status, err)
			}
			if resp.CheckInTime != nil {
				t.Errorf("expected CheckInTime nil for status %s", status)
			}
			if resp.CheckOutTime != nil {
				t.Errorf("expected CheckOutTime nil for status %s", status)
			}
		})
	}
}

func TestChildAttendanceService_Update_NotePreservedAfterStatusChange(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
		Note:   "Important note",
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Change status — note should be preserved
	sick := models.ChildAttendanceStatusSick
	resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		Status: &sick,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Note != "Important note" {
		t.Errorf("expected note to be preserved, got '%s'", resp.Note)
	}
}

func TestChildAttendanceService_Update_TimesClearedPersistsToDatabase(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	checkIn := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	createReq := &models.ChildAttendanceCreateRequest{
		Status:      models.ChildAttendanceStatusPresent,
		CheckInTime: &checkIn,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Change to sick — clear times
	sick := models.ChildAttendanceStatusSick
	_, _ = tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		Status: &sick,
	})

	// Re-fetch from database to verify times are actually persisted as NULL
	reloaded, err := tc.svc.GetByID(ctx, createResp.ID, tc.org.ID, tc.child.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if reloaded.CheckInTime != nil {
		t.Error("expected CheckInTime to be nil after reload from database")
	}
	if reloaded.CheckOutTime != nil {
		t.Error("expected CheckOutTime to be nil after reload from database")
	}
}

// ============================================================
// Update: check-in before check-out validation
// ============================================================

func TestChildAttendanceService_Update_CheckOutBeforeCheckIn_Rejected(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Set check-in at 15:00 and check-out at 07:00 — invalid
	checkIn := time.Date(2025, 6, 15, 15, 0, 0, 0, time.UTC)
	checkOut := time.Date(2025, 6, 15, 7, 0, 0, 0, time.UTC)
	_, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckInTime:  &checkIn,
		CheckOutTime: &checkOut,
	})
	if err == nil {
		t.Fatal("expected error when check-out is before check-in, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildAttendanceService_Update_EqualCheckInAndCheckOut_Rejected(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Same time for both — invalid (not strictly before)
	sameTime := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	_, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckInTime:  &sameTime,
		CheckOutTime: &sameTime,
	})
	if err == nil {
		t.Fatal("expected error when check-in equals check-out, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildAttendanceService_Update_CheckOutBeforeExistingCheckIn_Rejected(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	checkIn := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	createReq := &models.ChildAttendanceCreateRequest{
		Status:      models.ChildAttendanceStatusPresent,
		CheckInTime: &checkIn,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Set check-out to 09:00, which is before the existing check-in at 10:00
	earlyCheckOut := time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC)
	_, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckOutTime: &earlyCheckOut,
	})
	if err == nil {
		t.Fatal("expected error when check-out is before existing check-in, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildAttendanceService_Update_CheckInAfterExistingCheckOut_Rejected(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	checkIn := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	createReq := &models.ChildAttendanceCreateRequest{
		Status:      models.ChildAttendanceStatusPresent,
		CheckInTime: &checkIn,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// First set a valid check-out
	checkOut := time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC)
	_, _ = tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckOutTime: &checkOut,
	})

	// Now update check-in to 17:00, which is after the existing check-out at 16:00
	lateCheckIn := time.Date(2025, 6, 15, 17, 0, 0, 0, time.UTC)
	_, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckInTime: &lateCheckIn,
	})
	if err == nil {
		t.Fatal("expected error when check-in is after existing check-out, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildAttendanceService_Update_ValidCheckInBeforeCheckOut_Accepted(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Set valid times: check-in at 07:30, check-out at 15:30
	checkIn := time.Date(2025, 6, 15, 7, 30, 0, 0, time.UTC)
	checkOut := time.Date(2025, 6, 15, 15, 30, 0, 0, time.UTC)
	resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckInTime:  &checkIn,
		CheckOutTime: &checkOut,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !resp.CheckInTime.Equal(checkIn) {
		t.Errorf("expected CheckInTime %v, got %v", checkIn, resp.CheckInTime)
	}
	if !resp.CheckOutTime.Equal(checkOut) {
		t.Errorf("expected CheckOutTime %v, got %v", checkOut, resp.CheckOutTime)
	}
}

func TestChildAttendanceService_Update_CheckOutOneMinuteAfterCheckIn_Accepted(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Minimal valid gap: 1 minute apart
	checkIn := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	checkOut := time.Date(2025, 6, 15, 8, 1, 0, 0, time.UTC)
	resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckInTime:  &checkIn,
		CheckOutTime: &checkOut,
	})
	if err != nil {
		t.Fatalf("expected no error for 1 minute gap, got %v", err)
	}
	if !resp.CheckInTime.Equal(checkIn) {
		t.Errorf("expected CheckInTime %v, got %v", checkIn, resp.CheckInTime)
	}
	if !resp.CheckOutTime.Equal(checkOut) {
		t.Errorf("expected CheckOutTime %v, got %v", checkOut, resp.CheckOutTime)
	}
}

func TestChildAttendanceService_Update_CheckOutOneSecondAfterCheckIn_Accepted(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Smallest valid gap: 1 second apart
	checkIn := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	checkOut := time.Date(2025, 6, 15, 8, 0, 1, 0, time.UTC)
	_, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckInTime:  &checkIn,
		CheckOutTime: &checkOut,
	})
	if err != nil {
		t.Fatalf("expected no error for 1 second gap, got %v", err)
	}
}

func TestChildAttendanceService_Update_CheckOutOneNanosecondBeforeCheckIn_Rejected(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Check-out is 1 nanosecond before check-in
	checkIn := time.Date(2025, 6, 15, 8, 0, 0, 1, time.UTC)
	checkOut := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	_, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckInTime:  &checkIn,
		CheckOutTime: &checkOut,
	})
	if err == nil {
		t.Fatal("expected error for nanosecond-before check-out, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildAttendanceService_Update_OnlyCheckInNoCheckOut_NoValidation(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Update only check-in time, no check-out — should not trigger validation
	newCheckIn := time.Date(2025, 6, 15, 23, 59, 0, 0, time.UTC)
	resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckInTime: &newCheckIn,
	})
	if err != nil {
		t.Fatalf("expected no error when only setting check-in, got %v", err)
	}
	if resp.CheckOutTime != nil {
		t.Error("expected CheckOutTime to remain nil")
	}
}

func TestChildAttendanceService_Update_OnlyCheckOutNoExistingCheckIn_NoValidation(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	// Create a record directly with no check-in time (non-present status)
	date := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	attendance := &models.ChildAttendance{
		ChildID:        tc.child.ID,
		OrganizationID: tc.org.ID,
		Date:           date,
		Status:         models.ChildAttendanceStatusPresent,
		RecordedBy:     tc.userID,
	}
	if err := tc.attendanceStore.Create(ctx, attendance); err != nil {
		t.Fatalf("failed to create attendance: %v", err)
	}

	// Update only check-out, no existing check-in — validation not triggered
	checkOut := time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC)
	resp, err := tc.svc.Update(ctx, attendance.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckOutTime: &checkOut,
	})
	if err != nil {
		t.Fatalf("expected no error when setting check-out without check-in, got %v", err)
	}
	if resp.CheckOutTime == nil {
		t.Error("expected CheckOutTime to be set")
	}
}

func TestChildAttendanceService_Update_NonPresentStatusSkipsTimeValidation(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Send invalid times (check-out before check-in) PLUS a non-present status.
	// The status change clears times, so validation should not trigger.
	badCheckIn := time.Date(2025, 6, 15, 15, 0, 0, 0, time.UTC)
	badCheckOut := time.Date(2025, 6, 15, 7, 0, 0, 0, time.UTC)
	absent := models.ChildAttendanceStatusAbsent
	resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckInTime:  &badCheckIn,
		CheckOutTime: &badCheckOut,
		Status:       &absent,
	})
	if err != nil {
		t.Fatalf("expected no error (non-present status clears times), got %v", err)
	}
	if resp.CheckInTime != nil {
		t.Error("expected CheckInTime to be nil (non-present status)")
	}
	if resp.CheckOutTime != nil {
		t.Error("expected CheckOutTime to be nil (non-present status)")
	}
}

func TestChildAttendanceService_Update_ValidTimesNotCorruptedByValidation(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	checkIn := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	createReq := &models.ChildAttendanceCreateRequest{
		Status:      models.ChildAttendanceStatusPresent,
		CheckInTime: &checkIn,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Set valid check-out — should pass and persist correctly
	checkOut := time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC)
	_, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckOutTime: &checkOut,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify by re-fetching
	reloaded, err := tc.svc.GetByID(ctx, createResp.ID, tc.org.ID, tc.child.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !reloaded.CheckInTime.Equal(checkIn) {
		t.Errorf("check-in was corrupted: expected %v, got %v", checkIn, reloaded.CheckInTime)
	}
	if !reloaded.CheckOutTime.Equal(checkOut) {
		t.Errorf("check-out was corrupted: expected %v, got %v", checkOut, reloaded.CheckOutTime)
	}
}

func TestChildAttendanceService_Update_RejectedUpdateDoesNotPersist(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	checkIn := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	createReq := &models.ChildAttendanceCreateRequest{
		Status:      models.ChildAttendanceStatusPresent,
		CheckInTime: &checkIn,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Try to set an invalid check-out (before check-in)
	badCheckOut := time.Date(2025, 6, 15, 7, 0, 0, 0, time.UTC)
	_, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckOutTime: &badCheckOut,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify the original record is unchanged
	reloaded, err := tc.svc.GetByID(ctx, createResp.ID, tc.org.ID, tc.child.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !reloaded.CheckInTime.Equal(checkIn) {
		t.Errorf("check-in should be unchanged, expected %v, got %v", checkIn, reloaded.CheckInTime)
	}
	if reloaded.CheckOutTime != nil {
		t.Error("check-out should still be nil after rejected update")
	}
}

func TestChildAttendanceService_Update_SwapTimesInSingleRequest_Rejected(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	checkIn := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	checkOut := time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC)
	createReq := &models.ChildAttendanceCreateRequest{
		Status:      models.ChildAttendanceStatusPresent,
		CheckInTime: &checkIn,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)
	_, _ = tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckOutTime: &checkOut,
	})

	// Try to swap: check-in = old check-out, check-out = old check-in
	_, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckInTime:  &checkOut, // 16:00
		CheckOutTime: &checkIn,  // 08:00
	})
	if err == nil {
		t.Fatal("expected error when swapping check-in/check-out times, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildAttendanceService_Update_MidnightBoundary_Accepted(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	createReq := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	createResp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, createReq, tc.userID)

	// Check-in at midnight, check-out at 23:59 — valid full-day
	checkIn := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	checkOut := time.Date(2025, 6, 15, 23, 59, 0, 0, time.UTC)
	resp, err := tc.svc.Update(ctx, createResp.ID, tc.org.ID, tc.child.ID, &models.ChildAttendanceUpdateRequest{
		CheckInTime:  &checkIn,
		CheckOutTime: &checkOut,
	})
	if err != nil {
		t.Fatalf("expected no error for midnight boundary, got %v", err)
	}
	if !resp.CheckInTime.Equal(checkIn) {
		t.Errorf("expected CheckInTime %v, got %v", checkIn, resp.CheckInTime)
	}
	if !resp.CheckOutTime.Equal(checkOut) {
		t.Errorf("expected CheckOutTime %v, got %v", checkOut, resp.CheckOutTime)
	}
}

// ============================================================
// Delete tests
// ============================================================

func TestChildAttendanceService_Delete(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	resp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)

	err := tc.svc.Delete(ctx, resp.ID, tc.org.ID, tc.child.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's gone
	_, err = tc.svc.GetByID(ctx, resp.ID, tc.org.ID, tc.child.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestChildAttendanceService_Delete_WrongOrg(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	resp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)

	err := tc.svc.Delete(ctx, resp.ID, 999, tc.child.ID)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
}

func TestChildAttendanceService_Delete_WrongChild(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	resp, _ := tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)

	err := tc.svc.Delete(ctx, resp.ID, tc.org.ID, 999)
	if err == nil {
		t.Fatal("expected error for wrong child, got nil")
	}
}

func TestChildAttendanceService_Delete_NotFound(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	err := tc.svc.Delete(ctx, 9999, tc.org.ID, tc.child.ID)
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
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	req := &models.ChildAttendanceCreateRequest{
		Status: models.ChildAttendanceStatusPresent,
	}
	_, _ = tc.svc.Create(ctx, tc.org.ID, tc.child.ID, req, tc.userID)

	records, total, err := tc.svc.ListByDate(ctx, tc.org.ID, today, 10, 0)
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
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	farFuture := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	records, total, err := tc.svc.ListByDate(ctx, tc.org.ID, farFuture, 10, 0)
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
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	day1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	if err := tc.attendanceStore.Create(ctx, &models.ChildAttendance{ChildID: tc.child.ID, OrganizationID: tc.org.ID, Date: day1, Status: models.ChildAttendanceStatusPresent, RecordedBy: tc.userID}); err != nil {
		t.Fatalf("failed to create attendance: %v", err)
	}
	if err := tc.attendanceStore.Create(ctx, &models.ChildAttendance{ChildID: tc.child.ID, OrganizationID: tc.org.ID, Date: day2, Status: models.ChildAttendanceStatusSick, RecordedBy: tc.userID}); err != nil {
		t.Fatalf("failed to create attendance: %v", err)
	}

	records, total, err := tc.svc.ListByChild(ctx, tc.child.ID, tc.org.ID, day1, day2, 10, 0)
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
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	_, _, err := tc.svc.ListByChild(ctx, tc.child.ID, 999, from, to, 10, 0)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
}

func TestChildAttendanceService_ListByChild_ChildNotFound(t *testing.T) {
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	_, _, err := tc.svc.ListByChild(ctx, 999, tc.org.ID, from, to, 10, 0)
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
	user := createTestUser(t, db, "Recorder", "recorder@test.com", "password")
	attendanceStore := store.NewChildAttendanceStore(db)
	childStore := store.NewChildStore(db)
	svc := NewChildAttendanceService(attendanceStore, childStore)

	org := createTestOrganization(t, db, "Test Org")
	child1 := createTestChild(t, db, "C1", "L", org.ID)
	child2 := createTestChild(t, db, "C2", "L", org.ID)
	child3 := createTestChild(t, db, "C3", "L", org.ID)

	ctx := context.Background()
	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	now := time.Now()

	if err := attendanceStore.Create(ctx, &models.ChildAttendance{ChildID: child1.ID, OrganizationID: org.ID, Date: today, Status: models.ChildAttendanceStatusPresent, RecordedBy: user.ID, CheckInTime: &now}); err != nil {
		t.Fatalf("failed to create attendance: %v", err)
	}
	if err := attendanceStore.Create(ctx, &models.ChildAttendance{ChildID: child2.ID, OrganizationID: org.ID, Date: today, Status: models.ChildAttendanceStatusPresent, RecordedBy: user.ID, CheckInTime: &now}); err != nil {
		t.Fatalf("failed to create attendance: %v", err)
	}
	if err := attendanceStore.Create(ctx, &models.ChildAttendance{ChildID: child3.ID, OrganizationID: org.ID, Date: today, Status: models.ChildAttendanceStatusSick, RecordedBy: user.ID}); err != nil {
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
	tc := setupChildAttendanceTest(t)
	ctx := context.Background()

	emptyDay := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	summary, err := tc.svc.GetDailySummary(ctx, tc.org.ID, emptyDay)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if summary.TotalChildren != 0 {
		t.Errorf("expected 0 total, got %d", summary.TotalChildren)
	}
}
