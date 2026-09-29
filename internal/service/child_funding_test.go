package service

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// --- findPeriodForDate ---

func TestFindPeriodForDate_NoPeriods(t *testing.T) {
	date := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	got := findPeriodForDate(nil, date)
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestFindPeriodForDate_BeforeAll(t *testing.T) {
	periods := []models.GovernmentFundingPeriod{
		{Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: timePtr(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC))}},
	}
	got := findPeriodForDate(periods, time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC))
	if got != nil {
		t.Errorf("expected nil for date before all periods, got %+v", got)
	}
}

func TestFindPeriodForDate_AfterAll(t *testing.T) {
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	periods := []models.GovernmentFundingPeriod{
		{Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: &to}},
	}
	got := findPeriodForDate(periods, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if got != nil {
		t.Errorf("expected nil for date after all periods, got %+v", got)
	}
}

func TestFindPeriodForDate_ExactStart(t *testing.T) {
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	periods := []models.GovernmentFundingPeriod{
		{ID: 10, Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: &to}},
	}
	got := findPeriodForDate(periods, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	if got == nil {
		t.Fatal("expected period, got nil")
	}
	if got.ID != 10 {
		t.Errorf("expected period ID 10, got %d", got.ID)
	}
}

func TestFindPeriodForDate_OpenEnded(t *testing.T) {
	periods := []models.GovernmentFundingPeriod{
		{ID: 20, Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: nil}},
	}
	got := findPeriodForDate(periods, time.Date(2030, 6, 15, 0, 0, 0, 0, time.UTC))
	if got == nil {
		t.Fatal("expected open-ended period to match far future date, got nil")
	}
	if got.ID != 20 {
		t.Errorf("expected period ID 20, got %d", got.ID)
	}
}

func TestFindPeriodForDate_SelectsCorrect(t *testing.T) {
	to1 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	to2 := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	periods := []models.GovernmentFundingPeriod{
		{ID: 1, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &to1}},
		{ID: 2, Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: &to2}},
	}
	got := findPeriodForDate(periods, time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC))
	if got == nil {
		t.Fatal("expected period, got nil")
	}
	if got.ID != 2 {
		t.Errorf("expected period ID 2, got %d", got.ID)
	}
}

func TestFindPeriodForDate_ExactEnd(t *testing.T) {
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	periods := []models.GovernmentFundingPeriod{
		{ID: 30, Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: &to}},
	}
	// Period.IsActiveOn is inclusive on To, so exact end date should match
	got := findPeriodForDate(periods, time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC))
	if got == nil {
		t.Fatal("expected period at exact end date (inclusive), got nil")
	}
	if got.ID != 30 {
		t.Errorf("expected period ID 30, got %d", got.ID)
	}
}

func TestFindPeriodForDate_OverlappingPeriodsLogsError(t *testing.T) {
	// Two periods that overlap on 2025-06-15
	to1 := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	to2 := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	periods := []models.GovernmentFundingPeriod{
		{ID: 1, Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: &to1}},
		{ID: 2, Period: models.Period{From: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), To: &to2}},
	}

	// Capture slog output
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(oldLogger)

	got := findPeriodForDate(periods, time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC))
	switch {
	case got == nil:
		t.Fatal("expected a period, got nil")
	case got.ID != 1:
		t.Errorf("expected first matching period (ID 1), got %d", got.ID)
	}

	logOutput := buf.String()
	if logOutput == "" {
		t.Error("expected slog.Error for overlapping periods, got no log output")
	}
	if !bytes.Contains(buf.Bytes(), []byte("overlapping funding periods detected")) {
		t.Errorf("expected log message about overlapping periods, got: %s", logOutput)
	}
}

func TestFindPeriodForDate_NonOverlappingDoesNotLog(t *testing.T) {
	to1 := time.Date(2025, 5, 31, 0, 0, 0, 0, time.UTC)
	periods := []models.GovernmentFundingPeriod{
		{ID: 1, Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: &to1}},
		{ID: 2, Period: models.Period{From: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), To: nil}},
	}

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(oldLogger)

	got := findPeriodForDate(periods, time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC))
	switch {
	case got == nil:
		t.Fatal("expected period, got nil")
	case got.ID != 2:
		t.Errorf("expected period ID 2, got %d", got.ID)
	}

	if buf.Len() > 0 {
		t.Errorf("expected no log output for non-overlapping periods, got: %s", buf.String())
	}
}

// --- matchFundingProperties ---

func TestMatchFundingProperties_NilPeriod(t *testing.T) {
	props := models.ContractProperties{"care_type": "ganztag"}
	got := matchFundingProperties(3, props, nil)
	if got != nil {
		t.Errorf("expected nil for nil period, got %+v", got)
	}
}

func TestMatchFundingProperties_NoAgeMatch(t *testing.T) {
	period := &models.GovernmentFundingPeriod{
		Properties: []models.GovernmentFundingProperty{
			{Key: "care_type", Value: "ganztag", Payment: 10000, MinAge: intPtr(0), MaxAge: intPtr(2)},
		},
	}
	props := models.ContractProperties{"care_type": "ganztag"}
	got := matchFundingProperties(5, props, period)
	if len(got) != 0 {
		t.Errorf("expected no matches for age 5 (max 2), got %d", len(got))
	}
}

func TestMatchFundingProperties_AgeExactMinBoundary(t *testing.T) {
	period := &models.GovernmentFundingPeriod{
		Properties: []models.GovernmentFundingProperty{
			{Key: "care_type", Value: "ganztag", Payment: 10000, MinAge: intPtr(3), MaxAge: intPtr(6)},
		},
	}
	props := models.ContractProperties{"care_type": "ganztag"}
	got := matchFundingProperties(3, props, period)
	if len(got) != 1 {
		t.Errorf("expected 1 match at min boundary age 3, got %d", len(got))
	}
}

func TestMatchFundingProperties_AgeExactMaxBoundary(t *testing.T) {
	period := &models.GovernmentFundingPeriod{
		Properties: []models.GovernmentFundingProperty{
			{Key: "care_type", Value: "ganztag", Payment: 10000, MinAge: intPtr(0), MaxAge: intPtr(3)},
		},
	}
	props := models.ContractProperties{"care_type": "ganztag"}
	got := matchFundingProperties(3, props, period)
	if len(got) != 1 {
		t.Errorf("expected 1 match at max boundary age 3, got %d", len(got))
	}
}

func TestMatchFundingProperties_AgeOneAboveMax(t *testing.T) {
	period := &models.GovernmentFundingPeriod{
		Properties: []models.GovernmentFundingProperty{
			{Key: "care_type", Value: "ganztag", Payment: 10000, MinAge: intPtr(0), MaxAge: intPtr(3)},
		},
	}
	props := models.ContractProperties{"care_type": "ganztag"}
	got := matchFundingProperties(4, props, period)
	if len(got) != 0 {
		t.Errorf("expected 0 matches for age 4 (max 3), got %d", len(got))
	}
}

func TestMatchFundingProperties_NilAgeRange(t *testing.T) {
	period := &models.GovernmentFundingPeriod{
		Properties: []models.GovernmentFundingProperty{
			{Key: "care_type", Value: "ganztag", Payment: 10000, MinAge: nil, MaxAge: nil},
		},
	}
	props := models.ContractProperties{"care_type": "ganztag"}
	got := matchFundingProperties(99, props, period)
	if len(got) != 1 {
		t.Errorf("expected 1 match with nil age range for any age, got %d", len(got))
	}
}

func TestMatchFundingProperties_KeyNotInContract(t *testing.T) {
	period := &models.GovernmentFundingPeriod{
		Properties: []models.GovernmentFundingProperty{
			{Key: "supplements", Value: "ndh", Payment: 5000, MinAge: nil, MaxAge: nil},
		},
	}
	// Contract has care_type but not supplements - should not match
	props := models.ContractProperties{"care_type": "ganztag"}
	got := matchFundingProperties(3, props, period)
	if len(got) != 0 {
		t.Errorf("expected 0 matches when key not in contract, got %d", len(got))
	}
}

func TestMatchFundingProperties_MultipleMatches(t *testing.T) {
	period := &models.GovernmentFundingPeriod{
		Properties: []models.GovernmentFundingProperty{
			{Key: "care_type", Value: "ganztag", Payment: 10000, MinAge: nil, MaxAge: nil},
			{Key: "supplements", Value: "ndh", Payment: 5000, MinAge: nil, MaxAge: nil},
			{Key: "supplements", Value: "mss", Payment: 3000, MinAge: nil, MaxAge: nil},
		},
	}
	props := models.ContractProperties{
		"care_type":   "ganztag",
		"supplements": []string{"ndh", "mss"},
	}
	got := matchFundingProperties(3, props, period)
	if len(got) != 3 {
		t.Errorf("expected 3 matches, got %d", len(got))
	}
}

// TestMatchFundingProperties_DuplicateConfigDedupes asserts the correctness guard
// against a funding configuration that accidentally contains two properties with
// the same (key, value) and overlapping age ranges — without dedup, the child's
// payment would be double-counted.
func TestMatchFundingProperties_DuplicateConfigDedupes(t *testing.T) {
	period := &models.GovernmentFundingPeriod{
		ID: 42,
		Properties: []models.GovernmentFundingProperty{
			{ID: 1, Key: "care_type", Value: "ganztag", Payment: 100000, MinAge: intPtr(0), MaxAge: intPtr(3)},
			{ID: 2, Key: "care_type", Value: "ganztag", Payment: 100000, MinAge: intPtr(3), MaxAge: intPtr(6)},
		},
	}
	props := models.ContractProperties{"care_type": "ganztag"}

	var buf bytes.Buffer
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})))
	defer slog.SetDefault(oldLogger)

	// Age 3 is in both ranges — without dedup we'd match twice.
	got := matchFundingProperties(3, props, period)
	if len(got) != 1 {
		t.Errorf("expected 1 match after dedup, got %d", len(got))
	}
	if got[0].ID != 1 {
		t.Errorf("expected first match (ID 1) to be kept, got ID %d", got[0].ID)
	}
	if !bytes.Contains(buf.Bytes(), []byte("duplicate funding property matched")) {
		t.Errorf("expected duplicate-property log message, got: %s", buf.String())
	}
}

// TestSumChildFundingMatch_DuplicatePaymentNotDoubleCounted verifies the
// callers that sum payments (used by financials + staffing) see the deduped
// result: the bug would have been silent inflation of funding income.
func TestSumChildFundingMatch_DuplicatePaymentNotDoubleCounted(t *testing.T) {
	period := &models.GovernmentFundingPeriod{
		Properties: []models.GovernmentFundingProperty{
			{ID: 1, Key: "care_type", Value: "ganztag", Payment: 100000, Requirement: 0.2, MinAge: nil, MaxAge: nil},
			{ID: 2, Key: "care_type", Value: "ganztag", Payment: 100000, Requirement: 0.2, MinAge: nil, MaxAge: nil},
		},
	}
	props := models.ContractProperties{"care_type": "ganztag"}

	// Silence the expected error log.
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelError})))
	defer slog.SetDefault(oldLogger)

	payment, requirement := sumChildFundingMatch(3, props, period)
	if payment != 100000 {
		t.Errorf("expected payment 100000 (no double count), got %d", payment)
	}
	if requirement != 0.2 {
		t.Errorf("expected requirement 0.2 (no double count), got %f", requirement)
	}
}

// --- calculateChildFunding ---

func TestCalculateChildFunding_NilPeriod(t *testing.T) {
	props := models.ContractProperties{"care_type": "ganztag", "supplements": []string{"ndh"}}
	result := calculateChildFunding(3, props, nil)

	if result.Funding != 0 {
		t.Errorf("expected 0 funding with nil period, got %d", result.Funding)
	}
	if len(result.MatchedProperties) != 0 {
		t.Errorf("expected 0 matched properties, got %d", len(result.MatchedProperties))
	}
	if len(result.UnmatchedProperties) != 2 {
		t.Errorf("expected 2 unmatched properties (care_type + ndh), got %d", len(result.UnmatchedProperties))
	}
}

func TestCalculateChildFunding_AllMatched(t *testing.T) {
	period := &models.GovernmentFundingPeriod{
		Properties: []models.GovernmentFundingProperty{
			{Key: "care_type", Value: "ganztag", Payment: 10000, MinAge: nil, MaxAge: nil},
			{Key: "supplements", Value: "ndh", Payment: 5000, MinAge: nil, MaxAge: nil},
		},
	}
	props := models.ContractProperties{
		"care_type":   "ganztag",
		"supplements": []string{"ndh"},
	}
	result := calculateChildFunding(3, props, period)

	if result.Funding != 15000 {
		t.Errorf("expected 15000 funding, got %d", result.Funding)
	}
	if len(result.MatchedProperties) != 2 {
		t.Errorf("expected 2 matched properties, got %d", len(result.MatchedProperties))
	}
	if len(result.UnmatchedProperties) != 0 {
		t.Errorf("expected 0 unmatched properties, got %d", len(result.UnmatchedProperties))
	}
}

func TestCalculateChildFunding_PartialMatch(t *testing.T) {
	period := &models.GovernmentFundingPeriod{
		Properties: []models.GovernmentFundingProperty{
			{Key: "care_type", Value: "ganztag", Payment: 10000, MinAge: nil, MaxAge: nil},
		},
	}
	props := models.ContractProperties{
		"care_type":   "ganztag",
		"supplements": []string{"ndh"},
	}
	result := calculateChildFunding(3, props, period)

	if result.Funding != 10000 {
		t.Errorf("expected 10000 funding, got %d", result.Funding)
	}
	if len(result.MatchedProperties) != 1 {
		t.Errorf("expected 1 matched property, got %d", len(result.MatchedProperties))
	}
	if len(result.UnmatchedProperties) != 1 {
		t.Errorf("expected 1 unmatched property (ndh), got %d", len(result.UnmatchedProperties))
	}
}

func TestCalculateChildFunding_PaymentAccumulation(t *testing.T) {
	period := &models.GovernmentFundingPeriod{
		Properties: []models.GovernmentFundingProperty{
			{Key: "care_type", Value: "ganztag", Payment: 166847, MinAge: nil, MaxAge: nil},
			{Key: "supplements", Value: "ndh", Payment: 3456, MinAge: nil, MaxAge: nil},
			{Key: "supplements", Value: "mss", Payment: 7890, MinAge: nil, MaxAge: nil},
		},
	}
	props := models.ContractProperties{
		"care_type":   "ganztag",
		"supplements": []string{"ndh", "mss"},
	}
	result := calculateChildFunding(3, props, period)

	expected := 166847 + 3456 + 7890
	if result.Funding != expected {
		t.Errorf("expected funding %d, got %d", expected, result.Funding)
	}
}

func TestCalculateChildFunding_RequirementAccumulation(t *testing.T) {
	period := &models.GovernmentFundingPeriod{
		Properties: []models.GovernmentFundingProperty{
			{Key: "care_type", Value: "ganztag", Payment: 10000, Requirement: 0.5, MinAge: nil, MaxAge: nil},
			{Key: "supplements", Value: "ndh", Payment: 2000, Requirement: 0.25, MinAge: nil, MaxAge: nil},
		},
	}
	props := models.ContractProperties{
		"care_type":   "ganztag",
		"supplements": []string{"ndh"},
	}
	result := calculateChildFunding(3, props, period)

	expectedReq := 0.75
	if result.Requirement != expectedReq {
		t.Errorf("expected requirement %f, got %f", expectedReq, result.Requirement)
	}
}

func TestCalculateChildFunding_NoPropertyMatch(t *testing.T) {
	period := &models.GovernmentFundingPeriod{
		Properties: []models.GovernmentFundingProperty{
			{Key: "care_type", Value: "ganztag", Payment: 10000, MinAge: nil, MaxAge: nil},
		},
	}
	// Contract has a different value than what funding expects
	props := models.ContractProperties{"care_type": "teilzeit"}
	result := calculateChildFunding(3, props, period)

	if result.Funding != 0 {
		t.Errorf("expected 0 funding when property value doesn't match, got %d", result.Funding)
	}
	if len(result.MatchedProperties) != 0 {
		t.Errorf("expected 0 matched, got %d", len(result.MatchedProperties))
	}
	if len(result.UnmatchedProperties) != 1 {
		t.Errorf("expected 1 unmatched (care_type:teilzeit), got %d", len(result.UnmatchedProperties))
	}
}

// --- getAllContractKeyValues ---

func TestGetAllContractKeyValues_NilProperties(t *testing.T) {
	got := getAllContractKeyValues(nil)
	if got == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 entries, got %d", len(got))
	}
}

func TestGetAllContractKeyValues_EmptyMap(t *testing.T) {
	props := models.ContractProperties{}
	got := getAllContractKeyValues(props)
	if got == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 entries for empty map, got %d", len(got))
	}
}

func TestGetAllContractKeyValues_ScalarProperty(t *testing.T) {
	props := models.ContractProperties{"care_type": "ganztag"}
	got := getAllContractKeyValues(props)
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if got[0].Key != "care_type" || got[0].Value != "ganztag" {
		t.Errorf("expected care_type:ganztag, got %s:%s", got[0].Key, got[0].Value)
	}
}

func TestGetAllContractKeyValues_ArrayProperty(t *testing.T) {
	props := models.ContractProperties{"supplements": []string{"ndh", "mss"}}
	got := getAllContractKeyValues(props)
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	values := map[string]bool{}
	for _, entry := range got {
		if entry.Key != "supplements" {
			t.Errorf("expected key 'supplements', got %q", entry.Key)
		}
		values[entry.Value] = true
	}
	if !values["ndh"] || !values["mss"] {
		t.Errorf("expected ndh and mss values, got %v", values)
	}
}

// =========================================
// Funding Calculation Tests
// =========================================

func TestChildService_CalculateFunding_BasicCalculation(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	// Create org with government funding
	org := createTestOrganization(t, db, "Test Org")
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")

	// Assign funding to org
	// Funding is now automatically looked up by org.State ("berlin")

	// Create funding period covering our test date
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)

	// Create properties with age filter (ages 3-6)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 100000, 3, 7) // 1000.00 EUR
	createTestFundingProperty(t, db, period.ID, "supplements", "ndh", 50000, 3, 7)    // 500.00 EUR

	// Create child (born 2022-01-15, age 3 on 2025-01-27)
	child := createTestChild(t, db, "Max", "Mustermann", org.ID)
	child.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	// Create contract with attributes
	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag", "supplements": []string{"ndh"}},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Calculate funding
	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := statsSvc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify weekly hours basis is set from funding period (not pay plan)
	if result.WeeklyHoursBasis != 39.0 {
		t.Errorf("WeeklyHoursBasis = %f, want 39.0", result.WeeklyHoursBasis)
	}

	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}

	cf := result.Children[0]
	if cf.ChildID != child.ID {
		t.Errorf("ChildID = %d, want %d", cf.ChildID, child.ID)
	}
	if cf.Funding != 150000 { // 1000.00 + 500.00 = 1500.00 EUR = 150000 cents
		t.Errorf("Funding = %d, want 150000 (cents)", cf.Funding)
	}
	if cf.Requirement != 0.2 { // 0.1 + 0.1 = 0.2 (two matched properties with Requirement=0.1 each)
		t.Errorf("Requirement = %f, want 0.2", cf.Requirement)
	}
	if len(cf.MatchedProperties) != 2 {
		t.Errorf("MatchedProperties = %v, want 2 items", cf.MatchedProperties)
	}
	if len(cf.UnmatchedProperties) != 0 {
		t.Errorf("UnmatchedProperties = %v, want 0 items", cf.UnmatchedProperties)
	}
}

func TestChildService_CalculateFunding_NoFundingAssigned(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	// Create org WITHOUT government funding
	org := createTestOrganization(t, db, "Test Org")

	// Create child with contract
	child := createTestChild(t, db, "Max", "Mustermann", org.ID)
	child.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag", "supplements": []string{"ndh"}},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Calculate funding
	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := statsSvc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}

	cf := result.Children[0]
	if cf.Funding != 0 {
		t.Errorf("Funding = %d, want 0 (no funding assigned)", cf.Funding)
	}
	if len(cf.UnmatchedProperties) != 2 {
		t.Errorf("UnmatchedProperties = %v, want 2 items", cf.UnmatchedProperties)
	}
}

func TestChildService_CalculateFunding_NoMatchingPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	// Create org with funding
	org := createTestOrganization(t, db, "Test Org")
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	// Funding is now automatically looked up by org.State ("berlin")

	// Create period that doesn't cover our test date
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &to, 39.0)

	// Create child with contract
	child := createTestChild(t, db, "Max", "Mustermann", org.ID)
	child.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})

	// Calculate funding for date outside period
	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := statsSvc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cf := result.Children[0]
	if cf.Funding != 0 {
		t.Errorf("Funding = %d, want 0 (no matching period)", cf.Funding)
	}
}

func TestChildService_CalculateFunding_NoMatchingAgeProperty(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	// Create org with funding
	org := createTestOrganization(t, db, "Test Org")
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	// Funding is now automatically looked up by org.State ("berlin")

	// Create period
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)

	// Create property for ages 0-2 only
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 100000, 0, 2)

	// Create child age 3 (doesn't match 0-2 property)
	child := createTestChild(t, db, "Max", "Mustermann", org.ID)
	child.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC) // Age 3 on 2025-01-27
	db.Save(child)

	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})

	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := statsSvc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cf := result.Children[0]
	if cf.Funding != 0 {
		t.Errorf("Funding = %d, want 0 (no matching age property)", cf.Funding)
	}
	if len(cf.UnmatchedProperties) != 1 {
		t.Errorf("UnmatchedProperties = %v, want 1 item (care_type:ganztag)", cf.UnmatchedProperties)
	}
}

func TestChildService_CalculateFunding_PartialAttributeMatch(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	// Create org with funding
	org := createTestOrganization(t, db, "Test Org")
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	// Funding is now automatically looked up by org.State ("berlin")

	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 100000, 3, 7)
	// "unknown_key" property does NOT exist

	child := createTestChild(t, db, "Max", "Mustermann", org.ID)
	child.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	// Inserted directly rather than through CreateContract, which now refuses a
	// key the funding configuration does not declare. This test is about the
	// READ path -- what the calculator does with a property that matches
	// nothing -- and that question still has an answer for every contract
	// stored before validation existed. Going through the write path would test
	// the write path instead, and (because the error was discarded) left
	// result.Children empty and this test panicking on an index rather than
	// failing on an assertion.
	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, fromDate, nil, 1,
		models.ContractProperties{"care_type": "ganztag", "unknown_key": "xyz"})

	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := statsSvc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cf := result.Children[0]
	if cf.Funding != 100000 {
		t.Errorf("Funding = %d, want 100000", cf.Funding)
	}
	if len(cf.MatchedProperties) != 1 {
		t.Errorf("MatchedProperties = %v, want 1 item", cf.MatchedProperties)
	}
	if len(cf.UnmatchedProperties) != 1 {
		t.Errorf("UnmatchedProperties = %v, want 1 item", cf.UnmatchedProperties)
	}
}

func TestChildService_CalculateFunding_SingleProperty(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	// Funding is now automatically looked up by org.State ("berlin")

	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 100000, 3, 7)

	child := createTestChild(t, db, "Max", "Mustermann", org.ID)
	child.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})

	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := statsSvc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cf := result.Children[0]
	if cf.Funding != 100000 {
		t.Errorf("Funding = %d, want 100000", cf.Funding)
	}
	if len(cf.MatchedProperties) != 1 {
		t.Errorf("MatchedProperties = %v, want 1 item", cf.MatchedProperties)
	}
}

func TestChildService_CalculateFunding_ChildNoActiveOnDate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	// Funding is now automatically looked up by org.State ("berlin")

	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 100000, 3, 7)

	// Child with active contract
	childActive := createTestChild(t, db, "Active", "Child", org.ID)
	childActive.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(childActive)
	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, childActive.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})

	// Child with NO contract (should not appear in results)
	childNoContract := createTestChild(t, db, "NoContract", "Child", org.ID)
	childNoContract.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(childNoContract)

	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := statsSvc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should only include child with active contract
	if len(result.Children) != 1 {
		t.Errorf("expected 1 child (with active contract), got %d", len(result.Children))
	}
	if result.Children[0].ChildName != "Active Child" {
		t.Errorf("expected Active Child, got %s", result.Children[0].ChildName)
	}
}

// SECURITY TEST: Cross-organization funding calculation
func TestChildService_CalculateFunding_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Both orgs have state="berlin", so they share the same funding
	funding := createTestGovernmentFunding(t, db, "Funding")

	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 100000, 3, 7)

	// Child in org1
	child1 := createTestChild(t, db, "Org1", "Child", org1.ID)
	child1.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child1)
	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, child1.ID, org1.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})

	// Child in org2
	child2 := createTestChild(t, db, "Org2", "Child", org2.ID)
	child2.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child2)
	_, _ = svc.CreateContract(ctx, child2.ID, org2.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})

	// Calculate funding for org1 - should NOT include org2's child
	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := statsSvc.CalculateFunding(ctx, org1.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Children) != 1 {
		t.Errorf("expected 1 child from org1, got %d", len(result.Children))
	}

	for _, cf := range result.Children {
		if cf.ChildName == "Org2 Child" {
			t.Error("SECURITY: org2's child leaked into org1's funding calculation")
		}
	}
}

func TestChildService_CalculateFunding_WeeklyHoursFromFundingPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")

	// Create funding period with different weekly hours (40.0 instead of 39.0)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 40.0)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 100000, 3, 7)

	child := createTestChild(t, db, "Max", "Mustermann", org.ID)
	child.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := statsSvc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.WeeklyHoursBasis != 40.0 {
		t.Errorf("WeeklyHoursBasis = %f, want 40.0 (from funding period)", result.WeeklyHoursBasis)
	}
}

func TestChildService_CalculateFunding_NoMatchingPeriod_WeeklyHoursZero(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")

	// Create period that doesn't cover our test date
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &to, 39.0)

	child := createTestChild(t, db, "Max", "Mustermann", org.ID)
	child.Birthdate = time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	fromDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID:  1,
		From:       fromDate,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})

	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	result, err := statsSvc.CalculateFunding(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.WeeklyHoursBasis != 0 {
		t.Errorf("WeeklyHoursBasis = %f, want 0 (no matching period)", result.WeeklyHoursBasis)
	}
}
