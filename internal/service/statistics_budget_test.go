package service

import (
	"context"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

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

// TestGetFinancials_BudgetOneEntryPerItem documented the (then-only)
// app-layer protection in calculate.go's `break // only first active
// entry per item` against double-counting two overlapping entries.
// Migration 000016 added a DB-level GIST exclusion constraint so the
// "two overlapping entries on the same item" data state can no
// longer exist — the protection has moved one layer down. Convert
// the test to assert the new gate: directly seeding the corrupt
// state must fail, and a single-entry calc still works.
func TestGetFinancials_BudgetItemOverlapImpossibleAtDBLayer(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	item := createTestBudgetItem(t, db, "Rent", org.ID, "expense", false)
	createTestBudgetItemEntry(t, db, item.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 30000, "first")

	// Attempt to seed an overlapping entry; the GIST EXCLUSION
	// constraint added in 000016 must reject. testutil's helper
	// t.Fatal-s on error, so use raw GORM here to test the rejection.
	overlap := &models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period:       models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		AmountCents:  70000,
		Notes:        "second",
	}
	if err := db.Create(overlap).Error; err == nil {
		t.Fatal("expected DB exclusion constraint to reject overlapping entry, got nil")
	}

	// And the single-entry case still computes the expected total.
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetFinancials(ctx, org.ID, &from, &to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	dp := result.DataPoints[0]
	if dp.BudgetExpenses != 30000 {
		t.Errorf("BudgetExpenses = %d, want 30000 (single entry)", dp.BudgetExpenses)
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
