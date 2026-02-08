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

func setupAttendanceTest(t *testing.T) (*AttendanceService, *store.AttendanceStore, *store.ChildStore, *models.Organization, *models.Child) {
	t.Helper()
	db := setupTestDB(t)
	db.AutoMigrate(&models.Attendance{})

	attendanceStore := store.NewAttendanceStore(db)
	childStore := store.NewChildStore(db)
	svc := NewAttendanceService(attendanceStore, childStore)

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "Emma", "Schmidt", org.ID)

	return svc, attendanceStore, childStore, org, child
}

func TestAttendanceService_CheckIn(t *testing.T) {
	svc, _, _, org, child := setupAttendanceTest(t)
	ctx := context.Background()

	req := &models.AttendanceCheckInRequest{
		ChildID: child.ID,
		Note:    "Arrived with father",
	}

	resp, err := svc.CheckIn(ctx, org.ID, req, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.ChildID != child.ID {
		t.Errorf("expected ChildID %d, got %d", child.ID, resp.ChildID)
	}
	if resp.Status != models.AttendanceStatusPresent {
		t.Errorf("expected status 'present', got '%s'", resp.Status)
	}
	if resp.CheckInTime == nil {
		t.Error("expected CheckInTime to be set")
	}
	if resp.Note != "Arrived with father" {
		t.Errorf("expected note 'Arrived with father', got '%s'", resp.Note)
	}
}

func TestAttendanceService_CheckIn_ChildNotFound(t *testing.T) {
	svc, _, _, org, _ := setupAttendanceTest(t)
	ctx := context.Background()

	req := &models.AttendanceCheckInRequest{
		ChildID: 999,
	}

	_, err := svc.CheckIn(ctx, org.ID, req, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestAttendanceService_CheckIn_WrongOrg(t *testing.T) {
	svc, _, _, _, child := setupAttendanceTest(t)
	ctx := context.Background()

	req := &models.AttendanceCheckInRequest{
		ChildID: child.ID,
	}

	// Use a different org ID
	_, err := svc.CheckIn(ctx, 999, req, 1)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
}

func TestAttendanceService_CheckIn_DuplicateToday(t *testing.T) {
	svc, _, _, org, child := setupAttendanceTest(t)
	ctx := context.Background()

	req := &models.AttendanceCheckInRequest{
		ChildID: child.ID,
	}

	// First check-in should succeed
	_, err := svc.CheckIn(ctx, org.ID, req, 1)
	if err != nil {
		t.Fatalf("first check-in failed: %v", err)
	}

	// Second check-in should fail (duplicate)
	_, err = svc.CheckIn(ctx, org.ID, req, 1)
	if err == nil {
		t.Fatal("expected error for duplicate check-in, got nil")
	}
}

func TestAttendanceService_CheckOut(t *testing.T) {
	svc, _, _, org, child := setupAttendanceTest(t)
	ctx := context.Background()

	// First check in
	checkInReq := &models.AttendanceCheckInRequest{
		ChildID: child.ID,
	}
	checkInResp, _ := svc.CheckIn(ctx, org.ID, checkInReq, 1)

	// Then check out
	checkOutReq := &models.AttendanceCheckOutRequest{
		Note: "Picked up by mother",
	}
	resp, err := svc.CheckOut(ctx, checkInResp.ID, org.ID, checkOutReq)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.CheckOutTime == nil {
		t.Error("expected CheckOutTime to be set")
	}
}

func TestAttendanceService_CheckOut_AlreadyCheckedOut(t *testing.T) {
	svc, _, _, org, child := setupAttendanceTest(t)
	ctx := context.Background()

	checkInReq := &models.AttendanceCheckInRequest{ChildID: child.ID}
	checkInResp, _ := svc.CheckIn(ctx, org.ID, checkInReq, 1)

	checkOutReq := &models.AttendanceCheckOutRequest{}
	svc.CheckOut(ctx, checkInResp.ID, org.ID, checkOutReq)

	// Try to check out again
	_, err := svc.CheckOut(ctx, checkInResp.ID, org.ID, checkOutReq)
	if err == nil {
		t.Fatal("expected error for double check-out, got nil")
	}
}

func TestAttendanceService_MarkAbsent(t *testing.T) {
	svc, _, _, org, child := setupAttendanceTest(t)
	ctx := context.Background()

	req := &models.AttendanceMarkAbsentRequest{
		ChildID: child.ID,
		Date:    "2025-06-15",
		Status:  models.AttendanceStatusSick,
		Note:    "Has a cold",
	}

	resp, err := svc.MarkAbsent(ctx, org.ID, req, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Status != models.AttendanceStatusSick {
		t.Errorf("expected status 'sick', got '%s'", resp.Status)
	}
	if resp.Note != "Has a cold" {
		t.Errorf("expected note 'Has a cold', got '%s'", resp.Note)
	}
}

func TestAttendanceService_MarkAbsent_InvalidStatus(t *testing.T) {
	svc, _, _, org, child := setupAttendanceTest(t)
	ctx := context.Background()

	req := &models.AttendanceMarkAbsentRequest{
		ChildID: child.ID,
		Date:    "2025-06-15",
		Status:  "invalid",
	}

	_, err := svc.MarkAbsent(ctx, org.ID, req, 1)
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
}

func TestAttendanceService_MarkAbsent_PresentStatus(t *testing.T) {
	svc, _, _, org, child := setupAttendanceTest(t)
	ctx := context.Background()

	req := &models.AttendanceMarkAbsentRequest{
		ChildID: child.ID,
		Date:    "2025-06-15",
		Status:  models.AttendanceStatusPresent,
	}

	_, err := svc.MarkAbsent(ctx, org.ID, req, 1)
	if err == nil {
		t.Fatal("expected error when marking as present via absent endpoint, got nil")
	}
}

func TestAttendanceService_MarkAbsent_InvalidDate(t *testing.T) {
	svc, _, _, org, child := setupAttendanceTest(t)
	ctx := context.Background()

	req := &models.AttendanceMarkAbsentRequest{
		ChildID: child.ID,
		Date:    "invalid-date",
		Status:  models.AttendanceStatusSick,
	}

	_, err := svc.MarkAbsent(ctx, org.ID, req, 1)
	if err == nil {
		t.Fatal("expected error for invalid date, got nil")
	}
}

func TestAttendanceService_Delete(t *testing.T) {
	svc, _, _, org, child := setupAttendanceTest(t)
	ctx := context.Background()

	req := &models.AttendanceCheckInRequest{ChildID: child.ID}
	resp, _ := svc.CheckIn(ctx, org.ID, req, 1)

	err := svc.Delete(ctx, resp.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's gone
	_, err = svc.GetByID(ctx, resp.ID, org.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestAttendanceService_Delete_WrongOrg(t *testing.T) {
	svc, _, _, org, child := setupAttendanceTest(t)
	ctx := context.Background()

	req := &models.AttendanceCheckInRequest{ChildID: child.ID}
	resp, _ := svc.CheckIn(ctx, org.ID, req, 1)

	// Try to delete from wrong org
	err := svc.Delete(ctx, resp.ID, 999)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
}

func TestAttendanceService_ListByDate(t *testing.T) {
	svc, _, _, org, child := setupAttendanceTest(t)
	ctx := context.Background()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	req := &models.AttendanceCheckInRequest{ChildID: child.ID}
	svc.CheckIn(ctx, org.ID, req, 1)

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

func TestAttendanceService_GetDailySummary(t *testing.T) {
	svc, attendanceStore, _, org, _ := setupAttendanceTest(t)
	ctx := context.Background()

	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	now := time.Now()

	// Create attendance records directly in store for specific date
	db := attendanceStore
	child1 := &models.Child{Person: models.Person{FirstName: "C1", LastName: "L", OrganizationID: org.ID, Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}}
	child2 := &models.Child{Person: models.Person{FirstName: "C2", LastName: "L", OrganizationID: org.ID, Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}}
	child3 := &models.Child{Person: models.Person{FirstName: "C3", LastName: "L", OrganizationID: org.ID, Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(&models.Attendance{ChildID: child1.ID, OrganizationID: org.ID, Date: today, Status: models.AttendanceStatusPresent, RecordedBy: 1, CheckInTime: &now})
	db.Create(&models.Attendance{ChildID: child2.ID, OrganizationID: org.ID, Date: today, Status: models.AttendanceStatusPresent, RecordedBy: 1, CheckInTime: &now})
	db.Create(&models.Attendance{ChildID: child3.ID, OrganizationID: org.ID, Date: today, Status: models.AttendanceStatusSick, RecordedBy: 1})

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
