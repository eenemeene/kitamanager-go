package service

import (
	"context"
	"math"
	"strings"
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
	sectionStore := store.NewSectionStore(db)
	billStore := store.NewGovernmentFundingBillPeriodStore(db)
	return NewStatisticsService(childStore, employeeStore, orgStore, fundingStore, payPlanStore, budgetItemStore, sectionStore, billStore)
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
		Label:       strings.ToUpper(value[:1]) + value[1:],
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
	for i := range 2 {
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
	createTestFundingPropertyFull(t, db, fundingPeriod.ID, "care_type", "ganztag", "Ganztag", 166847, 0.25, 0, 6)

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
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]any{"grade": "S8a", "step": 3})

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
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]any{"grade": "S8a", "step": 3})

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
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]any{"grade": "S11b", "step": 5})

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
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]any{"grade": "S9", "step": 1})

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
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]any{"grade": "S8a", "step": 3})

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
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp1.ID).Updates(map[string]any{"grade": "S8a", "step": 3})

	// Non-pedagogical staff (kitchen, admin, etc.)
	emp2 := createTestEmployee(t, db, "Emp", "Kitchen", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp2.ID, payplan.ID, contractFrom, nil, 39.0, "non_pedagogical", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp2.ID).Updates(map[string]any{"grade": "S8a", "step": 3})

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
	createTestFundingPropertyFull(t, db, fundingPeriod.ID, "care_type", "ganztag", "Ganztag", 100000, 0.25, 0, 6)

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
	for i := range 2 {
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
	createTestFundingPropertyFull(t, db, fundingPeriod.ID, "care_type", "ganztag", "Ganztag", 80000, 0.25, 0, 6)
	createTestFundingPropertyFull(t, db, fundingPeriod.ID, "care_type", "halbtag", "Halbtag", 40000, 0.12, 0, 6)

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
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp1.ID).Updates(map[string]any{"grade": "S8a", "step": 3})

	// Employee 2: S11b step 5, 20h (part-time)
	emp2 := createTestEmployee(t, db, "Emp", "Two", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp2.ID, payplan.ID, contractFrom, nil, 20.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp2.ID).Updates(map[string]any{"grade": "S11b", "step": 5})

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
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]any{"grade": "S8a", "step": 1})

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
	createTestFundingPropertyFull(t, db, fundingPeriod.ID, "care_type", "ganztag", "Ganztag", 100000, 0.25, 0, 6)

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
	for i := range 6 {
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
	for i := range 2 {
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
	createTestFundingPropertyFull(t, db, fundingPeriod.ID, "care_type", "ganztag", "Ganztag", 100000, 0.25, 0, 6)

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
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]any{"grade": "S8a", "step": 3})

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
	for i := range 2 {
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
	for i := range 3 {
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
	for i := range 3 {
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
	if fd.Label != "Ganztag" {
		t.Errorf("FundingDetail label = %q, want %q", fd.Label, "Ganztag")
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

func TestGetFinancials_FundingDetails_Label(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")
	section := getDefaultSection(t, db, org.ID)

	// Government funding with explicit labels
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag (bis 9h)", 100000, 0.25, 0, 6)
	createTestFundingPropertyFull(t, db, period.ID, "integration", "integration a", "Integration A", 50000, 0.1, -1, -1)

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

	// Build a label map by key:value
	labelMap := make(map[string]string)
	for _, fd := range dp.FundingDetails {
		labelMap[fd.Key+":"+fd.Value] = fd.Label
	}

	if labelMap["care_type:ganztag"] != "Ganztag (bis 9h)" {
		t.Errorf("care_type:ganztag label = %q, want %q", labelMap["care_type:ganztag"], "Ganztag (bis 9h)")
	}
	if labelMap["integration:integration a"] != "Integration A" {
		t.Errorf("integration:integration a label = %q, want %q", labelMap["integration:integration a"], "Integration A")
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
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]any{"grade": "S8a", "step": 3})

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
		db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]any{"grade": c.grade, "step": c.step})
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
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp1.ID).Updates(map[string]any{"grade": "S8a", "step": 3})

	emp2 := createTestEmployee(t, db, "Emp", "Two", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp2.ID, payplan.ID, contractFrom, nil, 39.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp2.ID).Updates(map[string]any{"grade": "S8a", "step": 4})

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
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]any{"grade": "S8a", "step": 3})

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

// --- GetOccupancy Tests ---

func TestStatisticsService_GetOccupancy_Basic(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Create funding with age groups and care types
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)

	// care_type properties with age ranges
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 10000, 0.25, 0, 2)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 8000, 0.15, 3, 6)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "halbtag", "Halbtag", 6000, 0.20, 0, 2)

	// supplement property
	createTestFundingPropertyFull(t, db, period.ID, "integration", "integration_a", "Integration A", 5000, 0.0, 0, 6)

	section := getDefaultSection(t, db, org.ID)

	// Child 1: born 2022 (age ~2 in Jan 2024), ganztag + integration_a
	child1 := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Young", LastName: "Child", Birthdate: time.Date(2022, 6, 15, 0, 0, 0, 0, time.UTC)}}
	db.Create(child1)
	createTestChildContract(t, db, child1.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section.ID,
		models.ContractProperties{"care_type": "ganztag", "integration": "integration_a"})

	// Child 2: born 2020 (age ~3 in Jan 2024), ganztag, no supplements
	child2 := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Older", LastName: "Child", Birthdate: time.Date(2020, 3, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(child2)
	createTestChildContract(t, db, child2.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section.ID,
		models.ContractProperties{"care_type": "ganztag"})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetOccupancy(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should have age groups, care types, and supplement types
	if len(result.AgeGroups) == 0 {
		t.Error("expected age groups")
	}
	if len(result.CareTypes) == 0 {
		t.Error("expected care types")
	}
	if len(result.SupplementTypes) == 0 {
		t.Error("expected supplement types")
	}

	if len(result.DataPoints) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(result.DataPoints))
	}

	dp := result.DataPoints[0]
	if dp.Total != 2 {
		t.Errorf("expected total=2, got %d", dp.Total)
	}

	// Check supplement count
	if dp.BySupplement["integration_a"] != 1 {
		t.Errorf("expected integration_a=1, got %d", dp.BySupplement["integration_a"])
	}
}

func TestStatisticsService_GetOccupancy_NoFunding(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// No funding configured => should still return without error, empty structure
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetOccupancy(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 3 {
		t.Fatalf("expected 3 data points, got %d", len(result.DataPoints))
	}

	// All totals should be 0
	for _, dp := range result.DataPoints {
		if dp.Total != 0 {
			t.Errorf("expected total=0, got %d for %s", dp.Total, dp.Date)
		}
	}
}

func TestStatisticsService_GetOccupancy_SectionFilter(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 10000, 0.25, 0, 6)

	section1 := getDefaultSection(t, db, org.ID)
	section2 := createTestSection(t, db, "Krippe", org.ID, false)

	// Child in section1
	child1 := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "S1", LastName: "Child", Birthdate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(child1)
	createTestChildContract(t, db, child1.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section1.ID,
		models.ContractProperties{"care_type": "ganztag"})

	// Child in section2
	child2 := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "S2", LastName: "Child", Birthdate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(child2)
	createTestChildContract(t, db, child2.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section2.ID,
		models.ContractProperties{"care_type": "ganztag"})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Filter section2 => only 1 child
	result, err := svc.GetOccupancy(ctx, org.ID, &from, &to, &section2.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.DataPoints[0].Total != 1 {
		t.Errorf("expected total=1, got %d", result.DataPoints[0].Total)
	}
}

func TestStatisticsService_GetOccupancy_ContractEndsDuringRange(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 10000, 0.25, 0, 6)

	section := getDefaultSection(t, db, org.ID)

	// Child with contract ending in Feb 2024
	child := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Leaving", LastName: "Child", Birthdate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(child)
	contractEnd := time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &contractEnd, section.ID,
		models.ContractProperties{"care_type": "ganztag"})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetOccupancy(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Jan=1, Feb=1, Mar=0, Apr=0
	expected := []int{1, 1, 0, 0}
	for i, want := range expected {
		if result.DataPoints[i].Total != want {
			t.Errorf("month %d: total=%d, want %d", i, result.DataPoints[i].Total, want)
		}
	}
}

// --- Employee Staffing Hours Tests ---

func TestStatisticsService_GetEmployeeStaffingHours_Basic(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	payplan := createTestPayPlan(t, db, "TV-L", org.ID)

	// Create 2 employees with contracts from Jan 2024, ongoing
	emp1 := createTestEmployee(t, db, "Anna", "Mueller", org.ID)
	emp2 := createTestEmployee(t, db, "Bob", "Schmidt", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestEmployeeContractWithCategory(t, db, emp1.ID, payplan.ID, contractFrom, nil, 30.0, "qualified", section.ID)
	createTestEmployeeContractWithCategory(t, db, emp2.ID, payplan.ID, contractFrom, nil, 20.0, "supplementary", section.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetEmployeeStaffingHours(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 3 months: Jan, Feb, Mar
	if len(result.Dates) != 3 {
		t.Fatalf("expected 3 dates, got %d", len(result.Dates))
	}

	// 2 employees, sorted alphabetically (Mueller, Schmidt)
	if len(result.Employees) != 2 {
		t.Fatalf("expected 2 employees, got %d", len(result.Employees))
	}

	// Anna Mueller comes first (M < S)
	if result.Employees[0].LastName != "Mueller" {
		t.Errorf("expected first employee to be Mueller, got %s", result.Employees[0].LastName)
	}
	if result.Employees[1].LastName != "Schmidt" {
		t.Errorf("expected second employee to be Schmidt, got %s", result.Employees[1].LastName)
	}

	// Check hours for each month
	for i := range 3 {
		if !almostEqual(result.Employees[0].MonthlyHours[i], 30.0, 0.01) {
			t.Errorf("Mueller month %d: got %v, want 30.0", i, result.Employees[0].MonthlyHours[i])
		}
		if !almostEqual(result.Employees[1].MonthlyHours[i], 20.0, 0.01) {
			t.Errorf("Schmidt month %d: got %v, want 20.0", i, result.Employees[1].MonthlyHours[i])
		}
	}

	// Check staff categories
	if result.Employees[0].StaffCategory != "qualified" {
		t.Errorf("expected qualified, got %s", result.Employees[0].StaffCategory)
	}
	if result.Employees[1].StaffCategory != "supplementary" {
		t.Errorf("expected supplementary, got %s", result.Employees[1].StaffCategory)
	}
}

func TestStatisticsService_GetEmployeeStaffingHours_EmptyOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Empty Org")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetEmployeeStaffingHours(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Dates) != 3 {
		t.Fatalf("expected 3 dates, got %d", len(result.Dates))
	}
	if len(result.Employees) != 0 {
		t.Fatalf("expected 0 employees, got %d", len(result.Employees))
	}
}

func TestStatisticsService_GetEmployeeStaffingHours_ContractGaps(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	payplan := createTestPayPlan(t, db, "TV-L", org.ID)

	emp := createTestEmployee(t, db, "Charlie", "Brown", org.ID)

	// Contract 1: Jan-Feb 2024, 25 hours
	contract1End := time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &contract1End, 25.0, "qualified", section.ID)

	// Contract 2: Apr 2024 onward, 35 hours (gap in March)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID,
		time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC), nil, 35.0, "qualified", section.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetEmployeeStaffingHours(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Employees) != 1 {
		t.Fatalf("expected 1 employee, got %d", len(result.Employees))
	}

	hours := result.Employees[0].MonthlyHours
	// Jan=25, Feb=25, Mar=0 (gap), Apr=35, May=35
	expected := []float64{25.0, 25.0, 0.0, 35.0, 35.0}
	for i, want := range expected {
		if !almostEqual(hours[i], want, 0.01) {
			t.Errorf("month %d: got %v, want %v", i, hours[i], want)
		}
	}
}

func TestStatisticsService_GetEmployeeStaffingHours_SectionFilter(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section1 := getDefaultSection(t, db, org.ID)
	section2 := createTestSection(t, db, "Section B", org.ID, false)
	payplan := createTestPayPlan(t, db, "TV-L", org.ID)

	emp1 := createTestEmployee(t, db, "Anna", "Mueller", org.ID)
	emp2 := createTestEmployee(t, db, "Bob", "Schmidt", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestEmployeeContractWithCategory(t, db, emp1.ID, payplan.ID, contractFrom, nil, 30.0, "qualified", section1.ID)
	createTestEmployeeContractWithCategory(t, db, emp2.ID, payplan.ID, contractFrom, nil, 20.0, "qualified", section2.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Filter by section1: should only return emp1
	result, err := svc.GetEmployeeStaffingHours(ctx, org.ID, &from, &to, &section1.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Employees) != 1 {
		t.Fatalf("expected 1 employee, got %d", len(result.Employees))
	}
	if result.Employees[0].LastName != "Mueller" {
		t.Errorf("expected Mueller, got %s", result.Employees[0].LastName)
	}

	// Filter by section2: should only return emp2
	result, err = svc.GetEmployeeStaffingHours(ctx, org.ID, &from, &to, &section2.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Employees) != 1 {
		t.Fatalf("expected 1 employee, got %d", len(result.Employees))
	}
	if result.Employees[0].LastName != "Schmidt" {
		t.Errorf("expected Schmidt, got %s", result.Employees[0].LastName)
	}
}

// --- Edge case tests ---

func TestStatisticsService_GetFinancials_PayPlanPeriodTransition(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	section := getDefaultSection(t, db, org.ID)

	// Pay plan with TWO periods:
	// Period 1: Jan-Jun 2024, 39h, entry S8a/3 = 300000 cents, no employer contrib
	// Period 2: Jul-Dec 2024, 39h, entry S8a/3 = 350000 cents, no employer contrib
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	pp1To := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	pp1 := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &pp1To, 39.0, 0)
	createTestPayPlanEntry(t, db, pp1.ID, "S8a", 3, 300000, nil)

	pp2To := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	pp2 := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), &pp2To, 39.0, 0)
	createTestPayPlanEntry(t, db, pp2.ID, "S8a", 3, 350000, nil)

	// Employee: S8a step 3, full-time 39h, contract covers entire 2024
	emp := createTestEmployee(t, db, "Emp", "One", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, contractFrom, nil, 39.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]any{"grade": "S8a", "step": 3})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 12 {
		t.Fatalf("expected 12 data points, got %d", len(result.DataPoints))
	}

	// Jan-Jun: salary = 300000 (full-time)
	for i := range 6 {
		dp := result.DataPoints[i]
		if dp.GrossSalary != 300000 {
			t.Errorf("month %d (%s): GrossSalary = %d, want 300000", i+1, dp.Date, dp.GrossSalary)
		}
	}
	// Jul-Dec: salary = 350000 (full-time)
	for i := 6; i < 12; i++ {
		dp := result.DataPoints[i]
		if dp.GrossSalary != 350000 {
			t.Errorf("month %d (%s): GrossSalary = %d, want 350000", i+1, dp.Date, dp.GrossSalary)
		}
	}
}

func TestStatisticsService_GetFinancials_ChildAgeBoundaryChangesRate(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Funding with age-based rates:
	// U3 (ages 0-2): care_type=ganztag -> 200000 cents
	// Ü3 (ages 3-6): care_type=ganztag -> 100000 cents
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag U3", 200000, 0.25, 0, 2)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag Ü3", 100000, 0.15, 3, 6)

	section := getDefaultSection(t, db, org.ID)

	// Child born 2021-06-15: turns 3 on 2024-06-15
	// On 2024-06-01 the child is still 2 (birthday hasn't happened yet in June)
	// On 2024-07-01 the child is 3
	child := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Age", LastName: "Boundary", Birthdate: time.Date(2021, 6, 15, 0, 0, 0, 0, time.UTC)}}
	db.Create(child)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, contractFrom, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 12 {
		t.Fatalf("expected 12 data points, got %d", len(result.DataPoints))
	}

	// Jan-Jun (months 0-5): child is age 2, matches U3 -> 200000 cents
	for i := range 6 {
		dp := result.DataPoints[i]
		if dp.FundingIncome != 200000 {
			t.Errorf("month %d (%s): FundingIncome = %d, want 200000 (U3 rate)", i+1, dp.Date, dp.FundingIncome)
		}
	}
	// Jul-Dec (months 6-11): child is age 3, matches Ü3 -> 100000 cents
	for i := 6; i < 12; i++ {
		dp := result.DataPoints[i]
		if dp.FundingIncome != 100000 {
			t.Errorf("month %d (%s): FundingIncome = %d, want 100000 (Ü3 rate)", i+1, dp.Date, dp.FundingIncome)
		}
	}
}

func TestStatisticsService_GetFinancials_EmployeeZeroWeeklyHours(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	section := getDefaultSection(t, db, org.ID)

	// Pay plan: 39h full-time, 350000 cents/month, no employer contrib
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	ppPeriod := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, 0)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S8a", 3, 350000, nil)

	// Employee with 0 weekly hours (e.g., on leave but contract still active)
	emp := createTestEmployee(t, db, "Emp", "Zero", org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, contractFrom, nil, 0.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp.ID).Updates(map[string]any{"grade": "S8a", "step": 3})

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
		// Salary formula: 350000 * 0.0 / 39.0 = 0
		if dp.GrossSalary != 0 {
			t.Errorf("dp %d: GrossSalary = %d, want 0", i, dp.GrossSalary)
		}
		if dp.EmployerCosts != 0 {
			t.Errorf("dp %d: EmployerCosts = %d, want 0", i, dp.EmployerCosts)
		}
		// Employee should still be counted
		if dp.StaffCount != 1 {
			t.Errorf("dp %d: StaffCount = %d, want 1", i, dp.StaffCount)
		}
	}
}

func TestStatisticsService_GetOccupancy_ChildAgeGroupTransition(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Funding with two age groups:
	// U3 (ages 0-2): care_type=ganztag
	// Ü3 (ages 3-6): care_type=ganztag
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag U3", 200000, 0.25, 0, 2)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag Ü3", 100000, 0.15, 3, 6)

	section := getDefaultSection(t, db, org.ID)

	// Child born 2021-06-15: age 2 on 2024-06-01, age 3 on 2024-07-01
	child := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Transition", LastName: "Child", Birthdate: time.Date(2021, 6, 15, 0, 0, 0, 0, time.UTC)}}
	db.Create(child)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, contractFrom, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetOccupancy(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 12 {
		t.Fatalf("expected 12 data points, got %d", len(result.DataPoints))
	}

	// Age group labels: "0/1/2" for ages 0-2, "3+" for ages 3-6
	u3Label := formatAgeGroupLabel(0, 2)
	u6Label := formatAgeGroupLabel(3, 6)

	// Jan-Jun (months 0-5): child is 2 -> counted in U3 age group
	for i := range 6 {
		dp := result.DataPoints[i]
		if dp.Total != 1 {
			t.Errorf("month %d (%s): Total = %d, want 1", i+1, dp.Date, dp.Total)
		}
		u3Count := dp.ByAgeAndCareType[u3Label]["ganztag"]
		if u3Count != 1 {
			t.Errorf("month %d (%s): U3 ganztag count = %d, want 1", i+1, dp.Date, u3Count)
		}
		u6Count := dp.ByAgeAndCareType[u6Label]["ganztag"]
		if u6Count != 0 {
			t.Errorf("month %d (%s): Ü3 ganztag count = %d, want 0", i+1, dp.Date, u6Count)
		}
	}

	// Jul-Dec (months 6-11): child is 3 -> counted in Ü3 age group
	for i := 6; i < 12; i++ {
		dp := result.DataPoints[i]
		if dp.Total != 1 {
			t.Errorf("month %d (%s): Total = %d, want 1", i+1, dp.Date, dp.Total)
		}
		u3Count := dp.ByAgeAndCareType[u3Label]["ganztag"]
		if u3Count != 0 {
			t.Errorf("month %d (%s): U3 ganztag count = %d, want 0", i+1, dp.Date, u3Count)
		}
		u6Count := dp.ByAgeAndCareType[u6Label]["ganztag"]
		if u6Count != 1 {
			t.Errorf("month %d (%s): Ü3 ganztag count = %d, want 1", i+1, dp.Date, u6Count)
		}
	}
}

func TestStatisticsService_GetOccupancy_MultiplePeriodsWithDifferentRequirements(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Two funding periods with different care type configurations:
	// Period 1 (Jan-Jun): ganztag + halbtag available for ages 0-6
	// Period 2 (Jul-Dec): ganztag + halbtag available for ages 0-6 (same structure, different values)
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	p1To := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	p1 := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &p1To, 39.0)
	createTestFundingPropertyFull(t, db, p1.ID, "care_type", "ganztag", "Ganztag", 10000, 0.25, 0, 6)
	createTestFundingPropertyFull(t, db, p1.ID, "care_type", "halbtag", "Halbtag", 6000, 0.15, 0, 6)

	p2To := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	p2 := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), &p2To, 39.0)
	createTestFundingPropertyFull(t, db, p2.ID, "care_type", "ganztag", "Ganztag", 12000, 0.30, 0, 6)
	createTestFundingPropertyFull(t, db, p2.ID, "care_type", "halbtag", "Halbtag", 7000, 0.18, 0, 6)

	section := getDefaultSection(t, db, org.ID)

	// Child with ganztag contract covering all of 2024
	child := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Stable", LastName: "Child", Birthdate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(child)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, contractFrom, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetOccupancy(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 12 {
		t.Fatalf("expected 12 data points, got %d", len(result.DataPoints))
	}

	// The child should be counted in every month regardless of which funding period is active.
	// Occupancy counts are about headcount, not funding amounts.
	for i, dp := range result.DataPoints {
		if dp.Total != 1 {
			t.Errorf("month %d (%s): Total = %d, want 1", i+1, dp.Date, dp.Total)
		}
	}

	// Verify the age group structure is derived (from the most recent period per extractOccupancyStructure)
	if len(result.AgeGroups) == 0 {
		t.Error("expected at least one age group")
	}
	if len(result.CareTypes) == 0 {
		t.Error("expected at least one care type")
	}

	// The ganztag count should be 1 for every month in the appropriate age group
	ageLabel := formatAgeGroupLabel(0, 6)
	for i, dp := range result.DataPoints {
		ganztag := dp.ByAgeAndCareType[ageLabel]["ganztag"]
		if ganztag != 1 {
			t.Errorf("month %d (%s): ganztag count in %s = %d, want 1", i+1, dp.Date, ageLabel, ganztag)
		}
	}
}

// =============================================================================
// Complex multi-entity edge case tests
// =============================================================================

// Test 1: Multiple children with different ages matching different funding requirement rates.
// Verifies per-child age dispatch yields correct aggregate required hours.
func TestStatisticsService_GetStaffingHours_MultipleChildrenDifferentAgeRates(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Funding: U3 (ages 0-2) requirement=0.25, Ü3 (ages 3-6) requirement=0.15
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag U3", 200000, 0.25, 0, 2)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag Ü3", 100000, 0.15, 3, 6)

	section := getDefaultSection(t, db, org.ID)
	props := models.ContractProperties{"care_type": "ganztag"}
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// 3 children born 2023-01-01 → age 1 on 2024-06-01 (U3)
	for range 3 {
		c := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "U3", LastName: "Child", Birthdate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)}}
		db.Create(c)
		createTestChildContract(t, db, c.ID, contractFrom, nil, section.ID, props)
	}
	// 2 children born 2020-01-01 → age 4 on 2024-06-01 (Ü3)
	for range 2 {
		c := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "UE3", LastName: "Child", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}}
		db.Create(c)
		createTestChildContract(t, db, c.ID, contractFrom, nil, section.ID, props)
	}

	// 1 employee to have non-zero available hours
	payplan := createTestPayPlan(t, db, "TV-L", org.ID)
	emp := createTestEmployee(t, db, "Emp", "One", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, contractFrom, nil, 39.0, "qualified", section.ID)

	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetStaffingHours(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(result.DataPoints))
	}
	dp := result.DataPoints[0]

	// RequiredHours = 3*(0.25*39.0) + 2*(0.15*39.0) = 29.25 + 11.70 = 40.95
	wantRequired := 3*0.25*39.0 + 2*0.15*39.0
	if !almostEqual(dp.RequiredHours, wantRequired, 0.01) {
		t.Errorf("RequiredHours = %v, want %v", dp.RequiredHours, wantRequired)
	}
	if dp.ChildCount != 5 {
		t.Errorf("ChildCount = %d, want 5", dp.ChildCount)
	}
	if !almostEqual(dp.AvailableHours, 39.0, 0.01) {
		t.Errorf("AvailableHours = %v, want 39.0", dp.AvailableHours)
	}
}

// Test 2: Employee with consecutive contracts at different hours — verifies transition mid-range.
func TestStatisticsService_GetStaffingHours_EmployeeConsecutiveContracts(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 10000, 0.25, 0, 6)

	section := getDefaultSection(t, db, org.ID)

	// 1 child so we can verify the data points have content
	child := createTestChild(t, db, "Child", "One", org.ID)
	createTestChildContract(t, db, child.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	// Employee: 30h contract Jan-Mar, 39h contract Apr onwards
	payplan := createTestPayPlan(t, db, "TV-L", org.ID)
	emp := createTestEmployee(t, db, "Emp", "One", org.ID)
	contract1To := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &contract1To, 30.0, "qualified", section.ID)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, "qualified", section.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetStaffingHours(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 6 {
		t.Fatalf("expected 6 data points, got %d", len(result.DataPoints))
	}

	for i, dp := range result.DataPoints {
		if dp.StaffCount != 1 {
			t.Errorf("month %d: StaffCount = %d, want 1 (same employee)", i+1, dp.StaffCount)
		}
		if i < 3 {
			if !almostEqual(dp.AvailableHours, 30.0, 0.01) {
				t.Errorf("month %d: AvailableHours = %v, want 30.0", i+1, dp.AvailableHours)
			}
		} else {
			if !almostEqual(dp.AvailableHours, 39.0, 0.01) {
				t.Errorf("month %d: AvailableHours = %v, want 39.0", i+1, dp.AvailableHours)
			}
		}
	}
}

// Test 3: Child contract gap — child absent during gap months, counted before and after.
func TestStatisticsService_GetStaffingHours_ChildContractGap(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 10000, 0.25, 0, 6)

	section := getDefaultSection(t, db, org.ID)
	props := models.ContractProperties{"care_type": "ganztag"}

	// Child with gap: contract 1 Jan-Mar, contract 2 Jun onwards
	child := createTestChild(t, db, "Gap", "Child", org.ID)
	contract1To := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &contract1To, section.ID, props)
	createTestChildContract(t, db, child.ID, time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), nil, section.ID, props)

	// Employee present throughout
	payplan := createTestPayPlan(t, db, "TV-L", org.ID)
	emp := createTestEmployee(t, db, "Emp", "One", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp.ID, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 30.0, "qualified", section.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetStaffingHours(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 8 {
		t.Fatalf("expected 8 data points, got %d", len(result.DataPoints))
	}

	wantRequired := 0.25 * 39.0 // single child
	for i, dp := range result.DataPoints {
		month := i + 1 // Jan=1..Aug=8
		switch {
		case month <= 3: // Jan-Mar: active
			if dp.ChildCount != 1 {
				t.Errorf("month %d: ChildCount = %d, want 1", month, dp.ChildCount)
			}
			if !almostEqual(dp.RequiredHours, wantRequired, 0.01) {
				t.Errorf("month %d: RequiredHours = %v, want %v", month, dp.RequiredHours, wantRequired)
			}
		case month <= 5: // Apr-May: gap
			if dp.ChildCount != 0 {
				t.Errorf("month %d: ChildCount = %d, want 0 (gap)", month, dp.ChildCount)
			}
			if !almostEqual(dp.RequiredHours, 0.0, 0.01) {
				t.Errorf("month %d: RequiredHours = %v, want 0.0 (gap)", month, dp.RequiredHours)
			}
		default: // Jun-Aug: active again
			if dp.ChildCount != 1 {
				t.Errorf("month %d: ChildCount = %d, want 1", month, dp.ChildCount)
			}
			if !almostEqual(dp.RequiredHours, wantRequired, 0.01) {
				t.Errorf("month %d: RequiredHours = %v, want %v", month, dp.RequiredHours, wantRequired)
			}
		}
		// Employee always present
		if dp.StaffCount != 1 {
			t.Errorf("month %d: StaffCount = %d, want 1", month, dp.StaffCount)
		}
		if !almostEqual(dp.AvailableHours, 30.0, 0.01) {
			t.Errorf("month %d: AvailableHours = %v, want 30.0", month, dp.AvailableHours)
		}
	}
}

// Test 4: Full financials integration — multiple children (different ages/care types),
// multiple employees (different grades/steps/hours), budget items, all combined.
func TestStatisticsService_GetFinancials_FullIntegration(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Funding: U3 ganztag=200000, Ü3 ganztag=100000, halbtag=60000 (all ages)
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fpTo := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	fp := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &fpTo, 39.0)
	createTestFundingPropertyFull(t, db, fp.ID, "care_type", "ganztag", "Ganztag U3", 200000, 0.25, 0, 2)
	createTestFundingPropertyFull(t, db, fp.ID, "care_type", "ganztag", "Ganztag Ü3", 100000, 0.15, 3, 6)
	createTestFundingPropertyFull(t, db, fp.ID, "care_type", "halbtag", "Halbtag", 60000, 0.10, -1, -1) // all ages

	section := getDefaultSection(t, db, org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// 2 U3 ganztag children (born 2023 → age 1)
	for range 2 {
		c := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "U3", LastName: "GZ", Birthdate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)}}
		db.Create(c)
		createTestChildContract(t, db, c.ID, contractFrom, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})
	}
	// 1 Ü3 ganztag child (born 2020 → age 4)
	ue3 := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "UE3", LastName: "GZ", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(ue3)
	createTestChildContract(t, db, ue3.ID, contractFrom, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})
	// 1 halbtag child (born 2021 → age 3)
	ht := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "HT", LastName: "Child", Birthdate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(ht)
	createTestChildContract(t, db, ht.ID, contractFrom, nil, section.ID, models.ContractProperties{"care_type": "halbtag"})

	// Pay plan: S8a/3=350000, S4/2=280000, 22% employer contribution
	payplan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	ppPeriod := createTestPayPlanPeriodWithContrib(t, db, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, 2200)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S8a", 3, 350000, nil)
	createTestPayPlanEntry(t, db, ppPeriod.ID, "S4", 2, 280000, nil)

	// Employee 1: qualified, S8a/3, 39h (full-time)
	emp1 := createTestEmployee(t, db, "Emp", "Qualified", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp1.ID, payplan.ID, contractFrom, nil, 39.0, "qualified", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp1.ID).Updates(map[string]any{"grade": "S8a", "step": 3})

	// Employee 2: supplementary, S4/2, 20h (part-time)
	emp2 := createTestEmployee(t, db, "Emp", "Supplementary", org.ID)
	createTestEmployeeContractWithCategory(t, db, emp2.ID, payplan.ID, contractFrom, nil, 20.0, "supplementary", section.ID)
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", emp2.ID).Updates(map[string]any{"grade": "S4", "step": 2})

	// Budget items
	rent := createTestBudgetItem(t, db, "Rent", org.ID, "expense", false)
	createTestBudgetItemEntry(t, db, rent.ID, contractFrom, nil, 50000, "")
	meals := createTestBudgetItem(t, db, "Meals", org.ID, "expense", true) // per-child
	createTestBudgetItemEntry(t, db, meals.ID, contractFrom, nil, 3000, "")
	parentFees := createTestBudgetItem(t, db, "Parent Fees", org.ID, "income", true) // per-child
	createTestBudgetItemEntry(t, db, parentFees.ID, contractFrom, nil, 15000, "")

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

	// Funding: 2*200000 + 1*100000 + 1*60000 = 560000
	wantFunding := 2*200000 + 100000 + 60000
	if dp.FundingIncome != wantFunding {
		t.Errorf("FundingIncome = %d, want %d", dp.FundingIncome, wantFunding)
	}

	// Salary emp1: 350000 * 39/39 = 350000
	// Salary emp2: round(280000 * 20/39) = round(143589.74) = 143590
	grossEmp1 := 350000
	grossEmp2 := int(math.Round(280000.0 * 20.0 / 39.0))
	wantGross := grossEmp1 + grossEmp2
	if dp.GrossSalary != wantGross {
		t.Errorf("GrossSalary = %d, want %d", dp.GrossSalary, wantGross)
	}

	// Employer costs: round(350000*2200/10000) + round(143590*2200/10000)
	contribEmp1 := int(math.Round(float64(grossEmp1) * 2200.0 / 10000.0))
	contribEmp2 := int(math.Round(float64(grossEmp2) * 2200.0 / 10000.0))
	wantContrib := contribEmp1 + contribEmp2
	if dp.EmployerCosts != wantContrib {
		t.Errorf("EmployerCosts = %d, want %d", dp.EmployerCosts, wantContrib)
	}

	// Budget: income = 4 children * 15000 = 60000, expense = 50000 + 4*3000 = 62000
	childCount := 4
	wantBudgetIncome := childCount * 15000
	wantBudgetExpenses := 50000 + childCount*3000
	if dp.BudgetIncome != wantBudgetIncome {
		t.Errorf("BudgetIncome = %d, want %d", dp.BudgetIncome, wantBudgetIncome)
	}
	if dp.BudgetExpenses != wantBudgetExpenses {
		t.Errorf("BudgetExpenses = %d, want %d", dp.BudgetExpenses, wantBudgetExpenses)
	}

	// Totals
	wantTotalIncome := wantFunding + wantBudgetIncome
	wantTotalExpenses := wantGross + wantContrib + wantBudgetExpenses
	wantBalance := wantTotalIncome - wantTotalExpenses
	if dp.TotalIncome != wantTotalIncome {
		t.Errorf("TotalIncome = %d, want %d", dp.TotalIncome, wantTotalIncome)
	}
	if dp.TotalExpenses != wantTotalExpenses {
		t.Errorf("TotalExpenses = %d, want %d", dp.TotalExpenses, wantTotalExpenses)
	}
	if dp.Balance != wantBalance {
		t.Errorf("Balance = %d, want %d", dp.Balance, wantBalance)
	}
	if dp.ChildCount != childCount {
		t.Errorf("ChildCount = %d, want %d", dp.ChildCount, childCount)
	}
	if dp.StaffCount != 2 {
		t.Errorf("StaffCount = %d, want 2", dp.StaffCount)
	}
}

// Test 5: Financials with funding period transition mid-range — different payment rates before/after.
func TestStatisticsService_GetFinancials_FundingPeriodTransitionMidRange(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Two funding periods: Jan-Jun (ganztag=100000) and Jul-Dec (ganztag=150000)
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	p1To := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	p1 := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &p1To, 39.0)
	createTestFundingPropertyFull(t, db, p1.ID, "care_type", "ganztag", "Ganztag H1", 100000, 0.25, 0, 6)

	p2To := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	p2 := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), &p2To, 39.0)
	createTestFundingPropertyFull(t, db, p2.ID, "care_type", "ganztag", "Ganztag H2", 150000, 0.25, 0, 6)

	section := getDefaultSection(t, db, org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	props := models.ContractProperties{"care_type": "ganztag"}

	// 2 children with ganztag
	for range 2 {
		child := createTestChild(t, db, "Child", "X", org.ID)
		createTestChildContract(t, db, child.ID, contractFrom, nil, section.ID, props)
	}

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 12 {
		t.Fatalf("expected 12 data points, got %d", len(result.DataPoints))
	}

	for i, dp := range result.DataPoints {
		month := i + 1
		if month <= 6 {
			// 2 children * 100000 = 200000
			if dp.FundingIncome != 200000 {
				t.Errorf("month %d (%s): FundingIncome = %d, want 200000", month, dp.Date, dp.FundingIncome)
			}
		} else {
			// 2 children * 150000 = 300000
			if dp.FundingIncome != 300000 {
				t.Errorf("month %d (%s): FundingIncome = %d, want 300000", month, dp.Date, dp.FundingIncome)
			}
		}
		if dp.ChildCount != 2 {
			t.Errorf("month %d: ChildCount = %d, want 2", month, dp.ChildCount)
		}
	}
}

// Test 6: Financials — children with multiple matching properties (care_type + supplement)
// verifying correct per-child and aggregate funding income.
func TestStatisticsService_GetFinancials_MultipleMatchingProperties(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Funding: care_type ganztag=100000, integration_a=50000 (no age filter)
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fpTo := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	fp := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &fpTo, 39.0)
	createTestFundingPropertyFull(t, db, fp.ID, "care_type", "ganztag", "Ganztag", 100000, 0.25, -1, -1)
	createTestFundingPropertyFull(t, db, fp.ID, "integration", "integration_a", "Integration A", 50000, 0.10, -1, -1)

	section := getDefaultSection(t, db, org.ID)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Child A: ganztag + integration_a → 150000
	childA := createTestChild(t, db, "Child", "A", org.ID)
	createTestChildContract(t, db, childA.ID, contractFrom, nil, section.ID, models.ContractProperties{
		"care_type":   "ganztag",
		"integration": "integration_a",
	})

	// Child B: ganztag only → 100000
	childB := createTestChild(t, db, "Child", "B", org.ID)
	createTestChildContract(t, db, childB.ID, contractFrom, nil, section.ID, models.ContractProperties{
		"care_type": "ganztag",
	})

	// Child C: ganztag + integration_a → 150000
	childC := createTestChild(t, db, "Child", "C", org.ID)
	createTestChildContract(t, db, childC.ID, contractFrom, nil, section.ID, models.ContractProperties{
		"care_type":   "ganztag",
		"integration": "integration_a",
	})

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

	// Total: 150000 + 100000 + 150000 = 400000
	if dp.FundingIncome != 400000 {
		t.Errorf("FundingIncome = %d, want 400000", dp.FundingIncome)
	}
	if dp.ChildCount != 3 {
		t.Errorf("ChildCount = %d, want 3", dp.ChildCount)
	}

	// Verify FundingDetails: care_type:ganztag=300000, integration:integration_a=100000
	detailMap := make(map[string]int)
	for _, d := range dp.FundingDetails {
		detailMap[d.Key+":"+d.Value] = d.AmountCents
	}
	if detailMap["care_type:ganztag"] != 300000 {
		t.Errorf("FundingDetail care_type:ganztag = %d, want 300000", detailMap["care_type:ganztag"])
	}
	if detailMap["integration:integration_a"] != 100000 {
		t.Errorf("FundingDetail integration:integration_a = %d, want 100000", detailMap["integration:integration_a"])
	}
}

// Test 7: Occupancy full matrix — multiple children across age groups, care types, and supplements.
func TestStatisticsService_GetOccupancy_FullMatrix(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Funding: U3 (0-2) and Ü3 (3-6) × ganztag + halbtag, plus supplements integration_a, ndh
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fpTo := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	fp := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &fpTo, 39.0)
	createTestFundingPropertyFull(t, db, fp.ID, "care_type", "ganztag", "Ganztag U3", 200000, 0.25, 0, 2)
	createTestFundingPropertyFull(t, db, fp.ID, "care_type", "ganztag", "Ganztag Ü3", 100000, 0.15, 3, 6)
	createTestFundingPropertyFull(t, db, fp.ID, "care_type", "halbtag", "Halbtag U3", 120000, 0.15, 0, 2)
	createTestFundingPropertyFull(t, db, fp.ID, "care_type", "halbtag", "Halbtag Ü3", 80000, 0.10, 3, 6)
	createTestFundingPropertyFull(t, db, fp.ID, "integration", "integration_a", "Integration A", 50000, 0.10, -1, -1)
	createTestFundingPropertyFull(t, db, fp.ID, "supplements", "ndh", "NDH", 30000, 0.05, -1, -1)

	section := getDefaultSection(t, db, org.ID)
	contractFrom := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	type childSpec struct {
		birthYear int
		birthMon  time.Month
		props     models.ContractProperties
	}
	children := []childSpec{
		// Child A: age 1 (U3), ganztag, integration_a
		{2023, 1, models.ContractProperties{"care_type": "ganztag", "integration": "integration_a"}},
		// Child B: age 2 (U3), halbtag, ndh
		{2022, 1, models.ContractProperties{"care_type": "halbtag", "supplements": "ndh"}},
		// Child C: age 2 (U3), ganztag (no supplements)
		{2022, 6, models.ContractProperties{"care_type": "ganztag"}},
		// Child D: age 4 (Ü3), ganztag, ndh
		{2020, 1, models.ContractProperties{"care_type": "ganztag", "supplements": "ndh"}},
		// Child E: age 5 (Ü3), halbtag
		{2019, 1, models.ContractProperties{"care_type": "halbtag"}},
		// Child F: age 3 (Ü3), ganztag, integration_a + ndh
		{2021, 1, models.ContractProperties{"care_type": "ganztag", "integration": "integration_a", "supplements": "ndh"}},
	}

	for i, spec := range children {
		c := &models.Child{Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      string(rune('A' + i)),
			LastName:       "Child",
			Birthdate:      time.Date(spec.birthYear, spec.birthMon, 1, 0, 0, 0, 0, time.UTC),
		}}
		db.Create(c)
		createTestChildContract(t, db, c.ID, contractFrom, nil, section.ID, spec.props)
	}

	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetOccupancy(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(result.DataPoints))
	}
	dp := result.DataPoints[0]

	if dp.Total != 6 {
		t.Errorf("Total = %d, want 6", dp.Total)
	}

	u3Label := formatAgeGroupLabel(0, 2) // "0/1/2"
	u6Label := formatAgeGroupLabel(3, 6) // "3+"

	// ByAgeAndCareType checks
	checks := []struct {
		ageLabel string
		careType string
		want     int
	}{
		{u3Label, "ganztag", 2}, // A, C
		{u3Label, "halbtag", 1}, // B
		{u6Label, "ganztag", 2}, // D, F
		{u6Label, "halbtag", 1}, // E
	}
	for _, check := range checks {
		got := dp.ByAgeAndCareType[check.ageLabel][check.careType]
		if got != check.want {
			t.Errorf("ByAgeAndCareType[%s][%s] = %d, want %d", check.ageLabel, check.careType, got, check.want)
		}
	}

	// BySupplement checks
	// integration_a: A, F → 2
	if dp.BySupplement["integration_a"] != 2 {
		t.Errorf("BySupplement[integration_a] = %d, want 2", dp.BySupplement["integration_a"])
	}
	// ndh: B, D, F → 3
	if dp.BySupplement["ndh"] != 3 {
		t.Errorf("BySupplement[ndh] = %d, want 3", dp.BySupplement["ndh"])
	}

	// Verify structure metadata
	if len(result.AgeGroups) != 2 {
		t.Errorf("expected 2 age groups, got %d", len(result.AgeGroups))
	}
	if len(result.CareTypes) != 2 {
		t.Errorf("expected 2 care types, got %d", len(result.CareTypes))
	}
	if len(result.SupplementTypes) != 2 {
		t.Errorf("expected 2 supplement types, got %d", len(result.SupplementTypes))
	}
}

// Test 8: EmployeeStaffingHours — multiple employees with mid-range transitions, verifying
// sorting, staff category from latest contract, and correct monthly hours.
func TestStatisticsService_GetEmployeeStaffingHours_MultipleEmployeesTransition(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	payplan := createTestPayPlan(t, db, "TV-L", org.ID)

	// Employee "Alpha, A": contract 1 (Jan-Mar, 30h, supplementary), contract 2 (Apr-ongoing, 39h, qualified)
	empA := createTestEmployee(t, db, "A", "Alpha", org.ID)
	c1To := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	createTestEmployeeContractWithCategory(t, db, empA.ID, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &c1To, 30.0, "supplementary", section.ID)
	createTestEmployeeContractWithCategory(t, db, empA.ID, payplan.ID, time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, "qualified", section.ID)

	// Employee "Beta, B": ongoing contract (Jan-ongoing, 25h, qualified)
	empB := createTestEmployee(t, db, "B", "Beta", org.ID)
	createTestEmployeeContractWithCategory(t, db, empB.ID, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 25.0, "qualified", section.ID)

	// Employee "Alpha, C": ongoing contract (Jan-ongoing, 20h, qualified)
	// Same last name as empA to verify first name sorting
	empC := createTestEmployee(t, db, "C", "Alpha", org.ID)
	createTestEmployeeContractWithCategory(t, db, empC.ID, payplan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 20.0, "qualified", section.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetEmployeeStaffingHours(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Dates) != 6 {
		t.Fatalf("expected 6 dates, got %d", len(result.Dates))
	}
	if len(result.Employees) != 3 {
		t.Fatalf("expected 3 employees, got %d", len(result.Employees))
	}

	// Verify sorting: Alpha A, Alpha C, Beta B
	if result.Employees[0].LastName != "Alpha" || result.Employees[0].FirstName != "A" {
		t.Errorf("first employee = %s %s, want A Alpha", result.Employees[0].FirstName, result.Employees[0].LastName)
	}
	if result.Employees[1].LastName != "Alpha" || result.Employees[1].FirstName != "C" {
		t.Errorf("second employee = %s %s, want C Alpha", result.Employees[1].FirstName, result.Employees[1].LastName)
	}
	if result.Employees[2].LastName != "Beta" || result.Employees[2].FirstName != "B" {
		t.Errorf("third employee = %s %s, want B Beta", result.Employees[2].FirstName, result.Employees[2].LastName)
	}

	// Alpha A: staff category should be "qualified" (from latest contract starting Apr)
	if result.Employees[0].StaffCategory != "qualified" {
		t.Errorf("Alpha A StaffCategory = %s, want qualified", result.Employees[0].StaffCategory)
	}

	// Alpha A monthly hours: [30, 30, 30, 39, 39, 39]
	wantAlphaA := []float64{30, 30, 30, 39, 39, 39}
	for i, want := range wantAlphaA {
		got := result.Employees[0].MonthlyHours[i]
		if !almostEqual(got, want, 0.01) {
			t.Errorf("Alpha A month %d: hours = %v, want %v", i+1, got, want)
		}
	}

	// Alpha C: constant 20h
	for i := range 6 {
		got := result.Employees[1].MonthlyHours[i]
		if !almostEqual(got, 20.0, 0.01) {
			t.Errorf("Alpha C month %d: hours = %v, want 20.0", i+1, got)
		}
	}

	// Beta B: constant 25h
	for i := range 6 {
		got := result.Employees[2].MonthlyHours[i]
		if !almostEqual(got, 25.0, 0.01) {
			t.Errorf("Beta B month %d: hours = %v, want 25.0", i+1, got)
		}
	}
}

// ========== Actual Funding Integration Tests ==========

func TestStatisticsService_GetFinancials_ActualFunding_BillMatchesMonth(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "Bill User", "bill1@test.com", "password")

	// Create a bill for January 2025
	toJan := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	bill := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: &toJan},
		FileName:       "jan.xlsx",
		FileSha256:     "hash1",
		FacilityName:   "Kita Test",
		FacilityTotal:  500000,
		CreatedBy:      user.ID,
	}
	if err := db.Create(bill).Error; err != nil {
		t.Fatalf("create bill: %v", err)
	}

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("GetFinancials() error = %v", err)
	}

	// January should have actual funding
	janDP := findDataPoint(t, result.DataPoints, "2025-01-01")
	if janDP.ActualFunding == nil {
		t.Fatal("expected ActualFunding to be set for January")
	}
	if *janDP.ActualFunding != 500000 {
		t.Errorf("ActualFunding = %d, want 500000", *janDP.ActualFunding)
	}

	// February should have no actual funding
	febDP := findDataPoint(t, result.DataPoints, "2025-02-01")
	if febDP.ActualFunding != nil {
		t.Errorf("expected ActualFunding to be nil for February, got %d", *febDP.ActualFunding)
	}

	// March should have no actual funding
	marDP := findDataPoint(t, result.DataPoints, "2025-03-01")
	if marDP.ActualFunding != nil {
		t.Errorf("expected ActualFunding to be nil for March, got %d", *marDP.ActualFunding)
	}
}

func TestStatisticsService_GetFinancials_ActualFunding_SingleBill(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "Bill User", "bill2@test.com", "password")

	// Create a single bill for February (only one bill per month per org is allowed)
	toFeb := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)
	bill := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), To: &toFeb},
		FileName:       "feb.xlsx",
		FileSha256:     "febhash",
		FacilityName:   "Kita",
		FacilityTotal:  350000,
		CreatedBy:      user.ID,
	}
	if err := db.Create(bill).Error; err != nil {
		t.Fatalf("create bill: %v", err)
	}

	from := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("GetFinancials() error = %v", err)
	}

	dp := findDataPoint(t, result.DataPoints, "2025-02-01")
	if dp.ActualFunding == nil {
		t.Fatal("expected ActualFunding to be set")
	}
	if *dp.ActualFunding != 350000 {
		t.Errorf("ActualFunding = %d, want 350000", *dp.ActualFunding)
	}
}

func TestStatisticsService_GetFinancials_ActualFunding_NoBills(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("GetFinancials() error = %v", err)
	}

	for _, dp := range result.DataPoints {
		if dp.ActualFunding != nil {
			t.Errorf("expected ActualFunding nil for %s when no bills exist, got %d", dp.Date, *dp.ActualFunding)
		}
	}
}

func TestStatisticsService_GetFinancials_ActualFunding_BillOutsideRange(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "Bill User", "bill4@test.com", "password")

	// Create a bill for December 2024 — outside the query range
	toDec := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	bill := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC), To: &toDec},
		FileName:       "dec.xlsx",
		FileSha256:     "hash",
		FacilityName:   "Kita",
		FacilityTotal:  400000,
		CreatedBy:      user.ID,
	}
	if err := db.Create(bill).Error; err != nil {
		t.Fatalf("create bill: %v", err)
	}

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("GetFinancials() error = %v", err)
	}

	for _, dp := range result.DataPoints {
		if dp.ActualFunding != nil {
			t.Errorf("expected ActualFunding nil for %s (bill is outside range), got %d", dp.Date, *dp.ActualFunding)
		}
	}
}

func TestStatisticsService_GetFinancials_ActualFunding_DifferentOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	user := createTestUser(t, db, "Bill User", "bill5@test.com", "password")

	// Create bill for org2 in January
	toJan := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	bill := &models.GovernmentFundingBillPeriod{
		OrganizationID: org2.ID,
		Period:         models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: &toJan},
		FileName:       "org2.xlsx",
		FileSha256:     "hash",
		FacilityName:   "Kita Org2",
		FacilityTotal:  999999,
		CreatedBy:      user.ID,
	}
	if err := db.Create(bill).Error; err != nil {
		t.Fatalf("create bill: %v", err)
	}

	// Query org1 — should not see org2's bill
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org1.ID, &from, &to)
	if err != nil {
		t.Fatalf("GetFinancials() error = %v", err)
	}

	for _, dp := range result.DataPoints {
		if dp.ActualFunding != nil {
			t.Errorf("expected ActualFunding nil for org1 (bill belongs to org2), got %d", *dp.ActualFunding)
		}
	}
}

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

func findDataPoint(t *testing.T, dps []models.FinancialDataPoint, date string) models.FinancialDataPoint {
	t.Helper()
	for _, dp := range dps {
		if dp.Date == date {
			return dp
		}
	}
	t.Fatalf("data point for %s not found", date)
	return models.FinancialDataPoint{}
}
