package service

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

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

// calculateFinancials computes monthly income, expenses, and balance
// for the given date range.
//
// Income sources:
//   - Government funding: matched per child by age + contract properties
//   - Budget items with category "income" (fixed or per-child)
//
// Expense sources:
//   - Gross salary: pay plan entry amount pro-rated by weekly hours
//   - Employer contributions: gross × employer contribution rate
//   - Budget items with category "expense" (fixed or per-child)
//
// Each month uses the first-of-month as reference date.
// A child/employee is counted if their contract IsActiveOn that date.
func calculateFinancials(
	children []models.Child,
	employees []models.Employee,
	payPlans map[uint]*models.PayPlan,
	fundingPeriods []models.GovernmentFundingPeriod,
	budgetItems []models.BudgetItem,
	start, end time.Time,
) []models.FinancialDataPoint {
	// Pre-build indexes: O(1) lookups in hot loop
	fundingPeriodIdx := buildFundingPeriodIndex(fundingPeriods, start, end)
	payPlanIdx := buildPayPlanIndex(payPlans, start, end)

	type fundingDetailAccum struct {
		amount int
		label  string
	}
	dataPoints := make([]models.FinancialDataPoint, 0, monthCount(start, end))
	for date := start; !date.After(end); date = date.AddDate(0, 1, 0) {
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
					for _, fp := range matchFundingProperties(age, contract.Properties, fundingPeriod) {
						fundingIncome += fp.Payment
						mapKey := fp.Key + ":" + fp.Value
						existing := fundingDetailMap[mapKey]
						fundingDetailMap[mapKey] = fundingDetailAccum{
							amount: existing.amount + fp.Payment,
							label:  fp.Label,
						}
					}
					break
				}
			}
		}

		// Convert funding detail map to sorted slice
		fundingDetails := make([]models.FinancialFundingDetail, 0, len(fundingDetailMap))
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
		salaryByCategory := make(map[string][2]int) // [0]=gross, [1]=contrib
		for i := range employees {
			emp := &employees[i]
			for j := range emp.Contracts {
				ec := &emp.Contracts[j]
				if !ec.IsActiveOn(date) {
					continue
				}
				staffCount++

				ppDateMap := payPlanIdx[ec.PayPlanID]
				if ppDateMap == nil {
					break
				}
				resolved := ppDateMap[date]
				if resolved == nil {
					break
				}
				entry := resolved.entryIndex[gradeStepKey{ec.Grade, ec.Step}]
				if entry == nil {
					break
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
				break // one active contract per employee per month
			}
		}

		// Convert salary-by-category map to sorted slice
		salaryDetails := make([]models.FinancialSalaryDetail, 0, len(salaryByCategory))
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

	return dataPoints
}

// calculateStaffingHours computes monthly required vs available staffing
// hours for the given date range.
//
// Required hours: for each active child, the requirement from matched
// government funding properties is summed, then multiplied by the period's
// full-time weekly hours.
//
// Available hours: sum of weekly hours from active employee contracts.
// Only contracts passed in are counted — the caller is responsible for
// filtering to the desired staff categories.
//
// Each month uses the first-of-month as reference date.
func calculateStaffingHours(
	children []models.Child,
	employees []models.Employee,
	fundingPeriods []models.GovernmentFundingPeriod,
	start, end time.Time,
) []models.StaffingHoursDataPoint {
	fundingPeriodIdx := buildFundingPeriodIndex(fundingPeriods, start, end)

	dataPoints := make([]models.StaffingHoursDataPoint, 0, monthCount(start, end))
	for date := start; !date.After(end); date = date.AddDate(0, 1, 0) {
		dp := models.StaffingHoursDataPoint{
			Date: date.Format(models.DateFormat),
		}

		// Calculate required hours from children
		requiredHours := 0.0
		childCount := 0
		period := fundingPeriodIdx[date]
		for i := range children {
			child := &children[i]
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
		for i := range employees {
			emp := &employees[i]
			hasActive := false
			for j := range emp.Contracts {
				if emp.Contracts[j].IsActiveOn(date) {
					availableHours += emp.Contracts[j].WeeklyHours
					hasActive = true
					break // one active contract per employee per month
				}
			}
			if hasActive {
				staffCount++
			}
		}

		dp.RequiredHours = requiredHours
		dp.AvailableHours = availableHours
		dp.ChildCount = childCount
		dp.StaffCount = staffCount
		dataPoints = append(dataPoints, dp)
	}

	return dataPoints
}

// calculateEmployeeStaffingHours returns a per-employee monthly grid of
// contracted weekly hours. Each row is one employee, each column is one
// month. Employees are sorted by last name, then first name.
//
// The staff category for each row is taken from the employee's most recent
// contract (by start date).
func calculateEmployeeStaffingHours(
	employees []models.Employee,
	start, end time.Time,
) (dates []string, rows []models.EmployeeStaffingHoursRow) {
	numMonths := monthCount(start, end)
	dates = make([]string, 0, numMonths)
	for date := start; !date.After(end); date = date.AddDate(0, 1, 0) {
		dates = append(dates, date.Format(models.DateFormat))
	}

	rows = make([]models.EmployeeStaffingHoursRow, 0, len(employees))
	for i := range employees {
		emp := &employees[i]

		// Determine staff category from the most recent contract
		staffCategory := ""
		if len(emp.Contracts) > 0 {
			latest := emp.Contracts[0]
			for _, c := range emp.Contracts[1:] {
				if c.From.After(latest.From) {
					latest = c
				}
			}
			staffCategory = latest.StaffCategory
		}

		monthlyHours := make([]float64, numMonths)
		monthIdx := 0
		for date := start; !date.After(end); date = date.AddDate(0, 1, 0) {
			for j := range emp.Contracts {
				contract := &emp.Contracts[j]
				if contract.IsActiveOn(date) {
					monthlyHours[monthIdx] = contract.WeeklyHours
					break
				}
			}
			monthIdx++
		}

		rows = append(rows, models.EmployeeStaffingHoursRow{
			EmployeeID:    emp.ID,
			FirstName:     emp.FirstName,
			LastName:      emp.LastName,
			StaffCategory: staffCategory,
			MonthlyHours:  monthlyHours,
		})
	}

	// Sort by last name, first name
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].LastName != rows[j].LastName {
			return rows[i].LastName < rows[j].LastName
		}
		return rows[i].FirstName < rows[j].FirstName
	})

	return dates, rows
}

// calculateOccupancy computes monthly child counts broken down by
// age group × care type, plus supplement counts.
//
// Age groups, care types, and supplement types are derived from the
// government funding configuration (most recent period). If no funding
// is configured, returns empty structure with zero counts.
//
// Each month uses the first-of-month as reference date.
func calculateOccupancy(
	children []models.Child,
	fundingPeriods []models.GovernmentFundingPeriod,
	start, end time.Time,
) *models.OccupancyResponse {
	ageGroups, careTypes, supplementTypes := extractOccupancyStructure(fundingPeriods)

	dataPoints := make([]models.OccupancyDataPoint, 0, monthCount(start, end))
	for date := start; !date.After(end); date = date.AddDate(0, 1, 0) {
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
	}
}

// calculateAgeDistribution counts children by age bucket (0, 1, 2, 3, 4, 5, 6+)
// with gender breakdown, using contracts active on the given date.
//
// Only children with at least one contract active on the date are counted.
// Each child is counted at most once (first active contract wins).
func calculateAgeDistribution(
	children []models.Child,
	date time.Time,
) *models.AgeDistributionResponse {
	buckets := []models.AgeDistributionBucket{
		{AgeLabel: "0", MinAge: 0, MaxAge: intPtr(0), Count: 0},
		{AgeLabel: "1", MinAge: 1, MaxAge: intPtr(1), Count: 0},
		{AgeLabel: "2", MinAge: 2, MaxAge: intPtr(2), Count: 0},
		{AgeLabel: "3", MinAge: 3, MaxAge: intPtr(3), Count: 0},
		{AgeLabel: "4", MinAge: 4, MaxAge: intPtr(4), Count: 0},
		{AgeLabel: "5", MinAge: 5, MaxAge: intPtr(5), Count: 0},
		{AgeLabel: "6+", MinAge: 6, MaxAge: nil, Count: 0},
	}

	totalCount := 0
	for _, child := range children {
		// Only count children with active contracts
		hasActive := false
		for j := range child.Contracts {
			if child.Contracts[j].IsActiveOn(date) {
				hasActive = true
				break
			}
		}
		if !hasActive {
			continue
		}

		age := validation.CalculateAgeOnDate(child.Birthdate, date)
		totalCount++

		for i := range buckets {
			bucket := &buckets[i]
			matches := false
			if bucket.MaxAge == nil {
				matches = age >= bucket.MinAge
			} else {
				matches = age >= bucket.MinAge && age <= *bucket.MaxAge
			}

			if matches {
				bucket.Count++
				switch child.Gender {
				case string(models.GenderMale):
					bucket.MaleCount++
				case string(models.GenderFemale):
					bucket.FemaleCount++
				case string(models.GenderDiverse):
					bucket.DiverseCount++
				}
				break
			}
		}
	}

	return &models.AgeDistributionResponse{
		Date:         date.Format(models.DateFormat),
		TotalCount:   totalCount,
		Distribution: buckets,
	}
}

// calculateContractPropertiesDistribution counts how many children have
// each contract property key/value pair on the given date.
//
// Labels are resolved from the funding periods' properties.
// Only children with at least one active contract are counted.
// Results are sorted by key, then value.
func calculateContractPropertiesDistribution(
	children []models.Child,
	fundingPeriods []models.GovernmentFundingPeriod,
	date time.Time,
) *models.ContractPropertiesDistributionResponse {
	// Build label map from funding periods
	labelMap := buildFundingLabelMap(fundingPeriods)

	distribution := make(map[string]map[string]int)
	totalChildren := 0

	for _, child := range children {
		counted := false
		for _, contract := range child.Contracts {
			if !contract.IsActiveOn(date) {
				continue
			}
			if !counted {
				totalChildren++
				counted = true
			}
			if contract.Properties == nil {
				continue
			}
			for key := range contract.Properties {
				values := contract.Properties.GetAllValues(key)
				for _, value := range values {
					if distribution[key] == nil {
						distribution[key] = make(map[string]int)
					}
					distribution[key][value]++
				}
			}
		}
	}

	// Flatten to sorted slice
	properties := make([]models.ContractPropertyCount, 0, len(distribution))
	keys := make([]string, 0, len(distribution))
	for key := range distribution {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		values := make([]string, 0, len(distribution[key]))
		for value := range distribution[key] {
			values = append(values, value)
		}
		sort.Strings(values)
		for _, value := range values {
			properties = append(properties, models.ContractPropertyCount{
				Key:   key,
				Value: value,
				Label: labelMap[key+":"+value],
				Count: distribution[key][value],
			})
		}
	}

	return &models.ContractPropertiesDistributionResponse{
		Date:          date.Format(models.DateFormat),
		TotalChildren: totalChildren,
		Properties:    properties,
	}
}

// buildFundingLabelMap builds a "key:value" → label lookup from government
// funding period properties. Used by calculateContractPropertiesDistribution.
func buildFundingLabelMap(periods []models.GovernmentFundingPeriod) map[string]string {
	labelMap := make(map[string]string)
	for _, period := range periods {
		for _, prop := range period.Properties {
			if prop.Label != "" {
				key := prop.Key + ":" + prop.Value
				if _, exists := labelMap[key]; !exists {
					labelMap[key] = prop.Label
				}
			}
		}
	}
	return labelMap
}

// calculateFunding computes per-child government funding for all children
// with contracts active on the given date. For each child, matches
// contract properties against funding properties by key/value and age,
// returning matched and unmatched properties with payment amounts.
//
// If no funding period covers the given date, all children get 0 funding
// and all their contract properties are listed as unmatched.
func calculateFunding(
	children []models.Child,
	fundingPeriods []models.GovernmentFundingPeriod,
	date time.Time,
) *models.ChildrenFundingResponse {
	period := findPeriodForDate(fundingPeriods, date)

	response := &models.ChildrenFundingResponse{
		Date:     date,
		Children: make([]models.ChildFundingResponse, 0, len(children)),
	}

	if period != nil {
		response.WeeklyHoursBasis = period.FullTimeWeeklyHours
	}

	for _, child := range children {
		// Find active contract
		var activeContract *models.ChildContract
		for i := range child.Contracts {
			if child.Contracts[i].IsActiveOn(date) {
				activeContract = &child.Contracts[i]
				break
			}
		}
		if activeContract == nil {
			continue
		}

		childAge := validation.CalculateAgeOnDate(child.Birthdate, date)
		childFunding := calculateChildFunding(childAge, activeContract.Properties, period)
		childFunding.ChildID = child.ID
		childFunding.ChildName = child.FirstName + " " + child.LastName
		childFunding.Age = childAge

		response.Children = append(response.Children, childFunding)
	}

	return response
}

// calculateChildFunding calculates funding for a single child based on
// their age and contract properties. It matches contract properties against
// government funding properties using Key/Value matching.
//
// Returns matched properties (with accumulated payment and requirement)
// and unmatched contract properties.
func calculateChildFunding(age int, properties models.ContractProperties, period *models.GovernmentFundingPeriod) models.ChildFundingResponse {
	result := models.ChildFundingResponse{
		MatchedProperties:   []models.ChildFundingMatchedProp{},
		UnmatchedProperties: []models.ChildFundingMatchedProp{},
	}

	contractKeyValues := getAllContractKeyValues(properties)

	if period == nil {
		result.UnmatchedProperties = contractKeyValues
		return result
	}

	matches := matchFundingProperties(age, properties, period)
	matchedSet := make(map[string]bool, len(matches))
	for _, fp := range matches {
		result.Funding += fp.Payment
		result.Requirement += fp.Requirement
		kvKey := fp.Key + ":" + fp.Value
		if !matchedSet[kvKey] {
			matchedSet[kvKey] = true
			result.MatchedProperties = append(result.MatchedProperties, models.ChildFundingMatchedProp{
				Key:   fp.Key,
				Value: fp.Value,
			})
		}
	}

	for _, kv := range contractKeyValues {
		kvKey := kv.Key + ":" + kv.Value
		if !matchedSet[kvKey] {
			result.UnmatchedProperties = append(result.UnmatchedProperties, kv)
		}
	}

	return result
}

// intPtr returns a pointer to an int.
func intPtr(i int) *int {
	return &i
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
