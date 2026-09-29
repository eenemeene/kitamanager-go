package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

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
		CreatedBy:      &user.ID,
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
		CreatedBy:      &user.ID,
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
		CreatedBy:      &user.ID,
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
		CreatedBy:      &user.ID,
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

// TestStatisticsService_GetFinancials_AttributedCorrections asserts the
// financials response carries both keyings of the same money: arrival, which
// answers "what did we receive in March?", and attribution, which answers "was
// March funded correctly?".
//
// The bill is the shape ISBJ actually sends -- March's regular row plus a
// correction for January -- and the two questions have different answers for
// both months, which is the whole reason the month has to be persisted.
func TestStatisticsService_GetFinancials_AttributedCorrections(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Attributed Org")
	user := createTestUser(t, db, "Attr User", "attr_fin@test.com", "password")

	jan := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	mar := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	toMar := time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC)

	bill := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: mar, To: &toMar},
		FileName:       "mar.xlsx",
		FileSha256:     "attr-hash-mar",
		FacilityName:   "Kita Sonnenschein",
		FacilityTotal:  94752,
		CreatedBy:      &user.ID,
		Children: []models.GovernmentFundingBillChild{
			{
				VoucherNumber: "GB-12345678901-02",
				ChildName:     "Musterkind, Max",
				BirthDate:     "01.20",
				District:      1,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "ganztag", Amount: 94650,
						RowType: models.RowTypeRegular, BillingMonth: &mar},
					{Key: "care_type", Value: "ganztag", Amount: 102,
						RowType: models.RowTypeCorrection, BillingMonth: &jan},
				},
			},
		},
	}
	if err := db.Create(bill).Error; err != nil {
		t.Fatalf("create bill: %v", err)
	}

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &mar)
	if err != nil {
		t.Fatalf("GetFinancials() error = %v", err)
	}

	marDP := findDataPoint(t, result.DataPoints, "2025-03-01")
	// Arrival: the correction was paid out in March, so it counts there.
	if marDP.ActualFundingCorrection == nil || *marDP.ActualFundingCorrection != 102 {
		t.Errorf("March arrival correction = %v, want 102", marDP.ActualFundingCorrection)
	}
	// Attribution: March's own row only. The correction belongs to January.
	if marDP.ActualFundingCorrectionAttributed == nil || *marDP.ActualFundingCorrectionAttributed != 0 {
		t.Errorf("March attributed correction = %v, want 0", marDP.ActualFundingCorrectionAttributed)
	}
	if marDP.ActualFundingRegularAttributed == nil || *marDP.ActualFundingRegularAttributed != 94650 {
		t.Errorf("March attributed regular = %v, want 94650", marDP.ActualFundingRegularAttributed)
	}

	janDP := findDataPoint(t, result.DataPoints, "2025-01-01")
	// No bill arrived in January, so the arrival-keyed fields stay unset --
	// including ActualFunding, which the UI uses to decide the month has a
	// bill at all and must not start reporting one on the strength of a
	// correction that arrived two months later.
	if janDP.ActualFunding != nil {
		t.Errorf("January ActualFunding = %d, want nil", *janDP.ActualFunding)
	}
	if janDP.ActualFundingCorrection != nil {
		t.Errorf("January arrival correction = %d, want nil", *janDP.ActualFundingCorrection)
	}
	// Attribution reaches back into January even though nothing arrived there.
	if janDP.ActualFundingCorrectionAttributed == nil {
		t.Fatal("January attributed correction is nil; the March bill corrected it")
	}
	if *janDP.ActualFundingCorrectionAttributed != 102 {
		t.Errorf("January attributed correction = %d, want 102", *janDP.ActualFundingCorrectionAttributed)
	}

	// Whichever keying is read, the same total must come back out.
	sumArrival, sumAttributed := 0, 0
	for _, dp := range result.DataPoints {
		if dp.ActualFundingRegular != nil {
			sumArrival += *dp.ActualFundingRegular
		}
		if dp.ActualFundingCorrection != nil {
			sumArrival += *dp.ActualFundingCorrection
		}
		if dp.ActualFundingRegularAttributed != nil {
			sumAttributed += *dp.ActualFundingRegularAttributed
		}
		if dp.ActualFundingCorrectionAttributed != nil {
			sumAttributed += *dp.ActualFundingCorrectionAttributed
		}
	}
	if sumArrival != sumAttributed || sumArrival != 94752 {
		t.Errorf("totals: arrival %d, attributed %d, want both 94752", sumArrival, sumAttributed)
	}
}

// TestStatisticsService_GetFinancials_AttributedSetForEveryMonth pins the
// nil/zero distinction the two keyings depend on.
//
// A pure Korrektur-Abrechnung -- a bill whose every row is about an earlier
// month, which ISBJ does send -- has a bill for its own month and nothing
// attributed to it. Leaving the attributed fields nil there made them
// indistinguishable from "this server cannot compute them", and a client doing
// `attributed ?? arrival` counted the corrections twice: once in the month the
// bill arrived, once in the month it corrects. 102 cents of correction read as
// 204.
func TestStatisticsService_GetFinancials_AttributedSetForEveryMonth(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Orphan Attr Org")
	user := createTestUser(t, db, "Orphan User", "orphan_attr@test.com", "password")

	jan := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	mar := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	toMar := time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC)

	bill := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: mar, To: &toMar},
		FileName:       "mar-korrektur.xlsx",
		FileSha256:     "orphan-attr-mar",
		FacilityName:   "Kita Sonnenschein",
		FacilityTotal:  102,
		CreatedBy:      &user.ID,
		Children: []models.GovernmentFundingBillChild{
			{
				VoucherNumber: "GB-12345678901-02",
				ChildName:     "Musterkind, Max",
				BirthDate:     "01.20",
				District:      1,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "ganztag", Amount: 102,
						RowType: models.RowTypeCorrection, BillingMonth: &jan},
				},
			},
		},
	}
	if err := db.Create(bill).Error; err != nil {
		t.Fatalf("create bill: %v", err)
	}

	from := jan
	result, err := svc.GetFinancials(ctx, org.ID, &from, &mar)
	if err != nil {
		t.Fatalf("GetFinancials() error = %v", err)
	}

	// Every month in range carries both attributed figures, whether or not
	// anything is attributed to it. Nil is reserved for "could not compute".
	for _, dp := range result.DataPoints {
		if dp.ActualFundingRegularAttributed == nil {
			t.Errorf("%s: ActualFundingRegularAttributed is nil; nil means the server could not compute it", dp.Date)
		}
		if dp.ActualFundingCorrectionAttributed == nil {
			t.Errorf("%s: ActualFundingCorrectionAttributed is nil", dp.Date)
		}
	}

	marDP := findDataPoint(t, result.DataPoints, "2025-03-01")
	// March has a bill, and arrival-keyed it carries the whole correction.
	if marDP.ActualFunding == nil || *marDP.ActualFunding != 102 {
		t.Errorf("March ActualFunding = %v, want 102", marDP.ActualFunding)
	}
	if marDP.ActualFundingCorrection == nil || *marDP.ActualFundingCorrection != 102 {
		t.Errorf("March arrival correction = %v, want 102", marDP.ActualFundingCorrection)
	}
	// Attributed, March has nothing: every row of its bill is about January.
	if marDP.ActualFundingCorrectionAttributed == nil || *marDP.ActualFundingCorrectionAttributed != 0 {
		t.Errorf("March attributed correction = %v, want 0", marDP.ActualFundingCorrectionAttributed)
	}
	if marDP.ActualFundingRegularAttributed == nil || *marDP.ActualFundingRegularAttributed != 0 {
		t.Errorf("March attributed regular = %v, want 0", marDP.ActualFundingRegularAttributed)
	}

	janDP := findDataPoint(t, result.DataPoints, "2025-01-01")
	if janDP.ActualFundingCorrectionAttributed == nil || *janDP.ActualFundingCorrectionAttributed != 102 {
		t.Errorf("January attributed correction = %v, want 102", janDP.ActualFundingCorrectionAttributed)
	}

	// The reading a client actually makes: attributed where present, arrival
	// only as a fallback. It must total the bill, not twice the bill.
	attributedOrArrival := func(attr, arrival *int) int {
		if attr != nil {
			return *attr
		}
		if arrival != nil {
			return *arrival
		}
		return 0
	}
	total := 0
	for _, dp := range result.DataPoints {
		total += attributedOrArrival(dp.ActualFundingRegularAttributed, dp.ActualFundingRegular)
		total += attributedOrArrival(dp.ActualFundingCorrectionAttributed, dp.ActualFundingCorrection)
	}
	if total != 102 {
		t.Errorf("client-side total = %d, want 102 (the bill); 204 means the correction was counted twice", total)
	}
}
