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

// --- Pre-computed index types for O(1) lookups in hot loops ---

// gradeStepKey is used for O(1) lookup of pay plan entries by grade+step.
type gradeStepKey struct {
	Grade string
	Step  int
}

// resolvedPayPlanPeriod holds a pay plan period plus its pre-built entry index.
type resolvedPayPlanPeriod struct {
	period     *models.PayPlanPeriod
	entryIndex map[gradeStepKey]*models.PayPlanEntry
}

// buildFundingPeriodIndex pre-computes which funding period is active for each
// first-of-month in [start, end]. Built once, then O(1) lookup per month.
func buildFundingPeriodIndex(periods []models.GovernmentFundingPeriod, start, end time.Time) map[time.Time]*models.GovernmentFundingPeriod {
	idx := make(map[time.Time]*models.GovernmentFundingPeriod)
	for date := start; !date.After(end); date = date.AddDate(0, 1, 0) {
		idx[date] = findPeriodForDate(periods, date)
	}
	return idx
}

// buildEntryIndex creates an O(1) lookup map from (grade, step) to entry.
func buildEntryIndex(entries []models.PayPlanEntry) map[gradeStepKey]*models.PayPlanEntry {
	idx := make(map[gradeStepKey]*models.PayPlanEntry, len(entries))
	for i := range entries {
		e := &entries[i]
		idx[gradeStepKey{e.Grade, e.Step}] = e
	}
	return idx
}

// buildPayPlanIndex pre-builds per-payplan period+entry indexes for the full
// date range. Returns map[payPlanID]map[date]*resolvedPayPlanPeriod.
func buildPayPlanIndex(payPlanMap map[uint]*models.PayPlan, start, end time.Time) map[uint]map[time.Time]*resolvedPayPlanPeriod {
	idx := make(map[uint]map[time.Time]*resolvedPayPlanPeriod, len(payPlanMap))
	for ppID, pp := range payPlanMap {
		// Pre-build entry indexes for all periods
		entryIndexes := make(map[uint]map[gradeStepKey]*models.PayPlanEntry, len(pp.Periods))
		for i := range pp.Periods {
			entryIndexes[pp.Periods[i].ID] = buildEntryIndex(pp.Periods[i].Entries)
		}
		dateMap := make(map[time.Time]*resolvedPayPlanPeriod)
		for date := start; !date.After(end); date = date.AddDate(0, 1, 0) {
			period := findPayPlanPeriodForDate(pp.Periods, date)
			if period != nil {
				dateMap[date] = &resolvedPayPlanPeriod{
					period:     period,
					entryIndex: entryIndexes[period.ID],
				}
			}
		}
		idx[ppID] = dateMap
	}
	return idx
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

	// Pre-build funding period index: O(1) lookup per month
	fundingPeriodIdx := buildFundingPeriodIndex(fundingPeriods, rangeStart, rangeEnd)

	// Generate data points for each month
	dataPoints := make([]models.StaffingHoursDataPoint, 0, monthCount(rangeStart, rangeEnd))
	for date := rangeStart; !date.After(rangeEnd); date = date.AddDate(0, 1, 0) {
		dp := models.StaffingHoursDataPoint{
			Date: date.Format(models.DateFormat),
		}

		// Calculate required hours from children
		requiredHours := 0.0
		childCount := 0
		period := fundingPeriodIdx[date]
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
		employeeSeen := make(map[uint]bool, len(employeeContracts))
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

	// Batch-fetch unique pay plans in a single query
	payPlanIDs := make([]uint, 0)
	payPlanIDSeen := make(map[uint]bool)
	for i := range employeeContracts {
		ppID := employeeContracts[i].PayPlanID
		if ppID != 0 && !payPlanIDSeen[ppID] {
			payPlanIDSeen[ppID] = true
			payPlanIDs = append(payPlanIDs, ppID)
		}
	}
	payPlanMap, err := s.payPlanStore.FindByIDsWithPeriods(ctx, payPlanIDs)
	if err != nil {
		payPlanMap = make(map[uint]*models.PayPlan) // non-fatal: proceed without pay plans
	}

	// Pre-build funding period index and pay plan indexes: O(1) lookups in hot loop
	fundingPeriodIdx := buildFundingPeriodIndex(fundingPeriods, rangeStart, rangeEnd)
	payPlanIdx := buildPayPlanIndex(payPlanMap, rangeStart, rangeEnd)

	// Fetch budget items with entries for this organization
	budgetItems, err := s.budgetItemStore.FindByOrganizationWithEntries(ctx, orgID)
	if err != nil {
		budgetItems = nil // non-fatal: proceed without budget items
	}

	// Generate data points for each month
	type fundingDetailAccum struct {
		amount int
		label  string
	}
	dataPoints := make([]models.FinancialDataPoint, 0, monthCount(rangeStart, rangeEnd))
	for date := rangeStart; !date.After(rangeEnd); date = date.AddDate(0, 1, 0) {
		dp := models.FinancialDataPoint{
			Date: date.Format(models.DateFormat),
		}

		// Income: government funding for active children
		fundingIncome := 0
		childCount := 0
		fundingDetailMap := make(map[string]fundingDetailAccum) // "key:value" → {amount, label}
		fundingPeriod := fundingPeriodIdx[date]
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
								existing := fundingDetailMap[mapKey]
								fundingDetailMap[mapKey] = fundingDetailAccum{
									amount: existing.amount + fp.Payment,
									label:  fp.Label,
								}
							}
						}
					}
					break
				}
			}
		}

		// Convert funding detail map to sorted slice
		var fundingDetails []models.FinancialFundingDetail
		for mapKey, accum := range fundingDetailMap {
			parts := strings.SplitN(mapKey, ":", 2)
			fundingDetails = append(fundingDetails, models.FinancialFundingDetail{
				Key:         parts[0],
				Value:       parts[1],
				Label:       accum.label,
				AmountCents: accum.amount,
			})
		}
		sort.Slice(fundingDetails, func(i, j int) bool {
			if fundingDetails[i].Key != fundingDetails[j].Key {
				return fundingDetails[i].Key < fundingDetails[j].Key
			}
			return fundingDetails[i].Value < fundingDetails[j].Value
		})

		// Expenses: employee salaries using pre-built pay plan indexes
		grossSalary := 0
		employerCosts := 0
		staffCount := 0
		employeeSeen := make(map[uint]bool, len(employeeContracts))
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

			ppDateMap := payPlanIdx[ec.PayPlanID]
			if ppDateMap == nil {
				continue
			}
			resolved := ppDateMap[date]
			if resolved == nil {
				continue
			}
			entry := resolved.entryIndex[gradeStepKey{ec.Grade, ec.Step}]
			if entry == nil {
				continue
			}

			gross := int(math.Round(float64(entry.MonthlyAmount) * ec.WeeklyHours / resolved.period.WeeklyHours))
			contrib := int(math.Round(float64(gross) * float64(resolved.period.EmployerContributionRate) / 10000.0))
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
	dataPoints := make([]models.OccupancyDataPoint, 0, monthCount(rangeStart, rangeEnd))
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
func extractOccupancyStructure(periods []models.GovernmentFundingPeriod) ([]models.OccupancyAgeGroup, []models.OccupancyCareType, []models.OccupancySupplementType) {
	// Use the most recent period (periods are ordered DESC by from_date)
	if len(periods) == 0 {
		return nil, nil, nil
	}
	period := periods[0]

	type ageKey struct {
		minAge, maxAge int
	}
	ageGroupSet := make(map[ageKey]bool)
	careTypeSet := make(map[string]models.OccupancyCareType)
	supplementSet := make(map[string]models.OccupancySupplementType)

	for _, prop := range period.Properties {
		if prop.Key == "care_type" {
			if _, exists := careTypeSet[prop.Value]; !exists {
				careTypeSet[prop.Value] = models.OccupancyCareType{
					Value: prop.Value,
					Label: prop.Label,
				}
			}
			if prop.MinAge != nil && prop.MaxAge != nil {
				ageGroupSet[ageKey{*prop.MinAge, *prop.MaxAge}] = true
			}
		} else {
			if _, exists := supplementSet[prop.Value]; !exists {
				supplementSet[prop.Value] = models.OccupancySupplementType{
					Key:   prop.Key,
					Value: prop.Value,
					Label: prop.Label,
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
	var careTypes []models.OccupancyCareType
	for _, ct := range careTypeSet {
		careTypes = append(careTypes, ct)
	}
	sort.Slice(careTypes, func(i, j int) bool {
		return careTypes[i].Value < careTypes[j].Value
	})

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

// findAgeGroupLabel returns the label of the age group that matches the given age.
func findAgeGroupLabel(age int, ageGroups []models.OccupancyAgeGroup) string {
	for _, ag := range ageGroups {
		if age >= ag.MinAge && age <= ag.MaxAge {
			return ag.Label
		}
	}
	return ""
}

// monthCount returns the number of months in [start, end] (inclusive on both ends).
func monthCount(start, end time.Time) int {
	months := (end.Year()-start.Year())*12 + int(end.Month()-start.Month()) + 1
	if months < 0 {
		return 0
	}
	return months
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
