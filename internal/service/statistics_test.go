package service

import (
	"context"
	"math"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func createStatisticsService(db *gorm.DB) *StatisticsService {
	childStore := store.NewChildStore(db)
	employeeStore := store.NewEmployeeStore(db)
	orgStore := store.NewOrganizationStore(db)
	fundingStore := store.NewGovernmentFundingStore(db)
	payPlanStore := store.NewPayPlanStore(db)
	budgetItemStore := store.NewBudgetItemStore(db)
	return NewStatisticsService(childStore, employeeStore, orgStore, fundingStore, payPlanStore, budgetItemStore)
}

func createTestEmployeeContractWithCategory(t *testing.T, db *gorm.DB, employeeID uint, payplanID uint, from time.Time, to *time.Time, weeklyHours float64, staffCategory string, sectionID uint) *models.EmployeeContract {
	t.Helper()
	contract := &models.EmployeeContract{
		EmployeeID: employeeID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: from, To: to},
			SectionID: sectionID,
		},
		StaffCategory: staffCategory,
		WeeklyHours:   weeklyHours,
		PayPlanID:     payplanID,
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("failed to create test employee contract: %v", err)
	}
	return contract
}

func createTestChildContract(t *testing.T, db *gorm.DB, childID uint, from time.Time, to *time.Time, sectionID uint, properties models.ContractProperties) *models.ChildContract {
	t.Helper()
	contract := &models.ChildContract{
		ChildID: childID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: from, To: to},
			SectionID:  sectionID,
			Properties: properties,
		},
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("failed to create test child contract: %v", err)
	}
	return contract
}

func createTestFundingPropertyWithRequirement(t *testing.T, db *gorm.DB, periodID uint, key, value string, requirement float64, minAge, maxAge int) *models.GovernmentFundingProperty {
	t.Helper()
	var minAgePtr, maxAgePtr *int
	if minAge >= 0 {
		minAgePtr = &minAge
	}
	if maxAge >= 0 {
		maxAgePtr = &maxAge
	}
	prop := &models.GovernmentFundingProperty{
		PeriodID:    periodID,
		Key:         key,
		Value:       value,
		Payment:     10000,
		Requirement: requirement,
		MinAge:      minAgePtr,
		MaxAge:      maxAgePtr,
	}
	if err := db.Create(prop).Error; err != nil {
		t.Fatalf("failed to create test funding property: %v", err)
	}
	return prop
}

func almostEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestStatisticsService_GetStaffingHours_Basic(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	// Create org with state "berlin"
	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Create government funding with period covering 2024
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)

	// Funding property: care_type=ganztag, requirement=0.25, ages 0-6
	createTestFundingPropertyWithRequirement(t, db, period.ID, "care_type", "ganztag", 0.25, 0, 6)

	section := getDefaultSection(t, db, org.ID)

	// 2 children with contracts from 2024-01-01, ongoing
	child1 := createTestChild(t, db, "Child", "One", org.ID)
	child2 := createTestChild(t, db, "Child", "Two", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	props := models.ContractProperties{"care_type": "ganztag"}
	createTestChildContract(t, db, child1.ID, contractFrom, nil, section.ID, props)
	createTestChildContract(t, db, child2.ID, contractFrom, nil, section.ID, props)

	// 2 employees with qualified contracts, 30 hours each
	payplan := createTestPayPlan(t, db, "TV-L", org.ID)
	emp1 := createTestEmployee(t, db, "Emp", "One", org.ID)
	emp2 := createTestEmployee(t, db, "Emp", "Two", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp1.ID, payplan.ID, contractFrom, nil, 30.0, "qualified", section.ID)
	createTestEmployeeContractWithCategory(t, db, emp2.ID, payplan.ID, contractFrom, nil, 30.0, "qualified", section.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetStaffingHours(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 6 data points: Jan, Feb, Mar, Apr, May, Jun
	if len(result.DataPoints) != 6 {
		t.Fatalf("expected 6 data points, got %d", len(result.DataPoints))
	}

	for i, dp := range result.DataPoints {
		// RequiredHours = 2 children * 0.25 requirement * 39.0 full-time hours = 19.5
		if !almostEqual(dp.RequiredHours, 19.5, 0.01) {
			t.Errorf("data point %d: RequiredHours = %v, want 19.5", i, dp.RequiredHours)
		}
		// AvailableHours = 2 employees * 30.0 hours = 60.0
		if !almostEqual(dp.AvailableHours, 60.0, 0.01) {
			t.Errorf("data point %d: AvailableHours = %v, want 60.0", i, dp.AvailableHours)
		}
		if dp.ChildCount != 2 {
			t.Errorf("data point %d: ChildCount = %d, want 2", i, dp.ChildCount)
		}
		if dp.StaffCount != 2 {
			t.Errorf("data point %d: StaffCount = %d, want 2", i, dp.StaffCount)
		}
	}
}

func TestStatisticsService_GetStaffingHours_NoChildren(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyWithRequirement(t, db, period.ID, "care_type", "ganztag", 0.25, 0, 6)

	section := getDefaultSection(t, db, org.ID)

	// Only employees, no children
	payplan := createTestPayPlan(t, db, "TV-L", org.ID)
	emp := createTestEmployee(t, db, "Emp", "One", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, contractFrom, nil, 30.0, "qualified", section.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetStaffingHours(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for i, dp := range result.DataPoints {
		if !almostEqual(dp.RequiredHours, 0.0, 0.01) {
			t.Errorf("data point %d: RequiredHours = %v, want 0.0", i, dp.RequiredHours)
		}
		if !almostEqual(dp.AvailableHours, 30.0, 0.01) {
			t.Errorf("data point %d: AvailableHours = %v, want 30.0", i, dp.AvailableHours)
		}
		if dp.ChildCount != 0 {
			t.Errorf("data point %d: ChildCount = %d, want 0", i, dp.ChildCount)
		}
		if dp.StaffCount != 1 {
			t.Errorf("data point %d: StaffCount = %d, want 1", i, dp.StaffCount)
		}
	}
}

func TestStatisticsService_GetStaffingHours_NoEmployees(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyWithRequirement(t, db, period.ID, "care_type", "ganztag", 0.25, 0, 6)

	section := getDefaultSection(t, db, org.ID)

	// Only children, no employees
	child := createTestChild(t, db, "Child", "One", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	props := models.ContractProperties{"care_type": "ganztag"}
	createTestChildContract(t, db, child.ID, contractFrom, nil, section.ID, props)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetStaffingHours(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for i, dp := range result.DataPoints {
		if !almostEqual(dp.AvailableHours, 0.0, 0.01) {
			t.Errorf("data point %d: AvailableHours = %v, want 0.0", i, dp.AvailableHours)
		}
		// RequiredHours should be > 0 since child has contract with matching funding
		if !almostEqual(dp.RequiredHours, 0.25*39.0, 0.01) {
			t.Errorf("data point %d: RequiredHours = %v, want %v", i, dp.RequiredHours, 0.25*39.0)
		}
		if dp.ChildCount != 1 {
			t.Errorf("data point %d: ChildCount = %d, want 1", i, dp.ChildCount)
		}
		if dp.StaffCount != 0 {
			t.Errorf("data point %d: StaffCount = %d, want 0", i, dp.StaffCount)
		}
	}
}

func TestStatisticsService_GetStaffingHours_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetStaffingHours(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should still return data points (3: Jan, Feb, Mar)
	if len(result.DataPoints) != 3 {
		t.Fatalf("expected 3 data points, got %d", len(result.DataPoints))
	}

	for i, dp := range result.DataPoints {
		if !almostEqual(dp.RequiredHours, 0.0, 0.01) {
			t.Errorf("data point %d: RequiredHours = %v, want 0.0", i, dp.RequiredHours)
		}
		if !almostEqual(dp.AvailableHours, 0.0, 0.01) {
			t.Errorf("data point %d: AvailableHours = %v, want 0.0", i, dp.AvailableHours)
		}
		if dp.ChildCount != 0 {
			t.Errorf("data point %d: ChildCount = %d, want 0", i, dp.ChildCount)
		}
		if dp.StaffCount != 0 {
			t.Errorf("data point %d: StaffCount = %d, want 0", i, dp.StaffCount)
		}
	}
}

func TestStatisticsService_GetStaffingHours_SectionFilter(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyWithRequirement(t, db, period.ID, "care_type", "ganztag", 0.25, 0, 6)

	section1 := createTestSection(t, db, "Krippe", org.ID, false)
	section2 := createTestSection(t, db, "Elementar", org.ID, false)

	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	props := models.ContractProperties{"care_type": "ganztag"}

	// Child in section 1
	child1 := createTestChild(t, db, "Child", "One", org.ID)
	createTestChildContract(t, db, child1.ID, contractFrom, nil, section1.ID, props)

	// Child in section 2
	child2 := createTestChild(t, db, "Child", "Two", org.ID)
	createTestChildContract(t, db, child2.ID, contractFrom, nil, section2.ID, props)

	// Employee in section 1
	payplan := createTestPayPlan(t, db, "TV-L", org.ID)
	emp1 := createTestEmployee(t, db, "Emp", "One", org.ID)
	empContract1 := &models.EmployeeContract{
		EmployeeID: emp1.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: contractFrom, To: nil},
			SectionID: section1.ID,
		},
		StaffCategory: "qualified",
		WeeklyHours:   30.0,
		PayPlanID:     payplan.ID,
	}
	if err := db.Create(empContract1).Error; err != nil {
		t.Fatalf("failed to create employee contract: %v", err)
	}

	// Employee in section 2
	emp2 := createTestEmployee(t, db, "Emp", "Two", org.ID)
	empContract2 := &models.EmployeeContract{
		EmployeeID: emp2.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: contractFrom, To: nil},
			SectionID: section2.ID,
		},
		StaffCategory: "qualified",
		WeeklyHours:   25.0,
		PayPlanID:     payplan.ID,
	}
	if err := db.Create(empContract2).Error; err != nil {
		t.Fatalf("failed to create employee contract: %v", err)
	}

	// Filter by section 1
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetStaffingHours(ctx, org.ID, &from, &to, &section1.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for i, dp := range result.DataPoints {
		// Only 1 child in section 1
		if dp.ChildCount != 1 {
			t.Errorf("data point %d: ChildCount = %d, want 1", i, dp.ChildCount)
		}
		// Only 1 employee in section 1, 30 hours
		if dp.StaffCount != 1 {
			t.Errorf("data point %d: StaffCount = %d, want 1", i, dp.StaffCount)
		}
		if !almostEqual(dp.AvailableHours, 30.0, 0.01) {
			t.Errorf("data point %d: AvailableHours = %v, want 30.0", i, dp.AvailableHours)
		}
		// 1 child * 0.25 * 39.0 = 9.75
		if !almostEqual(dp.RequiredHours, 9.75, 0.01) {
			t.Errorf("data point %d: RequiredHours = %v, want 9.75", i, dp.RequiredHours)
		}
	}
}

func TestStatisticsService_GetStaffingHours_CustomDateRange(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Range from 2024-03 to 2024-08 should give 6 data points
	from := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetStaffingHours(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 6 data points: Mar, Apr, May, Jun, Jul, Aug
	if len(result.DataPoints) != 6 {
		t.Fatalf("expected 6 data points, got %d", len(result.DataPoints))
	}

	// Verify first and last dates
	if result.DataPoints[0].Date != "2024-03-01" {
		t.Errorf("first data point date = %v, want 2024-03-01", result.DataPoints[0].Date)
	}
	if result.DataPoints[5].Date != "2024-08-01" {
		t.Errorf("last data point date = %v, want 2024-08-01", result.DataPoints[5].Date)
	}
}

func TestStatisticsService_GetStaffingHours_DefaultDateRange(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Pass nil for from/to to use default Kita year range:
	// 1 month before previous Kita year through end of next Kita year
	result, err := svc.GetStaffingHours(ctx, org.ID, nil, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the range spans 3 Kita years + 1 extra month
	now := time.Now()
	kitaYearStartYear := now.Year()
	if now.Month() < time.August {
		kitaYearStartYear--
	}
	from := time.Date(kitaYearStartYear-1, time.July, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(kitaYearStartYear+2, time.August, 1, 0, 0, 0, 0, time.UTC)
	expectedPoints := 0
	for d := from; !d.After(to); d = d.AddDate(0, 1, 0) {
		expectedPoints++
	}
	if len(result.DataPoints) != expectedPoints {
		t.Errorf("expected %d data points, got %d", expectedPoints, len(result.DataPoints))
	}
}

func TestStatisticsService_GetStaffingHours_ContractStartsMidRange(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyWithRequirement(t, db, period.ID, "care_type", "ganztag", 0.25, 0, 6)

	section := getDefaultSection(t, db, org.ID)

	// Child contract starts 2024-03-01 (mid-range)
	child := createTestChild(t, db, "Child", "One", org.ID)
	contractFrom := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	props := models.ContractProperties{"care_type": "ganztag"}
	createTestChildContract(t, db, child.ID, contractFrom, nil, section.ID, props)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetStaffingHours(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 6 {
		t.Fatalf("expected 6 data points, got %d", len(result.DataPoints))
	}

	// Jan and Feb: child not yet active
	for i := 0; i < 2; i++ {
		if result.DataPoints[i].ChildCount != 0 {
			t.Errorf("data point %d: ChildCount = %d, want 0 (contract not started)", i, result.DataPoints[i].ChildCount)
		}
		if !almostEqual(result.DataPoints[i].RequiredHours, 0.0, 0.01) {
			t.Errorf("data point %d: RequiredHours = %v, want 0.0", i, result.DataPoints[i].RequiredHours)
		}
	}

	// Mar through Jun: child active
	for i := 2; i < 6; i++ {
		if result.DataPoints[i].ChildCount != 1 {
			t.Errorf("data point %d: ChildCount = %d, want 1", i, result.DataPoints[i].ChildCount)
		}
		if !almostEqual(result.DataPoints[i].RequiredHours, 0.25*39.0, 0.01) {
			t.Errorf("data point %d: RequiredHours = %v, want %v", i, result.DataPoints[i].RequiredHours, 0.25*39.0)
		}
	}
}

func TestStatisticsService_GetStaffingHours_OngoingContracts(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyWithRequirement(t, db, period.ID, "care_type", "ganztag", 0.25, 0, 6)

	section := getDefaultSection(t, db, org.ID)

	// Contracts with To = nil (ongoing)
	child := createTestChild(t, db, "Child", "One", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	props := models.ContractProperties{"care_type": "ganztag"}
	createTestChildContract(t, db, child.ID, contractFrom, nil, section.ID, props)

	payplan := createTestPayPlan(t, db, "TV-L", org.ID)
	emp := createTestEmployee(t, db, "Emp", "One", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, contractFrom, nil, 35.0, "qualified", section.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetStaffingHours(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// All data points should have the child and employee active
	for i, dp := range result.DataPoints {
		if dp.ChildCount != 1 {
			t.Errorf("data point %d: ChildCount = %d, want 1", i, dp.ChildCount)
		}
		if dp.StaffCount != 1 {
			t.Errorf("data point %d: StaffCount = %d, want 1", i, dp.StaffCount)
		}
		if !almostEqual(dp.AvailableHours, 35.0, 0.01) {
			t.Errorf("data point %d: AvailableHours = %v, want 35.0", i, dp.AvailableHours)
		}
	}
}

func TestStatisticsService_GetStaffingHours_NonPedagogicalExcluded(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	section := getDefaultSection(t, db, org.ID)

	payplan := createTestPayPlan(t, db, "TV-L", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Qualified employee (should be counted)
	emp1 := createTestEmployee(t, db, "Emp", "Qualified", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp1.ID, payplan.ID, contractFrom, nil, 30.0, "qualified", section.ID)

	// Supplementary employee (should be counted)
	emp2 := createTestEmployee(t, db, "Emp", "Supplementary", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp2.ID, payplan.ID, contractFrom, nil, 20.0, "supplementary", section.ID)

	// Non-pedagogical employee (should NOT be counted)
	emp3 := createTestEmployee(t, db, "Emp", "Kitchen", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp3.ID, payplan.ID, contractFrom, nil, 40.0, "non_pedagogical", section.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetStaffingHours(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for i, dp := range result.DataPoints {
		// Only qualified + supplementary = 30 + 20 = 50 hours
		if !almostEqual(dp.AvailableHours, 50.0, 0.01) {
			t.Errorf("data point %d: AvailableHours = %v, want 50.0", i, dp.AvailableHours)
		}
		// Only 2 pedagogical staff counted
		if dp.StaffCount != 2 {
			t.Errorf("data point %d: StaffCount = %d, want 2", i, dp.StaffCount)
		}
	}
}

func TestStatisticsService_GetStaffingHours_NoFundingForState(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	// Org with state "hamburg" - no funding exists for this state
	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "hamburg")

	section := getDefaultSection(t, db, org.ID)

	// Child with contract
	child := createTestChild(t, db, "Child", "One", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	props := models.ContractProperties{"care_type": "ganztag"}
	createTestChildContract(t, db, child.ID, contractFrom, nil, section.ID, props)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetStaffingHours(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for i, dp := range result.DataPoints {
		// No funding for state, so required hours = 0
		if !almostEqual(dp.RequiredHours, 0.0, 0.01) {
			t.Errorf("data point %d: RequiredHours = %v, want 0.0", i, dp.RequiredHours)
		}
		// Child still counted
		if dp.ChildCount != 1 {
			t.Errorf("data point %d: ChildCount = %d, want 1", i, dp.ChildCount)
		}
	}
}

// ============================================================
// GetFinancials tests
// ============================================================

func TestStatisticsService_GetFinancials_Basic(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Government funding: care_type=ganztag -> 166847 cents payment
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, fundingPeriod.ID, "care_type", "ganztag", 166847, 0.25, 0, 6)

	section := getDefaultSection(t, db, org.ID)

	// 1 child with ganztag contract
	child := createTestChild(t, db, "Child", "One", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, contractFrom, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	// Pay plan: S8a step 3, 350000 cents/month, 39h full-time, 22% employer contrib
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	ppPeriod := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, 2200)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S8a", 3, 350000, nil)

	// 1 employee: S8a step 3, 39h (full-time)
	emp := createTestEmployee(t, db, "Emp", "One", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, contractFrom, nil, 39.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]interface{}{"grade": "S8a", "step": 3})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 3 {
		t.Fatalf("expected 3 data points, got %d", len(result.DataPoints))
	}

	for i, dp := range result.DataPoints {
		// Income: 1 child * 166847 cents
		if dp.FundingIncome != 166847 {
			t.Errorf("dp %d: FundingIncome = %d, want 166847", i, dp.FundingIncome)
		}
		// Gross salary: 350000 * 39/39 = 350000
		if dp.GrossSalary != 350000 {
			t.Errorf("dp %d: GrossSalary = %d, want 350000", i, dp.GrossSalary)
		}
		// Employer costs: 350000 * 2200/10000 = 77000
		if dp.EmployerCosts != 77000 {
			t.Errorf("dp %d: EmployerCosts = %d, want 77000", i, dp.EmployerCosts)
		}
		// Totals
		if dp.TotalIncome != 166847 {
			t.Errorf("dp %d: TotalIncome = %d, want 166847", i, dp.TotalIncome)
		}
		expectedExpenses := 350000 + 77000
		if dp.TotalExpenses != expectedExpenses {
			t.Errorf("dp %d: TotalExpenses = %d, want %d", i, dp.TotalExpenses, expectedExpenses)
		}
		expectedBalance := 166847 - expectedExpenses
		if dp.Balance != expectedBalance {
			t.Errorf("dp %d: Balance = %d, want %d", i, dp.Balance, expectedBalance)
		}
		if dp.ChildCount != 1 {
			t.Errorf("dp %d: ChildCount = %d, want 1", i, dp.ChildCount)
		}
		if dp.StaffCount != 1 {
			t.Errorf("dp %d: StaffCount = %d, want 1", i, dp.StaffCount)
		}
	}
}

func TestStatisticsService_GetFinancials_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 3 {
		t.Fatalf("expected 3 data points, got %d", len(result.DataPoints))
	}

	for i, dp := range result.DataPoints {
		if dp.FundingIncome != 0 {
			t.Errorf("dp %d: FundingIncome = %d, want 0", i, dp.FundingIncome)
		}
		if dp.GrossSalary != 0 {
			t.Errorf("dp %d: GrossSalary = %d, want 0", i, dp.GrossSalary)
		}
		if dp.EmployerCosts != 0 {
			t.Errorf("dp %d: EmployerCosts = %d, want 0", i, dp.EmployerCosts)
		}
		if dp.TotalIncome != 0 {
			t.Errorf("dp %d: TotalIncome = %d, want 0", i, dp.TotalIncome)
		}
		if dp.TotalExpenses != 0 {
			t.Errorf("dp %d: TotalExpenses = %d, want 0", i, dp.TotalExpenses)
		}
		if dp.Balance != 0 {
			t.Errorf("dp %d: Balance = %d, want 0", i, dp.Balance)
		}
		if dp.ChildCount != 0 {
			t.Errorf("dp %d: ChildCount = %d, want 0", i, dp.ChildCount)
		}
		if dp.StaffCount != 0 {
			t.Errorf("dp %d: StaffCount = %d, want 0", i, dp.StaffCount)
		}
	}
}

func TestStatisticsService_GetFinancials_ProRataSalary(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	section := getDefaultSection(t, db, org.ID)

	// Pay plan: 39h full-time, 350000 cents/month at S8a step 3, no employer contrib
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	ppPeriod := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, 0)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S8a", 3, 350000, nil)

	// Employee works 30h/week (part-time)
	emp := createTestEmployee(t, db, "Emp", "PartTime", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, contractFrom, nil, 30.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]interface{}{"grade": "S8a", "step": 3})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(result.DataPoints))
	}

	dp := result.DataPoints[0]
	// Pro-rated: 350000 * 30/39 = 269231 (rounded)
	expected := int(math.Round(350000.0 * 30.0 / 39.0))
	if dp.GrossSalary != expected {
		t.Errorf("GrossSalary = %d, want %d", dp.GrossSalary, expected)
	}
}

func TestStatisticsService_GetFinancials_EmployerContribution(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	section := getDefaultSection(t, db, org.ID)

	// Pay plan: 39h, 400000 cents/month, 22.50% employer contribution (2250 hundredths)
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	ppPeriod := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, 2250)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S11b", 5, 400000, nil)

	emp := createTestEmployee(t, db, "Emp", "One", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, contractFrom, nil, 39.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]interface{}{"grade": "S11b", "step": 5})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if dp.GrossSalary != 400000 {
		t.Errorf("GrossSalary = %d, want 400000", dp.GrossSalary)
	}
	// Employer: 400000 * 2250/10000 = 90000
	expectedContrib := int(math.Round(400000.0 * 2250.0 / 10000.0))
	if dp.EmployerCosts != expectedContrib {
		t.Errorf("EmployerCosts = %d, want %d", dp.EmployerCosts, expectedContrib)
	}
}

func TestStatisticsService_GetFinancials_MissingPayPlanEntry(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	section := getDefaultSection(t, db, org.ID)

	// Pay plan with S8a step 3, but employee is S9 step 1 (no matching entry)
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	ppPeriod := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, 2200)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S8a", 3, 350000, nil)

	emp := createTestEmployee(t, db, "Emp", "One", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, contractFrom, nil, 39.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]interface{}{"grade": "S9", "step": 1})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	// No matching entry -> salary should be 0, but staff still counted
	if dp.GrossSalary != 0 {
		t.Errorf("GrossSalary = %d, want 0 (no matching pay plan entry)", dp.GrossSalary)
	}
	if dp.EmployerCosts != 0 {
		t.Errorf("EmployerCosts = %d, want 0", dp.EmployerCosts)
	}
	if dp.StaffCount != 1 {
		t.Errorf("StaffCount = %d, want 1 (employee still counted)", dp.StaffCount)
	}
}

func TestStatisticsService_GetFinancials_NoPayPlanPeriodForDate(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	section := getDefaultSection(t, db, org.ID)

	// Pay plan period covers only 2025, but we query 2024
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	ppPeriod := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, 2200)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S8a", 3, 350000, nil)

	emp := createTestEmployee(t, db, "Emp", "One", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, contractFrom, nil, 39.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]interface{}{"grade": "S8a", "step": 3})

	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if dp.GrossSalary != 0 {
		t.Errorf("GrossSalary = %d, want 0 (no pay plan period for 2024)", dp.GrossSalary)
	}
	if dp.StaffCount != 1 {
		t.Errorf("StaffCount = %d, want 1", dp.StaffCount)
	}
}

func TestStatisticsService_GetFinancials_NoFundingForState(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	// Org with state "hamburg" - no funding exists
	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "hamburg")

	section := getDefaultSection(t, db, org.ID)

	child := createTestChild(t, db, "Child", "One", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, contractFrom, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if dp.FundingIncome != 0 {
		t.Errorf("FundingIncome = %d, want 0 (no funding for state)", dp.FundingIncome)
	}
	if dp.ChildCount != 1 {
		t.Errorf("ChildCount = %d, want 1 (child still counted)", dp.ChildCount)
	}
}

func TestStatisticsService_GetFinancials_AllStaffIncluded(t *testing.T) {
	// Financials should include ALL staff categories (not just pedagogical)
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	section := getDefaultSection(t, db, org.ID)

	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	ppPeriod := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, 0)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S8a", 3, 300000, nil)

	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Qualified staff
	emp1 := createTestEmployee(t, db, "Emp", "Qualified", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp1.ID, payplan.ID, contractFrom, nil, 39.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp1.ID).Updates(map[string]interface{}{"grade": "S8a", "step": 3})

	// Non-pedagogical staff (kitchen, admin, etc.)
	emp2 := createTestEmployee(t, db, "Emp", "Kitchen", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp2.ID, payplan.ID, contractFrom, nil, 39.0, "non_pedagogical", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp2.ID).Updates(map[string]interface{}{"grade": "S8a", "step": 3})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	// Both employees should contribute to salary
	if dp.GrossSalary != 600000 {
		t.Errorf("GrossSalary = %d, want 600000 (2 employees * 300000)", dp.GrossSalary)
	}
	if dp.StaffCount != 2 {
		t.Errorf("StaffCount = %d, want 2 (all staff categories)", dp.StaffCount)
	}
}

func TestStatisticsService_GetFinancials_ContractStartsMidRange(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, fundingPeriod.ID, "care_type", "ganztag", 100000, 0.25, 0, 6)

	section := getDefaultSection(t, db, org.ID)

	// Child contract starts March (mid-range)
	child := createTestChild(t, db, "Child", "One", org.ID)
	contractFrom := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, contractFrom, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Jan, Feb: no child income
	for i := 0; i < 2; i++ {
		if result.DataPoints[i].FundingIncome != 0 {
			t.Errorf("dp %d: FundingIncome = %d, want 0 (contract not started)", i, result.DataPoints[i].FundingIncome)
		}
		if result.DataPoints[i].ChildCount != 0 {
			t.Errorf("dp %d: ChildCount = %d, want 0", i, result.DataPoints[i].ChildCount)
		}
	}
	// Mar-Jun: child active
	for i := 2; i < 6; i++ {
		if result.DataPoints[i].FundingIncome != 100000 {
			t.Errorf("dp %d: FundingIncome = %d, want 100000", i, result.DataPoints[i].FundingIncome)
		}
		if result.DataPoints[i].ChildCount != 1 {
			t.Errorf("dp %d: ChildCount = %d, want 1", i, result.DataPoints[i].ChildCount)
		}
	}
}

func TestStatisticsService_GetFinancials_MultipleChildren(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, fundingPeriod.ID, "care_type", "ganztag", 80000, 0.25, 0, 6)
	createTestFundingPropertyFull(t, db, fundingPeriod.ID, "care_type", "halbtag", 40000, 0.12, 0, 6)

	section := getDefaultSection(t, db, org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// 2 ganztag children
	child1 := createTestChild(t, db, "Child", "One", org.ID)
	createTestChildContract(t, db, child1.ID, contractFrom, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})
	child2 := createTestChild(t, db, "Child", "Two", org.ID)
	createTestChildContract(t, db, child2.ID, contractFrom, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	// 1 halbtag child
	child3 := createTestChild(t, db, "Child", "Three", org.ID)
	createTestChildContract(t, db, child3.ID, contractFrom, nil, section.ID, models.ContractProperties{"care_type": "halbtag"})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	expectedIncome := 2*80000 + 1*40000
	if dp.FundingIncome != expectedIncome {
		t.Errorf("FundingIncome = %d, want %d", dp.FundingIncome, expectedIncome)
	}
	if dp.ChildCount != 3 {
		t.Errorf("ChildCount = %d, want 3", dp.ChildCount)
	}
}

func TestStatisticsService_GetFinancials_MultipleEmployees(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	section := getDefaultSection(t, db, org.ID)

	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	ppPeriod := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, 2000)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S8a", 3, 300000, nil)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S11b", 5, 450000, nil)

	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Employee 1: S8a step 3, 39h
	emp1 := createTestEmployee(t, db, "Emp", "One", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp1.ID, payplan.ID, contractFrom, nil, 39.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp1.ID).Updates(map[string]interface{}{"grade": "S8a", "step": 3})

	// Employee 2: S11b step 5, 20h (part-time)
	emp2 := createTestEmployee(t, db, "Emp", "Two", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp2.ID, payplan.ID, contractFrom, nil, 20.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp2.ID).Updates(map[string]interface{}{"grade": "S11b", "step": 5})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	// Emp1: 300000 * 39/39 = 300000
	// Emp2: 450000 * 20/39 = 230769 (rounded)
	expectedEmp2 := int(math.Round(450000.0 * 20.0 / 39.0))
	expectedGross := 300000 + expectedEmp2
	if dp.GrossSalary != expectedGross {
		t.Errorf("GrossSalary = %d, want %d", dp.GrossSalary, expectedGross)
	}
	// Employer: each gross * 2000/10000
	expectedContrib := int(math.Round(300000.0*2000.0/10000.0)) + int(math.Round(float64(expectedEmp2)*2000.0/10000.0))
	if dp.EmployerCosts != expectedContrib {
		t.Errorf("EmployerCosts = %d, want %d", dp.EmployerCosts, expectedContrib)
	}
	if dp.StaffCount != 2 {
		t.Errorf("StaffCount = %d, want 2", dp.StaffCount)
	}
}

func TestStatisticsService_GetFinancials_BalanceNegative(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	section := getDefaultSection(t, db, org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// No children (no income), but has employee salary
	payplan := createTestPayPlan(t, db, "TVöD", org.ID)
	ppPeriod := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, contractFrom, nil, 39.0, 0)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S8a", 1, 250000, nil)

	emp := createTestEmployee(t, db, "Emp", "One", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, contractFrom, nil, 39.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]interface{}{"grade": "S8a", "step": 1})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if dp.TotalIncome != 0 {
		t.Errorf("TotalIncome = %d, want 0", dp.TotalIncome)
	}
	expectedExpenses := 250000
	if dp.TotalExpenses != expectedExpenses {
		t.Errorf("TotalExpenses = %d, want %d", dp.TotalExpenses, expectedExpenses)
	}
	if dp.Balance >= 0 {
		t.Errorf("Balance = %d, want negative", dp.Balance)
	}
	if dp.Balance != -expectedExpenses {
		t.Errorf("Balance = %d, want %d", dp.Balance, -expectedExpenses)
	}
}

func TestStatisticsService_GetFinancials_DefaultDateRange(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// nil from/to -> default Kita year range
	result, err := svc.GetFinancials(ctx, org.ID, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the range spans 3 Kita years + 1 extra month
	now := time.Now()
	kitaYearStartYear := now.Year()
	if now.Month() < time.August {
		kitaYearStartYear--
	}
	from := time.Date(kitaYearStartYear-1, time.July, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(kitaYearStartYear+2, time.August, 1, 0, 0, 0, 0, time.UTC)
	expectedPoints := 0
	for d := from; !d.After(to); d = d.AddDate(0, 1, 0) {
		expectedPoints++
	}
	if len(result.DataPoints) != expectedPoints {
		t.Errorf("expected %d data points (default range), got %d", expectedPoints, len(result.DataPoints))
	}
}

func TestStatisticsService_GetFinancials_UnmatchedChildProperty(t *testing.T) {
	// Child has a property not matching any funding property -> no income from it
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	// Funding only covers "ganztag"
	createTestFundingPropertyFull(t, db, fundingPeriod.ID, "care_type", "ganztag", 100000, 0.25, 0, 6)

	section := getDefaultSection(t, db, org.ID)

	// Child has "halbtag" - won't match
	child := createTestChild(t, db, "Child", "One", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, contractFrom, nil, section.ID, models.ContractProperties{"care_type": "halbtag"})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if dp.FundingIncome != 0 {
		t.Errorf("FundingIncome = %d, want 0 (no matching funding property)", dp.FundingIncome)
	}
	if dp.ChildCount != 1 {
		t.Errorf("ChildCount = %d, want 1 (child still counted)", dp.ChildCount)
	}
}

func TestStatisticsService_GetFinancials_EmployeeNoPayPlanEntries(t *testing.T) {
	// Employee with a pay plan that has no periods/entries should result in 0 salary
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	section := getDefaultSection(t, db, org.ID)

	// Create a pay plan with no periods or entries
	emptyPayPlan := createTestPayPlan(t, db, "Empty Pay Plan", org.ID)

	// Create employee contract with the empty pay plan
	emp := createTestEmployee(t, db, "Emp", "NoPlan", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	contract := &models.EmployeeContract{
		EmployeeID: emp.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: contractFrom},
			SectionID: section.ID,
		},
		StaffCategory: "qualified",
		WeeklyHours:   39.0,
		PayPlanID:     emptyPayPlan.ID,
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("failed to create employee contract: %v", err)
	}

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if dp.GrossSalary != 0 {
		t.Errorf("GrossSalary = %d, want 0 (no pay plan entries)", dp.GrossSalary)
	}
}

func TestStatisticsService_GetStaffingHours_FundingPeriodChange(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")

	// Period 1: Jan-Jun with 39 hours
	toDate1 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	period1 := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate1, 39.0)
	createTestFundingPropertyWithRequirement(t, db, period1.ID, "care_type", "ganztag", 0.25, 0, 6)

	// Period 2: Jul-Dec with 40 hours
	toDate2 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period2 := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), &toDate2, 40.0)
	createTestFundingPropertyWithRequirement(t, db, period2.ID, "care_type", "ganztag", 0.25, 0, 6)

	section := getDefaultSection(t, db, org.ID)

	// Child with ongoing contract
	child := createTestChild(t, db, "Child", "One", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	props := models.ContractProperties{"care_type": "ganztag"}
	createTestChildContract(t, db, child.ID, contractFrom, nil, section.ID, props)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetStaffingHours(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 12 {
		t.Fatalf("expected 12 data points, got %d", len(result.DataPoints))
	}

	// Jan-Jun: 1 child * 0.25 * 39.0 = 9.75
	expectedFirst := 0.25 * 39.0
	for i := 0; i < 6; i++ {
		if !almostEqual(result.DataPoints[i].RequiredHours, expectedFirst, 0.01) {
			t.Errorf("data point %d (period 1): RequiredHours = %v, want %v", i, result.DataPoints[i].RequiredHours, expectedFirst)
		}
	}

	// Jul-Dec: 1 child * 0.25 * 40.0 = 10.0
	expectedSecond := 0.25 * 40.0
	for i := 6; i < 12; i++ {
		if !almostEqual(result.DataPoints[i].RequiredHours, expectedSecond, 0.01) {
			t.Errorf("data point %d (period 2): RequiredHours = %v, want %v", i, result.DataPoints[i].RequiredHours, expectedSecond)
		}
	}
}

// ============================================================
// GetFinancials - Budget Items tests
// ============================================================

func TestGetFinancials_BudgetExpenseFixed(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Fixed expense budget item: 50000 cents/month (not per-child)
	item := createTestBudgetItem(t, db, "Rent", org.ID, "expense", false)
	createTestBudgetItemEntry(t, db, item.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 50000, "")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for i, dp := range result.DataPoints {
		if dp.BudgetExpenses != 50000 {
			t.Errorf("dp %d: BudgetExpenses = %d, want 50000", i, dp.BudgetExpenses)
		}
		if dp.BudgetIncome != 0 {
			t.Errorf("dp %d: BudgetIncome = %d, want 0", i, dp.BudgetIncome)
		}
		if dp.TotalExpenses != 50000 {
			t.Errorf("dp %d: TotalExpenses = %d, want 50000", i, dp.TotalExpenses)
		}
		if dp.Balance != -50000 {
			t.Errorf("dp %d: Balance = %d, want -50000", i, dp.Balance)
		}
	}
}

func TestGetFinancials_BudgetIncomeFixed(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	item := createTestBudgetItem(t, db, "Donations", org.ID, "income", false)
	createTestBudgetItemEntry(t, db, item.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 100000, "")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for i, dp := range result.DataPoints {
		if dp.BudgetIncome != 100000 {
			t.Errorf("dp %d: BudgetIncome = %d, want 100000", i, dp.BudgetIncome)
		}
		if dp.BudgetExpenses != 0 {
			t.Errorf("dp %d: BudgetExpenses = %d, want 0", i, dp.BudgetExpenses)
		}
		if dp.TotalIncome != 100000 {
			t.Errorf("dp %d: TotalIncome = %d, want 100000", i, dp.TotalIncome)
		}
	}
}

func TestGetFinancials_BudgetExpensePerChild(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")
	section := getDefaultSection(t, db, org.ID)

	// 3 children with active contracts
	for _, name := range []string{"A", "B", "C"} {
		child := createTestChild(t, db, "Child", name, org.ID)
		createTestChildContract(t, db, child.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section.ID, nil)
	}

	// Per-child expense: 2000 cents/child/month
	item := createTestBudgetItem(t, db, "Meals", org.ID, "expense", true)
	createTestBudgetItemEntry(t, db, item.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 2000, "")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	// 3 children * 2000 = 6000
	if dp.BudgetExpenses != 6000 {
		t.Errorf("BudgetExpenses = %d, want 6000", dp.BudgetExpenses)
	}
	if dp.TotalExpenses != 6000 {
		t.Errorf("TotalExpenses = %d, want 6000", dp.TotalExpenses)
	}
}

func TestGetFinancials_BudgetIncomePerChild(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")
	section := getDefaultSection(t, db, org.ID)

	// 2 children
	for _, name := range []string{"A", "B"} {
		child := createTestChild(t, db, "Child", name, org.ID)
		createTestChildContract(t, db, child.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section.ID, nil)
	}

	// Per-child income: 15000 cents/child/month
	item := createTestBudgetItem(t, db, "Parent Fees", org.ID, "income", true)
	createTestBudgetItemEntry(t, db, item.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 15000, "")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	// 2 children * 15000 = 30000
	if dp.BudgetIncome != 30000 {
		t.Errorf("BudgetIncome = %d, want 30000", dp.BudgetIncome)
	}
	if dp.TotalIncome != 30000 {
		t.Errorf("TotalIncome = %d, want 30000", dp.TotalIncome)
	}
}

func TestGetFinancials_BudgetPerChildNoChildren(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Per-child item but no children
	item := createTestBudgetItem(t, db, "Meals", org.ID, "expense", true)
	createTestBudgetItemEntry(t, db, item.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 5000, "")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if dp.BudgetExpenses != 0 {
		t.Errorf("BudgetExpenses = %d, want 0 (no children)", dp.BudgetExpenses)
	}
}

func TestGetFinancials_BudgetMultipleMixed(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")
	section := getDefaultSection(t, db, org.ID)

	// 2 children
	for _, name := range []string{"A", "B"} {
		child := createTestChild(t, db, "Child", name, org.ID)
		createTestChildContract(t, db, child.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section.ID, nil)
	}

	// Fixed income: 80000
	item1 := createTestBudgetItem(t, db, "Donations", org.ID, "income", false)
	createTestBudgetItemEntry(t, db, item1.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 80000, "")

	// Per-child income: 10000/child -> 20000
	item2 := createTestBudgetItem(t, db, "Parent Fees", org.ID, "income", true)
	createTestBudgetItemEntry(t, db, item2.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 10000, "")

	// Fixed expense: 50000
	item3 := createTestBudgetItem(t, db, "Rent", org.ID, "expense", false)
	createTestBudgetItemEntry(t, db, item3.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 50000, "")

	// Per-child expense: 3000/child -> 6000
	item4 := createTestBudgetItem(t, db, "Meals", org.ID, "expense", true)
	createTestBudgetItemEntry(t, db, item4.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 3000, "")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	expectedIncome := 80000 + 20000
	expectedExpenses := 50000 + 6000
	if dp.BudgetIncome != expectedIncome {
		t.Errorf("BudgetIncome = %d, want %d", dp.BudgetIncome, expectedIncome)
	}
	if dp.BudgetExpenses != expectedExpenses {
		t.Errorf("BudgetExpenses = %d, want %d", dp.BudgetExpenses, expectedExpenses)
	}
	if dp.TotalIncome != expectedIncome {
		t.Errorf("TotalIncome = %d, want %d", dp.TotalIncome, expectedIncome)
	}
	if dp.TotalExpenses != expectedExpenses {
		t.Errorf("TotalExpenses = %d, want %d", dp.TotalExpenses, expectedExpenses)
	}
	if dp.Balance != expectedIncome-expectedExpenses {
		t.Errorf("Balance = %d, want %d", dp.Balance, expectedIncome-expectedExpenses)
	}
}

func TestGetFinancials_BudgetEntryNotActive(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Entry covers 2025, but we query 2024
	item := createTestBudgetItem(t, db, "Rent", org.ID, "expense", false)
	createTestBudgetItemEntry(t, db, item.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 50000, "")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for i, dp := range result.DataPoints {
		if dp.BudgetExpenses != 0 {
			t.Errorf("dp %d: BudgetExpenses = %d, want 0 (entry not active)", i, dp.BudgetExpenses)
		}
	}
}

func TestGetFinancials_BudgetEntryStartsMidRange(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Entry starts March 2024
	item := createTestBudgetItem(t, db, "Rent", org.ID, "expense", false)
	createTestBudgetItemEntry(t, db, item.ID, time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), nil, 40000, "")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Jan, Feb: 0
	for i := 0; i < 2; i++ {
		if result.DataPoints[i].BudgetExpenses != 0 {
			t.Errorf("dp %d: BudgetExpenses = %d, want 0 (entry not started)", i, result.DataPoints[i].BudgetExpenses)
		}
	}
	// Mar-Jun: 40000
	for i := 2; i < 6; i++ {
		if result.DataPoints[i].BudgetExpenses != 40000 {
			t.Errorf("dp %d: BudgetExpenses = %d, want 40000", i, result.DataPoints[i].BudgetExpenses)
		}
	}
}

func TestGetFinancials_BudgetOneEntryPerItem(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Two overlapping entries on same item: only first active one is counted
	item := createTestBudgetItem(t, db, "Rent", org.ID, "expense", false)
	createTestBudgetItemEntry(t, db, item.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 30000, "first")
	createTestBudgetItemEntry(t, db, item.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 70000, "second")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	// Should be either 30000 or 70000, but NOT 100000 (both combined)
	if dp.BudgetExpenses == 100000 {
		t.Errorf("BudgetExpenses = 100000, should only count first active entry, not both")
	}
	if dp.BudgetExpenses != 30000 && dp.BudgetExpenses != 70000 {
		t.Errorf("BudgetExpenses = %d, want 30000 or 70000 (first active entry)", dp.BudgetExpenses)
	}
}

func TestGetFinancials_BudgetWithSalariesAndFunding(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Government funding
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, fundingPeriod.ID, "care_type", "ganztag", 100000, 0.25, 0, 6)

	section := getDefaultSection(t, db, org.ID)

	// 1 child
	child := createTestChild(t, db, "Child", "One", org.ID)
	createTestChildContract(t, db, child.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	// Pay plan + employee
	payplan := createTestPayPlan(t, db, "TVöD", org.ID)
	ppPeriod := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, 2000)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S8a", 3, 300000, nil)
	emp := createTestEmployee(t, db, "Emp", "One", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]interface{}{"grade": "S8a", "step": 3})

	// Budget items
	incomeItem := createTestBudgetItem(t, db, "Parent Fees", org.ID, "income", true)
	createTestBudgetItemEntry(t, db, incomeItem.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 20000, "")

	expenseItem := createTestBudgetItem(t, db, "Rent", org.ID, "expense", false)
	createTestBudgetItemEntry(t, db, expenseItem.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 50000, "")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	// Funding: 100000
	if dp.FundingIncome != 100000 {
		t.Errorf("FundingIncome = %d, want 100000", dp.FundingIncome)
	}
	// Budget income: 1 child * 20000 = 20000
	if dp.BudgetIncome != 20000 {
		t.Errorf("BudgetIncome = %d, want 20000", dp.BudgetIncome)
	}
	// TotalIncome = funding + budget income
	if dp.TotalIncome != 120000 {
		t.Errorf("TotalIncome = %d, want 120000", dp.TotalIncome)
	}
	// Gross: 300000, Employer: 300000 * 2000/10000 = 60000
	if dp.GrossSalary != 300000 {
		t.Errorf("GrossSalary = %d, want 300000", dp.GrossSalary)
	}
	if dp.EmployerCosts != 60000 {
		t.Errorf("EmployerCosts = %d, want 60000", dp.EmployerCosts)
	}
	if dp.BudgetExpenses != 50000 {
		t.Errorf("BudgetExpenses = %d, want 50000", dp.BudgetExpenses)
	}
	// TotalExpenses = salary + employer + budget expenses
	expectedExpenses := 300000 + 60000 + 50000
	if dp.TotalExpenses != expectedExpenses {
		t.Errorf("TotalExpenses = %d, want %d", dp.TotalExpenses, expectedExpenses)
	}
	expectedBalance := 120000 - expectedExpenses
	if dp.Balance != expectedBalance {
		t.Errorf("Balance = %d, want %d", dp.Balance, expectedBalance)
	}
}

func TestGetFinancials_BudgetPerChildCountChanges(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")
	section := getDefaultSection(t, db, org.ID)

	// Child A: active from Jan
	childA := createTestChild(t, db, "Child", "A", org.ID)
	createTestChildContract(t, db, childA.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section.ID, nil)

	// Child B: active from March (mid-range)
	childB := createTestChild(t, db, "Child", "B", org.ID)
	createTestChildContract(t, db, childB.ID, time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), nil, section.ID, nil)

	// Per-child expense: 10000 cents/child/month
	item := createTestBudgetItem(t, db, "Meals", org.ID, "expense", true)
	createTestBudgetItemEntry(t, db, item.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 10000, "")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Jan, Feb: 1 child * 10000 = 10000
	for i := 0; i < 2; i++ {
		if result.DataPoints[i].BudgetExpenses != 10000 {
			t.Errorf("dp %d: BudgetExpenses = %d, want 10000 (1 child)", i, result.DataPoints[i].BudgetExpenses)
		}
		if result.DataPoints[i].ChildCount != 1 {
			t.Errorf("dp %d: ChildCount = %d, want 1", i, result.DataPoints[i].ChildCount)
		}
	}
	// Mar-Jun: 2 children * 10000 = 20000
	for i := 2; i < 6; i++ {
		if result.DataPoints[i].BudgetExpenses != 20000 {
			t.Errorf("dp %d: BudgetExpenses = %d, want 20000 (2 children)", i, result.DataPoints[i].BudgetExpenses)
		}
		if result.DataPoints[i].ChildCount != 2 {
			t.Errorf("dp %d: ChildCount = %d, want 2", i, result.DataPoints[i].ChildCount)
		}
	}
}

func TestGetFinancials_BudgetEntryEndsMidRange(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Entry active Jan-Mar 2024 only (to_date = 2024-03-31)
	item := createTestBudgetItem(t, db, "Insurance", org.ID, "expense", false)
	to := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	createTestBudgetItemEntry(t, db, item.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &to, 25000, "")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	toQuery := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &toQuery)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Jan-Mar: 25000
	for i := 0; i < 3; i++ {
		if result.DataPoints[i].BudgetExpenses != 25000 {
			t.Errorf("dp %d: BudgetExpenses = %d, want 25000 (entry active)", i, result.DataPoints[i].BudgetExpenses)
		}
	}
	// Apr-Jun: 0 (entry expired)
	for i := 3; i < 6; i++ {
		if result.DataPoints[i].BudgetExpenses != 0 {
			t.Errorf("dp %d: BudgetExpenses = %d, want 0 (entry expired)", i, result.DataPoints[i].BudgetExpenses)
		}
	}
}

func TestGetFinancials_BudgetEntryExpiredBeforeRange(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Entry entirely in the past (2023), query range is 2024
	item := createTestBudgetItem(t, db, "Old Insurance", org.ID, "expense", false)
	to := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	createTestBudgetItemEntry(t, db, item.ID, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), &to, 30000, "")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	toQuery := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &toQuery)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for i, dp := range result.DataPoints {
		if dp.BudgetExpenses != 0 {
			t.Errorf("dp %d: BudgetExpenses = %d, want 0 (entry expired before range)", i, dp.BudgetExpenses)
		}
	}
}

func TestGetFinancials_BudgetEntryTransition(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Budget item with two consecutive entries at different amounts
	item := createTestBudgetItem(t, db, "Rent", org.ID, "expense", false)

	// First entry: Jan-Mar at 40000
	to1 := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	createTestBudgetItemEntry(t, db, item.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &to1, 40000, "old rate")

	// Second entry: Apr onward at 45000
	createTestBudgetItemEntry(t, db, item.ID, time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC), nil, 45000, "new rate")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	toQuery := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &toQuery)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Jan-Mar: 40000 (first entry)
	for i := 0; i < 3; i++ {
		if result.DataPoints[i].BudgetExpenses != 40000 {
			t.Errorf("dp %d: BudgetExpenses = %d, want 40000 (old rate)", i, result.DataPoints[i].BudgetExpenses)
		}
	}
	// Apr-Jun: 45000 (second entry)
	for i := 3; i < 6; i++ {
		if result.DataPoints[i].BudgetExpenses != 45000 {
			t.Errorf("dp %d: BudgetExpenses = %d, want 45000 (new rate)", i, result.DataPoints[i].BudgetExpenses)
		}
	}
}

func TestGetFinancials_BudgetItemDetails_SingleIncome(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	item := createTestBudgetItem(t, db, "Donations", org.ID, "income", false)
	createTestBudgetItemEntry(t, db, item.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 100000, "")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if len(dp.BudgetItemDetails) != 1 {
		t.Fatalf("expected 1 budget item detail, got %d", len(dp.BudgetItemDetails))
	}
	d := dp.BudgetItemDetails[0]
	if d.Name != "Donations" {
		t.Errorf("Name = %q, want %q", d.Name, "Donations")
	}
	if d.Category != "income" {
		t.Errorf("Category = %q, want %q", d.Category, "income")
	}
	if d.AmountCents != 100000 {
		t.Errorf("AmountCents = %d, want 100000", d.AmountCents)
	}
}

func TestGetFinancials_BudgetItemDetails_MixedIncomeExpense(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	incomeItem := createTestBudgetItem(t, db, "Donations", org.ID, "income", false)
	createTestBudgetItemEntry(t, db, incomeItem.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 80000, "")

	expenseItem := createTestBudgetItem(t, db, "Rent", org.ID, "expense", false)
	createTestBudgetItemEntry(t, db, expenseItem.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 50000, "")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if len(dp.BudgetItemDetails) != 2 {
		t.Fatalf("expected 2 budget item details, got %d", len(dp.BudgetItemDetails))
	}
	if dp.BudgetIncome != 80000 {
		t.Errorf("BudgetIncome = %d, want 80000", dp.BudgetIncome)
	}
	if dp.BudgetExpenses != 50000 {
		t.Errorf("BudgetExpenses = %d, want 50000", dp.BudgetExpenses)
	}

	// Verify each detail has the correct category
	incomeFound, expenseFound := false, false
	for _, d := range dp.BudgetItemDetails {
		if d.Name == "Donations" && d.Category == "income" && d.AmountCents == 80000 {
			incomeFound = true
		}
		if d.Name == "Rent" && d.Category == "expense" && d.AmountCents == 50000 {
			expenseFound = true
		}
	}
	if !incomeFound {
		t.Error("missing income budget item detail for Donations")
	}
	if !expenseFound {
		t.Error("missing expense budget item detail for Rent")
	}
}

func TestGetFinancials_BudgetItemDetails_PerChild(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")
	section := getDefaultSection(t, db, org.ID)

	// 2 children
	child1 := createTestChild(t, db, "Child", "One", org.ID)
	child2 := createTestChild(t, db, "Child", "Two", org.ID)
	createTestChildContract(t, db, child1.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section.ID, nil)
	createTestChildContract(t, db, child2.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section.ID, nil)

	// Per-child expense: 3000 cents/child/month
	item := createTestBudgetItem(t, db, "Meals", org.ID, "expense", true)
	createTestBudgetItemEntry(t, db, item.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 3000, "")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if len(dp.BudgetItemDetails) != 1 {
		t.Fatalf("expected 1 budget item detail, got %d", len(dp.BudgetItemDetails))
	}
	d := dp.BudgetItemDetails[0]
	// 2 children * 3000 = 6000
	if d.AmountCents != 6000 {
		t.Errorf("AmountCents = %d, want 6000", d.AmountCents)
	}
	if d.Name != "Meals" {
		t.Errorf("Name = %q, want %q", d.Name, "Meals")
	}
}

func TestGetFinancials_FundingDetails_SingleProperty(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")
	section := getDefaultSection(t, db, org.ID)

	// Government funding with one property
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 166847, 0, 6)

	// 1 child with matching contract
	child := createTestChild(t, db, "Child", "One", org.ID)
	props := models.ContractProperties{"care_type": "ganztag"}
	createTestChildContract(t, db, child.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section.ID, props)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if len(dp.FundingDetails) != 1 {
		t.Fatalf("expected 1 funding detail, got %d", len(dp.FundingDetails))
	}
	fd := dp.FundingDetails[0]
	if fd.Key != "care_type" || fd.Value != "ganztag" {
		t.Errorf("FundingDetail key/value = %q/%q, want care_type/ganztag", fd.Key, fd.Value)
	}
	if fd.AmountCents != 166847 {
		t.Errorf("AmountCents = %d, want 166847", fd.AmountCents)
	}
	if dp.FundingIncome != 166847 {
		t.Errorf("FundingIncome = %d, want 166847", dp.FundingIncome)
	}
}

func TestGetFinancials_FundingDetails_MultipleProperties(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")
	section := getDefaultSection(t, db, org.ID)

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 100000, 0, 6)
	createTestFundingProperty(t, db, period.ID, "integration", "integration a", 50000, -1, -1)

	// 1 child with both properties
	child := createTestChild(t, db, "Child", "One", org.ID)
	props := models.ContractProperties{"care_type": "ganztag", "integration": "integration a"}
	createTestChildContract(t, db, child.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section.ID, props)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if len(dp.FundingDetails) != 2 {
		t.Fatalf("expected 2 funding details, got %d", len(dp.FundingDetails))
	}

	// Total should be 100000 + 50000 = 150000
	if dp.FundingIncome != 150000 {
		t.Errorf("FundingIncome = %d, want 150000", dp.FundingIncome)
	}

	// Check both details exist (sorted by key then value)
	detailMap := make(map[string]int)
	for _, fd := range dp.FundingDetails {
		detailMap[fd.Key+":"+fd.Value] = fd.AmountCents
	}
	if detailMap["care_type:ganztag"] != 100000 {
		t.Errorf("care_type:ganztag amount = %d, want 100000", detailMap["care_type:ganztag"])
	}
	if detailMap["integration:integration a"] != 50000 {
		t.Errorf("integration:integration a amount = %d, want 50000", detailMap["integration:integration a"])
	}
}

func TestGetFinancials_FundingDetails_MultipleChildren(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")
	section := getDefaultSection(t, db, org.ID)

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 80000, 0, 6)

	// 2 children both matching the same property
	props := models.ContractProperties{"care_type": "ganztag"}
	child1 := createTestChild(t, db, "Child", "One", org.ID)
	child2 := createTestChild(t, db, "Child", "Two", org.ID)
	createTestChildContract(t, db, child1.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section.ID, props)
	createTestChildContract(t, db, child2.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section.ID, props)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if len(dp.FundingDetails) != 1 {
		t.Fatalf("expected 1 funding detail, got %d", len(dp.FundingDetails))
	}

	fd := dp.FundingDetails[0]
	// 2 children * 80000 = 160000
	if fd.AmountCents != 160000 {
		t.Errorf("AmountCents = %d, want 160000", fd.AmountCents)
	}
	if dp.FundingIncome != 160000 {
		t.Errorf("FundingIncome = %d, want 160000", dp.FundingIncome)
	}
}

func TestGetFinancials_SalaryDetails_SingleCategory(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")
	section := getDefaultSection(t, db, org.ID)

	// Pay plan: S8a step 3, 350000 cents/month, 39h full-time, 22% employer contrib
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	ppPeriod := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, 2200)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S8a", 3, 350000, nil)

	emp := createTestEmployee(t, db, "Emp", "One", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, contractFrom, nil, 39.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]interface{}{"grade": "S8a", "step": 3})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if len(dp.SalaryDetails) != 1 {
		t.Fatalf("expected 1 salary detail, got %d", len(dp.SalaryDetails))
	}
	sd := dp.SalaryDetails[0]
	if sd.StaffCategory != "qualified" {
		t.Errorf("StaffCategory = %q, want %q", sd.StaffCategory, "qualified")
	}
	if sd.GrossSalary != 350000 {
		t.Errorf("GrossSalary = %d, want 350000", sd.GrossSalary)
	}
	// Employer: 350000 * 2200/10000 = 77000
	if sd.EmployerCosts != 77000 {
		t.Errorf("EmployerCosts = %d, want 77000", sd.EmployerCosts)
	}
}

func TestGetFinancials_SalaryDetails_MultipleCategories(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")
	section := getDefaultSection(t, db, org.ID)

	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	ppPeriod := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, 2200)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S8a", 3, 350000, nil)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S4", 2, 280000, nil)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S3", 1, 250000, nil)

	categories := []struct {
		firstName     string
		staffCategory string
		grade         string
		step          int
		monthly       int
	}{
		{"Emp", "qualified", "S8a", 3, 350000},
		{"Sup", "supplementary", "S4", 2, 280000},
		{"Non", "non_pedagogical", "S3", 1, 250000},
	}

	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, c := range categories {
		emp := createTestEmployee(t, db, c.firstName, "Test", org.ID)
		createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, contractFrom, nil, 39.0, c.staffCategory, section.ID)
		db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]interface{}{"grade": c.grade, "step": c.step})
	}

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if len(dp.SalaryDetails) != 3 {
		t.Fatalf("expected 3 salary details, got %d", len(dp.SalaryDetails))
	}

	// Should be sorted alphabetically by staff_category
	expectedOrder := []string{"non_pedagogical", "qualified", "supplementary"}
	for i, sd := range dp.SalaryDetails {
		if sd.StaffCategory != expectedOrder[i] {
			t.Errorf("SalaryDetails[%d].StaffCategory = %q, want %q", i, sd.StaffCategory, expectedOrder[i])
		}
	}

	// Verify aggregates match sum of details
	totalGross := 0
	totalEmployer := 0
	for _, sd := range dp.SalaryDetails {
		totalGross += sd.GrossSalary
		totalEmployer += sd.EmployerCosts
	}
	if totalGross != dp.GrossSalary {
		t.Errorf("sum of detail GrossSalary = %d, want %d (aggregate)", totalGross, dp.GrossSalary)
	}
	if totalEmployer != dp.EmployerCosts {
		t.Errorf("sum of detail EmployerCosts = %d, want %d (aggregate)", totalEmployer, dp.EmployerCosts)
	}
}

func TestGetFinancials_SalaryDetails_SameCategory(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")
	section := getDefaultSection(t, db, org.ID)

	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	ppPeriod := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, 2200)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S8a", 3, 350000, nil)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S8a", 4, 380000, nil)

	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	emp1 := createTestEmployee(t, db, "Emp", "One", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp1.ID, payplan.ID, contractFrom, nil, 39.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp1.ID).Updates(map[string]interface{}{"grade": "S8a", "step": 3})

	emp2 := createTestEmployee(t, db, "Emp", "Two", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp2.ID, payplan.ID, contractFrom, nil, 39.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp2.ID).Updates(map[string]interface{}{"grade": "S8a", "step": 4})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if len(dp.SalaryDetails) != 1 {
		t.Fatalf("expected 1 salary detail (same category summed), got %d", len(dp.SalaryDetails))
	}
	sd := dp.SalaryDetails[0]
	if sd.StaffCategory != "qualified" {
		t.Errorf("StaffCategory = %q, want %q", sd.StaffCategory, "qualified")
	}
	// 350000 + 380000 = 730000
	if sd.GrossSalary != 730000 {
		t.Errorf("GrossSalary = %d, want 730000", sd.GrossSalary)
	}
	// Employer: round(350000*0.22) + round(380000*0.22) = 77000 + 83600 = 160600
	expectedEmployer := int(math.Round(350000.0*2200.0/10000.0)) + int(math.Round(380000.0*2200.0/10000.0))
	if sd.EmployerCosts != expectedEmployer {
		t.Errorf("EmployerCosts = %d, want %d", sd.EmployerCosts, expectedEmployer)
	}
}

func TestGetFinancials_SalaryDetails_ProRata(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")
	section := getDefaultSection(t, db, org.ID)

	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	ppPeriod := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, 2200)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S8a", 3, 350000, nil)

	emp := createTestEmployee(t, db, "Emp", "Part", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, contractFrom, nil, 20.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]interface{}{"grade": "S8a", "step": 3})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if len(dp.SalaryDetails) != 1 {
		t.Fatalf("expected 1 salary detail, got %d", len(dp.SalaryDetails))
	}
	sd := dp.SalaryDetails[0]
	// Pro-rated: 350000 * 20/39
	expectedGross := int(math.Round(350000.0 * 20.0 / 39.0))
	if sd.GrossSalary != expectedGross {
		t.Errorf("GrossSalary = %d, want %d", sd.GrossSalary, expectedGross)
	}
	expectedEmployer := int(math.Round(float64(expectedGross) * 2200.0 / 10000.0))
	if sd.EmployerCosts != expectedEmployer {
		t.Errorf("EmployerCosts = %d, want %d", sd.EmployerCosts, expectedEmployer)
	}
}

func TestGetFinancials_SalaryDetails_NoEmployees(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.DataPoints[0]
	if len(dp.SalaryDetails) != 0 {
		t.Errorf("expected no salary details, got %d", len(dp.SalaryDetails))
	}
}
