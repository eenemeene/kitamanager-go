package service

import (
	"bytes"
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
