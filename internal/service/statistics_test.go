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
	return NewStatisticsService(childStore, employeeStore, orgStore, fundingStore)
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

	// Pass nil for from/to to use default range: 12 months back to 6 months forward = 19 data points
	result, err := svc.GetStaffingHours(ctx, org.ID, nil, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Default is 12 months back + current month + 6 months forward = 19 data points
	if len(result.DataPoints) != 19 {
		t.Errorf("expected 19 data points, got %d", len(result.DataPoints))
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
