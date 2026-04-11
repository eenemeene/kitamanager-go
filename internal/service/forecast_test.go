package service

import (
	"context"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// setupForecastTestData creates a complete test environment with org, funding, pay plan,
// employees, children, and budget items. Returns all created entities for use in forecast tests.
type forecastTestData struct {
	org           *models.Organization
	section       *models.Section
	payplan       *models.PayPlan
	payplanPeriod *models.PayPlanPeriod
	fundingPeriod *models.GovernmentFundingPeriod
	emp1          *models.Employee
	emp2          *models.Employee
	child1        *models.Child
	child2        *models.Child
	budgetItem    *models.BudgetItem
}

func setupForecastTestData(t *testing.T) (*StatisticsService, forecastTestData) {
	t.Helper()
	db := setupTestDB(t)
	svc := createStatisticsService(db)

	org := createTestOrganization(t, db, "Forecast Org")
	db.Model(org).Update("state", "berlin")
	section := getDefaultSection(t, db, org.ID)

	// Government funding: care_type=ganztag with payment and requirement
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingTo := time.Date(2027, 7, 31, 0, 0, 0, 0, time.UTC)
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID,
		time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC), &fundingTo, 39.0)
	createTestFundingPropertyFull(t, db, fundingPeriod.ID,
		"care_type", "ganztag", "Ganztag", 100000, 0.25, 0, 6) // 1000.00 EUR, 0.25 requirement

	// Pay plan with period and entry
	payplan := createTestPayPlan(t, db, "TV-L", org.ID)
	ppTo := time.Date(2027, 7, 31, 0, 0, 0, 0, time.UTC)
	payplanPeriod := createTestPayPlanPeriodWithContrib(t, db, payplan.ID,
		time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC), &ppTo, 39.0, 2000) // 20% employer contribution
	createTestPayPlanEntry(t, db, payplanPeriod.ID, "S8a", 3, 350000, nil) // 3500.00 EUR

	// 2 employees with qualified contracts (Grade/Step must match pay plan entry)
	emp1 := createTestEmployee(t, db, "Emp", "One", org.ID)
	emp2 := createTestEmployee(t, db, "Emp", "Two", org.ID)
	contractFrom := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	emp1Contract := createTestEmployeeContractWithCategory(t, db, emp1.ID, payplan.ID, contractFrom, nil, 39.0, "qualified", section.ID)
	db.Model(emp1Contract).Updates(map[string]interface{}{"grade": "S8a", "step": 3})
	emp2Contract := createTestEmployeeContractWithCategory(t, db, emp2.ID, payplan.ID, contractFrom, nil, 30.0, "qualified", section.ID)
	db.Model(emp2Contract).Updates(map[string]interface{}{"grade": "S8a", "step": 3})

	// 2 children with ganztag contracts
	child1 := createTestChild(t, db, "Child", "One", org.ID)
	child2 := createTestChild(t, db, "Child", "Two", org.ID)
	props := models.ContractProperties{"care_type": "ganztag"}
	createTestChildContract(t, db, child1.ID, contractFrom, nil, section.ID, props)
	createTestChildContract(t, db, child2.ID, contractFrom, nil, section.ID, props)

	// Budget item: income, 500 EUR/month per child
	budgetItem := createTestBudgetItem(t, db, "Elternbeiträge", org.ID, "income", true)
	createTestBudgetItemEntry(t, db, budgetItem.ID, contractFrom, nil, 50000, "Monthly")

	return svc, forecastTestData{
		org: org, section: section, payplan: payplan, payplanPeriod: payplanPeriod,
		fundingPeriod: fundingPeriod, emp1: emp1, emp2: emp2, child1: child1, child2: child2,
		budgetItem: budgetItem,
	}
}

func TestGetForecast_EmptyOverlay(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ForecastRequest{From: &from, To: &to}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify all four response sections are present
	if result.Financials == nil {
		t.Fatal("expected financials in response")
	}
	if result.StaffingHours == nil {
		t.Fatal("expected staffing_hours in response")
	}
	if result.Occupancy == nil {
		t.Fatal("expected occupancy in response")
	}
	if result.EmployeeStaffingHours == nil {
		t.Fatal("expected employee_staffing_hours in response")
	}

	// 6 months of data
	if len(result.Financials.DataPoints) != 6 {
		t.Errorf("expected 6 financial data points, got %d", len(result.Financials.DataPoints))
	}
	if len(result.StaffingHours.DataPoints) != 6 {
		t.Errorf("expected 6 staffing data points, got %d", len(result.StaffingHours.DataPoints))
	}

	// Check staffing: 2 employees, 39+30=69 available hours
	dp := result.StaffingHours.DataPoints[0]
	if dp.StaffCount != 2 {
		t.Errorf("expected staff_count=2, got %d", dp.StaffCount)
	}
	if !almostEqual(dp.AvailableHours, 69.0, 0.01) {
		t.Errorf("expected available_hours=69.0, got %v", dp.AvailableHours)
	}
	// Required: 2 children * 0.25 * 39.0 = 19.5
	if !almostEqual(dp.RequiredHours, 19.5, 0.01) {
		t.Errorf("expected required_hours=19.5, got %v", dp.RequiredHours)
	}
}

func TestGetForecast_AddEmployee(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	contractFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	req := &models.ForecastRequest{
		From: &from,
		To:   &to,
		AddEmployees: []models.ForecastAddEmployee{
			{
				FirstName: "New",
				LastName:  "Employee",
				Gender:    "female",
				Birthdate: time.Date(1985, 6, 15, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ForecastAddEmployeeContract{
					{
						From:          contractFrom,
						SectionID:     td.section.ID,
						StaffCategory: "qualified",
						Grade:         "S8a",
						Step:          3,
						WeeklyHours:   20.0,
						PayPlanID:     td.payplan.ID,
					},
				},
			},
		},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Now 3 staff members: 39 + 30 + 20 = 89 available hours
	dp := result.StaffingHours.DataPoints[0]
	if dp.StaffCount != 3 {
		t.Errorf("expected staff_count=3, got %d", dp.StaffCount)
	}
	if !almostEqual(dp.AvailableHours, 89.0, 0.01) {
		t.Errorf("expected available_hours=89.0, got %v", dp.AvailableHours)
	}
}

func TestGetForecast_RemoveEmployee(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	req := &models.ForecastRequest{
		From:              &from,
		To:                &to,
		RemoveEmployeeIDs: []uint{td.emp2.ID},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Only 1 employee left: 39 available hours
	dp := result.StaffingHours.DataPoints[0]
	if dp.StaffCount != 1 {
		t.Errorf("expected staff_count=1, got %d", dp.StaffCount)
	}
	if !almostEqual(dp.AvailableHours, 39.0, 0.01) {
		t.Errorf("expected available_hours=39.0, got %v", dp.AvailableHours)
	}
}

func TestGetForecast_AddChild(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	req := &models.ForecastRequest{
		From: &from,
		To:   &to,
		AddChildren: []models.ForecastAddChild{
			{
				FirstName: "New",
				LastName:  "Child",
				Gender:    "male",
				Birthdate: time.Date(2023, 5, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ForecastAddChildContract{
					{
						From:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
						SectionID:  td.section.ID,
						Properties: models.ContractProperties{"care_type": "ganztag"},
					},
				},
			},
		},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Now 3 children: 3 * 0.25 * 39.0 = 29.25 required hours
	dp := result.StaffingHours.DataPoints[0]
	if dp.ChildCount != 3 {
		t.Errorf("expected child_count=3, got %d", dp.ChildCount)
	}
	if !almostEqual(dp.RequiredHours, 29.25, 0.01) {
		t.Errorf("expected required_hours=29.25, got %v", dp.RequiredHours)
	}
}

func TestGetForecast_RemoveChild(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	req := &models.ForecastRequest{
		From:           &from,
		To:             &to,
		RemoveChildIDs: []uint{td.child1.ID},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 1 child: 1 * 0.25 * 39.0 = 9.75
	dp := result.StaffingHours.DataPoints[0]
	if dp.ChildCount != 1 {
		t.Errorf("expected child_count=1, got %d", dp.ChildCount)
	}
	if !almostEqual(dp.RequiredHours, 9.75, 0.01) {
		t.Errorf("expected required_hours=9.75, got %v", dp.RequiredHours)
	}
}

func TestGetForecast_AddFundingPeriod(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	// The existing funding period runs 2025-08-01 to 2027-07-31.
	// Query AFTER it ends so only the overlay period is active.
	// Children are born 2020-01-01 (from testutil), so at 2027-08-01 they're age 7.
	// Use MaxAge=10 so age-7 children still match the overlay funding property.
	from := time.Date(2027, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2027, 12, 1, 0, 0, 0, 0, time.UTC)

	req := &models.ForecastRequest{
		From: &from,
		To:   &to,
		AddFundingPeriods: []models.ForecastAddFundingPeriod{
			{
				From:                time.Date(2027, 8, 1, 0, 0, 0, 0, time.UTC),
				FullTimeWeeklyHours: 39.0,
				Properties: []models.ForecastFundingProperty{
					{
						Key:         "care_type",
						Value:       "ganztag",
						Label:       "Ganztag",
						Payment:     120000, // 1200.00 EUR (higher than original 1000.00)
						Requirement: 0.30,   // higher requirement than original 0.25
						MinAge:      intPtr(0),
						MaxAge:      intPtr(10),
					},
				},
			},
		},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.StaffingHours.DataPoints) == 0 {
		t.Fatal("no staffing data points returned")
	}
	// Only overlay period is active after 2027-07-31.
	// 2 children * 0.30 * 39.0 = 23.4
	dp := result.StaffingHours.DataPoints[0]
	if !almostEqual(dp.RequiredHours, 23.4, 0.01) {
		t.Errorf("expected required_hours=23.4, got %v", dp.RequiredHours)
	}
}

func TestGetForecast_AddBudgetItem(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	req := &models.ForecastRequest{
		From: &from,
		To:   &to,
		AddBudgetItems: []models.ForecastAddBudgetItem{
			{
				Name:     "New Rent",
				Category: "expense",
				PerChild: false,
				Entries: []models.ForecastBudgetItemEntry{
					{
						From:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
						AmountCents: 200000, // 2000.00 EUR
					},
				},
			},
		},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Budget expenses should include the new 2000.00 EUR rent
	dp := result.Financials.DataPoints[0]
	if dp.BudgetExpenses < 200000 {
		t.Errorf("expected budget_expenses >= 200000, got %d", dp.BudgetExpenses)
	}
}

func TestGetForecast_RemoveBudgetItem(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	// First check baseline has budget income
	baseReq := &models.ForecastRequest{From: &from, To: &to}
	baseResult, err := svc.GetForecast(ctx, td.org.ID, baseReq)
	if err != nil {
		t.Fatalf("baseline error: %v", err)
	}
	baseBudgetIncome := baseResult.Financials.DataPoints[0].BudgetIncome

	// Now remove the budget item
	req := &models.ForecastRequest{
		From:                &from,
		To:                  &to,
		RemoveBudgetItemIDs: []uint{td.budgetItem.ID},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.Financials.DataPoints[0]
	if dp.BudgetIncome >= baseBudgetIncome && baseBudgetIncome > 0 {
		t.Errorf("expected budget_income to decrease after removing budget item, got %d (baseline %d)", dp.BudgetIncome, baseBudgetIncome)
	}
}

func TestGetForecast_ValidateOverlay_WrongOrg(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()
	db := setupTestDB(t) // fresh DB for other org
	otherOrg := createTestOrganization(t, db, "Other Org")

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	// Try to remove an employee from a different org
	req := &models.ForecastRequest{
		From:              &from,
		To:                &to,
		RemoveEmployeeIDs: []uint{td.emp1.ID},
	}
	_, err := svc.GetForecast(ctx, otherOrg.ID, req)
	if err == nil {
		t.Fatal("expected error when removing employee from wrong org")
	}
}

func TestGetForecast_ValidateOverlay_InvalidPayPlan(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	// Try to add employee with non-existent pay plan
	req := &models.ForecastRequest{
		From: &from,
		To:   &to,
		AddEmployees: []models.ForecastAddEmployee{
			{
				FirstName: "Bad",
				LastName:  "Employee",
				Gender:    "male",
				Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ForecastAddEmployeeContract{
					{
						From:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
						SectionID:     td.section.ID,
						StaffCategory: "qualified",
						WeeklyHours:   39.0,
						PayPlanID:     999999, // non-existent
					},
				},
			},
		},
	}

	_, err := svc.GetForecast(ctx, td.org.ID, req)
	if err == nil {
		t.Fatal("expected error for non-existent pay plan")
	}
}

func TestApplyOverlay_EndEmployeeContract(t *testing.T) {
	endDate := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	ds := &DataSet{
		Employees: []models.Employee{
			{
				Person: models.Person{ID: 1},
				Contracts: []models.EmployeeContract{
					{ID: 10, EmployeeID: 1, BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}}},
				},
			},
		},
	}

	req := &models.ForecastRequest{
		EndEmployeeContracts: []models.ForecastEndContract{
			{ContractID: 10, EndDate: endDate},
		},
	}

	applyOverlay(ds, req, nil)

	if ds.Employees[0].Contracts[0].To == nil {
		t.Fatal("expected contract To to be set")
	}
	if !ds.Employees[0].Contracts[0].To.Equal(endDate) {
		t.Errorf("expected contract To=%v, got %v", endDate, *ds.Employees[0].Contracts[0].To)
	}
}

func TestApplyOverlay_EndChildContract(t *testing.T) {
	endDate := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	ds := &DataSet{
		Children: []models.Child{
			{
				Person: models.Person{ID: 1},
				Contracts: []models.ChildContract{
					{ID: 20, ChildID: 1, BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}}},
				},
			},
		},
	}

	req := &models.ForecastRequest{
		EndChildContracts: []models.ForecastEndContract{
			{ContractID: 20, EndDate: endDate},
		},
	}

	applyOverlay(ds, req, nil)

	if ds.Children[0].Contracts[0].To == nil {
		t.Fatal("expected contract To to be set")
	}
	if !ds.Children[0].Contracts[0].To.Equal(endDate) {
		t.Errorf("expected contract To=%v, got %v", endDate, *ds.Children[0].Contracts[0].To)
	}
}

func TestDataSet_PedagogicalEmployees(t *testing.T) {
	ds := &DataSet{
		Employees: []models.Employee{
			{
				Person: models.Person{ID: 1, FirstName: "Qualified"},
				Contracts: []models.EmployeeContract{
					{StaffCategory: "qualified", WeeklyHours: 39.0},
				},
			},
			{
				Person: models.Person{ID: 2, FirstName: "NonPed"},
				Contracts: []models.EmployeeContract{
					{StaffCategory: "non_pedagogical", WeeklyHours: 20.0},
				},
			},
			{
				Person: models.Person{ID: 3, FirstName: "Mixed"},
				Contracts: []models.EmployeeContract{
					{StaffCategory: "qualified", WeeklyHours: 30.0},
					{StaffCategory: "non_pedagogical", WeeklyHours: 10.0},
				},
			},
		},
	}

	ped := ds.PedagogicalEmployees()
	if len(ped) != 2 {
		t.Fatalf("expected 2 pedagogical employees, got %d", len(ped))
	}

	// Employee 1: all contracts kept
	if ped[0].ID != 1 {
		t.Errorf("expected first ped employee ID=1, got %d", ped[0].ID)
	}

	// Employee 3: only qualified contract kept
	if ped[1].ID != 3 {
		t.Errorf("expected second ped employee ID=3, got %d", ped[1].ID)
	}
	if len(ped[1].Contracts) != 1 {
		t.Errorf("expected 1 contract for mixed employee, got %d", len(ped[1].Contracts))
	}
	if ped[1].Contracts[0].StaffCategory != "qualified" {
		t.Errorf("expected qualified contract, got %s", ped[1].Contracts[0].StaffCategory)
	}

	// Verify original DataSet is not mutated (employee 3 still has 2 contracts)
	if len(ds.Employees[2].Contracts) != 2 {
		t.Errorf("original employee should still have 2 contracts, got %d", len(ds.Employees[2].Contracts))
	}
}

func TestApplyOverlay_AddPayPlanPeriod(t *testing.T) {
	ds := &DataSet{
		PayPlans: map[uint]*models.PayPlan{
			1: {
				ID:      1,
				Periods: []models.PayPlanPeriod{},
			},
		},
	}

	req := &models.ForecastRequest{
		AddPayPlanPeriods: []models.ForecastAddPayPlanPeriod{
			{
				PayPlanID:                1,
				From:                     time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
				WeeklyHours:              39.0,
				EmployerContributionRate: 2200,
				Entries: []models.ForecastPayPlanEntry{
					{Grade: "S8a", Step: 3, MonthlyAmount: 380000},
				},
			},
		},
	}

	applyOverlay(ds, req, nil)

	pp := ds.PayPlans[1]
	if len(pp.Periods) != 1 {
		t.Fatalf("expected 1 period, got %d", len(pp.Periods))
	}
	if len(pp.Periods[0].Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(pp.Periods[0].Entries))
	}
	if pp.Periods[0].Entries[0].MonthlyAmount != 380000 {
		t.Errorf("expected monthly_amount=380000, got %d", pp.Periods[0].Entries[0].MonthlyAmount)
	}
}

func TestApplyOverlay_AddContractToExistingEmployee(t *testing.T) {
	ds := &DataSet{
		Employees: []models.Employee{
			{
				Person: models.Person{ID: 5},
				Contracts: []models.EmployeeContract{
					{ID: 50, EmployeeID: 5, BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}}},
				},
			},
		},
	}

	req := &models.ForecastRequest{
		AddEmployeeContracts: []models.ForecastAddEmployeeContract{
			{
				EmployeeID:    5,
				From:          time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
				SectionID:     1,
				StaffCategory: "supplementary",
				WeeklyHours:   20.0,
				PayPlanID:     1,
			},
		},
	}

	applyOverlay(ds, req, nil)

	if len(ds.Employees[0].Contracts) != 2 {
		t.Fatalf("expected 2 contracts, got %d", len(ds.Employees[0].Contracts))
	}
	if ds.Employees[0].Contracts[1].StaffCategory != "supplementary" {
		t.Errorf("expected supplementary, got %s", ds.Employees[0].Contracts[1].StaffCategory)
	}
}

func TestApplyOverlay_AddFundingPeriod_UnitTest(t *testing.T) {
	// Simulate: 1 child with ganztag, no existing funding, overlay adds a period
	child := models.Child{
		Person: models.Person{ID: 1, Birthdate: time.Date(2023, 5, 1, 0, 0, 0, 0, time.UTC)},
		Contracts: []models.ChildContract{
			{
				ID:      1,
				ChildID: 1,
				BaseContract: models.BaseContract{
					Period:     models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
					Properties: models.ContractProperties{"care_type": "ganztag"},
				},
			},
		},
	}

	ds := &DataSet{
		Children:       []models.Child{child},
		FundingPeriods: []models.GovernmentFundingPeriod{},
		PayPlans:       make(map[uint]*models.PayPlan),
	}

	req := &models.ForecastRequest{
		AddFundingPeriods: []models.ForecastAddFundingPeriod{
			{
				From:                time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				FullTimeWeeklyHours: 39.0,
				Properties: []models.ForecastFundingProperty{
					{
						Key:         "care_type",
						Value:       "ganztag",
						Label:       "Ganztag",
						Payment:     100000,
						Requirement: 0.25,
						MinAge:      intPtr(0),
						MaxAge:      intPtr(6),
					},
				},
			},
		},
	}

	applyOverlay(ds, req, nil)

	if len(ds.FundingPeriods) != 1 {
		t.Fatalf("expected 1 funding period after overlay, got %d", len(ds.FundingPeriods))
	}

	if len(ds.FundingPeriods[0].Properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(ds.FundingPeriods[0].Properties))
	}

	// Now call calculateStaffingHours directly
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	dataPoints := calculateStaffingHours(ds.Children, nil, ds.FundingPeriods, start, end)

	if len(dataPoints) == 0 {
		t.Fatal("no data points")
	}

	dp := dataPoints[0]
	t.Logf("child_count=%d, required=%v, date=%s", dp.ChildCount, dp.RequiredHours, dp.Date)
	// 1 child * 0.25 * 39.0 = 9.75
	if !almostEqual(dp.RequiredHours, 9.75, 0.01) {
		t.Errorf("expected required_hours=9.75, got %v", dp.RequiredHours)
	}
}

// TestGetForecast_RemoveAndReAddChild removes a child and adds back a virtual child
// with identical data. All calculations should produce the same results as the baseline.
func TestGetForecast_RemoveAndReAddChild(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	// Baseline: no overlay
	baseReq := &models.ForecastRequest{From: &from, To: &to}
	baseResult, err := svc.GetForecast(ctx, td.org.ID, baseReq)
	if err != nil {
		t.Fatalf("baseline error: %v", err)
	}

	// Remove child1 and add back a virtual child with same birthdate, same contract
	contractFrom := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ForecastRequest{
		From:           &from,
		To:             &to,
		RemoveChildIDs: []uint{td.child1.ID},
		AddChildren: []models.ForecastAddChild{
			{
				FirstName: "Virtual",
				LastName:  "Child",
				Gender:    "female",
				Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), // same as test children
				Contracts: []models.ForecastAddChildContract{
					{
						From:       contractFrom,
						SectionID:  td.section.ID,
						Properties: models.ContractProperties{"care_type": "ganztag"},
					},
				},
			},
		},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should have same child count, required hours, and funding income
	for i := range baseResult.StaffingHours.DataPoints {
		baseDp := baseResult.StaffingHours.DataPoints[i]
		dp := result.StaffingHours.DataPoints[i]
		if dp.ChildCount != baseDp.ChildCount {
			t.Errorf("month %d: child_count %d != baseline %d", i, dp.ChildCount, baseDp.ChildCount)
		}
		if !almostEqual(dp.RequiredHours, baseDp.RequiredHours, 0.01) {
			t.Errorf("month %d: required_hours %v != baseline %v", i, dp.RequiredHours, baseDp.RequiredHours)
		}
	}
	for i := range baseResult.Financials.DataPoints {
		baseDp := baseResult.Financials.DataPoints[i]
		dp := result.Financials.DataPoints[i]
		if dp.FundingIncome != baseDp.FundingIncome {
			t.Errorf("month %d: funding_income %d != baseline %d", i, dp.FundingIncome, baseDp.FundingIncome)
		}
	}
}

// TestGetForecast_RemoveAndReAddEmployee removes an employee and adds back
// a virtual one with identical data. Staffing and salary should match baseline.
func TestGetForecast_RemoveAndReAddEmployee(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	// Baseline
	baseReq := &models.ForecastRequest{From: &from, To: &to}
	baseResult, err := svc.GetForecast(ctx, td.org.ID, baseReq)
	if err != nil {
		t.Fatalf("baseline error: %v", err)
	}

	// Remove emp1 (39h qualified) and re-add identical virtual employee
	contractFrom := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	req := &models.ForecastRequest{
		From:              &from,
		To:                &to,
		RemoveEmployeeIDs: []uint{td.emp1.ID},
		AddEmployees: []models.ForecastAddEmployee{
			{
				FirstName: "Virtual",
				LastName:  "Employee",
				Gender:    "male",
				Birthdate: time.Date(1985, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ForecastAddEmployeeContract{
					{
						From:          contractFrom,
						SectionID:     td.section.ID,
						StaffCategory: "qualified",
						Grade:         "S8a",
						Step:          3,
						WeeklyHours:   39.0,
						PayPlanID:     td.payplan.ID,
					},
				},
			},
		},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for i := range baseResult.StaffingHours.DataPoints {
		baseDp := baseResult.StaffingHours.DataPoints[i]
		dp := result.StaffingHours.DataPoints[i]
		if dp.StaffCount != baseDp.StaffCount {
			t.Errorf("month %d: staff_count %d != baseline %d", i, dp.StaffCount, baseDp.StaffCount)
		}
		if !almostEqual(dp.AvailableHours, baseDp.AvailableHours, 0.01) {
			t.Errorf("month %d: available_hours %v != baseline %v", i, dp.AvailableHours, baseDp.AvailableHours)
		}
	}
	for i := range baseResult.Financials.DataPoints {
		baseDp := baseResult.Financials.DataPoints[i]
		dp := result.Financials.DataPoints[i]
		if dp.GrossSalary != baseDp.GrossSalary {
			t.Errorf("month %d: gross_salary %d != baseline %d", i, dp.GrossSalary, baseDp.GrossSalary)
		}
	}
}

// TestGetForecast_FutureChildNoImpact adds a child starting 10 years in the future.
// It should have no impact on the queried date range.
func TestGetForecast_FutureChildNoImpact(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	// Baseline
	baseReq := &models.ForecastRequest{From: &from, To: &to}
	baseResult, err := svc.GetForecast(ctx, td.org.ID, baseReq)
	if err != nil {
		t.Fatalf("baseline error: %v", err)
	}

	// Add child starting in 2036
	req := &models.ForecastRequest{
		From: &from,
		To:   &to,
		AddChildren: []models.ForecastAddChild{
			{
				FirstName: "Future",
				LastName:  "Child",
				Gender:    "female",
				Birthdate: time.Date(2033, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ForecastAddChildContract{
					{
						From:       time.Date(2036, 8, 1, 0, 0, 0, 0, time.UTC),
						SectionID:  td.section.ID,
						Properties: models.ContractProperties{"care_type": "ganztag"},
					},
				},
			},
		},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Child count and required hours should be identical to baseline
	for i := range baseResult.StaffingHours.DataPoints {
		baseDp := baseResult.StaffingHours.DataPoints[i]
		dp := result.StaffingHours.DataPoints[i]
		if dp.ChildCount != baseDp.ChildCount {
			t.Errorf("month %d: child_count %d != baseline %d", i, dp.ChildCount, baseDp.ChildCount)
		}
		if !almostEqual(dp.RequiredHours, baseDp.RequiredHours, 0.01) {
			t.Errorf("month %d: required_hours %v != baseline %v", i, dp.RequiredHours, baseDp.RequiredHours)
		}
	}
}

// TestGetForecast_FutureEmployeeNoImpact adds an employee starting 10 years in the future.
func TestGetForecast_FutureEmployeeNoImpact(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	baseReq := &models.ForecastRequest{From: &from, To: &to}
	baseResult, err := svc.GetForecast(ctx, td.org.ID, baseReq)
	if err != nil {
		t.Fatalf("baseline error: %v", err)
	}

	req := &models.ForecastRequest{
		From: &from,
		To:   &to,
		AddEmployees: []models.ForecastAddEmployee{
			{
				FirstName: "Future",
				LastName:  "Employee",
				Gender:    "male",
				Birthdate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ForecastAddEmployeeContract{
					{
						From:          time.Date(2036, 8, 1, 0, 0, 0, 0, time.UTC),
						SectionID:     td.section.ID,
						StaffCategory: "qualified",
						Grade:         "S8a",
						Step:          3,
						WeeklyHours:   39.0,
						PayPlanID:     td.payplan.ID,
					},
				},
			},
		},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for i := range baseResult.StaffingHours.DataPoints {
		baseDp := baseResult.StaffingHours.DataPoints[i]
		dp := result.StaffingHours.DataPoints[i]
		if dp.StaffCount != baseDp.StaffCount {
			t.Errorf("month %d: staff_count %d != baseline %d", i, dp.StaffCount, baseDp.StaffCount)
		}
		if !almostEqual(dp.AvailableHours, baseDp.AvailableHours, 0.01) {
			t.Errorf("month %d: available_hours %v != baseline %v", i, dp.AvailableHours, baseDp.AvailableHours)
		}
	}
}

// TestGetForecast_PayPlanSalaryIncrease adds a new pay plan period with 10% higher salaries.
// Salary costs in the new period should increase by ~10%.
func TestGetForecast_PayPlanSalaryIncrease(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	// Query range spanning the boundary: existing period ends 2027-07-31, new starts 2027-08-01
	from := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2027, 9, 1, 0, 0, 0, 0, time.UTC)

	// Original: 350000 cents (3500 EUR). 10% increase: 385000 cents (3850 EUR).
	req := &models.ForecastRequest{
		From: &from,
		To:   &to,
		AddPayPlanPeriods: []models.ForecastAddPayPlanPeriod{
			{
				PayPlanID:                td.payplan.ID,
				From:                     time.Date(2027, 8, 1, 0, 0, 0, 0, time.UTC),
				WeeklyHours:              39.0,
				EmployerContributionRate: 2000, // same 20%
				Entries: []models.ForecastPayPlanEntry{
					{Grade: "S8a", Step: 3, MonthlyAmount: 385000}, // +10%
				},
			},
		},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Financials.DataPoints) < 2 {
		t.Fatalf("expected at least 2 data points, got %d", len(result.Financials.DataPoints))
	}

	// July 2027: still on old period (350000 base)
	dpJuly := result.Financials.DataPoints[0]
	// August 2027: new period (385000 base)
	dpAug := result.Financials.DataPoints[1]

	if dpAug.GrossSalary <= dpJuly.GrossSalary {
		t.Errorf("expected August salary (%d) > July salary (%d) after 10%% increase",
			dpAug.GrossSalary, dpJuly.GrossSalary)
	}

	// Verify ~10% increase: emp1=385000*39/39=385000, emp2=385000*30/39≈296154
	// Old: emp1=350000, emp2=350000*30/39≈269231. Total old=619231, new=681154
	// Ratio should be ~1.10
	ratio := float64(dpAug.GrossSalary) / float64(dpJuly.GrossSalary)
	if !almostEqual(ratio, 1.10, 0.01) {
		t.Errorf("expected ~10%% salary increase, got ratio %.4f (july=%d, aug=%d)",
			ratio, dpJuly.GrossSalary, dpAug.GrossSalary)
	}
}

// TestGetForecast_EndEmployeeContractMidRange ends an employee contract in the middle
// of the queried range. Earlier months should have more available hours than later ones.
func TestGetForecast_EndEmployeeContractMidRange(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	// Baseline
	baseReq := &models.ForecastRequest{From: &from, To: &to}
	baseResult, err := svc.GetForecast(ctx, td.org.ID, baseReq)
	if err != nil {
		t.Fatalf("baseline error: %v", err)
	}

	// Replace emp2 with a virtual employee whose contract ends 2026-03-31.
	// This simulates an employee leaving mid-range.
	contractFrom := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	contractTo := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	req := &models.ForecastRequest{
		From:              &from,
		To:                &to,
		RemoveEmployeeIDs: []uint{td.emp2.ID},
		AddEmployees: []models.ForecastAddEmployee{
			{
				FirstName: "Temp",
				LastName:  "Worker",
				Gender:    "female",
				Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ForecastAddEmployeeContract{
					{
						From:          contractFrom,
						To:            &contractTo,
						SectionID:     td.section.ID,
						StaffCategory: "qualified",
						Grade:         "S8a",
						Step:          3,
						WeeklyHours:   30.0,
						PayPlanID:     td.payplan.ID,
					},
				},
			},
		},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Jan-Mar: 2 staff (emp1=39h + virtual=30h = 69h) — same as baseline
	// Apr-May: 1 staff (emp1=39h only)
	dpJan := result.StaffingHours.DataPoints[0]
	dpApr := result.StaffingHours.DataPoints[3]

	if dpJan.StaffCount != baseResult.StaffingHours.DataPoints[0].StaffCount {
		t.Errorf("Jan staff_count=%d, expected baseline %d", dpJan.StaffCount, baseResult.StaffingHours.DataPoints[0].StaffCount)
	}
	if !almostEqual(dpJan.AvailableHours, 69.0, 0.01) {
		t.Errorf("Jan available_hours=%v, expected 69.0", dpJan.AvailableHours)
	}
	if dpApr.StaffCount != 1 {
		t.Errorf("Apr staff_count=%d, expected 1", dpApr.StaffCount)
	}
	if !almostEqual(dpApr.AvailableHours, 39.0, 0.01) {
		t.Errorf("Apr available_hours=%v, expected 39.0", dpApr.AvailableHours)
	}
}

// TestGetForecast_AddEmployeeMidRange adds an employee starting in the middle of the range.
// Only months after the start date should include the new employee.
func TestGetForecast_AddEmployeeMidRange(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	req := &models.ForecastRequest{
		From: &from,
		To:   &to,
		AddEmployees: []models.ForecastAddEmployee{
			{
				FirstName: "MidYear",
				LastName:  "Hire",
				Gender:    "male",
				Birthdate: time.Date(1995, 3, 15, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ForecastAddEmployeeContract{
					{
						From:          time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
						SectionID:     td.section.ID,
						StaffCategory: "qualified",
						Grade:         "S8a",
						Step:          3,
						WeeklyHours:   25.0,
						PayPlanID:     td.payplan.ID,
					},
				},
			},
		},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Jan-Mar: 2 staff, 69h (unchanged)
	dpJan := result.StaffingHours.DataPoints[0]
	if dpJan.StaffCount != 2 {
		t.Errorf("Jan staff_count=%d, expected 2", dpJan.StaffCount)
	}
	if !almostEqual(dpJan.AvailableHours, 69.0, 0.01) {
		t.Errorf("Jan available_hours=%v, expected 69.0", dpJan.AvailableHours)
	}

	// Apr+: 3 staff, 69+25=94h
	dpApr := result.StaffingHours.DataPoints[3]
	if dpApr.StaffCount != 3 {
		t.Errorf("Apr staff_count=%d, expected 3", dpApr.StaffCount)
	}
	if !almostEqual(dpApr.AvailableHours, 94.0, 0.01) {
		t.Errorf("Apr available_hours=%v, expected 94.0", dpApr.AvailableHours)
	}
}

// TestGetForecast_AddNonPedagogicalEmployee adds a non-pedagogical employee.
// Staffing hours should be unchanged but salary costs should increase.
func TestGetForecast_AddNonPedagogicalEmployee(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	baseReq := &models.ForecastRequest{From: &from, To: &to}
	baseResult, err := svc.GetForecast(ctx, td.org.ID, baseReq)
	if err != nil {
		t.Fatalf("baseline error: %v", err)
	}

	req := &models.ForecastRequest{
		From: &from,
		To:   &to,
		AddEmployees: []models.ForecastAddEmployee{
			{
				FirstName: "Cook",
				LastName:  "Helper",
				Gender:    "female",
				Birthdate: time.Date(1988, 7, 20, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ForecastAddEmployeeContract{
					{
						From:          time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
						SectionID:     td.section.ID,
						StaffCategory: "non_pedagogical",
						Grade:         "S8a",
						Step:          3,
						WeeklyHours:   20.0,
						PayPlanID:     td.payplan.ID,
					},
				},
			},
		},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Staffing hours: unchanged (non-pedagogical not counted)
	for i := range baseResult.StaffingHours.DataPoints {
		baseDp := baseResult.StaffingHours.DataPoints[i]
		dp := result.StaffingHours.DataPoints[i]
		if dp.StaffCount != baseDp.StaffCount {
			t.Errorf("month %d: staff_count %d != baseline %d (non-ped should not affect staffing)", i, dp.StaffCount, baseDp.StaffCount)
		}
		if !almostEqual(dp.AvailableHours, baseDp.AvailableHours, 0.01) {
			t.Errorf("month %d: available_hours %v != baseline %v", i, dp.AvailableHours, baseDp.AvailableHours)
		}
	}

	// Salary costs: should increase (non-pedagogical still gets paid)
	for i := range baseResult.Financials.DataPoints {
		baseDp := baseResult.Financials.DataPoints[i]
		dp := result.Financials.DataPoints[i]
		if dp.GrossSalary <= baseDp.GrossSalary {
			t.Errorf("month %d: gross_salary %d should be > baseline %d (non-ped adds cost)", i, dp.GrossSalary, baseDp.GrossSalary)
		}
	}
}

// TestGetForecast_ChildWithUnmatchedCareType adds a child whose care_type doesn't
// match any funding property. The child should be counted but contribute 0 required hours.
func TestGetForecast_ChildWithUnmatchedCareType(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	baseReq := &models.ForecastRequest{From: &from, To: &to}
	baseResult, err := svc.GetForecast(ctx, td.org.ID, baseReq)
	if err != nil {
		t.Fatalf("baseline error: %v", err)
	}

	// Add child with care_type "halbtag" which has no matching funding property
	req := &models.ForecastRequest{
		From: &from,
		To:   &to,
		AddChildren: []models.ForecastAddChild{
			{
				FirstName: "Halbtag",
				LastName:  "Child",
				Gender:    "male",
				Birthdate: time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ForecastAddChildContract{
					{
						From:       time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
						SectionID:  td.section.ID,
						Properties: models.ContractProperties{"care_type": "halbtag"},
					},
				},
			},
		},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Child count increases by 1
	dp := result.StaffingHours.DataPoints[0]
	baseDp := baseResult.StaffingHours.DataPoints[0]
	if dp.ChildCount != baseDp.ChildCount+1 {
		t.Errorf("child_count=%d, expected baseline+1=%d", dp.ChildCount, baseDp.ChildCount+1)
	}

	// Required hours unchanged (halbtag has no matching funding, so 0 requirement added)
	if !almostEqual(dp.RequiredHours, baseDp.RequiredHours, 0.01) {
		t.Errorf("required_hours=%v, expected baseline %v (unmatched care_type adds 0)", dp.RequiredHours, baseDp.RequiredHours)
	}
}

// TestGetForecast_CombinedOverlay tests multiple overlay types in a single request:
// remove one employee, add another, remove one child, add two new children.
func TestGetForecast_CombinedOverlay(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	contractFrom := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)

	req := &models.ForecastRequest{
		From:              &from,
		To:                &to,
		RemoveEmployeeIDs: []uint{td.emp2.ID},   // remove 30h employee
		RemoveChildIDs:    []uint{td.child1.ID}, // remove 1 child
		AddEmployees: []models.ForecastAddEmployee{
			{
				FirstName: "New",
				LastName:  "Staff",
				Gender:    "female",
				Birthdate: time.Date(1992, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ForecastAddEmployeeContract{
					{
						From:          contractFrom,
						SectionID:     td.section.ID,
						StaffCategory: "qualified",
						Grade:         "S8a",
						Step:          3,
						WeeklyHours:   35.0,
						PayPlanID:     td.payplan.ID,
					},
				},
			},
		},
		AddChildren: []models.ForecastAddChild{
			{
				FirstName: "Extra", LastName: "ChildA", Gender: "female",
				Birthdate: time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ForecastAddChildContract{
					{From: contractFrom, SectionID: td.section.ID, Properties: models.ContractProperties{"care_type": "ganztag"}},
				},
			},
			{
				FirstName: "Extra", LastName: "ChildB", Gender: "male",
				Birthdate: time.Date(2022, 7, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ForecastAddChildContract{
					{From: contractFrom, SectionID: td.section.ID, Properties: models.ContractProperties{"care_type": "ganztag"}},
				},
			},
		},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dp := result.StaffingHours.DataPoints[0]

	// Employees: removed emp2(30h), kept emp1(39h), added new(35h) → 2 staff, 74h
	if dp.StaffCount != 2 {
		t.Errorf("staff_count=%d, expected 2", dp.StaffCount)
	}
	if !almostEqual(dp.AvailableHours, 74.0, 0.01) {
		t.Errorf("available_hours=%v, expected 74.0", dp.AvailableHours)
	}

	// Children: had 2, removed 1, added 2 → 3 children
	// 3 * 0.25 * 39.0 = 29.25
	if dp.ChildCount != 3 {
		t.Errorf("child_count=%d, expected 3", dp.ChildCount)
	}
	if !almostEqual(dp.RequiredHours, 29.25, 0.01) {
		t.Errorf("required_hours=%v, expected 29.25", dp.RequiredHours)
	}
}

// TestGetForecast_PerChildBudgetItemWithAddedChildren tests that adding children
// increases income from per-child budget items (e.g. Elternbeiträge).
func TestGetForecast_PerChildBudgetItemWithAddedChildren(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	// Baseline: 2 children, budget item is per_child income at 500 EUR/month
	baseReq := &models.ForecastRequest{From: &from, To: &to}
	baseResult, err := svc.GetForecast(ctx, td.org.ID, baseReq)
	if err != nil {
		t.Fatalf("baseline error: %v", err)
	}

	// Add 1 more child → 3 children, per-child income should increase by 50%
	req := &models.ForecastRequest{
		From: &from,
		To:   &to,
		AddChildren: []models.ForecastAddChild{
			{
				FirstName: "Third", LastName: "Kid", Gender: "male",
				Birthdate: time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ForecastAddChildContract{
					{
						From:       time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
						SectionID:  td.section.ID,
						Properties: models.ContractProperties{"care_type": "ganztag"},
					},
				},
			},
		},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for i := range baseResult.Financials.DataPoints {
		baseDp := baseResult.Financials.DataPoints[i]
		dp := result.Financials.DataPoints[i]
		// Budget income should scale with child count: 3/2 = 1.5x
		if baseDp.BudgetIncome > 0 {
			ratio := float64(dp.BudgetIncome) / float64(baseDp.BudgetIncome)
			if !almostEqual(ratio, 1.5, 0.01) {
				t.Errorf("month %d: budget_income ratio=%.4f, expected 1.5 (base=%d, overlay=%d)",
					i, ratio, baseDp.BudgetIncome, dp.BudgetIncome)
			}
		}
	}
}

// TestGetForecast_EmployerContributionRateChange adds a new pay plan period
// with the same salary but a higher employer contribution rate.
func TestGetForecast_EmployerContributionRateChange(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	// Query range spanning the boundary
	from := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2027, 9, 1, 0, 0, 0, 0, time.UTC)

	// Same salary (350000), but employer contribution goes from 20% to 25%
	req := &models.ForecastRequest{
		From: &from,
		To:   &to,
		AddPayPlanPeriods: []models.ForecastAddPayPlanPeriod{
			{
				PayPlanID:                td.payplan.ID,
				From:                     time.Date(2027, 8, 1, 0, 0, 0, 0, time.UTC),
				WeeklyHours:              39.0,
				EmployerContributionRate: 2500, // 25% (was 2000 = 20%)
				Entries: []models.ForecastPayPlanEntry{
					{Grade: "S8a", Step: 3, MonthlyAmount: 350000}, // same salary
				},
			},
		},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Financials.DataPoints) < 2 {
		t.Fatalf("expected at least 2 data points, got %d", len(result.Financials.DataPoints))
	}

	dpJuly := result.Financials.DataPoints[0]
	dpAug := result.Financials.DataPoints[1]

	// Gross salary should be the same
	if dpJuly.GrossSalary != dpAug.GrossSalary {
		t.Errorf("gross salary should be same: july=%d, aug=%d", dpJuly.GrossSalary, dpAug.GrossSalary)
	}

	// Employer costs should increase: 25% vs 20% = 1.25x
	if dpAug.EmployerCosts <= dpJuly.EmployerCosts {
		t.Errorf("expected Aug employer_costs (%d) > July (%d) after rate increase",
			dpAug.EmployerCosts, dpJuly.EmployerCosts)
	}
	ratio := float64(dpAug.EmployerCosts) / float64(dpJuly.EmployerCosts)
	if !almostEqual(ratio, 1.25, 0.01) {
		t.Errorf("expected ~25%% increase in employer costs, got ratio %.4f (july=%d, aug=%d)",
			ratio, dpJuly.EmployerCosts, dpAug.EmployerCosts)
	}
}

// TestGetForecast_EndChildContractMidRange ends a child's contract mid-range.
// Later months should have fewer children and less required staffing.
func TestGetForecast_EndChildContractMidRange(t *testing.T) {
	svc, td := setupForecastTestData(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	// Replace child1 with a virtual child whose contract ends 2026-03-31
	contractFrom := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	contractTo := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	req := &models.ForecastRequest{
		From:           &from,
		To:             &to,
		RemoveChildIDs: []uint{td.child1.ID},
		AddChildren: []models.ForecastAddChild{
			{
				FirstName: "Leaving", LastName: "Child", Gender: "female",
				Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ForecastAddChildContract{
					{
						From:       contractFrom,
						To:         &contractTo,
						SectionID:  td.section.ID,
						Properties: models.ContractProperties{"care_type": "ganztag"},
					},
				},
			},
		},
	}

	result, err := svc.GetForecast(ctx, td.org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Jan-Mar: 2 children (child2 + virtual), 2 * 0.25 * 39 = 19.5
	dpJan := result.StaffingHours.DataPoints[0]
	if dpJan.ChildCount != 2 {
		t.Errorf("Jan child_count=%d, expected 2", dpJan.ChildCount)
	}
	if !almostEqual(dpJan.RequiredHours, 19.5, 0.01) {
		t.Errorf("Jan required_hours=%v, expected 19.5", dpJan.RequiredHours)
	}

	// Apr+: 1 child (only child2), 1 * 0.25 * 39 = 9.75
	dpApr := result.StaffingHours.DataPoints[3]
	if dpApr.ChildCount != 1 {
		t.Errorf("Apr child_count=%d, expected 1", dpApr.ChildCount)
	}
	if !almostEqual(dpApr.RequiredHours, 9.75, 0.01) {
		t.Errorf("Apr required_hours=%v, expected 9.75", dpApr.RequiredHours)
	}
}
