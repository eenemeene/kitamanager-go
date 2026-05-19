package service

import (
	"context"
	"errors"
	"math"
	"strings"
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
	plans, total, err := svc.List(ctx, org.ID, "", 2, 0)
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
	plans, _, err = svc.List(ctx, org.ID, "", 2, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(plans) != 1 {
		t.Errorf("expected 1 plan on second page, got %d", len(plans))
	}
}

// TestPayPlanService_List_PopulatesPeriodsCount is the regression
// for the "Pay Plans list shows Periods: 0 even though the detail
// view shows many" bug. Pre-fix the list DTO (PayPlanResponse) had
// no count field; the frontend rendered a hard-coded 0 for every
// row. Post-fix PayPlanService.List runs a GROUP BY query on
// pay_plan_periods and populates PayPlanResponse.PeriodsCount.
//
// This test also guards the zero-period edge case (a freshly
// created pay plan with no periods at all): the count must be 0,
// not "missing".
func TestPayPlanService_List_PopulatesPeriodsCount(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	// Plan A: 3 periods. The exact From/To values don't matter for
	// the count — they just need to be non-overlapping so the
	// constraint added in migration 000022 doesn't reject the insert.
	planA := createTestPayPlan(t, db, "Plan A", org.ID)
	createTestPayPlanPeriod(t, db, planA.ID, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), timePtr(time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)), 39.0)
	createTestPayPlanPeriod(t, db, planA.ID, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), timePtr(time.Date(2021, 12, 31, 0, 0, 0, 0, time.UTC)), 39.0)
	createTestPayPlanPeriod(t, db, planA.ID, time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)

	// Plan B: 1 period.
	planB := createTestPayPlan(t, db, "Plan B", org.ID)
	createTestPayPlanPeriod(t, db, planB.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 40.0)

	// Plan C: 0 periods — the freshly-created case the original
	// bug masked behind a 0 default.
	createTestPayPlan(t, db, "Plan C", org.ID)

	plans, total, err := svc.List(ctx, org.ID, "", 100, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}

	got := map[string]int{}
	for _, p := range plans {
		got[p.Name] = p.PeriodsCount
	}
	want := map[string]int{
		"Plan A": 3,
		"Plan B": 1,
		"Plan C": 0,
	}
	for name, expected := range want {
		if got[name] != expected {
			t.Errorf("PeriodsCount[%q] = %d, want %d (full map: %+v)", name, got[name], expected, got)
		}
	}
}

func TestPayPlanService_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	plans, total, err := svc.List(ctx, org.ID, "", 10, 0)
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

func TestPayPlanService_Update_DuplicateNameReturnsConflict(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	createTestPayPlan(t, db, "Plan A", org.ID)
	planB := createTestPayPlan(t, db, "Plan B", org.ID)

	collidingName := "Plan A"
	req := models.PayPlanUpdateRequest{Name: &collidingName}
	_, err := svc.Update(ctx, planB.ID, org.ID, &req)
	if err == nil {
		t.Fatal("expected conflict on rename collision, got nil")
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
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

func TestPayPlanService_Delete_BlockedByEmployeeContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, &to, 39.0)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)

	emp := createTestEmployee(t, db, "Anna", "Schmidt", org.ID)
	createTestEmployeeContract(t, db, emp.ID, payplan.ID, from, &to, "S8a", 3, 39.0)

	err := svc.Delete(ctx, payplan.ID, org.ID)
	if err == nil {
		t.Fatal("expected delete to fail because contract still references pay plan")
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}

	// Periods and entries must NOT have been pre-deleted before the failed pay-plan delete.
	resp, err := svc.GetByID(ctx, payplan.ID, org.ID, nil)
	if err != nil {
		t.Fatalf("pay plan should still exist after blocked delete: %v", err)
	}
	if len(resp.Periods) != 1 {
		t.Errorf("period count after blocked delete = %d, want 1 (atomicity check)", len(resp.Periods))
	} else if len(resp.Periods[0].Entries) != 1 {
		t.Errorf("entry count after blocked delete = %d, want 1 (atomicity check)", len(resp.Periods[0].Entries))
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

func TestPayPlanService_CreatePeriod_FieldValidation(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()
	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	jan1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	dec31 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name        string
		req         models.PayPlanPeriodCreateRequest
		wantMessage string
	}{
		{
			name: "from after to",
			req: models.PayPlanPeriodCreateRequest{
				From: dec31, To: &jan1, WeeklyHours: 39.0, EmployerContributionRate: 2200,
			},
			wantMessage: "to must be on or after from",
		},
		{
			name: "weekly_hours zero",
			req: models.PayPlanPeriodCreateRequest{
				From: jan1, To: &dec31, WeeklyHours: 0, EmployerContributionRate: 2200,
			},
			wantMessage: "weekly_hours must be > 0",
		},
		{
			name: "weekly_hours negative",
			req: models.PayPlanPeriodCreateRequest{
				From: jan1, To: &dec31, WeeklyHours: -5, EmployerContributionRate: 2200,
			},
			wantMessage: "weekly_hours must be > 0",
		},
		{
			name: "weekly_hours over max",
			req: models.PayPlanPeriodCreateRequest{
				From: jan1, To: &dec31, WeeklyHours: 200, EmployerContributionRate: 2200,
			},
			wantMessage: "weekly_hours cannot exceed",
		},
		{
			name: "employer_rate negative",
			req: models.PayPlanPeriodCreateRequest{
				From: jan1, To: &dec31, WeeklyHours: 39, EmployerContributionRate: -1,
			},
			wantMessage: "employer_contribution_rate must be in",
		},
		{
			name: "employer_rate too large",
			req: models.PayPlanPeriodCreateRequest{
				From: jan1, To: &dec31, WeeklyHours: 39, EmployerContributionRate: 10001,
			},
			wantMessage: "employer_contribution_rate must be in",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreatePeriod(ctx, payplan.ID, org.ID, &tc.req)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !errors.Is(err, apperror.ErrBadRequest) {
				t.Fatalf("expected ErrBadRequest, got %v", err)
			}
			var appErr *apperror.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("expected AppError, got %T", err)
			}
			if !strings.Contains(appErr.Message, tc.wantMessage) {
				t.Errorf("message %q does not contain %q", appErr.Message, tc.wantMessage)
			}
		})
	}
}

func TestPayPlanService_CreatePeriod_FromEqualsToAllowed(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()
	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	day := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	req := models.PayPlanPeriodCreateRequest{
		From: day, To: &day, WeeklyHours: 39, EmployerContributionRate: 0,
	}
	if _, err := svc.CreatePeriod(ctx, payplan.ID, org.ID, &req); err != nil {
		t.Fatalf("from==to should be a valid (single-day) period, got %v", err)
	}
}

func TestPayPlanService_UpdatePeriod_FromAfterToRejected(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()
	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, &to, 39.0)

	bad := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	req := models.PayPlanPeriodUpdateRequest{
		From: to, To: &bad, WeeklyHours: 39, EmployerContributionRate: 2200,
	}
	_, err := svc.UpdatePeriod(ctx, period.ID, payplan.ID, org.ID, &req)
	if err == nil {
		t.Fatal("expected from>to to be rejected on update")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
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

func TestPayPlanService_CreateEntry_DuplicateGradeStepReturnsConflict(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)

	req := models.PayPlanEntryCreateRequest{Grade: "S8a", Step: 3, MonthlyAmount: 400000}
	if _, err := svc.CreateEntry(ctx, period.ID, payplan.ID, org.ID, &req); err != nil {
		t.Fatalf("first create should succeed, got %v", err)
	}

	_, err := svc.CreateEntry(ctx, period.ID, payplan.ID, org.ID, &req)
	if err == nil {
		t.Fatal("expected duplicate (grade, step) to be rejected")
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestPayPlanService_CreateEntry_FieldValidation(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()
	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)

	cases := []struct {
		name string
		req  models.PayPlanEntryCreateRequest
	}{
		{"empty grade", models.PayPlanEntryCreateRequest{Grade: "  ", Step: 1, MonthlyAmount: 100000}},
		{"step zero", models.PayPlanEntryCreateRequest{Grade: "S8a", Step: 0, MonthlyAmount: 100000}},
		{"negative monthly_amount", models.PayPlanEntryCreateRequest{Grade: "S8a", Step: 1, MonthlyAmount: -1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreateEntry(ctx, period.ID, payplan.ID, org.ID, &tc.req)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !errors.Is(err, apperror.ErrBadRequest) {
				t.Errorf("expected ErrBadRequest, got %v", err)
			}
		})
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

func TestPayPlanService_UpdateEntry_DuplicateGradeStepReturnsConflict(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriod(t, db, payplan.ID, from, nil, 39.0)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)
	target := createTestPayPlanEntry(t, db, period.ID, "S8a", 4, 420000, nil)

	// Try to update target to collide with the (S8a, 3) entry.
	req := models.PayPlanEntryUpdateRequest{
		Grade: "S8a", Step: 3, MonthlyAmount: 999000,
	}
	_, err := svc.UpdateEntry(ctx, target.ID, period.ID, payplan.ID, org.ID, &req)
	if err == nil {
		t.Fatal("expected duplicate-key conflict on update")
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
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

// Import tests

func TestPayPlanService_Import_CreatesNew(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	stepMin := 2
	data := &models.PayPlanDetailResponse{
		Name: "TVöD-SuE",
		Periods: []models.PayPlanPeriodResponse{
			{
				From:                     from,
				To:                       &to,
				WeeklyHours:              39.0,
				EmployerContributionRate: 2100,
				Entries: []models.PayPlanEntryResponse{
					{Grade: "S8a", Step: 1, MonthlyAmount: 300000},
					{Grade: "S8a", Step: 2, MonthlyAmount: 350000, StepMinYears: &stepMin},
				},
			},
		},
	}

	resp, err := svc.Import(ctx, org.ID, data)
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
	if len(resp.Periods) != 1 {
		t.Fatalf("expected 1 period, got %d", len(resp.Periods))
	}
	if resp.Periods[0].EmployerContributionRate != 2100 {
		t.Errorf("EmployerContributionRate = %d, want 2100", resp.Periods[0].EmployerContributionRate)
	}
	if len(resp.Periods[0].Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(resp.Periods[0].Entries))
	}
}

func TestPayPlanService_Import_UpsertPreservesID(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	// First import: create the pay plan.
	from1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data1 := &models.PayPlanDetailResponse{
		Name: "TV eene meene",
		Periods: []models.PayPlanPeriodResponse{
			{
				From:        from1,
				WeeklyHours: 39.0,
				Entries: []models.PayPlanEntryResponse{
					{Grade: "5", Step: 1, MonthlyAmount: 198900},
				},
			},
		},
	}
	resp1, err := svc.Import(ctx, org.ID, data1)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	originalID := resp1.ID

	// Second import: same name, different data → should upsert.
	from2 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	data2 := &models.PayPlanDetailResponse{
		Name: "TV eene meene",
		Periods: []models.PayPlanPeriodResponse{
			{
				From:                     from2,
				WeeklyHours:              39.0,
				EmployerContributionRate: 2120,
				Entries: []models.PayPlanEntryResponse{
					{Grade: "5", Step: 1, MonthlyAmount: 210000},
					{Grade: "5", Step: 2, MonthlyAmount: 230000},
				},
			},
		},
	}
	resp2, err := svc.Import(ctx, org.ID, data2)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}

	// Pay plan ID must be preserved.
	if resp2.ID != originalID {
		t.Errorf("ID = %d, want %d (original)", resp2.ID, originalID)
	}
	if resp2.Name != "TV eene meene" {
		t.Errorf("Name = %v, want TV eene meene", resp2.Name)
	}
	// New data should be present.
	if len(resp2.Periods) != 1 {
		t.Fatalf("expected 1 period, got %d", len(resp2.Periods))
	}
	if resp2.Periods[0].EmployerContributionRate != 2120 {
		t.Errorf("EmployerContributionRate = %d, want 2120", resp2.Periods[0].EmployerContributionRate)
	}
	if len(resp2.Periods[0].Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(resp2.Periods[0].Entries))
	}
	if resp2.Periods[0].Entries[0].MonthlyAmount != 210000 {
		t.Errorf("first entry MonthlyAmount = %d, want 210000", resp2.Periods[0].Entries[0].MonthlyAmount)
	}
}

func TestPayPlanService_Import_UpsertReplacesOldPeriods(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	// First import: 2 periods with entries.
	from1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	from2 := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	data1 := &models.PayPlanDetailResponse{
		Name: "Test Plan",
		Periods: []models.PayPlanPeriodResponse{
			{
				From: from1, To: &to1, WeeklyHours: 39.0,
				Entries: []models.PayPlanEntryResponse{
					{Grade: "S8a", Step: 1, MonthlyAmount: 300000},
					{Grade: "S8a", Step: 2, MonthlyAmount: 320000},
					{Grade: "S8a", Step: 3, MonthlyAmount: 340000},
				},
			},
			{
				From: from2, WeeklyHours: 39.0,
				Entries: []models.PayPlanEntryResponse{
					{Grade: "S8a", Step: 1, MonthlyAmount: 310000},
				},
			},
		},
	}
	resp1, err := svc.Import(ctx, org.ID, data1)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if len(resp1.Periods) != 2 {
		t.Fatalf("expected 2 periods after first import, got %d", len(resp1.Periods))
	}

	// Second import: 1 period with 1 entry → old data must be fully replaced.
	from3 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	data2 := &models.PayPlanDetailResponse{
		Name: "Test Plan",
		Periods: []models.PayPlanPeriodResponse{
			{
				From: from3, WeeklyHours: 40.0,
				Entries: []models.PayPlanEntryResponse{
					{Grade: "S11b", Step: 1, MonthlyAmount: 500000},
				},
			},
		},
	}
	resp2, err := svc.Import(ctx, org.ID, data2)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}

	if len(resp2.Periods) != 1 {
		t.Fatalf("expected 1 period after upsert, got %d", len(resp2.Periods))
	}
	if resp2.Periods[0].WeeklyHours != 40.0 {
		t.Errorf("WeeklyHours = %f, want 40.0", resp2.Periods[0].WeeklyHours)
	}
	if len(resp2.Periods[0].Entries) != 1 {
		t.Fatalf("expected 1 entry after upsert, got %d", len(resp2.Periods[0].Entries))
	}
	if resp2.Periods[0].Entries[0].Grade != "S11b" {
		t.Errorf("Grade = %s, want S11b", resp2.Periods[0].Entries[0].Grade)
	}

	// Verify no stale data remains: list all pay plans, should be exactly 1.
	plans, total, err := svc.List(ctx, org.ID, "", 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 pay plan total, got %d", total)
	}
	if len(plans) != 1 {
		t.Errorf("expected 1 pay plan, got %d", len(plans))
	}
}

func TestPayPlanService_Import_DifferentOrgsSameName(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data := &models.PayPlanDetailResponse{
		Name: "Same Name Plan",
		Periods: []models.PayPlanPeriodResponse{
			{From: from, WeeklyHours: 39.0, Entries: []models.PayPlanEntryResponse{
				{Grade: "S8a", Step: 1, MonthlyAmount: 300000},
			}},
		},
	}

	resp1, err := svc.Import(ctx, org1.ID, data)
	if err != nil {
		t.Fatalf("import org1: %v", err)
	}

	resp2, err := svc.Import(ctx, org2.ID, data)
	if err != nil {
		t.Fatalf("import org2: %v", err)
	}

	// Must be different pay plan records.
	if resp1.ID == resp2.ID {
		t.Error("expected different IDs for pay plans in different orgs")
	}
	if resp1.OrganizationID != org1.ID {
		t.Errorf("org1 plan OrganizationID = %d, want %d", resp1.OrganizationID, org1.ID)
	}
	if resp2.OrganizationID != org2.ID {
		t.Errorf("org2 plan OrganizationID = %d, want %d", resp2.OrganizationID, org2.ID)
	}
}

func TestPayPlanService_Import_EmptyName(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	data := &models.PayPlanDetailResponse{
		Name: "",
	}

	_, err := svc.Import(ctx, org.ID, data)
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestPayPlanService_Import_WhitespaceName(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	data := &models.PayPlanDetailResponse{
		Name: "   ",
	}

	_, err := svc.Import(ctx, org.ID, data)
	if err == nil {
		t.Fatal("expected error for whitespace name, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestPayPlanService_Import_NoPeriods(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	data := &models.PayPlanDetailResponse{
		Name:    "Empty Plan",
		Periods: nil,
	}

	resp, err := svc.Import(ctx, org.ID, data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Name != "Empty Plan" {
		t.Errorf("Name = %v, want Empty Plan", resp.Name)
	}
	if len(resp.Periods) != 0 {
		t.Errorf("expected 0 periods, got %d", len(resp.Periods))
	}
}

func TestPayPlanService_Import_UpsertNoPeriodsClearsExisting(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	// First import: with periods and entries.
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data1 := &models.PayPlanDetailResponse{
		Name: "Wipeable Plan",
		Periods: []models.PayPlanPeriodResponse{
			{From: from, WeeklyHours: 39.0, Entries: []models.PayPlanEntryResponse{
				{Grade: "S8a", Step: 1, MonthlyAmount: 300000},
			}},
		},
	}
	_, err := svc.Import(ctx, org.ID, data1)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}

	// Second import: same name, no periods → should clear everything.
	data2 := &models.PayPlanDetailResponse{
		Name:    "Wipeable Plan",
		Periods: nil,
	}
	resp, err := svc.Import(ctx, org.ID, data2)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}

	if len(resp.Periods) != 0 {
		t.Errorf("expected 0 periods after upsert with empty data, got %d", len(resp.Periods))
	}
}

func TestPayPlanService_Import_MultiplePeriods(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	from1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	from2 := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	to2 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	from3 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	data := &models.PayPlanDetailResponse{
		Name: "Multi Period Plan",
		Periods: []models.PayPlanPeriodResponse{
			{
				From: from1, To: &to1, WeeklyHours: 39.0,
				EmployerContributionRate: 2000,
				Entries: []models.PayPlanEntryResponse{
					{Grade: "S8a", Step: 1, MonthlyAmount: 300000},
				},
			},
			{
				From: from2, To: &to2, WeeklyHours: 39.0,
				EmployerContributionRate: 2050,
				Entries: []models.PayPlanEntryResponse{
					{Grade: "S8a", Step: 1, MonthlyAmount: 310000},
					{Grade: "S8a", Step: 2, MonthlyAmount: 330000},
				},
			},
			{
				From: from3, WeeklyHours: 40.0,
				EmployerContributionRate: 2120,
				Entries: []models.PayPlanEntryResponse{
					{Grade: "S8a", Step: 1, MonthlyAmount: 320000},
				},
			},
		},
	}

	resp, err := svc.Import(ctx, org.ID, data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(resp.Periods) != 3 {
		t.Fatalf("expected 3 periods, got %d", len(resp.Periods))
	}

	// Periods are returned ordered by from_date DESC.
	totalEntries := 0
	for _, p := range resp.Periods {
		totalEntries += len(p.Entries)
	}
	if totalEntries != 4 {
		t.Errorf("expected 4 total entries, got %d", totalEntries)
	}
}

func TestPayPlanService_Import_StepMinYearsPreserved(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	stepMin2 := 2
	stepMin4 := 4
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data := &models.PayPlanDetailResponse{
		Name: "Step Min Plan",
		Periods: []models.PayPlanPeriodResponse{
			{
				From: from, WeeklyHours: 39.0,
				Entries: []models.PayPlanEntryResponse{
					{Grade: "S8a", Step: 1, MonthlyAmount: 300000, StepMinYears: nil},
					{Grade: "S8a", Step: 2, MonthlyAmount: 320000, StepMinYears: &stepMin2},
					{Grade: "S8a", Step: 3, MonthlyAmount: 340000, StepMinYears: &stepMin4},
				},
			},
		},
	}

	resp, err := svc.Import(ctx, org.ID, data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	entries := resp.Periods[0].Entries
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].StepMinYears != nil {
		t.Errorf("entry 0 StepMinYears = %v, want nil", entries[0].StepMinYears)
	}
	if entries[1].StepMinYears == nil || *entries[1].StepMinYears != 2 {
		t.Errorf("entry 1 StepMinYears = %v, want 2", entries[1].StepMinYears)
	}
	if entries[2].StepMinYears == nil || *entries[2].StepMinYears != 4 {
		t.Errorf("entry 2 StepMinYears = %v, want 4", entries[2].StepMinYears)
	}
}

// Import validation tests — YAML deserialisation does not run gin's binding
// tags, so the service layer has to enforce the same rules.

func TestPayPlanService_Import_RejectsBadFields(t *testing.T) {
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	negStepMin := -1

	cases := []struct {
		name        string
		data        models.PayPlanDetailResponse
		wantMessage string
	}{
		{
			name: "weekly_hours zero",
			data: models.PayPlanDetailResponse{
				Name: "P", Periods: []models.PayPlanPeriodResponse{
					{From: from, To: &to, WeeklyHours: 0, EmployerContributionRate: 2200},
				},
			},
			wantMessage: "weekly_hours must be > 0",
		},
		{
			name: "weekly_hours negative",
			data: models.PayPlanDetailResponse{
				Name: "P", Periods: []models.PayPlanPeriodResponse{
					{From: from, To: &to, WeeklyHours: -1, EmployerContributionRate: 2200},
				},
			},
			wantMessage: "weekly_hours must be > 0",
		},
		{
			name: "weekly_hours over 168",
			data: models.PayPlanDetailResponse{
				Name: "P", Periods: []models.PayPlanPeriodResponse{
					{From: from, To: &to, WeeklyHours: 200, EmployerContributionRate: 2200},
				},
			},
			wantMessage: "weekly_hours cannot exceed",
		},
		{
			name: "employer_rate negative",
			data: models.PayPlanDetailResponse{
				Name: "P", Periods: []models.PayPlanPeriodResponse{
					{From: from, To: &to, WeeklyHours: 39, EmployerContributionRate: -1},
				},
			},
			wantMessage: "employer_contribution_rate must be in",
		},
		{
			name: "employer_rate over max",
			data: models.PayPlanDetailResponse{
				Name: "P", Periods: []models.PayPlanPeriodResponse{
					{From: from, To: &to, WeeklyHours: 39, EmployerContributionRate: 99999},
				},
			},
			wantMessage: "employer_contribution_rate must be in",
		},
		{
			name: "from after to",
			data: models.PayPlanDetailResponse{
				Name: "P", Periods: []models.PayPlanPeriodResponse{
					{From: to, To: &from, WeeklyHours: 39, EmployerContributionRate: 2200},
				},
			},
			wantMessage: "to must be on or after from",
		},
		{
			name: "empty grade",
			data: models.PayPlanDetailResponse{
				Name: "P", Periods: []models.PayPlanPeriodResponse{
					{From: from, To: &to, WeeklyHours: 39, EmployerContributionRate: 2200,
						Entries: []models.PayPlanEntryResponse{
							{Grade: "   ", Step: 1, MonthlyAmount: 100000},
						}},
				},
			},
			wantMessage: "grade cannot be empty",
		},
		{
			name: "step zero",
			data: models.PayPlanDetailResponse{
				Name: "P", Periods: []models.PayPlanPeriodResponse{
					{From: from, To: &to, WeeklyHours: 39, EmployerContributionRate: 2200,
						Entries: []models.PayPlanEntryResponse{
							{Grade: "S8a", Step: 0, MonthlyAmount: 100000},
						}},
				},
			},
			wantMessage: "step must be >= 1",
		},
		{
			name: "negative monthly_amount",
			data: models.PayPlanDetailResponse{
				Name: "P", Periods: []models.PayPlanPeriodResponse{
					{From: from, To: &to, WeeklyHours: 39, EmployerContributionRate: 2200,
						Entries: []models.PayPlanEntryResponse{
							{Grade: "S8a", Step: 1, MonthlyAmount: -1},
						}},
				},
			},
			wantMessage: "monthly_amount must be >= 0",
		},
		{
			name: "negative step_min_years",
			data: models.PayPlanDetailResponse{
				Name: "P", Periods: []models.PayPlanPeriodResponse{
					{From: from, To: &to, WeeklyHours: 39, EmployerContributionRate: 2200,
						Entries: []models.PayPlanEntryResponse{
							{Grade: "S8a", Step: 1, MonthlyAmount: 100000, StepMinYears: &negStepMin},
						}},
				},
			},
			wantMessage: "step_min_years must be >= 0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := setupTestDB(t)
			svc := createPayPlanService(db)
			org := createTestOrganization(t, db, "Test Org")
			_, err := svc.Import(context.Background(), org.ID, &tc.data)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !errors.Is(err, apperror.ErrBadRequest) {
				t.Fatalf("expected ErrBadRequest, got %v", err)
			}
			var appErr *apperror.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("expected AppError, got %T", err)
			}
			if !strings.Contains(appErr.Message, tc.wantMessage) {
				t.Errorf("message %q does not contain %q", appErr.Message, tc.wantMessage)
			}

			// Nothing should have been persisted on failure.
			var count int64
			if err := db.Model(&models.PayPlan{}).Where("organization_id = ?", org.ID).Count(&count).Error; err != nil {
				t.Fatalf("count query failed: %v", err)
			}
			if count != 0 {
				t.Errorf("expected 0 pay plans persisted on rejection, got %d", count)
			}
		})
	}
}

func TestPayPlanService_Import_RejectsOverlappingPeriods(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	org := createTestOrganization(t, db, "Test Org")

	period1End := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	period2Start := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC) // overlaps with period1
	period2End := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	data := &models.PayPlanDetailResponse{
		Name: "Overlapping",
		Periods: []models.PayPlanPeriodResponse{
			{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &period1End, WeeklyHours: 39, EmployerContributionRate: 2200},
			{From: period2Start, To: &period2End, WeeklyHours: 39, EmployerContributionRate: 2200},
		},
	}
	_, err := svc.Import(context.Background(), org.ID, data)
	if err == nil {
		t.Fatal("expected overlap rejection")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestPayPlanService_Import_RejectsDuplicateEntriesWithinPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createPayPlanService(db)
	org := createTestOrganization(t, db, "Test Org")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data := &models.PayPlanDetailResponse{
		Name: "Dups",
		Periods: []models.PayPlanPeriodResponse{
			{From: from, WeeklyHours: 39, EmployerContributionRate: 2200,
				Entries: []models.PayPlanEntryResponse{
					{Grade: "S8a", Step: 1, MonthlyAmount: 100000},
					{Grade: " S8a ", Step: 1, MonthlyAmount: 110000}, // duplicate after trim
				}},
		},
	}
	_, err := svc.Import(context.Background(), org.ID, data)
	if err == nil {
		t.Fatal("expected duplicate-entry rejection")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}
