package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// StatisticsService handles cross-resource statistics calculations
type StatisticsService struct {
	childStore      store.ChildStorer
	employeeStore   store.EmployeeStorer
	orgStore        store.OrganizationStorer
	fundingStore    store.GovernmentFundingStorer
	payPlanStore    store.PayPlanStorer
	budgetItemStore store.BudgetItemStorer
}

// NewStatisticsService creates a new statistics service
func NewStatisticsService(childStore store.ChildStorer, employeeStore store.EmployeeStorer, orgStore store.OrganizationStorer, fundingStore store.GovernmentFundingStorer, payPlanStore store.PayPlanStorer, budgetItemStore store.BudgetItemStorer) *StatisticsService {
	return &StatisticsService{
		childStore:      childStore,
		employeeStore:   employeeStore,
		orgStore:        orgStore,
		fundingStore:    fundingStore,
		payPlanStore:    payPlanStore,
		budgetItemStore: budgetItemStore,
	}
}

// pedagogicalCategories lists staff categories counted toward staffing requirements
var pedagogicalCategories = []string{
	string(models.StaffCategoryQualified),
	string(models.StaffCategorySupplementary),
}

// snapDateRange returns a date range snapped to 1st-of-month with defaults.
// Defaults cover: 1 month before the previous Kita year through the end of the
// next Kita year. A Kita year runs Aug 1 – Jul 31.
func snapDateRange(from, to *time.Time) (time.Time, time.Time) {
	now := time.Now()
	var rangeStart, rangeEnd time.Time

	// Current Kita year starts on Aug 1 of this or last calendar year
	kitaYearStartYear := now.Year()
	if now.Month() < time.August {
		kitaYearStartYear--
	}

	if from != nil {
		rangeStart = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		// 1 month before the previous Kita year (= July of kitaYearStartYear-1)
		rangeStart = time.Date(kitaYearStartYear-1, time.July, 1, 0, 0, 0, 0, time.UTC)
	}
	if to != nil {
		rangeEnd = time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		// 1 month past the next Kita year (= August of kitaYearStartYear+2)
		rangeEnd = time.Date(kitaYearStartYear+2, time.August, 1, 0, 0, 0, 0, time.UTC)
	}
	return rangeStart, rangeEnd
}

// GetStaffingHours calculates monthly staffing hours data points
func (s *StatisticsService) GetStaffingHours(ctx context.Context, orgID uint, from, to *time.Time, sectionID *uint) (*models.StaffingHoursResponse, error) {
	rangeStart, rangeEnd := snapDateRange(from, to)

	// Fetch organization for state
	org, err := s.orgStore.FindByID(ctx, orgID)
	if err != nil {
		return nil, classifyStoreError(err, "organization")
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
func (s *StatisticsService) GetFinancials(ctx context.Context, orgID uint, from, to *time.Time) (*models.FinancialResponse, error) {
	rangeStart, rangeEnd := snapDateRange(from, to)

	// Fetch organization for state
	org, err := s.orgStore.FindByID(ctx, orgID)
	if err != nil {
		return nil, classifyStoreError(err, "organization")
	}

	// Fetch government funding with all periods and properties
	var fundingPeriods []models.GovernmentFundingPeriod
	funding, err := s.fundingStore.FindByStateWithDetails(ctx, org.State, 0, nil)
	if err == nil {
		fundingPeriods = funding.Periods
	}

	// Fetch children with contracts in range
	children, err := s.childStore.FindByOrganizationInDateRange(ctx, orgID, rangeStart, rangeEnd, nil)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children")
	}

	// Fetch ALL employee contracts in range (salaries apply to all staff, not just pedagogical)
	employeeContracts, err := s.employeeStore.FindContractsByOrganizationInDateRange(ctx, orgID, rangeStart, rangeEnd, nil, nil)
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

	// Fetch budget items with entries for this organization
	budgetItems, err := s.budgetItemStore.FindByOrganizationWithEntries(ctx, orgID)
	if err != nil {
		budgetItems = nil // non-fatal: proceed without budget items
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
		fundingDetailMap := make(map[string]int) // "key:value" → total cents
		fundingPeriod := findPeriodForDate(fundingPeriods, date)
		for i := range children {
			child := &children[i]
			for j := range child.Contracts {
				contract := &child.Contracts[j]
				if contract.IsActiveOn(date) {
					childCount++
					age := validation.CalculateAgeOnDate(child.Birthdate, date)
					if fundingPeriod != nil {
						for _, fp := range fundingPeriod.Properties {
							if !fp.MatchesAge(age) {
								continue
							}
							if contract.Properties.HasValue(fp.Key, fp.Value) {
								fundingIncome += fp.Payment
								mapKey := fp.Key + ":" + fp.Value
								fundingDetailMap[mapKey] += fp.Payment
							}
						}
					}
					break
				}
			}
		}

		// Convert funding detail map to sorted slice
		var fundingDetails []models.FinancialFundingDetail
		for mapKey, amount := range fundingDetailMap {
			parts := strings.SplitN(mapKey, ":", 2)
			fundingDetails = append(fundingDetails, models.FinancialFundingDetail{
				Key:         parts[0],
				Value:       parts[1],
				AmountCents: amount,
			})
		}
		sort.Slice(fundingDetails, func(i, j int) bool {
			if fundingDetails[i].Key != fundingDetails[j].Key {
				return fundingDetails[i].Key < fundingDetails[j].Key
			}
			return fundingDetails[i].Value < fundingDetails[j].Value
		})

		// Expenses: employee salaries
		grossSalary := 0
		employerCosts := 0
		staffCount := 0
		employeeSeen := make(map[uint]bool)
		salaryByCategory := make(map[string][2]int) // [0]=gross, [1]=contrib
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

			cat := ec.StaffCategory
			pair := salaryByCategory[cat]
			pair[0] += gross
			pair[1] += contrib
			salaryByCategory[cat] = pair
		}

		// Convert salary-by-category map to sorted slice
		var salaryDetails []models.FinancialSalaryDetail
		for cat, pair := range salaryByCategory {
			salaryDetails = append(salaryDetails, models.FinancialSalaryDetail{
				StaffCategory: cat,
				GrossSalary:   pair[0],
				EmployerCosts: pair[1],
			})
		}
		sort.Slice(salaryDetails, func(i, j int) bool {
			return salaryDetails[i].StaffCategory < salaryDetails[j].StaffCategory
		})

		// Budget items: income and expenses from budget items
		budgetIncome := 0
		budgetExpenses := 0
		var budgetItemDetails []models.FinancialBudgetItemDetail
		for i := range budgetItems {
			item := &budgetItems[i]
			for j := range item.Entries {
				entry := &item.Entries[j]
				if entry.IsActiveOn(date) {
					amount := entry.AmountCents
					if item.PerChild {
						amount *= childCount
					}
					if item.Category == string(models.BudgetItemCategoryIncome) {
						budgetIncome += amount
					} else {
						budgetExpenses += amount
					}
					budgetItemDetails = append(budgetItemDetails, models.FinancialBudgetItemDetail{
						Name:        item.Name,
						Category:    item.Category,
						AmountCents: amount,
					})
					break // only first active entry per item
				}
			}
		}

		dp.FundingIncome = fundingIncome
		dp.GrossSalary = grossSalary
		dp.EmployerCosts = employerCosts
		dp.BudgetIncome = budgetIncome
		dp.BudgetExpenses = budgetExpenses
		dp.TotalIncome = fundingIncome + budgetIncome
		dp.TotalExpenses = grossSalary + employerCosts + budgetExpenses
		dp.Balance = dp.TotalIncome - dp.TotalExpenses
		dp.ChildCount = childCount
		dp.StaffCount = staffCount
		dp.BudgetItemDetails = budgetItemDetails
		dp.FundingDetails = fundingDetails
		dp.SalaryDetails = salaryDetails
		dataPoints = append(dataPoints, dp)
	}

	return &models.FinancialResponse{
		DataPoints: dataPoints,
	}, nil
}

// GetOccupancy calculates monthly occupancy data points broken down by age group, care type, and supplements.
func (s *StatisticsService) GetOccupancy(ctx context.Context, orgID uint, from, to *time.Time, sectionID *uint) (*models.OccupancyResponse, error) {
	rangeStart, rangeEnd := snapDateRange(from, to)

	// Fetch organization for state
	org, err := s.orgStore.FindByID(ctx, orgID)
	if err != nil {
		return nil, classifyStoreError(err, "organization")
	}

	// Fetch government funding with all periods and properties
	var fundingPeriods []models.GovernmentFundingPeriod
	funding, err := s.fundingStore.FindByStateWithDetails(ctx, org.State, 0, nil)
	if err == nil {
		fundingPeriods = funding.Periods
	}

	// Extract table structure from funding configuration
	ageGroups, careTypes, supplementTypes := extractOccupancyStructure(fundingPeriods)

	// Fetch children with contracts in range
	children, err := s.childStore.FindByOrganizationInDateRange(ctx, orgID, rangeStart, rangeEnd, sectionID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children")
	}

	// Generate data points for each month
	var dataPoints []models.OccupancyDataPoint
	for date := rangeStart; !date.After(rangeEnd); date = date.AddDate(0, 1, 0) {
		dp := models.OccupancyDataPoint{
			Date:             date.Format(models.DateFormat),
			ByAgeAndCareType: make(map[string]map[string]int),
			BySupplement:     make(map[string]int),
		}

		// Initialize the nested maps
		for _, ag := range ageGroups {
			dp.ByAgeAndCareType[ag.Label] = make(map[string]int)
		}

		for i := range children {
			child := &children[i]
			for j := range child.Contracts {
				contract := &child.Contracts[j]
				if !contract.IsActiveOn(date) {
					continue
				}
				dp.Total++

				age := validation.CalculateAgeOnDate(child.Birthdate, date)
				ageLabel := findAgeGroupLabel(age, ageGroups)

				// Count by age group × care type
				careType := contract.Properties.GetScalarProperty("care_type")
				if ageLabel != "" && careType != "" {
					if dp.ByAgeAndCareType[ageLabel] == nil {
						dp.ByAgeAndCareType[ageLabel] = make(map[string]int)
					}
					dp.ByAgeAndCareType[ageLabel][careType]++
				}

				// Count supplements
				for _, st := range supplementTypes {
					if contract.Properties.HasValue(st.Key, st.Value) {
						dp.BySupplement[st.Value]++
					}
				}

				break // Only count each child once per month
			}
		}

		dataPoints = append(dataPoints, dp)
	}

	return &models.OccupancyResponse{
		AgeGroups:       ageGroups,
		CareTypes:       careTypes,
		SupplementTypes: supplementTypes,
		DataPoints:      dataPoints,
	}, nil
}

// extractOccupancyStructure derives age groups, care types, and supplement types
// from the government funding periods' properties.
func extractOccupancyStructure(periods []models.GovernmentFundingPeriod) ([]models.OccupancyAgeGroup, []string, []models.OccupancySupplementType) {
	// Use the most recent period (periods are ordered DESC by from_date)
	if len(periods) == 0 {
		return nil, nil, nil
	}
	period := periods[0]

	type ageKey struct {
		minAge, maxAge int
	}
	ageGroupSet := make(map[ageKey]bool)
	careTypeSet := make(map[string]bool)
	supplementSet := make(map[string]models.OccupancySupplementType)

	for _, prop := range period.Properties {
		if prop.Key == "care_type" {
			careTypeSet[prop.Value] = true
			if prop.MinAge != nil && prop.MaxAge != nil {
				ageGroupSet[ageKey{*prop.MinAge, *prop.MaxAge}] = true
			}
		} else {
			if _, exists := supplementSet[prop.Value]; !exists {
				supplementSet[prop.Value] = models.OccupancySupplementType{
					Key:   prop.Key,
					Value: prop.Value,
					Label: formatSupplementLabel(prop.Value),
				}
			}
		}
	}

	// Build sorted age groups
	var ageGroups []models.OccupancyAgeGroup
	for ak := range ageGroupSet {
		ageGroups = append(ageGroups, models.OccupancyAgeGroup{
			Label:  formatAgeGroupLabel(ak.minAge, ak.maxAge),
			MinAge: ak.minAge,
			MaxAge: ak.maxAge,
		})
	}
	sort.Slice(ageGroups, func(i, j int) bool {
		return ageGroups[i].MinAge < ageGroups[j].MinAge
	})

	// Build sorted care types
	var careTypes []string
	for ct := range careTypeSet {
		careTypes = append(careTypes, ct)
	}
	sort.Strings(careTypes)

	// Build sorted supplement types
	var supplements []models.OccupancySupplementType
	for _, st := range supplementSet {
		supplements = append(supplements, st)
	}
	sort.Slice(supplements, func(i, j int) bool {
		return supplements[i].Value < supplements[j].Value
	})

	return ageGroups, careTypes, supplements
}

// formatAgeGroupLabel formats an age range into a display label.
// Examples: {0,1}→"0/1", {2,2}→"2", {3,8}→"3+"
func formatAgeGroupLabel(minAge, maxAge int) string {
	if minAge == maxAge {
		return fmt.Sprintf("%d", minAge)
	}
	if maxAge >= 6 {
		return fmt.Sprintf("%d+", minAge)
	}
	// For small ranges like 0-1, use slash notation
	parts := make([]string, 0, maxAge-minAge+1)
	for i := minAge; i <= maxAge; i++ {
		parts = append(parts, fmt.Sprintf("%d", i))
	}
	return strings.Join(parts, "/")
}

// formatSupplementLabel formats a supplement value into a display label.
// Capitalizes words and replaces separators.
func formatSupplementLabel(value string) string {
	words := strings.FieldsFunc(value, func(r rune) bool {
		return r == '_' || r == '/' || r == '-'
	})
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// findAgeGroupLabel returns the label of the age group that matches the given age.
func findAgeGroupLabel(age int, ageGroups []models.OccupancyAgeGroup) string {
	for _, ag := range ageGroups {
		if age >= ag.MinAge && age <= ag.MaxAge {
			return ag.Label
		}
	}
	return ""
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
