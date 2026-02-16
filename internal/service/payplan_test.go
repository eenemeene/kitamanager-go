package service

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

// PayPlan CRUD tests

func TestPayPlanService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := models.PayPlanCreateRequest{
		Name: "TVöD-SuE",
	}

	resp, err := svc.Create(ctx, org.ID, &req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.ID == 0 {
		t.Error("expected ID to be set")
	}
	if resp.Name != "TVöD-SuE" {
		t.Errorf("Name = %v, want TVöD-SuE", resp.Name)
	}
	if resp.OrganizationID != org.ID {
		t.Errorf("OrganizationID = %d, want %d", resp.OrganizationID, org.ID)
	}
}

func TestPayPlanService_GetByID(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	// Create a period with entries
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, &to, 39.0)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)

	resp, err := svc.GetByID(ctx, payplan.ID, org.ID, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.ID != payplan.ID {
		t.Errorf("ID = %d, want %d", resp.ID, payplan.ID)
	}
	if resp.Name != "TVöD-SuE" {
		t.Errorf("Name = %v, want TVöD-SuE", resp.Name)
	}
	if len(resp.Periods) != 1 {
		t.Fatalf("expected 1 period, got %d", len(resp.Periods))
	}
	if len(resp.Periods[0].Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(resp.Periods[0].Entries))
	}
}

func TestPayPlanService_GetByID_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org1.ID)

	_, err := svc.GetByID(ctx, payplan.ID, org2.ID, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPayPlanService_GetByID_ActiveOnFilter(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	// Period 1: 2024-01-01 to 2024-06-30
	from1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	createTestPayPlanPeriod(t, db, payplan.ID, from1, &to1, 39.0)

	// Period 2: 2024-07-01 to 2024-12-31
	from2 := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	to2 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	createTestPayPlanPeriod(t, db, payplan.ID, from2, &to2, 39.0)

	// Filter to March 2024 - should only get period 1
	activeOn := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	resp, err := svc.GetByID(ctx, payplan.ID, org.ID, &activeOn)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(resp.Periods) != 1 {
		t.Fatalf("expected 1 period for activeOn March 2024, got %d", len(resp.Periods))
	}
	expectedFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if !resp.Periods[0].From.Equal(expectedFrom) {
		t.Errorf("expected period from %v, got %v", expectedFrom, resp.Periods[0].From)
	}
}

func TestPayPlanService_List(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	createTestPayPlan(t, db, "Plan A", org.ID)
	createTestPayPlan(t, db, "Plan B", org.ID)
	createTestPayPlan(t, db, "Plan C", org.ID)

	// Test pagination: first page
	plans, total, err := svc.List(ctx, org.ID, 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(plans) != 2 {
		t.Errorf("expected 2 plans on first page, got %d", len(plans))
	}

	// Second page
	plans, _, err = svc.List(ctx, org.ID, 2, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(plans) != 1 {
		t.Errorf("expected 1 plan on second page, got %d", len(plans))
	}
}

func TestPayPlanService_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	plans, total, err := svc.List(ctx, org.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(plans) != 0 {
		t.Errorf("expected 0 plans, got %d", len(plans))
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
}

func TestPayPlanService_Update(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "Original Name", org.ID)

	updatedName := "Updated Name"
	req := models.PayPlanUpdateRequest{
		Name: &updatedName,
	}

	resp, err := svc.Update(ctx, payplan.ID, org.ID, &req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Name != "Updated Name" {
		t.Errorf("Name = %v, want Updated Name", resp.Name)
	}
}

func TestPayPlanService_Update_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org1.ID)

	updName := "Updated"
	req := models.PayPlanUpdateRequest{
		Name: &updName,
	}

	_, err := svc.Update(ctx, payplan.ID, org2.ID, &req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPayPlanService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "To Delete", org.ID)

	err := svc.Delete(ctx, payplan.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's deleted
	_, err = svc.GetByID(ctx, payplan.ID, org.ID, nil)
	if err == nil {
		t.Error("expected error getting deleted pay plan")
	}
}

func TestPayPlanService_Delete_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org1.ID)

	err := svc.Delete(ctx, payplan.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Period CRUD tests

func TestPayPlanService_CreatePeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	// With To date
	fromDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req := models.PayPlanPeriodCreateRequest{
		From:        fromDate,
		To:          &toDate,
		WeeklyHours: 39.0,
	}

	resp, err := svc.CreatePeriod(ctx, payplan.ID, org.ID, &req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.ID == 0 {
		t.Error("expected ID to be set")
	}
	if resp.PayPlanID != payplan.ID {
		t.Errorf("PayPlanID = %d, want %d", resp.PayPlanID, payplan.ID)
	}
	if !resp.From.Equal(fromDate) {
		t.Errorf("From = %v, want %v", resp.From, fromDate)
	}
	if resp.To == nil || !resp.To.Equal(toDate) {
		t.Errorf("To = %v, want %v", resp.To, toDate)
	}
	if resp.WeeklyHours != 39.0 {
		t.Errorf("WeeklyHours = %f, want 39.0", resp.WeeklyHours)
	}
}

func TestPayPlanService_CreatePeriod_WithoutTo(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	req := models.PayPlanPeriodCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          nil,
		WeeklyHours: 39.0,
	}

	resp, err := svc.CreatePeriod(ctx, payplan.ID, org.ID, &req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.To != nil {
		t.Errorf("To = %v, want nil", resp.To)
	}
}

func TestPayPlanService_CreatePeriod_WrongPayPlan(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := models.PayPlanPeriodCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		WeeklyHours: 39.0,
	}

	_, err := svc.CreatePeriod(ctx, 999, org.ID, &req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPayPlanService_GetPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)

	resp, err := svc.GetPeriod(ctx, period.ID, payplan.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.ID != period.ID {
		t.Errorf("ID = %d, want %d", resp.ID, period.ID)
	}
	if resp.PayPlanID != payplan.ID {
		t.Errorf("PayPlanID = %d, want %d", resp.PayPlanID, payplan.ID)
	}
	// Entries should be preloaded
	if len(resp.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(resp.Entries))
	}
}

func TestPayPlanService_GetPeriod_WrongPayPlan(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan1 := createTestPayPlan(t, db, "Plan 1", org.ID)
	payplan2 := createTestPayPlan(t, db, "Plan 2", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan1.ID, from, nil, 39.0)

	_, err := svc.GetPeriod(ctx, period.ID, payplan2.ID, org.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPayPlanService_GetPeriod_WrongPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	_, err := svc.GetPeriod(ctx, 999, payplan.ID, org.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPayPlanService_UpdatePeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)

	newFrom := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	newTo := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req := models.PayPlanPeriodUpdateRequest{
		From:        newFrom,
		To:          &newTo,
		WeeklyHours: 38.5,
	}

	resp, err := svc.UpdatePeriod(ctx, period.ID, payplan.ID, org.ID, &req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !resp.From.Equal(newFrom) {
		t.Errorf("From = %v, want %v", resp.From, newFrom)
	}
	if resp.To == nil || !resp.To.Equal(newTo) {
		t.Errorf("To = %v, want %v", resp.To, newTo)
	}
	if resp.WeeklyHours != 38.5 {
		t.Errorf("WeeklyHours = %f, want 38.5", resp.WeeklyHours)
	}
}

func TestPayPlanService_UpdatePeriod_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org1.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)

	req := models.PayPlanPeriodUpdateRequest{
		From:        time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		WeeklyHours: 39.0,
	}

	_, err := svc.UpdatePeriod(ctx, period.ID, payplan.ID, org2.ID, &req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPayPlanService_UpdatePeriod_WrongPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	req := models.PayPlanPeriodUpdateRequest{
		From:        time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		WeeklyHours: 39.0,
	}

	_, err := svc.UpdatePeriod(ctx, 999, payplan.ID, org.ID, &req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPayPlanService_DeletePeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)

	err := svc.DeletePeriod(ctx, period.ID, payplan.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's deleted
	_, err = svc.GetPeriod(ctx, period.ID, payplan.ID, org.ID)
	if err == nil {
		t.Error("expected error getting deleted period")
	}
}

func TestPayPlanService_DeletePeriod_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org1.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)

	err := svc.DeletePeriod(ctx, period.ID, payplan.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPayPlanService_DeletePeriod_WrongPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	err := svc.DeletePeriod(ctx, 999, payplan.ID, org.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Entry CRUD tests

func TestPayPlanService_CreateEntry(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)

	stepMinYears := 3
	req := models.PayPlanEntryCreateRequest{
		Grade:         "S8a",
		Step:          3,
		MonthlyAmount: 400000,
		StepMinYears:  &stepMinYears,
	}

	resp, err := svc.CreateEntry(ctx, period.ID, payplan.ID, org.ID, &req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.ID == 0 {
		t.Error("expected ID to be set")
	}
	if resp.PeriodID != period.ID {
		t.Errorf("PeriodID = %d, want %d", resp.PeriodID, period.ID)
	}
	if resp.Grade != "S8a" {
		t.Errorf("Grade = %s, want S8a", resp.Grade)
	}
	if resp.Step != 3 {
		t.Errorf("Step = %d, want 3", resp.Step)
	}
	if resp.MonthlyAmount != 400000 {
		t.Errorf("MonthlyAmount = %d, want 400000", resp.MonthlyAmount)
	}
	if resp.StepMinYears == nil || *resp.StepMinYears != 3 {
		t.Errorf("StepMinYears = %v, want 3", resp.StepMinYears)
	}
}

func TestPayPlanService_CreateEntry_WrongPayPlan(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)

	req := models.PayPlanEntryCreateRequest{
		Grade:         "S8a",
		Step:          3,
		MonthlyAmount: 400000,
	}

	// Wrong payplan ID in the ownership chain
	_, err := svc.CreateEntry(ctx, period.ID, 999, org.ID, &req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPayPlanService_CreateEntry_WrongPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	req := models.PayPlanEntryCreateRequest{
		Grade:         "S8a",
		Step:          3,
		MonthlyAmount: 400000,
	}

	// Period 999 doesn't exist
	_, err := svc.CreateEntry(ctx, 999, payplan.ID, org.ID, &req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPayPlanService_GetEntry(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)
	entry := createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)

	resp, err := svc.GetEntry(ctx, entry.ID, period.ID, payplan.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.ID != entry.ID {
		t.Errorf("ID = %d, want %d", resp.ID, entry.ID)
	}
	if resp.Grade != "S8a" {
		t.Errorf("Grade = %s, want S8a", resp.Grade)
	}
}

func TestPayPlanService_GetEntry_WrongPayPlan(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)
	entry := createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)

	_, err := svc.GetEntry(ctx, entry.ID, period.ID, 999, org.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPayPlanService_GetEntry_WrongPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)
	entry := createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)

	// Create a second period to use as wrong period
	from2 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	period2 := createTestPayPlanPeriod(t, db, payplan.ID, from2, nil, 39.0)

	_, err := svc.GetEntry(ctx, entry.ID, period2.ID, payplan.ID, org.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPayPlanService_UpdateEntry(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)
	entry := createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)

	stepMinYears := 5
	req := models.PayPlanEntryUpdateRequest{
		Grade:         "S11b",
		Step:          4,
		MonthlyAmount: 500000,
		StepMinYears:  &stepMinYears,
	}

	resp, err := svc.UpdateEntry(ctx, entry.ID, period.ID, payplan.ID, org.ID, &req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Grade != "S11b" {
		t.Errorf("Grade = %s, want S11b", resp.Grade)
	}
	if resp.Step != 4 {
		t.Errorf("Step = %d, want 4", resp.Step)
	}
	if resp.MonthlyAmount != 500000 {
		t.Errorf("MonthlyAmount = %d, want 500000", resp.MonthlyAmount)
	}
	if resp.StepMinYears == nil || *resp.StepMinYears != 5 {
		t.Errorf("StepMinYears = %v, want 5", resp.StepMinYears)
	}
}

func TestPayPlanService_UpdateEntry_WrongPayPlan(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)
	entry := createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)

	req := models.PayPlanEntryUpdateRequest{
		Grade:         "S11b",
		Step:          4,
		MonthlyAmount: 500000,
	}

	_, err := svc.UpdateEntry(ctx, entry.ID, period.ID, 999, org.ID, &req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPayPlanService_UpdateEntry_WrongPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)
	entry := createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)

	from2 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	period2 := createTestPayPlanPeriod(t, db, payplan.ID, from2, nil, 39.0)

	req := models.PayPlanEntryUpdateRequest{
		Grade:         "S11b",
		Step:          4,
		MonthlyAmount: 500000,
	}

	_, err := svc.UpdateEntry(ctx, entry.ID, period2.ID, payplan.ID, org.ID, &req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPayPlanService_DeleteEntry(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)
	entry := createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)

	err := svc.DeleteEntry(ctx, entry.ID, period.ID, payplan.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's deleted
	_, err = svc.GetEntry(ctx, entry.ID, period.ID, payplan.ID, org.ID)
	if err == nil {
		t.Error("expected error getting deleted entry")
	}
}

func TestPayPlanService_DeleteEntry_WrongPayPlan(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)
	entry := createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)

	err := svc.DeleteEntry(ctx, entry.ID, period.ID, 999, org.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPayPlanService_DeleteEntry_WrongPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)
	entry := createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)

	from2 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	period2 := createTestPayPlanPeriod(t, db, payplan.ID, from2, nil, 39.0)

	err := svc.DeleteEntry(ctx, entry.ID, period2.ID, payplan.ID, org.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// CalculateSalary tests

func TestPayPlanService_CalculateSalary_FullTime(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	// Period: 2024-01-01 to 2024-12-31, weekly hours = 39.0
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, &to, 39.0)

	// Entry: grade S8a, step 3, monthly amount 400000 (4000 EUR)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)

	// Full-time: 39/39 = 100%
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	salary, err := svc.CalculateSalary(ctx, payplan.ID, "S8a", 3, 39.0, date)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := 400000 // 400000 * (39/39) = 400000
	if salary != expected {
		t.Errorf("salary = %d, want %d", salary, expected)
	}
}

func TestPayPlanService_CalculateSalary_PartTime(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, &to, 39.0)

	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)

	// Part-time: 20/39 = 51.28%
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	salary, err := svc.CalculateSalary(ctx, payplan.ID, "S8a", 3, 20.0, date)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := int(math.Round(400000 * (20.0 / 39.0))) // 205128
	if salary != expected {
		t.Errorf("salary = %d, want %d", salary, expected)
	}
}

func TestPayPlanService_CalculateSalary_NoActivePeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	// Period: 2024-01-01 to 2024-06-30
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, &to, 39.0)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)

	// Query for a date outside the period
	date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	_, err := svc.CalculateSalary(ctx, payplan.ID, "S8a", 3, 39.0, date)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPayPlanService_CalculateSalary_NoMatchingGradeStep(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, &to, 39.0)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)

	// Query for a grade/step that doesn't exist
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	_, err := svc.CalculateSalary(ctx, payplan.ID, "S11b", 5, 39.0, date)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// EmployerContributionRate tests

func TestPayPlanService_CreatePeriod_WithEmployerContributionRate(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	fromDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := models.PayPlanPeriodCreateRequest{
		From:                     fromDate,
		WeeklyHours:              39.0,
		EmployerContributionRate: 2200, // 22.00%
	}

	resp, err := svc.CreatePeriod(ctx, payplan.ID, org.ID, &req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.EmployerContributionRate != 2200 {
		t.Errorf("EmployerContributionRate = %d, want 2200", resp.EmployerContributionRate)
	}
}

func TestPayPlanService_CreatePeriod_DefaultEmployerContributionRate(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	req := models.PayPlanPeriodCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		WeeklyHours: 39.0,
	}

	resp, err := svc.CreatePeriod(ctx, payplan.ID, org.ID, &req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.EmployerContributionRate != 0 {
		t.Errorf("EmployerContributionRate = %d, want 0", resp.EmployerContributionRate)
	}
}

func TestPayPlanService_UpdatePeriod_EmployerContributionRate(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)

	req := models.PayPlanPeriodUpdateRequest{
		From:                     from,
		WeeklyHours:              39.0,
		EmployerContributionRate: 2350, // 23.50%
	}

	resp, err := svc.UpdatePeriod(ctx, period.ID, payplan.ID, org.ID, &req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.EmployerContributionRate != 2350 {
		t.Errorf("EmployerContributionRate = %d, want 2350", resp.EmployerContributionRate)
	}
}
