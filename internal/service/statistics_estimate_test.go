package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// === EstimateChildFunding tests ===

func TestStatisticsService_EstimateChildFunding_Basic(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 166847, 0.261, 0, 6)

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildFundingEstimateRequest{
		Birthdate:  time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC), // age 2
		Date:       &date,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	}

	result, err := svc.EstimateChildFunding(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Age != 2 {
		t.Errorf("expected age 2, got %d", result.Age)
	}
	if result.Funding != 166847 {
		t.Errorf("expected funding 166847, got %d", result.Funding)
	}
	if !almostEqual(result.Requirement, 0.261, 0.001) {
		t.Errorf("expected requirement 0.261, got %f", result.Requirement)
	}
	if len(result.MatchedProperties) != 1 {
		t.Errorf("expected 1 matched property, got %d", len(result.MatchedProperties))
	}
	if len(result.UnmatchedProperties) != 0 {
		t.Errorf("expected 0 unmatched properties, got %d", len(result.UnmatchedProperties))
	}
}

func TestStatisticsService_EstimateChildFunding_DefaultDate(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	// Wide period covering current date
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 100000, 0.1, 0, 99)

	// No date specified — should default to now, snapped to first of month
	req := &models.ChildFundingEstimateRequest{
		Birthdate:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Properties: models.ContractProperties{"care_type": "ganztag"},
	}

	result, err := svc.EstimateChildFunding(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Funding != 100000 {
		t.Errorf("expected funding 100000, got %d", result.Funding)
	}
}

func TestStatisticsService_EstimateChildFunding_NoFundingForState(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "hamburg") // no funding configured for hamburg

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildFundingEstimateRequest{
		Birthdate:  time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC),
		Date:       &date,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	}

	// Should succeed with zero funding (no funding periods)
	result, err := svc.EstimateChildFunding(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Funding != 0 {
		t.Errorf("expected funding 0 (no state funding), got %d", result.Funding)
	}
	// All properties should be unmatched
	if len(result.UnmatchedProperties) != 1 {
		t.Errorf("expected 1 unmatched property, got %d", len(result.UnmatchedProperties))
	}
}

func TestStatisticsService_EstimateChildFunding_NoActivePeriodForDate(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	// Period only covers 2025
	toDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 166847, 0.261, 0, 6)

	// Request for 2027 — outside period
	date := time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildFundingEstimateRequest{
		Birthdate:  time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC),
		Date:       &date,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	}

	result, err := svc.EstimateChildFunding(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Funding != 0 {
		t.Errorf("expected 0 funding (no active period), got %d", result.Funding)
	}
	if len(result.UnmatchedProperties) != 1 {
		t.Errorf("expected 1 unmatched property, got %d", len(result.UnmatchedProperties))
	}
}

func TestStatisticsService_EstimateChildFunding_UnmatchedProperty(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	// Funding only matches "ganztag", not "halbtag"
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 166847, 0.261, 0, 6)

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildFundingEstimateRequest{
		Birthdate:  time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC),
		Date:       &date,
		Properties: models.ContractProperties{"care_type": "halbtag"},
	}

	result, err := svc.EstimateChildFunding(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Funding != 0 {
		t.Errorf("expected 0 funding (unmatched care_type), got %d", result.Funding)
	}
	if len(result.UnmatchedProperties) != 1 {
		t.Errorf("expected 1 unmatched property, got %d", len(result.UnmatchedProperties))
	}
	if len(result.MatchedProperties) != 0 {
		t.Errorf("expected 0 matched properties, got %d", len(result.MatchedProperties))
	}
}

func TestStatisticsService_EstimateChildFunding_MultipleProperties(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 166847, 0.261, 0, 6)
	createTestFundingPropertyFull(t, db, period.ID, "supplement", "integration", "Integration", 50000, 0.1, 0, 6)

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildFundingEstimateRequest{
		Birthdate: time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC), // age 2
		Date:      &date,
		Properties: models.ContractProperties{
			"care_type":  "ganztag",
			"supplement": "integration",
		},
	}

	result, err := svc.EstimateChildFunding(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Funding != 166847+50000 {
		t.Errorf("expected funding %d, got %d", 166847+50000, result.Funding)
	}
	if len(result.MatchedProperties) != 2 {
		t.Errorf("expected 2 matched properties, got %d", len(result.MatchedProperties))
	}
}

func TestStatisticsService_EstimateChildFunding_PartialPropertyMatch(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 166847, 0.261, 0, 6)
	// No "supplement" funding configured

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildFundingEstimateRequest{
		Birthdate: time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC),
		Date:      &date,
		Properties: models.ContractProperties{
			"care_type":  "ganztag",
			"supplement": "integration", // no funding for this
		},
	}

	result, err := svc.EstimateChildFunding(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Only care_type matched
	if result.Funding != 166847 {
		t.Errorf("expected funding 166847, got %d", result.Funding)
	}
	if len(result.MatchedProperties) != 1 {
		t.Errorf("expected 1 matched property, got %d", len(result.MatchedProperties))
	}
	if len(result.UnmatchedProperties) != 1 {
		t.Errorf("expected 1 unmatched property, got %d", len(result.UnmatchedProperties))
	}
}

func TestStatisticsService_EstimateChildFunding_AgeBoundary(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	// Under-3 rate: higher payment
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag U3", 200000, 0.4, 0, 2)
	// Over-3 rate: lower payment
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag Ü3", 150000, 0.2, 3, 6)

	props := models.ContractProperties{"care_type": "ganztag"}

	// Child is 2 on 2026-04-01 (born 2023-05-15)
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildFundingEstimateRequest{
		Birthdate:  time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC),
		Date:       &date,
		Properties: props,
	}
	result, err := svc.EstimateChildFunding(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Age != 2 {
		t.Errorf("expected age 2, got %d", result.Age)
	}
	if result.Funding != 200000 {
		t.Errorf("expected under-3 funding 200000, got %d", result.Funding)
	}

	// Same child turns 3 by 2026-06-01 (born 2023-05-15, birthday passed)
	date2 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	req2 := &models.ChildFundingEstimateRequest{
		Birthdate:  time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC),
		Date:       &date2,
		Properties: props,
	}
	result2, err := svc.EstimateChildFunding(ctx, org.ID, req2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result2.Age != 3 {
		t.Errorf("expected age 3, got %d", result2.Age)
	}
	if result2.Funding != 150000 {
		t.Errorf("expected over-3 funding 150000, got %d", result2.Funding)
	}
}

func TestStatisticsService_EstimateChildFunding_ChildOutsideAgeRange(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	// Only covers ages 0-6
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 166847, 0.261, 0, 6)

	// Child is 8 years old
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildFundingEstimateRequest{
		Birthdate:  time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC), // age 8
		Date:       &date,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	}

	result, err := svc.EstimateChildFunding(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Age != 8 {
		t.Errorf("expected age 8, got %d", result.Age)
	}
	if result.Funding != 0 {
		t.Errorf("expected 0 funding (age outside range), got %d", result.Funding)
	}
}

func TestStatisticsService_EstimateChildFunding_InvalidOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildFundingEstimateRequest{
		Birthdate:  time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC),
		Date:       &date,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	}

	_, err := svc.EstimateChildFunding(ctx, 99999, req)
	if err == nil {
		t.Fatal("expected error for non-existent org")
	}
}

func TestStatisticsService_EstimateChildFunding_DateSnappedToFirstOfMonth(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 100000, 0.1, 0, 99)

	// Pass a date in the middle of month — should be snapped to 1st
	// Child born 2024-04-20, date=2026-04-15 => age calculated on 2026-04-01 = 1 (birthday not yet)
	date := time.Date(2026, 4, 15, 12, 30, 0, 0, time.UTC)
	req := &models.ChildFundingEstimateRequest{
		Birthdate:  time.Date(2024, 4, 20, 0, 0, 0, 0, time.UTC),
		Date:       &date,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	}

	result, err := svc.EstimateChildFunding(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// On 2026-04-01, child born 2024-04-20 is still 1 (birthday hasn't happened yet)
	if result.Age != 1 {
		t.Errorf("expected age 1 (date snapped to first of month), got %d", result.Age)
	}
}

func TestStatisticsService_EstimateChildFunding_EmptyProperties(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 166847, 0.261, 0, 6)

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ChildFundingEstimateRequest{
		Birthdate:  time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC),
		Date:       &date,
		Properties: models.ContractProperties{}, // empty
	}

	result, err := svc.EstimateChildFunding(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Funding != 0 {
		t.Errorf("expected 0 funding (no properties to match), got %d", result.Funding)
	}
}

// === EstimateEmployeeCost tests ===

func TestStatisticsService_EstimateEmployeeCost_Basic(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	toDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.4, 2200)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 380000, nil) // €3,800.00

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeCostEstimateRequest{
		PayPlanID:     payplan.ID,
		Grade:         "S8a",
		Step:          3,
		WeeklyHours:   39.4, // full-time
		Date:          &date,
		StaffCategory: "qualified",
	}

	result, err := svc.EstimateEmployeeCost(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.GrossSalary != 380000 {
		t.Errorf("expected gross salary 380000, got %d", result.GrossSalary)
	}
	// employer costs: 380000 * 2200 / 10000 = 83600
	if result.EmployerCosts != 83600 {
		t.Errorf("expected employer costs 83600, got %d", result.EmployerCosts)
	}
	if result.TotalMonthlyCost != 380000+83600 {
		t.Errorf("expected total %d, got %d", 380000+83600, result.TotalMonthlyCost)
	}
	if result.Grade != "S8a" {
		t.Errorf("expected grade S8a, got %s", result.Grade)
	}
	if result.Step != 3 {
		t.Errorf("expected step 3, got %d", result.Step)
	}
	if result.FullTimeMonthlyAmount != 380000 {
		t.Errorf("expected full-time amount 380000, got %d", result.FullTimeMonthlyAmount)
	}
	if result.PayPlanWeeklyHours != 39.4 {
		t.Errorf("expected pay plan weekly hours 39.4, got %f", result.PayPlanWeeklyHours)
	}
	if result.EmployerContributionRate != 2200 {
		t.Errorf("expected employer contribution rate 2200, got %d", result.EmployerContributionRate)
	}
	if result.StaffCategory != "qualified" {
		t.Errorf("expected staff category 'qualified', got %s", result.StaffCategory)
	}
}

func TestStatisticsService_EstimateEmployeeCost_PartTime(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	toDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.4, 2200)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 380000, nil)

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeCostEstimateRequest{
		PayPlanID:   payplan.ID,
		Grade:       "S8a",
		Step:        3,
		WeeklyHours: 19.7, // half-time
		Date:        &date,
	}

	result, err := svc.EstimateEmployeeCost(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// gross = 380000 * 19.7 / 39.4 = 190000
	if result.GrossSalary != 190000 {
		t.Errorf("expected gross salary 190000, got %d", result.GrossSalary)
	}
	// employer costs: 190000 * 2200 / 10000 = 41800
	if result.EmployerCosts != 41800 {
		t.Errorf("expected employer costs 41800, got %d", result.EmployerCosts)
	}
	if result.TotalMonthlyCost != 190000+41800 {
		t.Errorf("expected total %d, got %d", 190000+41800, result.TotalMonthlyCost)
	}
}

func TestStatisticsService_EstimateEmployeeCost_NonExactHoursRounding(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	toDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.4, 2200)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 380000, nil)

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeCostEstimateRequest{
		PayPlanID:   payplan.ID,
		Grade:       "S8a",
		Step:        3,
		WeeklyHours: 30.0, // non-exact division: 380000 * 30 / 39.4 = 289340.10...
		Date:        &date,
	}

	result, err := svc.EstimateEmployeeCost(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// gross = round(380000 * 30.0 / 39.4) = round(289340.101...) = 289340
	expectedGross := int(math.Round(float64(380000) * 30.0 / 39.4))
	if result.GrossSalary != expectedGross {
		t.Errorf("expected gross salary %d, got %d", expectedGross, result.GrossSalary)
	}
	// employer costs = round(289340 * 2200 / 10000) = round(63654.8) = 63655
	expectedContrib := int(math.Round(float64(expectedGross) * 2200.0 / 10000.0))
	if result.EmployerCosts != expectedContrib {
		t.Errorf("expected employer costs %d, got %d", expectedContrib, result.EmployerCosts)
	}
}

func TestStatisticsService_EstimateEmployeeCost_ZeroContributionRate(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	toDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.4, 0) // 0% employer contribution
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 380000, nil)

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeCostEstimateRequest{
		PayPlanID:   payplan.ID,
		Grade:       "S8a",
		Step:        3,
		WeeklyHours: 39.4,
		Date:        &date,
	}

	result, err := svc.EstimateEmployeeCost(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.EmployerCosts != 0 {
		t.Errorf("expected 0 employer costs, got %d", result.EmployerCosts)
	}
	if result.TotalMonthlyCost != 380000 {
		t.Errorf("expected total 380000, got %d", result.TotalMonthlyCost)
	}
}

func TestStatisticsService_EstimateEmployeeCost_PayPlanNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeCostEstimateRequest{
		PayPlanID:   99999, // doesn't exist
		Grade:       "S8a",
		Step:        3,
		WeeklyHours: 39.4,
		Date:        &date,
	}

	_, err := svc.EstimateEmployeeCost(ctx, org.ID, req)
	if err == nil {
		t.Fatal("expected error for non-existent pay plan")
	}
}

func TestStatisticsService_EstimateEmployeeCost_PayPlanWrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Pay plan belongs to org2
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org2.ID)
	toDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.4, 2200)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 380000, nil)

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeCostEstimateRequest{
		PayPlanID:   payplan.ID,
		Grade:       "S8a",
		Step:        3,
		WeeklyHours: 39.4,
		Date:        &date,
	}

	// Request using org1 should fail because pay plan belongs to org2
	_, err := svc.EstimateEmployeeCost(ctx, org1.ID, req)
	if err == nil {
		t.Fatal("expected error when pay plan belongs to different org")
	}
}

func TestStatisticsService_EstimateEmployeeCost_NoActivePeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	// Period only covers 2025
	toDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.4, 2200)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 380000, nil)

	// Request for 2027 — no active period
	date := time.Date(2027, 4, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeCostEstimateRequest{
		PayPlanID:   payplan.ID,
		Grade:       "S8a",
		Step:        3,
		WeeklyHours: 39.4,
		Date:        &date,
	}

	_, err := svc.EstimateEmployeeCost(ctx, org.ID, req)
	if err == nil {
		t.Fatal("expected error for no active pay plan period")
	}
}

func TestStatisticsService_EstimateEmployeeCost_GradeStepNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	toDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.4, 2200)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 380000, nil) // only S8a step 3

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	// Wrong grade
	req := &models.EmployeeCostEstimateRequest{
		PayPlanID:   payplan.ID,
		Grade:       "S11b",
		Step:        3,
		WeeklyHours: 39.4,
		Date:        &date,
	}
	_, err := svc.EstimateEmployeeCost(ctx, org.ID, req)
	if err == nil {
		t.Fatal("expected error for non-existent grade")
	}

	// Right grade, wrong step
	req2 := &models.EmployeeCostEstimateRequest{
		PayPlanID:   payplan.ID,
		Grade:       "S8a",
		Step:        6,
		WeeklyHours: 39.4,
		Date:        &date,
	}
	_, err = svc.EstimateEmployeeCost(ctx, org.ID, req2)
	if err == nil {
		t.Fatal("expected error for non-existent step")
	}
}

func TestStatisticsService_EstimateEmployeeCost_DefaultDate(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	// Open-ended period covering now
	period := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.4, 2200)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 380000, nil)

	// No date — should default to now, snapped to first of month
	req := &models.EmployeeCostEstimateRequest{
		PayPlanID:   payplan.ID,
		Grade:       "S8a",
		Step:        3,
		WeeklyHours: 39.4,
	}

	result, err := svc.EstimateEmployeeCost(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.GrossSalary != 380000 {
		t.Errorf("expected gross salary 380000, got %d", result.GrossSalary)
	}
	// Verify date is first of current month
	now := time.Now()
	expectedDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	if result.Date != expectedDate {
		t.Errorf("expected date %s, got %s", expectedDate, result.Date)
	}
}

func TestStatisticsService_EstimateEmployeeCost_MultipleGradesSamePayPlan(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	toDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.4, 2200)
	createTestPayPlanEntry(t, db, period.ID, "S3", 1, 280000, nil)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 380000, nil)
	createTestPayPlanEntry(t, db, period.ID, "S11b", 5, 450000, nil)

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	// S3 step 1
	req := &models.EmployeeCostEstimateRequest{
		PayPlanID: payplan.ID, Grade: "S3", Step: 1, WeeklyHours: 39.4, Date: &date,
	}
	r1, err := svc.EstimateEmployeeCost(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("S3 error: %v", err)
	}
	if r1.GrossSalary != 280000 {
		t.Errorf("S3 gross: expected 280000, got %d", r1.GrossSalary)
	}

	// S11b step 5
	req2 := &models.EmployeeCostEstimateRequest{
		PayPlanID: payplan.ID, Grade: "S11b", Step: 5, WeeklyHours: 39.4, Date: &date,
	}
	r2, err := svc.EstimateEmployeeCost(ctx, org.ID, req2)
	if err != nil {
		t.Fatalf("S11b error: %v", err)
	}
	if r2.GrossSalary != 450000 {
		t.Errorf("S11b gross: expected 450000, got %d", r2.GrossSalary)
	}
}

func TestStatisticsService_EstimateEmployeeCost_InvalidOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	toDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.4, 2200)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 380000, nil)

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeCostEstimateRequest{
		PayPlanID:   payplan.ID,
		Grade:       "S8a",
		Step:        3,
		WeeklyHours: 39.4,
		Date:        &date,
	}

	// Non-existent org — pay plan won't match
	_, err := svc.EstimateEmployeeCost(ctx, 99999, req)
	if err == nil {
		t.Fatal("expected error for non-existent org")
	}
}
