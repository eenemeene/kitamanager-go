package service

import (
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// --- Test helpers ---

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func datePtr(year int, month time.Month, day int) *time.Time {
	d := date(year, month, day)
	return &d
}

func intP(i int) *int { return &i }

func makeChild(id uint, birthdate time.Time, gender string, contracts []models.ChildContract) models.Child {
	return models.Child{
		Person: models.Person{
			ID:        id,
			FirstName: "Child",
			LastName:  "Test",
			Gender:    gender,
			Birthdate: birthdate,
		},
		Contracts: contracts,
	}
}

func makeChildContract(from time.Time, to *time.Time, props models.ContractProperties) models.ChildContract {
	return models.ChildContract{
		BaseContract: models.BaseContract{
			Period:     models.Period{From: from, To: to},
			Properties: props,
		},
	}
}

func makeEmployeeContract(id, empID uint, from time.Time, to *time.Time, category string, grade string, step int, hours float64, ppID uint) models.EmployeeContract {
	return models.EmployeeContract{
		ID:            id,
		EmployeeID:    empID,
		StaffCategory: category,
		Grade:         grade,
		Step:          step,
		WeeklyHours:   hours,
		PayPlanID:     ppID,
		BaseContract: models.BaseContract{
			Period: models.Period{From: from, To: to},
		},
	}
}

func makeEmployee(id uint, first, last string, contracts []models.EmployeeContract) models.Employee {
	return models.Employee{
		Person: models.Person{
			ID:        id,
			FirstName: first,
			LastName:  last,
		},
		Contracts: contracts,
	}
}

func makeFundingPeriod(from time.Time, to *time.Time, fullTimeHours float64, props []models.GovernmentFundingProperty) models.GovernmentFundingPeriod {
	return models.GovernmentFundingPeriod{
		Period:              models.Period{From: from, To: to},
		FullTimeWeeklyHours: fullTimeHours,
		Properties:          props,
	}
}

func makeFundingProp(key, value, label string, payment int, requirement float64, minAge, maxAge *int) models.GovernmentFundingProperty {
	return models.GovernmentFundingProperty{
		Key:         key,
		Value:       value,
		Label:       label,
		Payment:     payment,
		Requirement: requirement,
		MinAge:      minAge,
		MaxAge:      maxAge,
	}
}

func makePayPlan(id uint, periods []models.PayPlanPeriod) *models.PayPlan {
	return &models.PayPlan{
		ID:      id,
		Periods: periods,
	}
}

func makePayPlanPeriod(id uint, from time.Time, to *time.Time, weeklyHours float64, contribRate int, entries []models.PayPlanEntry) models.PayPlanPeriod {
	return models.PayPlanPeriod{
		ID:                       id,
		Period:                   models.Period{From: from, To: to},
		WeeklyHours:              weeklyHours,
		EmployerContributionRate: contribRate,
		Entries:                  entries,
	}
}

func makePayPlanEntry(grade string, step, monthlyAmount int) models.PayPlanEntry {
	return models.PayPlanEntry{
		Grade:         grade,
		Step:          step,
		MonthlyAmount: monthlyAmount,
	}
}

func makeBudgetItem(name, category string, perChild bool, entries []models.BudgetItemEntry) models.BudgetItem {
	return models.BudgetItem{
		Name:     name,
		Category: category,
		PerChild: perChild,
		Entries:  entries,
	}
}

func makeBudgetItemEntry(from time.Time, to *time.Time, amountCents int) models.BudgetItemEntry {
	return models.BudgetItemEntry{
		Period:      models.Period{From: from, To: to},
		AmountCents: amountCents,
	}
}

// ========================================================================
// monthCount
// ========================================================================

func TestMonthCount(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected int
	}{
		{"same month", date(2025, 1, 1), date(2025, 1, 1), 1},
		{"two months", date(2025, 1, 1), date(2025, 2, 1), 2},
		{"full year", date(2025, 1, 1), date(2025, 12, 1), 12},
		{"cross year", date(2024, 11, 1), date(2025, 2, 1), 4},
		{"end before start", date(2025, 3, 1), date(2025, 1, 1), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := monthCount(tt.start, tt.end)
			if got != tt.expected {
				t.Errorf("monthCount(%s, %s) = %d, want %d", tt.start.Format("2006-01"), tt.end.Format("2006-01"), got, tt.expected)
			}
		})
	}
}

// ========================================================================
// formatAgeGroupLabel
// ========================================================================

func TestFormatAgeGroupLabel(t *testing.T) {
	tests := []struct {
		minAge, maxAge int
		expected       string
	}{
		{2, 2, "2"},
		{0, 1, "0/1"},
		{3, 8, "3+"},
		{0, 0, "0"},
		{3, 5, "3/4/5"},
	}
	for _, tt := range tests {
		got := formatAgeGroupLabel(tt.minAge, tt.maxAge)
		if got != tt.expected {
			t.Errorf("formatAgeGroupLabel(%d, %d) = %q, want %q", tt.minAge, tt.maxAge, got, tt.expected)
		}
	}
}

// ========================================================================
// findAgeGroupLabel
// ========================================================================

func TestFindAgeGroupLabel(t *testing.T) {
	groups := []models.OccupancyAgeGroup{
		{Label: "0/1", MinAge: 0, MaxAge: 1},
		{Label: "2", MinAge: 2, MaxAge: 2},
		{Label: "3+", MinAge: 3, MaxAge: 99},
	}
	tests := []struct {
		age      int
		expected string
	}{
		{0, "0/1"},
		{1, "0/1"},
		{2, "2"},
		{3, "3+"},
		{10, "3+"},
	}
	for _, tt := range tests {
		got := findAgeGroupLabel(tt.age, groups)
		if got != tt.expected {
			t.Errorf("findAgeGroupLabel(%d) = %q, want %q", tt.age, got, tt.expected)
		}
	}
}

// ========================================================================
// buildFundingLabelMap
// ========================================================================

func TestBuildFundingLabelMap(t *testing.T) {
	periods := []models.GovernmentFundingPeriod{
		makeFundingPeriod(date(2024, 1, 1), nil, 39, []models.GovernmentFundingProperty{
			makeFundingProp("care_type", "ganztag", "Ganztag", 100000, 0.2, nil, nil),
			makeFundingProp("supplements", "ndh", "NDH", 50000, 0.1, nil, nil),
		}),
	}
	labelMap := buildFundingLabelMap(periods)

	if labelMap["care_type:ganztag"] != "Ganztag" {
		t.Errorf("expected 'Ganztag', got %q", labelMap["care_type:ganztag"])
	}
	if labelMap["supplements:ndh"] != "NDH" {
		t.Errorf("expected 'NDH', got %q", labelMap["supplements:ndh"])
	}
	if labelMap["nonexistent:key"] != "" {
		t.Errorf("expected empty string for missing key, got %q", labelMap["nonexistent:key"])
	}
}

func TestBuildFundingLabelMap_EmptyPeriods(t *testing.T) {
	labelMap := buildFundingLabelMap(nil)
	if len(labelMap) != 0 {
		t.Errorf("expected empty map, got %d entries", len(labelMap))
	}
}

// ========================================================================
// extractOccupancyStructure
// ========================================================================

func TestExtractOccupancyStructure(t *testing.T) {
	periods := []models.GovernmentFundingPeriod{
		makeFundingPeriod(date(2024, 1, 1), nil, 39, []models.GovernmentFundingProperty{
			makeFundingProp("care_type", "ganztag", "Ganztag", 100000, 0.2, intP(0), intP(2)),
			makeFundingProp("care_type", "ganztag", "Ganztag", 80000, 0.15, intP(3), intP(6)),
			makeFundingProp("care_type", "halbtag", "Halbtag", 60000, 0.1, intP(0), intP(2)),
			makeFundingProp("care_type", "halbtag", "Halbtag", 50000, 0.08, intP(3), intP(6)),
			makeFundingProp("supplements", "ndh", "NDH", 30000, 0.05, nil, nil),
		}),
	}

	ageGroups, careTypes, supplements := extractOccupancyStructure(periods)

	if len(ageGroups) != 2 {
		t.Fatalf("expected 2 age groups, got %d", len(ageGroups))
	}
	if ageGroups[0].MinAge != 0 || ageGroups[0].MaxAge != 2 {
		t.Errorf("first age group: expected 0-2, got %d-%d", ageGroups[0].MinAge, ageGroups[0].MaxAge)
	}
	if ageGroups[1].MinAge != 3 || ageGroups[1].MaxAge != 6 {
		t.Errorf("second age group: expected 3-6, got %d-%d", ageGroups[1].MinAge, ageGroups[1].MaxAge)
	}

	if len(careTypes) != 2 {
		t.Fatalf("expected 2 care types, got %d", len(careTypes))
	}
	// Sorted by value
	if careTypes[0].Value != "ganztag" {
		t.Errorf("expected first care type 'ganztag', got %q", careTypes[0].Value)
	}

	if len(supplements) != 1 {
		t.Fatalf("expected 1 supplement, got %d", len(supplements))
	}
	if supplements[0].Value != "ndh" {
		t.Errorf("expected supplement 'ndh', got %q", supplements[0].Value)
	}
}

func TestExtractOccupancyStructure_NoPeriods(t *testing.T) {
	ageGroups, careTypes, supplements := extractOccupancyStructure(nil)
	if ageGroups != nil || careTypes != nil || supplements != nil {
		t.Error("expected nil for all with no periods")
	}
}

// ========================================================================
// calculateAgeDistribution
// ========================================================================

func TestCalculateAgeDistribution_Basic(t *testing.T) {
	refDate := date(2025, 6, 1)
	children := []models.Child{
		makeChild(1, date(2023, 6, 1), "male", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), nil, nil),
		}),
		makeChild(2, date(2020, 1, 1), "female", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), nil, nil),
		}),
	}

	result := calculateAgeDistribution(children, refDate)

	if result.TotalCount != 2 {
		t.Errorf("expected total 2, got %d", result.TotalCount)
	}
	if result.Date != "2025-06-01" {
		t.Errorf("expected date '2025-06-01', got %q", result.Date)
	}
	// Child 1: age 2, Child 2: age 5
	for _, b := range result.Distribution {
		switch b.AgeLabel {
		case "2":
			if b.Count != 1 || b.MaleCount != 1 {
				t.Errorf("age 2 bucket: expected 1 male, got count=%d male=%d", b.Count, b.MaleCount)
			}
		case "5":
			if b.Count != 1 || b.FemaleCount != 1 {
				t.Errorf("age 5 bucket: expected 1 female, got count=%d female=%d", b.Count, b.FemaleCount)
			}
		default:
			if b.Count != 0 {
				t.Errorf("bucket %s: expected 0, got %d", b.AgeLabel, b.Count)
			}
		}
	}
}

func TestCalculateAgeDistribution_NoActiveContract(t *testing.T) {
	refDate := date(2025, 6, 1)
	children := []models.Child{
		makeChild(1, date(2023, 1, 1), "male", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), datePtr(2024, 12, 31), nil), // expired
		}),
	}

	result := calculateAgeDistribution(children, refDate)
	if result.TotalCount != 0 {
		t.Errorf("expected 0, got %d", result.TotalCount)
	}
}

func TestCalculateAgeDistribution_Empty(t *testing.T) {
	result := calculateAgeDistribution(nil, date(2025, 1, 1))
	if result.TotalCount != 0 {
		t.Errorf("expected 0, got %d", result.TotalCount)
	}
	if len(result.Distribution) != 7 {
		t.Errorf("expected 7 buckets, got %d", len(result.Distribution))
	}
}

func TestCalculateAgeDistribution_SixPlusBucket(t *testing.T) {
	refDate := date(2025, 6, 1)
	children := []models.Child{
		makeChild(1, date(2018, 1, 1), "diverse", []models.ChildContract{
			makeChildContract(date(2020, 1, 1), nil, nil),
		}),
	}
	result := calculateAgeDistribution(children, refDate)
	for _, b := range result.Distribution {
		if b.AgeLabel == "6+" && b.Count != 1 {
			t.Errorf("6+ bucket: expected 1, got %d", b.Count)
		}
		if b.AgeLabel == "6+" && b.DiverseCount != 1 {
			t.Errorf("6+ bucket: expected 1 diverse, got %d", b.DiverseCount)
		}
	}
}

// ========================================================================
// calculateContractPropertiesDistribution
// ========================================================================

func TestCalculateContractPropertiesDistribution_Basic(t *testing.T) {
	refDate := date(2025, 6, 1)
	children := []models.Child{
		makeChild(1, date(2022, 1, 1), "male", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), nil, models.ContractProperties{
				"care_type":   "ganztag",
				"supplements": []any{"ndh", "mss"},
			}),
		}),
		makeChild(2, date(2021, 1, 1), "female", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), nil, models.ContractProperties{
				"care_type": "halbtag",
			}),
		}),
	}

	periods := []models.GovernmentFundingPeriod{
		makeFundingPeriod(date(2024, 1, 1), nil, 39, []models.GovernmentFundingProperty{
			makeFundingProp("care_type", "ganztag", "Ganztag", 100000, 0, nil, nil),
			makeFundingProp("care_type", "halbtag", "Halbtag", 60000, 0, nil, nil),
			makeFundingProp("supplements", "ndh", "NDH", 30000, 0, nil, nil),
		}),
	}

	result := calculateContractPropertiesDistribution(children, periods, refDate)

	if result.TotalChildren != 2 {
		t.Errorf("expected 2 children, got %d", result.TotalChildren)
	}
	if result.Date != "2025-06-01" {
		t.Errorf("expected date '2025-06-01', got %q", result.Date)
	}

	expected := map[string]int{
		"care_type:ganztag": 1,
		"care_type:halbtag": 1,
		"supplements:mss":   1,
		"supplements:ndh":   1,
	}
	if len(result.Properties) != len(expected) {
		t.Fatalf("expected %d properties, got %d", len(expected), len(result.Properties))
	}
	for _, p := range result.Properties {
		key := p.Key + ":" + p.Value
		if expected[key] != p.Count {
			t.Errorf("property %s: expected %d, got %d", key, expected[key], p.Count)
		}
	}
}

func TestCalculateContractPropertiesDistribution_WithLabels(t *testing.T) {
	children := []models.Child{
		makeChild(1, date(2022, 1, 1), "male", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), nil, models.ContractProperties{"care_type": "ganztag"}),
		}),
	}
	periods := []models.GovernmentFundingPeriod{
		makeFundingPeriod(date(2024, 1, 1), nil, 39, []models.GovernmentFundingProperty{
			makeFundingProp("care_type", "ganztag", "Ganztag (bis 9h)", 100000, 0, nil, nil),
		}),
	}

	result := calculateContractPropertiesDistribution(children, periods, date(2025, 1, 1))
	if len(result.Properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(result.Properties))
	}
	if result.Properties[0].Label != "Ganztag (bis 9h)" {
		t.Errorf("expected label 'Ganztag (bis 9h)', got %q", result.Properties[0].Label)
	}
}

func TestCalculateContractPropertiesDistribution_NilFunding(t *testing.T) {
	children := []models.Child{
		makeChild(1, date(2022, 1, 1), "male", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), nil, models.ContractProperties{"care_type": "ganztag"}),
		}),
	}
	result := calculateContractPropertiesDistribution(children, nil, date(2025, 1, 1))
	if result.TotalChildren != 1 {
		t.Errorf("expected 1 child, got %d", result.TotalChildren)
	}
	// No labels from funding but still counts properties
	if len(result.Properties) != 1 {
		t.Errorf("expected 1 property, got %d", len(result.Properties))
	}
}

func TestCalculateContractPropertiesDistribution_SortedOutput(t *testing.T) {
	children := []models.Child{
		makeChild(1, date(2022, 1, 1), "male", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), nil, models.ContractProperties{
				"z_key": "b_val",
				"a_key": "a_val",
			}),
		}),
	}
	result := calculateContractPropertiesDistribution(children, nil, date(2025, 1, 1))
	if len(result.Properties) < 2 {
		t.Fatalf("expected at least 2 properties, got %d", len(result.Properties))
	}
	if result.Properties[0].Key != "a_key" {
		t.Errorf("expected first property key 'a_key', got %q", result.Properties[0].Key)
	}
}

// ========================================================================
// calculateFunding
// ========================================================================

func TestCalculateFunding_Basic(t *testing.T) {
	refDate := date(2025, 6, 1)
	children := []models.Child{
		makeChild(1, date(2022, 6, 1), "male", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), nil, models.ContractProperties{"care_type": "ganztag"}),
		}),
	}
	periods := []models.GovernmentFundingPeriod{
		makeFundingPeriod(date(2024, 1, 1), nil, 39.0, []models.GovernmentFundingProperty{
			makeFundingProp("care_type", "ganztag", "Ganztag", 166847, 0.261, intP(0), intP(6)),
		}),
	}

	result := calculateFunding(children, periods, refDate)

	if result.WeeklyHoursBasis != 39.0 {
		t.Errorf("expected weekly hours basis 39, got %f", result.WeeklyHoursBasis)
	}
	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}
	cf := result.Children[0]
	if cf.Funding != 166847 {
		t.Errorf("expected funding 166847, got %d", cf.Funding)
	}
	if cf.Requirement != 0.261 {
		t.Errorf("expected requirement 0.261, got %f", cf.Requirement)
	}
	if len(cf.MatchedProperties) != 1 {
		t.Errorf("expected 1 matched property, got %d", len(cf.MatchedProperties))
	}
	if len(cf.UnmatchedProperties) != 0 {
		t.Errorf("expected 0 unmatched, got %d", len(cf.UnmatchedProperties))
	}
}

func TestCalculateFunding_NoMatchingPeriod(t *testing.T) {
	children := []models.Child{
		makeChild(1, date(2022, 1, 1), "male", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), nil, models.ContractProperties{"care_type": "ganztag"}),
		}),
	}
	// Period doesn't cover the reference date
	periods := []models.GovernmentFundingPeriod{
		makeFundingPeriod(date(2020, 1, 1), datePtr(2023, 12, 31), 39, []models.GovernmentFundingProperty{
			makeFundingProp("care_type", "ganztag", "Ganztag", 100000, 0.2, nil, nil),
		}),
	}

	result := calculateFunding(children, periods, date(2025, 6, 1))
	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}
	if result.Children[0].Funding != 0 {
		t.Errorf("expected 0 funding, got %d", result.Children[0].Funding)
	}
	if len(result.Children[0].UnmatchedProperties) != 1 {
		t.Errorf("expected 1 unmatched property, got %d", len(result.Children[0].UnmatchedProperties))
	}
}

func TestCalculateFunding_NoActiveContract(t *testing.T) {
	children := []models.Child{
		makeChild(1, date(2022, 1, 1), "male", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), datePtr(2024, 12, 31), nil),
		}),
	}
	result := calculateFunding(children, nil, date(2025, 6, 1))
	if len(result.Children) != 0 {
		t.Errorf("expected 0 children in result, got %d", len(result.Children))
	}
}

func TestCalculateFunding_ChildNameAndAge(t *testing.T) {
	children := []models.Child{
		{
			Person: models.Person{
				ID:        42,
				FirstName: "Max",
				LastName:  "Mustermann",
				Birthdate: date(2022, 1, 15),
			},
			Contracts: []models.ChildContract{
				makeChildContract(date(2024, 1, 1), nil, nil),
			},
		},
	}
	result := calculateFunding(children, nil, date(2025, 6, 1))
	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child")
	}
	if result.Children[0].ChildName != "Max Mustermann" {
		t.Errorf("expected 'Max Mustermann', got %q", result.Children[0].ChildName)
	}
	if result.Children[0].Age != 3 {
		t.Errorf("expected age 3, got %d", result.Children[0].Age)
	}
	if result.Children[0].ChildID != 42 {
		t.Errorf("expected child ID 42, got %d", result.Children[0].ChildID)
	}
}

// ========================================================================
// calculateChildFunding
// ========================================================================

func TestCalculateChildFunding_NilPeriod_ScalarProp(t *testing.T) {
	props := models.ContractProperties{"care_type": "ganztag"}
	result := calculateChildFunding(3, props, nil)
	if result.Funding != 0 {
		t.Errorf("expected 0, got %d", result.Funding)
	}
	if len(result.UnmatchedProperties) != 1 {
		t.Errorf("expected 1 unmatched, got %d", len(result.UnmatchedProperties))
	}
}

func TestCalculateChildFunding_MultipleMatches(t *testing.T) {
	period := &models.GovernmentFundingPeriod{
		Period: models.Period{From: date(2024, 1, 1)},
		Properties: []models.GovernmentFundingProperty{
			makeFundingProp("care_type", "ganztag", "Ganztag", 100000, 0.2, nil, nil),
			makeFundingProp("supplements", "ndh", "NDH", 30000, 0.05, nil, nil),
		},
	}
	props := models.ContractProperties{
		"care_type":   "ganztag",
		"supplements": []any{"ndh"},
	}
	result := calculateChildFunding(3, props, period)
	if result.Funding != 130000 {
		t.Errorf("expected 130000, got %d", result.Funding)
	}
	if result.Requirement != 0.25 {
		t.Errorf("expected 0.25, got %f", result.Requirement)
	}
	if len(result.MatchedProperties) != 2 {
		t.Errorf("expected 2 matched, got %d", len(result.MatchedProperties))
	}
	if len(result.UnmatchedProperties) != 0 {
		t.Errorf("expected 0 unmatched, got %d", len(result.UnmatchedProperties))
	}
}

func TestCalculateChildFunding_AgeFiltered(t *testing.T) {
	period := &models.GovernmentFundingPeriod{
		Period: models.Period{From: date(2024, 1, 1)},
		Properties: []models.GovernmentFundingProperty{
			makeFundingProp("care_type", "ganztag", "Ganztag U3", 100000, 0.2, intP(0), intP(2)),
			makeFundingProp("care_type", "ganztag", "Ganztag 3+", 80000, 0.15, intP(3), intP(6)),
		},
	}
	props := models.ContractProperties{"care_type": "ganztag"}

	// Age 1: should match U3
	result := calculateChildFunding(1, props, period)
	if result.Funding != 100000 {
		t.Errorf("age 1: expected 100000, got %d", result.Funding)
	}

	// Age 4: should match 3+
	result = calculateChildFunding(4, props, period)
	if result.Funding != 80000 {
		t.Errorf("age 4: expected 80000, got %d", result.Funding)
	}
}

// ========================================================================
// calculateStaffingHours
// ========================================================================

func TestCalculateStaffingHours_Basic(t *testing.T) {
	start := date(2025, 1, 1)
	end := date(2025, 3, 1) // 3 months: Jan, Feb, Mar

	children := []models.Child{
		makeChild(1, date(2022, 1, 1), "male", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), nil, models.ContractProperties{"care_type": "ganztag"}),
		}),
	}
	periods := []models.GovernmentFundingPeriod{
		makeFundingPeriod(date(2024, 1, 1), nil, 39.0, []models.GovernmentFundingProperty{
			makeFundingProp("care_type", "ganztag", "Ganztag", 100000, 0.261, nil, nil),
		}),
	}
	contracts := []models.EmployeeContract{
		makeEmployeeContract(1, 1, date(2024, 1, 1), nil, "qualified", "S8a", 3, 39.0, 1),
	}

	result := calculateStaffingHours(children, contracts, periods, start, end)

	if len(result) != 3 {
		t.Fatalf("expected 3 data points, got %d", len(result))
	}
	for _, dp := range result {
		if dp.ChildCount != 1 {
			t.Errorf("%s: expected 1 child, got %d", dp.Date, dp.ChildCount)
		}
		if dp.StaffCount != 1 {
			t.Errorf("%s: expected 1 staff, got %d", dp.Date, dp.StaffCount)
		}
		if dp.AvailableHours != 39.0 {
			t.Errorf("%s: expected 39.0 available hours, got %f", dp.Date, dp.AvailableHours)
		}
		// Required: 0.261 * 39.0 = 10.179
		expectedRequired := 0.261 * 39.0
		if dp.RequiredHours < expectedRequired-0.01 || dp.RequiredHours > expectedRequired+0.01 {
			t.Errorf("%s: expected ~%f required hours, got %f", dp.Date, expectedRequired, dp.RequiredHours)
		}
	}
}

func TestCalculateStaffingHours_NoFunding(t *testing.T) {
	children := []models.Child{
		makeChild(1, date(2022, 1, 1), "male", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), nil, nil),
		}),
	}
	result := calculateStaffingHours(children, nil, nil, date(2025, 1, 1), date(2025, 1, 1))
	if len(result) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(result))
	}
	if result[0].RequiredHours != 0 {
		t.Errorf("expected 0 required hours without funding, got %f", result[0].RequiredHours)
	}
	if result[0].ChildCount != 1 {
		t.Errorf("expected child count 1, got %d", result[0].ChildCount)
	}
}

func TestCalculateStaffingHours_MultipleEmployeesDedup(t *testing.T) {
	// Same employee with two contracts in different periods
	contracts := []models.EmployeeContract{
		makeEmployeeContract(1, 10, date(2025, 1, 1), datePtr(2025, 1, 31), "qualified", "S8a", 3, 30.0, 1),
		makeEmployeeContract(2, 10, date(2025, 2, 1), nil, "qualified", "S8a", 3, 39.0, 1),
	}
	result := calculateStaffingHours(nil, contracts, nil, date(2025, 1, 1), date(2025, 2, 1))
	if len(result) != 2 {
		t.Fatalf("expected 2 data points, got %d", len(result))
	}
	// Jan: one contract active, staff count = 1
	if result[0].StaffCount != 1 {
		t.Errorf("Jan: expected 1 staff, got %d", result[0].StaffCount)
	}
	if result[0].AvailableHours != 30.0 {
		t.Errorf("Jan: expected 30 hours, got %f", result[0].AvailableHours)
	}
	// Feb: new contract
	if result[1].AvailableHours != 39.0 {
		t.Errorf("Feb: expected 39 hours, got %f", result[1].AvailableHours)
	}
}

// ========================================================================
// calculateEmployeeStaffingHours
// ========================================================================

func TestCalculateEmployeeStaffingHours_Basic(t *testing.T) {
	employees := []models.Employee{
		makeEmployee(1, "Anna", "Mueller", []models.EmployeeContract{
			makeEmployeeContract(1, 1, date(2024, 1, 1), nil, "qualified", "S8a", 3, 39.0, 1),
		}),
		makeEmployee(2, "Bob", "Zimmermann", []models.EmployeeContract{
			makeEmployeeContract(2, 2, date(2024, 1, 1), nil, "supplementary", "S3", 1, 20.0, 1),
		}),
	}

	dates, rows := calculateEmployeeStaffingHours(employees, date(2025, 1, 1), date(2025, 2, 1))

	if len(dates) != 2 {
		t.Fatalf("expected 2 dates, got %d", len(dates))
	}
	if dates[0] != "2025-01-01" || dates[1] != "2025-02-01" {
		t.Errorf("unexpected dates: %v", dates)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	// Sorted by last name: Mueller before Zimmermann
	if rows[0].LastName != "Mueller" {
		t.Errorf("expected Mueller first, got %q", rows[0].LastName)
	}
	if rows[0].StaffCategory != "qualified" {
		t.Errorf("expected 'qualified', got %q", rows[0].StaffCategory)
	}
	if rows[0].MonthlyHours[0] != 39.0 || rows[0].MonthlyHours[1] != 39.0 {
		t.Errorf("unexpected hours for Mueller: %v", rows[0].MonthlyHours)
	}
	if rows[1].MonthlyHours[0] != 20.0 {
		t.Errorf("unexpected hours for Zimmermann: %v", rows[1].MonthlyHours)
	}
}

func TestCalculateEmployeeStaffingHours_NoContract(t *testing.T) {
	employees := []models.Employee{
		makeEmployee(1, "Anna", "Mueller", nil),
	}
	dates, rows := calculateEmployeeStaffingHours(employees, date(2025, 1, 1), date(2025, 1, 1))
	if len(dates) != 1 {
		t.Fatalf("expected 1 date, got %d", len(dates))
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].MonthlyHours[0] != 0 {
		t.Errorf("expected 0 hours, got %f", rows[0].MonthlyHours[0])
	}
	if rows[0].StaffCategory != "" {
		t.Errorf("expected empty staff category, got %q", rows[0].StaffCategory)
	}
}

func TestCalculateEmployeeStaffingHours_ContractTransition(t *testing.T) {
	employees := []models.Employee{
		makeEmployee(1, "Anna", "Mueller", []models.EmployeeContract{
			makeEmployeeContract(1, 1, date(2025, 1, 1), datePtr(2025, 1, 31), "qualified", "S8a", 3, 30.0, 1),
			makeEmployeeContract(2, 1, date(2025, 2, 1), nil, "qualified", "S8a", 4, 39.0, 1),
		}),
	}
	_, rows := calculateEmployeeStaffingHours(employees, date(2025, 1, 1), date(2025, 3, 1))
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].MonthlyHours[0] != 30.0 {
		t.Errorf("Jan: expected 30, got %f", rows[0].MonthlyHours[0])
	}
	if rows[0].MonthlyHours[1] != 39.0 {
		t.Errorf("Feb: expected 39, got %f", rows[0].MonthlyHours[1])
	}
	if rows[0].MonthlyHours[2] != 39.0 {
		t.Errorf("Mar: expected 39, got %f", rows[0].MonthlyHours[2])
	}
}

func TestCalculateEmployeeStaffingHours_SortOrder(t *testing.T) {
	employees := []models.Employee{
		makeEmployee(1, "Zara", "Adams", nil),
		makeEmployee(2, "Anna", "Adams", nil),
		makeEmployee(3, "Bob", "Baker", nil),
	}
	_, rows := calculateEmployeeStaffingHours(employees, date(2025, 1, 1), date(2025, 1, 1))
	// Expected: Anna Adams, Zara Adams, Bob Baker
	if rows[0].FirstName != "Anna" || rows[1].FirstName != "Zara" || rows[2].FirstName != "Bob" {
		t.Errorf("unexpected sort: %s %s, %s %s, %s %s", rows[0].FirstName, rows[0].LastName, rows[1].FirstName, rows[1].LastName, rows[2].FirstName, rows[2].LastName)
	}
}

// ========================================================================
// calculateOccupancy
// ========================================================================

func TestCalculateOccupancy_Basic(t *testing.T) {
	children := []models.Child{
		makeChild(1, date(2024, 1, 1), "male", []models.ChildContract{
			makeChildContract(date(2024, 6, 1), nil, models.ContractProperties{
				"care_type":   "ganztag",
				"supplements": []any{"ndh"},
			}),
		}),
		makeChild(2, date(2022, 1, 1), "female", []models.ChildContract{
			makeChildContract(date(2024, 6, 1), nil, models.ContractProperties{
				"care_type": "halbtag",
			}),
		}),
	}
	periods := []models.GovernmentFundingPeriod{
		makeFundingPeriod(date(2024, 1, 1), nil, 39, []models.GovernmentFundingProperty{
			makeFundingProp("care_type", "ganztag", "Ganztag", 100000, 0.2, intP(0), intP(2)),
			makeFundingProp("care_type", "ganztag", "Ganztag", 80000, 0.15, intP(3), intP(6)),
			makeFundingProp("care_type", "halbtag", "Halbtag", 60000, 0.1, intP(0), intP(2)),
			makeFundingProp("care_type", "halbtag", "Halbtag", 50000, 0.08, intP(3), intP(6)),
			makeFundingProp("supplements", "ndh", "NDH", 30000, 0.05, nil, nil),
		}),
	}

	result := calculateOccupancy(children, periods, date(2025, 3, 1), date(2025, 3, 1))

	if len(result.AgeGroups) != 2 {
		t.Fatalf("expected 2 age groups, got %d", len(result.AgeGroups))
	}
	if len(result.CareTypes) != 2 {
		t.Fatalf("expected 2 care types, got %d", len(result.CareTypes))
	}
	if len(result.DataPoints) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(result.DataPoints))
	}

	dp := result.DataPoints[0]
	if dp.Total != 2 {
		t.Errorf("expected total 2, got %d", dp.Total)
	}
	// Child 1: age 1 (born 2024-01-01, ref 2025-03-01), care_type=ganztag → age group "0/1/2"
	if dp.ByAgeAndCareType["0/1/2"]["ganztag"] != 1 {
		t.Errorf("expected 1 child in 0/1/2 ganztag, got %d", dp.ByAgeAndCareType["0/1/2"]["ganztag"])
	}
	// Child 2: age 3, care_type=halbtag → age group "3+"
	if dp.ByAgeAndCareType["3+"]["halbtag"] != 1 {
		t.Errorf("expected 1 child in 3+ halbtag, got %d", dp.ByAgeAndCareType["3+"]["halbtag"])
	}
	// Supplements
	if dp.BySupplement["ndh"] != 1 {
		t.Errorf("expected 1 ndh supplement, got %d", dp.BySupplement["ndh"])
	}
}

func TestCalculateOccupancy_NoPeriods(t *testing.T) {
	children := []models.Child{
		makeChild(1, date(2022, 1, 1), "male", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), nil, models.ContractProperties{"care_type": "ganztag"}),
		}),
	}
	result := calculateOccupancy(children, nil, date(2025, 1, 1), date(2025, 1, 1))
	if len(result.DataPoints) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(result.DataPoints))
	}
	// Still counts total children even without funding structure
	if result.DataPoints[0].Total != 1 {
		t.Errorf("expected total 1, got %d", result.DataPoints[0].Total)
	}
}

func TestCalculateOccupancy_MultipleMonths(t *testing.T) {
	children := []models.Child{
		makeChild(1, date(2022, 1, 1), "male", []models.ChildContract{
			makeChildContract(date(2025, 2, 1), nil, models.ContractProperties{"care_type": "ganztag"}),
		}),
	}
	periods := []models.GovernmentFundingPeriod{
		makeFundingPeriod(date(2024, 1, 1), nil, 39, []models.GovernmentFundingProperty{
			makeFundingProp("care_type", "ganztag", "Ganztag", 100000, 0.2, intP(0), intP(6)),
		}),
	}

	result := calculateOccupancy(children, periods, date(2025, 1, 1), date(2025, 3, 1))
	if len(result.DataPoints) != 3 {
		t.Fatalf("expected 3 data points, got %d", len(result.DataPoints))
	}
	// Jan: no contract yet
	if result.DataPoints[0].Total != 0 {
		t.Errorf("Jan: expected 0, got %d", result.DataPoints[0].Total)
	}
	// Feb: contract starts
	if result.DataPoints[1].Total != 1 {
		t.Errorf("Feb: expected 1, got %d", result.DataPoints[1].Total)
	}
}

// ========================================================================
// calculateFinancials
// ========================================================================

func TestCalculateFinancials_Basic(t *testing.T) {
	start := date(2025, 1, 1)
	end := date(2025, 1, 1) // Single month

	children := []models.Child{
		makeChild(1, date(2022, 1, 1), "male", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), nil, models.ContractProperties{"care_type": "ganztag"}),
		}),
	}
	fundingPeriods := []models.GovernmentFundingPeriod{
		makeFundingPeriod(date(2024, 1, 1), nil, 39.0, []models.GovernmentFundingProperty{
			makeFundingProp("care_type", "ganztag", "Ganztag", 166847, 0.261, nil, nil),
		}),
	}
	ppID := uint(1)
	employeeContracts := []models.EmployeeContract{
		makeEmployeeContract(1, 1, date(2024, 1, 1), nil, "qualified", "S8a", 3, 39.0, ppID),
	}
	payPlans := map[uint]*models.PayPlan{
		ppID: makePayPlan(ppID, []models.PayPlanPeriod{
			makePayPlanPeriod(1, date(2024, 1, 1), nil, 39.0, 2200, []models.PayPlanEntry{
				makePayPlanEntry("S8a", 3, 350000), // 3500.00 EUR
			}),
		}),
	}

	result := calculateFinancials(children, employeeContracts, payPlans, fundingPeriods, nil, start, end)

	if len(result) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(result))
	}
	dp := result[0]

	if dp.FundingIncome != 166847 {
		t.Errorf("expected funding income 166847, got %d", dp.FundingIncome)
	}
	if dp.ChildCount != 1 {
		t.Errorf("expected 1 child, got %d", dp.ChildCount)
	}
	if dp.StaffCount != 1 {
		t.Errorf("expected 1 staff, got %d", dp.StaffCount)
	}
	// Gross: 350000 * 39/39 = 350000
	if dp.GrossSalary != 350000 {
		t.Errorf("expected gross 350000, got %d", dp.GrossSalary)
	}
	// Employer: 350000 * 2200/10000 = 77000
	if dp.EmployerCosts != 77000 {
		t.Errorf("expected employer costs 77000, got %d", dp.EmployerCosts)
	}
	if dp.TotalIncome != 166847 {
		t.Errorf("expected total income 166847, got %d", dp.TotalIncome)
	}
	if dp.TotalExpenses != 350000+77000 {
		t.Errorf("expected total expenses %d, got %d", 350000+77000, dp.TotalExpenses)
	}
	if dp.Balance != 166847-(350000+77000) {
		t.Errorf("expected balance %d, got %d", 166847-(350000+77000), dp.Balance)
	}
}

func TestCalculateFinancials_PartTimeEmployee(t *testing.T) {
	start := date(2025, 1, 1)
	end := date(2025, 1, 1)

	ppID := uint(1)
	employeeContracts := []models.EmployeeContract{
		makeEmployeeContract(1, 1, date(2024, 1, 1), nil, "qualified", "S8a", 3, 20.0, ppID),
	}
	payPlans := map[uint]*models.PayPlan{
		ppID: makePayPlan(ppID, []models.PayPlanPeriod{
			makePayPlanPeriod(1, date(2024, 1, 1), nil, 39.0, 2200, []models.PayPlanEntry{
				makePayPlanEntry("S8a", 3, 390000), // 3900.00 EUR
			}),
		}),
	}

	result := calculateFinancials(nil, employeeContracts, payPlans, nil, nil, start, end)
	if len(result) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(result))
	}
	// Gross: round(390000 * 20/39) = round(200000) = 200000
	if result[0].GrossSalary != 200000 {
		t.Errorf("expected gross 200000, got %d", result[0].GrossSalary)
	}
}

func TestCalculateFinancials_BudgetItems(t *testing.T) {
	start := date(2025, 1, 1)
	end := date(2025, 1, 1)

	children := []models.Child{
		makeChild(1, date(2022, 1, 1), "male", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), nil, nil),
		}),
		makeChild(2, date(2021, 1, 1), "female", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), nil, nil),
		}),
	}
	budgetItems := []models.BudgetItem{
		makeBudgetItem("Rent", "expense", false, []models.BudgetItemEntry{
			makeBudgetItemEntry(date(2024, 1, 1), nil, 200000), // 2000 EUR fixed
		}),
		makeBudgetItem("Meal fees", "income", true, []models.BudgetItemEntry{
			makeBudgetItemEntry(date(2024, 1, 1), nil, 5000), // 50 EUR per child
		}),
	}

	result := calculateFinancials(children, nil, nil, nil, budgetItems, start, end)
	if len(result) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(result))
	}
	dp := result[0]

	// Meal fees: 5000 * 2 children = 10000
	if dp.BudgetIncome != 10000 {
		t.Errorf("expected budget income 10000, got %d", dp.BudgetIncome)
	}
	// Rent: 200000 fixed
	if dp.BudgetExpenses != 200000 {
		t.Errorf("expected budget expenses 200000, got %d", dp.BudgetExpenses)
	}
	if dp.TotalIncome != 10000 {
		t.Errorf("expected total income 10000, got %d", dp.TotalIncome)
	}
	if dp.TotalExpenses != 200000 {
		t.Errorf("expected total expenses 200000, got %d", dp.TotalExpenses)
	}

	// Budget item details
	if len(dp.BudgetItemDetails) != 2 {
		t.Fatalf("expected 2 budget item details, got %d", len(dp.BudgetItemDetails))
	}
}

func TestCalculateFinancials_FundingDetails(t *testing.T) {
	start := date(2025, 1, 1)
	end := date(2025, 1, 1)

	children := []models.Child{
		makeChild(1, date(2022, 1, 1), "male", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), nil, models.ContractProperties{
				"care_type":   "ganztag",
				"supplements": []any{"ndh"},
			}),
		}),
		makeChild(2, date(2021, 1, 1), "female", []models.ChildContract{
			makeChildContract(date(2024, 1, 1), nil, models.ContractProperties{
				"care_type": "ganztag",
			}),
		}),
	}
	periods := []models.GovernmentFundingPeriod{
		makeFundingPeriod(date(2024, 1, 1), nil, 39, []models.GovernmentFundingProperty{
			makeFundingProp("care_type", "ganztag", "Ganztag", 100000, 0, nil, nil),
			makeFundingProp("supplements", "ndh", "NDH", 30000, 0, nil, nil),
		}),
	}

	result := calculateFinancials(children, nil, nil, periods, nil, start, end)
	dp := result[0]

	// 2 children × 100000 ganztag + 1 child × 30000 ndh = 230000
	if dp.FundingIncome != 230000 {
		t.Errorf("expected funding income 230000, got %d", dp.FundingIncome)
	}

	// Funding details should be sorted by key, value
	if len(dp.FundingDetails) != 2 {
		t.Fatalf("expected 2 funding details, got %d", len(dp.FundingDetails))
	}
	if dp.FundingDetails[0].Key != "care_type" || dp.FundingDetails[0].AmountCents != 200000 {
		t.Errorf("expected care_type detail with 200000, got %s:%d", dp.FundingDetails[0].Key, dp.FundingDetails[0].AmountCents)
	}
	if dp.FundingDetails[1].Key != "supplements" || dp.FundingDetails[1].AmountCents != 30000 {
		t.Errorf("expected supplements detail with 30000, got %s:%d", dp.FundingDetails[1].Key, dp.FundingDetails[1].AmountCents)
	}
}

func TestCalculateFinancials_SalaryDetails(t *testing.T) {
	start := date(2025, 1, 1)
	end := date(2025, 1, 1)

	ppID := uint(1)
	contracts := []models.EmployeeContract{
		makeEmployeeContract(1, 1, date(2024, 1, 1), nil, "qualified", "S8a", 3, 39.0, ppID),
		makeEmployeeContract(2, 2, date(2024, 1, 1), nil, "supplementary", "S3", 1, 20.0, ppID),
	}
	payPlans := map[uint]*models.PayPlan{
		ppID: makePayPlan(ppID, []models.PayPlanPeriod{
			makePayPlanPeriod(1, date(2024, 1, 1), nil, 39.0, 2000, []models.PayPlanEntry{
				makePayPlanEntry("S8a", 3, 350000),
				makePayPlanEntry("S3", 1, 250000),
			}),
		}),
	}

	result := calculateFinancials(nil, contracts, payPlans, nil, nil, start, end)
	dp := result[0]

	if len(dp.SalaryDetails) != 2 {
		t.Fatalf("expected 2 salary details, got %d", len(dp.SalaryDetails))
	}
	// Sorted by staff category: qualified, supplementary
	if dp.SalaryDetails[0].StaffCategory != "qualified" {
		t.Errorf("expected first salary detail 'qualified', got %q", dp.SalaryDetails[0].StaffCategory)
	}
	if dp.SalaryDetails[0].GrossSalary != 350000 {
		t.Errorf("qualified gross: expected 350000, got %d", dp.SalaryDetails[0].GrossSalary)
	}
}

func TestCalculateFinancials_NoPayPlan(t *testing.T) {
	// Employee has pay plan ID that doesn't exist in the map
	contracts := []models.EmployeeContract{
		makeEmployeeContract(1, 1, date(2024, 1, 1), nil, "qualified", "S8a", 3, 39.0, 999),
	}
	result := calculateFinancials(nil, contracts, nil, nil, nil, date(2025, 1, 1), date(2025, 1, 1))
	if result[0].GrossSalary != 0 {
		t.Errorf("expected 0 gross when no pay plan, got %d", result[0].GrossSalary)
	}
	if result[0].StaffCount != 1 {
		t.Errorf("expected 1 staff even without pay plan, got %d", result[0].StaffCount)
	}
}

func TestCalculateFinancials_MultipleMonths(t *testing.T) {
	result := calculateFinancials(nil, nil, nil, nil, nil, date(2025, 1, 1), date(2025, 6, 1))
	if len(result) != 6 {
		t.Errorf("expected 6 data points, got %d", len(result))
	}
	if result[0].Date != "2025-01-01" {
		t.Errorf("expected first date '2025-01-01', got %q", result[0].Date)
	}
	if result[5].Date != "2025-06-01" {
		t.Errorf("expected last date '2025-06-01', got %q", result[5].Date)
	}
}

func TestCalculateFinancials_Empty(t *testing.T) {
	result := calculateFinancials(nil, nil, nil, nil, nil, date(2025, 1, 1), date(2025, 1, 1))
	if len(result) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(result))
	}
	dp := result[0]
	if dp.FundingIncome != 0 || dp.GrossSalary != 0 || dp.Balance != 0 {
		t.Error("expected all zeros for empty data")
	}
}

func TestCalculateFinancials_BudgetItemExpiredEntry(t *testing.T) {
	budgetItems := []models.BudgetItem{
		makeBudgetItem("Old expense", "expense", false, []models.BudgetItemEntry{
			makeBudgetItemEntry(date(2023, 1, 1), datePtr(2024, 12, 31), 100000),
		}),
	}
	result := calculateFinancials(nil, nil, nil, nil, budgetItems, date(2025, 1, 1), date(2025, 1, 1))
	if result[0].BudgetExpenses != 0 {
		t.Errorf("expected 0 for expired budget entry, got %d", result[0].BudgetExpenses)
	}
}

// ========================================================================
// buildFundingPeriodIndex
// ========================================================================

func TestBuildFundingPeriodIndex(t *testing.T) {
	periods := []models.GovernmentFundingPeriod{
		makeFundingPeriod(date(2024, 1, 1), datePtr(2024, 12, 31), 39, nil),
		makeFundingPeriod(date(2025, 1, 1), nil, 39, nil),
	}

	idx := buildFundingPeriodIndex(periods, date(2024, 11, 1), date(2025, 2, 1))

	// Nov 2024: first period
	if idx[date(2024, 11, 1)] == nil {
		t.Error("expected a period for Nov 2024")
	}
	// Jan 2025: second period
	if idx[date(2025, 1, 1)] == nil {
		t.Error("expected a period for Jan 2025")
	}
}

// ========================================================================
// buildPayPlanIndex
// ========================================================================

func TestBuildPayPlanIndex(t *testing.T) {
	ppID := uint(1)
	payPlans := map[uint]*models.PayPlan{
		ppID: makePayPlan(ppID, []models.PayPlanPeriod{
			makePayPlanPeriod(10, date(2024, 1, 1), nil, 39.0, 2200, []models.PayPlanEntry{
				makePayPlanEntry("S8a", 3, 350000),
			}),
		}),
	}

	idx := buildPayPlanIndex(payPlans, date(2025, 1, 1), date(2025, 1, 1))

	resolved := idx[ppID][date(2025, 1, 1)]
	if resolved == nil {
		t.Fatal("expected resolved pay plan period")
	}
	entry := resolved.entryIndex[gradeStepKey{"S8a", 3}]
	if entry == nil {
		t.Fatal("expected entry for S8a/3")
	}
	if entry.MonthlyAmount != 350000 {
		t.Errorf("expected 350000, got %d", entry.MonthlyAmount)
	}
}

// ========================================================================
// intPtr
// ========================================================================

func TestIntPtr(t *testing.T) {
	p := intPtr(42)
	if *p != 42 {
		t.Errorf("expected 42, got %d", *p)
	}
}
