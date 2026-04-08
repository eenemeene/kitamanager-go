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

func TestGovernmentFundingService_List(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	// Create test funding
	funding := &models.GovernmentFunding{Name: "Berlin Funding", State: "berlin"}
	db.Create(funding)

	fundings, total, err := svc.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(fundings) != 1 {
		t.Errorf("expected 1 funding, got %d", len(fundings))
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
}

func TestGovernmentFundingService_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	fundings, total, err := svc.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(fundings) != 0 {
		t.Errorf("expected 0 fundings, got %d", len(fundings))
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
}

func TestGovernmentFundingService_GetByID(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Berlin Funding", State: "berlin"}
	db.Create(funding)

	found, err := svc.GetByID(ctx, funding.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.ID != funding.ID {
		t.Errorf("ID = %d, want %d", found.ID, funding.ID)
	}
	if found.Name != "Berlin Funding" {
		t.Errorf("Name = %s, want Berlin Funding", found.Name)
	}
}

func TestGovernmentFundingService_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 999)
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

func TestGovernmentFundingService_GetByIDWithDetails(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Berlin Funding", State: "berlin"}
	db.Create(funding)

	// Create periods
	period1 := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period1)

	period2 := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period2)

	// Test with limit 1 (default)
	found, err := svc.GetByIDWithDetails(ctx, funding.ID, 1, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.TotalPeriods != 2 {
		t.Errorf("TotalPeriods = %d, want 2", found.TotalPeriods)
	}
	if len(found.Periods) != 1 {
		t.Errorf("expected 1 period with limit=1, got %d", len(found.Periods))
	}

	// Test with limit 0 (all)
	found, err = svc.GetByIDWithDetails(ctx, funding.ID, 0, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(found.Periods) != 2 {
		t.Errorf("expected 2 periods with limit=0, got %d", len(found.Periods))
	}
}

func TestGovernmentFundingService_GetByState(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Berlin Funding", State: "berlin"}
	db.Create(funding)

	found, err := svc.GetByState(ctx, "berlin")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found.ID != funding.ID {
		t.Errorf("expected ID %d, got %d", funding.ID, found.ID)
	}
	if found.State != "berlin" {
		t.Errorf("expected state berlin, got %s", found.State)
	}
}

func TestGovernmentFundingService_GetByState_NotFound(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	_, err := svc.GetByState(ctx, "berlin")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGovernmentFundingService_GetByStateWithDetails(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Berlin Funding", State: "berlin"}
	db.Create(funding)

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period)

	minAge := 0
	maxAge := 3
	prop := &models.GovernmentFundingProperty{
		PeriodID: period.ID,
		Key:      "care_type",
		Value:    "ganztag",
		Label:    "Ganztag",
		Payment:  100000,
		MinAge:   &minAge,
		MaxAge:   &maxAge,
	}
	db.Create(prop)

	found, err := svc.GetByStateWithDetails(ctx, "berlin")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found.ID != funding.ID {
		t.Errorf("expected ID %d, got %d", funding.ID, found.ID)
	}
	if len(found.Periods) != 1 {
		t.Fatalf("expected 1 period, got %d", len(found.Periods))
	}
	if len(found.Periods[0].Properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(found.Periods[0].Properties))
	}
	if found.Periods[0].Properties[0].Key != "care_type" {
		t.Errorf("expected key care_type, got %s", found.Periods[0].Properties[0].Key)
	}
}

func TestGovernmentFundingService_GetByStateWithDetails_NotFound(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	_, err := svc.GetByStateWithDetails(ctx, "berlin")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGovernmentFundingService_Create(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	req := &models.GovernmentFundingCreateRequest{
		Name:  "Berlin Funding",
		State: "berlin",
	}

	result, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Name != "Berlin Funding" {
		t.Errorf("Name = %s, want Berlin Funding", result.Name)
	}
	if result.State != "berlin" {
		t.Errorf("State = %s, want berlin", result.State)
	}
	if result.ID == 0 {
		t.Error("expected ID to be set")
	}
}

func TestGovernmentFundingService_Create_InvalidState(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	req := &models.GovernmentFundingCreateRequest{
		Name:  "Invalid Funding",
		State: "invalid",
	}

	_, err := svc.Create(ctx, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestGovernmentFundingService_Create_WhitespaceName(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	req := &models.GovernmentFundingCreateRequest{
		Name:  "   ",
		State: "berlin",
	}

	_, err := svc.Create(ctx, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestGovernmentFundingService_Create_TrimsName(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	req := &models.GovernmentFundingCreateRequest{
		Name:  "  Berlin Funding  ",
		State: "berlin",
	}

	result, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Name != "Berlin Funding" {
		t.Errorf("Name = %s, want 'Berlin Funding' (trimmed)", result.Name)
	}
}

func TestGovernmentFundingService_Update(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Original Name", State: "berlin"}
	db.Create(funding)

	newName := "Updated Name"
	req := &models.GovernmentFundingUpdateRequest{Name: &newName}

	result, err := svc.Update(ctx, funding.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Name != "Updated Name" {
		t.Errorf("Name = %s, want Updated Name", result.Name)
	}
}

func TestGovernmentFundingService_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	newName := "Updated Name"
	req := &models.GovernmentFundingUpdateRequest{Name: &newName}

	_, err := svc.Update(ctx, 999, req)
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

func TestGovernmentFundingService_Update_WhitespaceName(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Original Name", State: "berlin"}
	db.Create(funding)

	newName := "   "
	req := &models.GovernmentFundingUpdateRequest{Name: &newName}

	_, err := svc.Update(ctx, funding.ID, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestGovernmentFundingService_Delete(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "To Delete", State: "berlin"}
	db.Create(funding)

	err := svc.Delete(ctx, funding.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify deleted
	_, err = svc.GetByID(ctx, funding.ID)
	if err == nil {
		t.Error("expected error getting deleted funding")
	}
}

// Period tests

func TestGovernmentFundingService_CreatePeriod(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	req := &models.GovernmentFundingPeriodCreateRequest{
		From:                time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		FullTimeWeeklyHours: 39.0,
		Comment:             "Test period",
	}

	result, err := svc.CreatePeriod(ctx, funding.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.GovernmentFundingID != funding.ID {
		t.Errorf("GovernmentFundingID = %d, want %d", result.GovernmentFundingID, funding.ID)
	}
	if result.Comment != "Test period" {
		t.Errorf("Comment = %s, want 'Test period'", result.Comment)
	}
	if result.FullTimeWeeklyHours != 39.0 {
		t.Errorf("FullTimeWeeklyHours = %f, want 39.0", result.FullTimeWeeklyHours)
	}
}

func TestGovernmentFundingService_CreatePeriod_FundingNotFound(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	req := &models.GovernmentFundingPeriodCreateRequest{
		From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	_, err := svc.CreatePeriod(ctx, 999, req)
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

func TestGovernmentFundingService_CreatePeriod_OverlapRejected(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	// Create first period
	to := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &to},
	}
	db.Create(period)

	// Try to create overlapping period
	req := &models.GovernmentFundingPeriodCreateRequest{
		From: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
	}

	_, err := svc.CreatePeriod(ctx, funding.ID, req)
	if err == nil {
		t.Fatal("expected error for overlapping period, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestGovernmentFundingService_UpdatePeriod(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		Comment:             "Original comment",
	}
	db.Create(period)

	newComment := "Updated comment"
	req := &models.GovernmentFundingPeriodUpdateRequest{
		Comment: &newComment,
	}

	result, err := svc.UpdatePeriod(ctx, period.ID, funding.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Comment != "Updated comment" {
		t.Errorf("Comment = %s, want 'Updated comment'", result.Comment)
	}
}

func TestGovernmentFundingService_UpdatePeriod_FullTimeWeeklyHours(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		FullTimeWeeklyHours: 39.0,
	}
	db.Create(period)

	newHours := 40.0
	req := &models.GovernmentFundingPeriodUpdateRequest{
		FullTimeWeeklyHours: &newHours,
	}

	result, err := svc.UpdatePeriod(ctx, period.ID, funding.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.FullTimeWeeklyHours != 40.0 {
		t.Errorf("FullTimeWeeklyHours = %f, want 40.0", result.FullTimeWeeklyHours)
	}
}

func TestGovernmentFundingService_UpdatePeriod_NotFound(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	newComment := "Updated comment"
	req := &models.GovernmentFundingPeriodUpdateRequest{
		Comment: &newComment,
	}

	_, err := svc.UpdatePeriod(ctx, 999, 999, req)
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

func TestGovernmentFundingService_DeletePeriod(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period)

	err := svc.DeletePeriod(ctx, period.ID, funding.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify deleted
	_, err = svc.GetPeriodByID(ctx, period.ID)
	if err == nil {
		t.Error("expected error getting deleted period")
	}
}

// Property tests

func TestGovernmentFundingService_CreateProperty(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period)

	minAge := 0
	maxAge := 3
	req := &models.GovernmentFundingPropertyCreateRequest{
		Key:         "care_type",
		Value:       "ganztag",
		Label:       "Ganztag",
		Payment:     100000,
		Requirement: 0.1,
		MinAge:      &minAge,
		MaxAge:      &maxAge,
	}

	result, err := svc.CreateProperty(ctx, funding.ID, period.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Key != "care_type" {
		t.Errorf("Key = %s, want care_type", result.Key)
	}
	if result.Value != "ganztag" {
		t.Errorf("Value = %s, want ganztag", result.Value)
	}
	if result.Payment != 100000 {
		t.Errorf("Payment = %d, want 100000", result.Payment)
	}
	if result.Label != "Ganztag" {
		t.Errorf("Label = %q, want %q", result.Label, "Ganztag")
	}
}

func TestGovernmentFundingService_CreateProperty_PeriodNotFound(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	req := &models.GovernmentFundingPropertyCreateRequest{
		Key:         "care_type",
		Value:       "ganztag",
		Label:       "Ganztag",
		Payment:     100000,
		Requirement: 0.1,
	}

	_, err := svc.CreateProperty(ctx, 999, 999, req)
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

func TestGovernmentFundingService_CreateProperty_InvalidAgeRange(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period)

	// min >= max is invalid
	minAge := 5
	maxAge := 3
	req := &models.GovernmentFundingPropertyCreateRequest{
		Key:         "care_type",
		Value:       "ganztag",
		Label:       "Ganztag",
		Payment:     100000,
		Requirement: 0.1,
		MinAge:      &minAge,
		MaxAge:      &maxAge,
	}

	_, err := svc.CreateProperty(ctx, funding.ID, period.ID, req)
	if err == nil {
		t.Fatal("expected error for invalid age range, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestGovernmentFundingService_CreateProperty_WhitespaceKey(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period)

	req := &models.GovernmentFundingPropertyCreateRequest{
		Key:         "   ",
		Value:       "ganztag",
		Label:       "Ganztag",
		Payment:     100000,
		Requirement: 0.1,
	}

	_, err := svc.CreateProperty(ctx, funding.ID, period.ID, req)
	if err == nil {
		t.Fatal("expected error for whitespace key, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestGovernmentFundingService_UpdateProperty(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period)

	property := &models.GovernmentFundingProperty{
		PeriodID:    period.ID,
		Key:         "care_type",
		Value:       "ganztag",
		Label:       "Ganztag",
		Payment:     100000,
		Requirement: 0.1,
	}
	db.Create(property)

	newPayment := 150000
	req := &models.GovernmentFundingPropertyUpdateRequest{
		Payment: &newPayment,
	}

	result, err := svc.UpdateProperty(ctx, property.ID, period.ID, funding.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Payment != 150000 {
		t.Errorf("Payment = %d, want 150000", result.Payment)
	}
}

func TestGovernmentFundingService_UpdateProperty_NotFound(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	newPayment := 150000
	req := &models.GovernmentFundingPropertyUpdateRequest{
		Payment: &newPayment,
	}

	_, err := svc.UpdateProperty(ctx, 999, 999, 999, req)
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

func TestGovernmentFundingService_DeleteProperty(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period)

	property := &models.GovernmentFundingProperty{
		PeriodID:    period.ID,
		Key:         "care_type",
		Value:       "ganztag",
		Label:       "Ganztag",
		Payment:     100000,
		Requirement: 0.1,
	}
	db.Create(property)

	err := svc.DeleteProperty(ctx, property.ID, period.ID, funding.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify deleted
	_, err = svc.GetPropertyByID(ctx, property.ID)
	if err == nil {
		t.Error("expected error getting deleted property")
	}
}

// Ownership validation tests

func TestGovernmentFundingService_UpdatePeriod_WrongFunding(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding1 := &models.GovernmentFunding{Name: "Funding 1", State: "berlin"}
	db.Create(funding1)

	funding2 := &models.GovernmentFunding{Name: "Funding 2", State: "berlin"}
	db.Create(funding2)

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding1.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period)

	newComment := "Hacked"
	req := &models.GovernmentFundingPeriodUpdateRequest{Comment: &newComment}

	// Try to update period via wrong funding ID
	_, err := svc.UpdatePeriod(ctx, period.ID, funding2.ID, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGovernmentFundingService_DeletePeriod_WrongFunding(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding1 := &models.GovernmentFunding{Name: "Funding 1", State: "berlin"}
	db.Create(funding1)

	funding2 := &models.GovernmentFunding{Name: "Funding 2", State: "berlin"}
	db.Create(funding2)

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding1.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period)

	// Try to delete period via wrong funding ID
	err := svc.DeletePeriod(ctx, period.ID, funding2.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// Verify period was NOT deleted
	_, err = svc.GetPeriodByID(ctx, period.ID)
	if err != nil {
		t.Error("period should still exist after failed delete with wrong funding ID")
	}
}

func TestGovernmentFundingService_CreateProperty_WrongFunding(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding1 := &models.GovernmentFunding{Name: "Funding 1", State: "berlin"}
	db.Create(funding1)

	funding2 := &models.GovernmentFunding{Name: "Funding 2", State: "berlin"}
	db.Create(funding2)

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding1.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period)

	req := &models.GovernmentFundingPropertyCreateRequest{
		Key: "care_type", Value: "ganztag", Label: "Ganztag", Payment: 100000, Requirement: 0.1,
	}

	// Try to create property via wrong funding ID
	_, err := svc.CreateProperty(ctx, funding2.ID, period.ID, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGovernmentFundingService_UpdateProperty_WrongPeriod(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	period1 := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period1)

	to := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	period2 := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), To: &to},
	}
	db.Create(period2)

	property := &models.GovernmentFundingProperty{
		PeriodID: period1.ID, Key: "care_type", Value: "ganztag", Label: "Ganztag", Payment: 100000, Requirement: 0.1,
	}
	db.Create(property)

	newPayment := 999999
	req := &models.GovernmentFundingPropertyUpdateRequest{Payment: &newPayment}

	// Try to update property via wrong period ID
	_, err := svc.UpdateProperty(ctx, property.ID, period2.ID, funding.ID, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGovernmentFundingService_DeleteProperty_WrongPeriod(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	period1 := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period1)

	to := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	period2 := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), To: &to},
	}
	db.Create(period2)

	property := &models.GovernmentFundingProperty{
		PeriodID: period1.ID, Key: "care_type", Value: "ganztag", Label: "Ganztag", Payment: 100000, Requirement: 0.1,
	}
	db.Create(property)

	// Try to delete property via wrong period ID
	err := svc.DeleteProperty(ctx, property.ID, period2.ID, funding.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// Verify property was NOT deleted
	_, err = svc.GetPropertyByID(ctx, property.ID)
	if err != nil {
		t.Error("property should still exist after failed delete with wrong period ID")
	}
}

// =========================================
// Nullable field clearing tests
// =========================================

func TestGovernmentFundingService_UpdateProperty_ClearNullableAgeFields(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := createTestGovernmentFunding(t, db, "Test Funding")
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)

	// Create property with age range set
	minAge := 0
	maxAge := 3
	property := &models.GovernmentFundingProperty{
		PeriodID:    period.ID,
		Key:         "care_type",
		Value:       "ganztag",
		Label:       "Ganztag",
		Payment:     100000,
		Requirement: 0.1,
		MinAge:      &minAge,
		MaxAge:      &maxAge,
	}
	db.Create(property)

	// Verify ages are set
	found, _ := svc.GetPropertyByID(ctx, property.ID)
	if found.MinAge == nil || *found.MinAge != 0 {
		t.Fatalf("setup: MinAge should be 0, got %v", found.MinAge)
	}
	if found.MaxAge == nil || *found.MaxAge != 3 {
		t.Fatalf("setup: MaxAge should be 3, got %v", found.MaxAge)
	}

	// Clear ages by sending nil (simulates frontend sending null)
	req := &models.GovernmentFundingPropertyUpdateRequest{
		MinAge: nil,
		MaxAge: nil,
	}
	updated, err := svc.UpdateProperty(ctx, property.ID, period.ID, funding.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.MinAge != nil {
		t.Errorf("MinAge should be nil after clearing, got %d", *updated.MinAge)
	}
	if updated.MaxAge != nil {
		t.Errorf("MaxAge should be nil after clearing, got %d", *updated.MaxAge)
	}

	// Verify persistence by re-fetching
	refetched, err := svc.GetPropertyByID(ctx, property.ID)
	if err != nil {
		t.Fatalf("re-fetch failed: %v", err)
	}
	if refetched.MinAge != nil {
		t.Errorf("MinAge should be nil after re-fetch, got %d", *refetched.MinAge)
	}
	if refetched.MaxAge != nil {
		t.Errorf("MaxAge should be nil after re-fetch, got %d", *refetched.MaxAge)
	}
}

func TestGovernmentFundingService_UpdatePeriod_ClearNullableTo(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := createTestGovernmentFunding(t, db, "Test Funding")

	// Create period with To set
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &to, 39.0)

	// Verify To is set
	found, _ := svc.GetPeriod(ctx, period.ID, funding.ID)
	if found.To == nil {
		t.Fatal("setup: To should be set")
	}

	// Clear To by sending nil (simulates frontend sending null to make open-ended)
	req := &models.GovernmentFundingPeriodUpdateRequest{
		To: nil,
	}
	updated, err := svc.UpdatePeriod(ctx, period.ID, funding.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.To != nil {
		t.Errorf("To should be nil after clearing, got %v", updated.To)
	}

	// Verify persistence by re-fetching
	refetched, err := svc.GetPeriod(ctx, period.ID, funding.ID)
	if err != nil {
		t.Fatalf("re-fetch failed: %v", err)
	}
	if refetched.To != nil {
		t.Errorf("To should be nil after re-fetch, got %v", refetched.To)
	}
}

// Period overlap validation tests

func TestPeriodsOverlap(t *testing.T) {
	tests := []struct {
		name     string
		from1    string
		to1      *string
		from2    string
		to2      *string
		expected bool
	}{
		{
			name:     "no overlap: period1 before period2",
			from1:    "2024-01-01",
			to1:      strPtr("2024-06-30"),
			from2:    "2024-07-01",
			to2:      strPtr("2024-12-31"),
			expected: false,
		},
		{
			name:     "no overlap: period2 before period1",
			from1:    "2024-07-01",
			to1:      strPtr("2024-12-31"),
			from2:    "2024-01-01",
			to2:      strPtr("2024-06-30"),
			expected: false,
		},
		{
			name:     "overlap: period1 contains period2",
			from1:    "2024-01-01",
			to1:      strPtr("2024-12-31"),
			from2:    "2024-03-01",
			to2:      strPtr("2024-06-30"),
			expected: true,
		},
		{
			name:     "overlap: partial at start",
			from1:    "2024-01-01",
			to1:      strPtr("2024-06-30"),
			from2:    "2024-05-01",
			to2:      strPtr("2024-12-31"),
			expected: true,
		},
		{
			name:     "overlap: period1 ongoing overlaps period2",
			from1:    "2024-01-01",
			to1:      nil,
			from2:    "2024-06-01",
			to2:      strPtr("2024-12-31"),
			expected: true,
		},
		{
			name:     "no overlap: period2 ends before period1 ongoing starts",
			from1:    "2024-07-01",
			to1:      nil,
			from2:    "2024-01-01",
			to2:      strPtr("2024-06-30"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from1, _ := time.Parse("2006-01-02", tt.from1)
			var to1 *time.Time
			if tt.to1 != nil {
				parsed, _ := time.Parse("2006-01-02", *tt.to1)
				to1 = &parsed
			}

			from2, _ := time.Parse("2006-01-02", tt.from2)
			var to2 *time.Time
			if tt.to2 != nil {
				parsed, _ := time.Parse("2006-01-02", *tt.to2)
				to2 = &parsed
			}

			result := periodsOverlap(from1, to1, from2, to2)
			if result != tt.expected {
				t.Errorf("periodsOverlap() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGovernmentFundingService_GetProperty(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period)

	minAge := 0
	maxAge := 3
	prop := &models.GovernmentFundingProperty{
		PeriodID:    period.ID,
		Key:         "care_type",
		Value:       "ganztag",
		Label:       "Ganztag",
		Payment:     100000,
		Requirement: 0.1,
		MinAge:      &minAge,
		MaxAge:      &maxAge,
	}
	db.Create(prop)

	result, err := svc.GetProperty(ctx, prop.ID, period.ID, funding.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ID != prop.ID {
		t.Errorf("ID = %d, want %d", result.ID, prop.ID)
	}
	if result.Key != "care_type" {
		t.Errorf("Key = %q, want %q", result.Key, "care_type")
	}
	if result.Payment != 100000 {
		t.Errorf("Payment = %d, want 100000", result.Payment)
	}
}

func TestGovernmentFundingService_GetProperty_WrongPeriod(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	period1 := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: timePtr(time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC))},
	}
	db.Create(period1)

	period2 := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period2)

	prop := &models.GovernmentFundingProperty{
		PeriodID: period1.ID,
		Key:      "care_type",
		Value:    "ganztag",
		Payment:  100000,
	}
	db.Create(prop)

	// Try to get the property via the wrong period.
	_, err := svc.GetProperty(ctx, prop.ID, period2.ID, funding.ID)
	if err == nil {
		t.Fatal("expected error for wrong period, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGovernmentFundingService_GetProperty_WrongFunding(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding1 := &models.GovernmentFunding{Name: "Funding 1", State: "berlin"}
	db.Create(funding1)
	funding2 := &models.GovernmentFunding{Name: "Funding 2", State: "berlin"}
	db.Create(funding2)

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding1.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period)

	prop := &models.GovernmentFundingProperty{
		PeriodID: period.ID,
		Key:      "care_type",
		Value:    "ganztag",
		Payment:  100000,
	}
	db.Create(prop)

	// Try to get the property via the wrong funding.
	_, err := svc.GetProperty(ctx, prop.ID, period.ID, funding2.ID)
	if err == nil {
		t.Fatal("expected error for wrong funding, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGovernmentFundingService_GetProperty_NotFound(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	ctx := context.Background()

	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period)

	_, err := svc.GetProperty(ctx, 99999, period.ID, funding.ID)
	if err == nil {
		t.Fatal("expected error for non-existent property, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func strPtr(s string) *string {
	return &s
}
