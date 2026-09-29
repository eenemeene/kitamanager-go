package service

import (
	"context"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

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
