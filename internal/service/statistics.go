package service

import (
	"context"
	"math"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// StatisticsService handles cross-resource statistics calculations
type StatisticsService struct {
	childStore    store.ChildStorer
	employeeStore store.EmployeeStorer
	orgStore      store.OrganizationStorer
	fundingStore  store.GovernmentFundingStorer
	payPlanStore  store.PayPlanStorer
	costStore     store.CostStorer
}

// NewStatisticsService creates a new statistics service
func NewStatisticsService(childStore store.ChildStorer, employeeStore store.EmployeeStorer, orgStore store.OrganizationStorer, fundingStore store.GovernmentFundingStorer, payPlanStore store.PayPlanStorer, costStore store.CostStorer) *StatisticsService {
	return &StatisticsService{
		childStore:    childStore,
		employeeStore: employeeStore,
		orgStore:      orgStore,
		fundingStore:  fundingStore,
		payPlanStore:  payPlanStore,
		costStore:     costStore,
	}
}

// pedagogicalCategories lists staff categories counted toward staffing requirements
var pedagogicalCategories = []string{
	string(models.StaffCategoryQualified),
	string(models.StaffCategorySupplementary),
}

// snapDateRange returns a date range snapped to 1st-of-month with defaults.
func snapDateRange(from, to *time.Time) (time.Time, time.Time) {
	now := time.Now()
	var rangeStart, rangeEnd time.Time
	if from != nil {
		rangeStart = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		rangeStart = time.Date(now.Year(), now.Month()-12, 1, 0, 0, 0, 0, time.UTC)
	}
	if to != nil {
		rangeEnd = time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		rangeEnd = time.Date(now.Year(), now.Month()+6, 1, 0, 0, 0, 0, time.UTC)
	}
	return rangeStart, rangeEnd
}

// GetStaffingHours calculates monthly staffing hours data points
func (s *StatisticsService) GetStaffingHours(ctx context.Context, orgID uint, from, to *time.Time, sectionID *uint) (*models.StaffingHoursResponse, error) {
	rangeStart, rangeEnd := snapDateRange(from, to)

	// Fetch organization for state
	org, err := s.orgStore.FindByID(ctx, orgID)
	if err != nil {
		return nil, apperror.NotFound("organization")
	}

	// Fetch government funding with all periods and properties
	var fundingPeriods []models.GovernmentFundingPeriod
	funding, err := s.fundingStore.FindByStateWithDetails(ctx, org.State, 0, nil)
	if err == nil {
		fundingPeriods = funding.Periods
	}
	// If no funding found, fundingPeriods stays nil — required hours will be 0

	// Fetch children with contracts in range
	children, err := s.childStore.FindByOrganizationInDateRange(ctx, orgID, rangeStart, rangeEnd, sectionID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children")
	}

	// Fetch employee contracts in range (pedagogical staff only)
	employeeContracts, err := s.employeeStore.FindContractsByOrganizationInDateRange(ctx, orgID, rangeStart, rangeEnd, pedagogicalCategories, sectionID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch employee contracts")
	}

	// Generate data points for each month
	var dataPoints []models.StaffingHoursDataPoint
	for date := rangeStart; !date.After(rangeEnd); date = date.AddDate(0, 1, 0) {
		dp := models.StaffingHoursDataPoint{
			Date: date.Format(models.DateFormat),
		}

		// Calculate required hours from children
		requiredHours := 0.0
		childCount := 0
		period := findPeriodForDate(fundingPeriods, date)
		for i := range children {
			child := &children[i]
			// Check if child has a contract active on this date
			for j := range child.Contracts {
				contract := &child.Contracts[j]
				if contract.IsActiveOn(date) {
					childCount++
					if period != nil {
						age := validation.CalculateAgeOnDate(child.Birthdate, date)
						requirement := sumChildRequirement(age, contract.Properties, period)
						requiredHours += requirement * period.FullTimeWeeklyHours
					}
					break // Only count each child once per month
				}
			}
		}

		// Calculate available hours from employee contracts
		availableHours := 0.0
		staffCount := 0
		employeeSeen := make(map[uint]bool)
		for i := range employeeContracts {
			ec := &employeeContracts[i]
			if ec.IsActiveOn(date) {
				availableHours += ec.WeeklyHours
				if !employeeSeen[ec.EmployeeID] {
					employeeSeen[ec.EmployeeID] = true
					staffCount++
				}
			}
		}

		dp.RequiredHours = requiredHours
		dp.AvailableHours = availableHours
		dp.ChildCount = childCount
		dp.StaffCount = staffCount
		dataPoints = append(dataPoints, dp)
	}

	return &models.StaffingHoursResponse{
		DataPoints: dataPoints,
	}, nil
}

// GetFinancials calculates monthly financial data points (income, expenses, balance)
func (s *StatisticsService) GetFinancials(ctx context.Context, orgID uint, from, to *time.Time, sectionID *uint) (*models.FinancialResponse, error) {
	rangeStart, rangeEnd := snapDateRange(from, to)

	// Fetch organization for state
	org, err := s.orgStore.FindByID(ctx, orgID)
	if err != nil {
		return nil, apperror.NotFound("organization")
	}

	// Fetch government funding with all periods and properties
	var fundingPeriods []models.GovernmentFundingPeriod
	funding, err := s.fundingStore.FindByStateWithDetails(ctx, org.State, 0, nil)
	if err == nil {
		fundingPeriods = funding.Periods
	}

	// Fetch children with contracts in range
	children, err := s.childStore.FindByOrganizationInDateRange(ctx, orgID, rangeStart, rangeEnd, sectionID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children")
	}

	// Fetch ALL employee contracts in range (salaries apply to all staff, not just pedagogical)
	employeeContracts, err := s.employeeStore.FindContractsByOrganizationInDateRange(ctx, orgID, rangeStart, rangeEnd, nil, sectionID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch employee contracts")
	}

	// Collect unique PayPlanIDs and fetch pay plans with periods+entries
	payPlanMap := make(map[uint]*models.PayPlan)
	for i := range employeeContracts {
		ppID := employeeContracts[i].PayPlanID
		if ppID == 0 || payPlanMap[ppID] != nil {
			continue
		}
		pp, err := s.payPlanStore.FindByIDWithPeriods(ctx, ppID, nil)
		if err != nil {
			continue // skip pay plans that can't be loaded
		}
		payPlanMap[ppID] = pp
	}

	// Fetch all costs with entries for operating cost calculation
	costs, err := s.costStore.FindByOrganizationWithEntries(ctx, orgID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch costs")
	}

	// Generate data points for each month
	var dataPoints []models.FinancialDataPoint
	for date := rangeStart; !date.After(rangeEnd); date = date.AddDate(0, 1, 0) {
		dp := models.FinancialDataPoint{
			Date: date.Format(models.DateFormat),
		}

		// Income: government funding for active children
		fundingIncome := 0
		childCount := 0
		fundingPeriod := findPeriodForDate(fundingPeriods, date)
		for i := range children {
			child := &children[i]
			for j := range child.Contracts {
				contract := &child.Contracts[j]
				if contract.IsActiveOn(date) {
					childCount++
					age := validation.CalculateAgeOnDate(child.Birthdate, date)
					payment, _ := sumChildFundingMatch(age, contract.Properties, fundingPeriod)
					fundingIncome += payment
					break
				}
			}
		}

		// Expenses: employee salaries
		grossSalary := 0
		employerCosts := 0
		staffCount := 0
		employeeSeen := make(map[uint]bool)
		for i := range employeeContracts {
			ec := &employeeContracts[i]
			if !ec.IsActiveOn(date) {
				continue
			}
			if !employeeSeen[ec.EmployeeID] {
				employeeSeen[ec.EmployeeID] = true
				staffCount++
			}

			pp := payPlanMap[ec.PayPlanID]
			if pp == nil {
				continue
			}
			period := findPayPlanPeriodForDate(pp.Periods, date)
			if period == nil {
				continue
			}
			entry := findPayPlanEntry(period.Entries, ec.Grade, ec.Step)
			if entry == nil {
				continue
			}

			gross := int(math.Round(float64(entry.MonthlyAmount) * ec.WeeklyHours / period.WeeklyHours))
			contrib := int(math.Round(float64(gross) * float64(period.EmployerContributionRate) / 10000.0))
			grossSalary += gross
			employerCosts += contrib
		}

		// Expenses: operating costs
		operatingCost := 0
		for i := range costs {
			for j := range costs[i].Entries {
				entry := &costs[i].Entries[j]
				if entry.IsActiveOn(date) {
					operatingCost += entry.AmountCents
					break // one active entry per cost category
				}
			}
		}

		dp.FundingIncome = fundingIncome
		dp.GrossSalary = grossSalary
		dp.EmployerCosts = employerCosts
		dp.OperatingCost = operatingCost
		dp.TotalIncome = fundingIncome
		dp.TotalExpenses = grossSalary + employerCosts + operatingCost
		dp.Balance = dp.TotalIncome - dp.TotalExpenses
		dp.ChildCount = childCount
		dp.StaffCount = staffCount
		dataPoints = append(dataPoints, dp)
	}

	return &models.FinancialResponse{
		DataPoints: dataPoints,
	}, nil
}

// findPayPlanPeriodForDate finds the pay plan period covering a date.
func findPayPlanPeriodForDate(periods []models.PayPlanPeriod, date time.Time) *models.PayPlanPeriod {
	for i := range periods {
		if periods[i].IsActiveOn(date) {
			return &periods[i]
		}
	}
	return nil
}

// findPayPlanEntry finds the entry matching grade+step in a period's entries.
func findPayPlanEntry(entries []models.PayPlanEntry, grade string, step int) *models.PayPlanEntry {
	for i := range entries {
		if entries[i].Grade == grade && entries[i].Step == step {
			return &entries[i]
		}
	}
	return nil
}
