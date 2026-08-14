package service

import (
	"context"
	"log/slog"
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
	sectionStore    store.SectionStorer
	billStore       store.GovernmentFundingBillPeriodStorer
}

// NewStatisticsService creates a new statistics service
func NewStatisticsService(childStore store.ChildStorer, employeeStore store.EmployeeStorer, orgStore store.OrganizationStorer, fundingStore store.GovernmentFundingStorer, payPlanStore store.PayPlanStorer, budgetItemStore store.BudgetItemStorer, sectionStore store.SectionStorer, billStore store.GovernmentFundingBillPeriodStorer) *StatisticsService {
	return &StatisticsService{
		childStore:      childStore,
		employeeStore:   employeeStore,
		orgStore:        orgStore,
		fundingStore:    fundingStore,
		payPlanStore:    payPlanStore,
		budgetItemStore: budgetItemStore,
		sectionStore:    sectionStore,
		billStore:       billStore,
	}
}

// pedagogicalCategories lists staff categories counted toward staffing requirements
var pedagogicalCategories = []string{
	string(models.StaffCategoryQualified),
	string(models.StaffCategorySupplementary),
}

// MaxStatisticsRangeMonths caps the span (in months) any statistics or
// forecast calculation will iterate over. The four calculators walk the
// range one month at a time and pre-build per-month indexes; an
// unbounded range (e.g. from=1900, to=2200) turns one request into
// 14,400+ tight loops over the entire dataset and is a trivial DoS
// vector if not capped here.
//
// Mirrors handlers.MaxDateRangeMonths so the bound is identical whether
// the user came in through a query-string endpoint (parseOptionalDatePair
// catches them at the handler) or the JSON-body forecast endpoint
// (snapAndValidateRange catches them at the service). Bumping this
// constant is a deliberate "we accept slower forecasts" decision —
// don't bump just to get a single request through.
const MaxStatisticsRangeMonths = 72

// snapAndValidateRange snaps the user-supplied date range to first-of-
// month with sensible Kita-year defaults, then enforces two invariants
// the calculators silently assume:
//
//   - end >= start: snapDateRange does not reorder, so an inverted
//     range (from=2026-06, to=2025-01) used to silently return zero
//     data points instead of an error.
//   - span <= MaxStatisticsRangeMonths: see the constant comment.
//
// Centralizing both checks here means every statistics endpoint —
// including the JSON-body forecast endpoint that bypasses the handler-
// level parseOptionalDatePair gate — gets the same bound. New endpoints
// MUST call this rather than snapDateRange directly.
func snapAndValidateRange(from, to *time.Time) (time.Time, time.Time, error) {
	start, end := snapDateRange(from, to)
	if end.Before(start) {
		return start, end, apperror.BadRequest("'to' date must not be before 'from' date")
	}
	if monthCount(start, end) > MaxStatisticsRangeMonths {
		return start, end, apperror.BadRequest(
			"date range must not exceed %d months", MaxStatisticsRangeMonths)
	}
	return start, end, nil
}

// snapDateRange returns a date range snapped to 1st-of-month with defaults.
// Defaults cover: 1 month before the previous Kita year through the end of the
// next Kita year. A Kita year runs Aug 1 – Jul 31.
//
// Uses models.Today() so the current Kita year is derived from the user's
// wall-clock calendar day in the application timezone — a request made at
// 23:30 UTC in late July is "still July" for a Berlin user even though
// the server's UTC clock has already rolled into August.
//
// Both `from` and `to` are INCLUSIVE of the month they fall in. The
// calculator loop is `for date := start; !date.After(end); date = date.AddDate(0, 1, 0)`,
// so when `end` is also a 1st-of-month date the loop emits a data point
// for that month before exiting. Concrete: with end=2026-07-01 the loop
// emits Jul 2026 and stops; July is NOT skipped. The frontend forecast
// page relies on this: it sends `to = ${year+1}-07-01` to mean "include
// all of July of the following year" and would silently lose one month
// per request if the convention ever flipped to exclusive. Tests in
// forecast_test.go (TestSnapDateRange_*Inclusive*) lock this in.
//
// Callers should prefer snapAndValidateRange so unbounded user input is
// rejected at the same place every time. snapDateRange remains exported-
// internal for unit tests that want to assert just the snap behavior.
func snapDateRange(from, to *time.Time) (time.Time, time.Time) {
	now := models.Today()
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

// loadFundingPeriods fetches government funding periods for the org's state.
// Returns nil (not error) if no funding is configured.
func (s *StatisticsService) loadFundingPeriods(ctx context.Context, state string) []models.GovernmentFundingPeriod {
	funding, err := s.fundingStore.FindByStateWithDetails(ctx, state, 0, nil)
	if err != nil {
		return nil
	}
	return funding.Periods
}

// loadOrgAndFunding fetches the organization and its government funding periods.
func (s *StatisticsService) loadOrgAndFunding(ctx context.Context, orgID uint) ([]models.GovernmentFundingPeriod, error) {
	org, err := s.orgStore.FindByID(ctx, orgID)
	if err != nil {
		return nil, classifyStoreError(err, "organization")
	}
	return s.loadFundingPeriods(ctx, org.State), nil
}

// loadPayPlans batch-fetches pay plans referenced by the given employees'
// contracts. A store error here means the calculation would silently
// undercount labor cost (calculate.go skips employees whose pay plan is
// missing) — propagate it instead of swallowing.
//
// Per-row data-quality issues (a contract referencing a payplan_id that
// no longer exists in the DB, but the store call succeeded with a partial
// map) are NOT errors — they're surfaced as CalculationWarnings from
// calculateFinancials so the caller can show them to the user.
func (s *StatisticsService) loadPayPlans(ctx context.Context, employees []models.Employee) (map[uint]*models.PayPlan, error) {
	payPlanIDs := make([]uint, 0)
	seen := make(map[uint]bool)
	for i := range employees {
		for j := range employees[i].Contracts {
			ppID := employees[i].Contracts[j].PayPlanID
			if ppID != 0 && !seen[ppID] {
				seen[ppID] = true
				payPlanIDs = append(payPlanIDs, ppID)
			}
		}
	}
	if len(payPlanIDs) == 0 {
		return make(map[uint]*models.PayPlan), nil
	}
	payPlanMap, err := s.payPlanStore.FindByIDsWithPeriods(ctx, payPlanIDs)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to load pay plans")
	}
	return payPlanMap, nil
}

// GetStaffingHours calculates monthly staffing hours data points
func (s *StatisticsService) GetStaffingHours(ctx context.Context, orgID uint, from, to *time.Time, sectionID *uint) (*models.StaffingHoursResponse, error) {
	rangeStart, rangeEnd, err := snapAndValidateRange(from, to)
	if err != nil {
		return nil, err
	}

	fundingPeriods, err := s.loadOrgAndFunding(ctx, orgID)
	if err != nil {
		return nil, err
	}

	children, err := s.childStore.FindByOrganizationInDateRange(ctx, orgID, rangeStart, rangeEnd, sectionID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children")
	}

	employees, err := s.employeeStore.FindByOrganizationInDateRange(ctx, orgID, rangeStart, rangeEnd, pedagogicalCategories, sectionID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch employees")
	}

	dataPoints := calculateStaffingHours(children, employees, fundingPeriods, rangeStart, rangeEnd)
	return &models.StaffingHoursResponse{DataPoints: dataPoints}, nil
}

// GetEmployeeStaffingHours returns per-employee monthly contracted hours
func (s *StatisticsService) GetEmployeeStaffingHours(ctx context.Context, orgID uint, from, to *time.Time, sectionID *uint) (*models.EmployeeStaffingHoursResponse, error) {
	rangeStart, rangeEnd, err := snapAndValidateRange(from, to)
	if err != nil {
		return nil, err
	}

	employees, err := s.employeeStore.FindByOrganizationInDateRange(ctx, orgID, rangeStart, rangeEnd, []string(nil), sectionID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch employees")
	}

	dates, rows := calculateEmployeeStaffingHours(employees, rangeStart, rangeEnd)
	return &models.EmployeeStaffingHoursResponse{Dates: dates, Employees: rows}, nil
}

// GetFinancials calculates monthly financial data points (income, expenses, balance)
func (s *StatisticsService) GetFinancials(ctx context.Context, orgID uint, from, to *time.Time, sectionID *uint) (*models.FinancialResponse, error) {
	rangeStart, rangeEnd, err := snapAndValidateRange(from, to)
	if err != nil {
		return nil, err
	}

	fundingPeriods, err := s.loadOrgAndFunding(ctx, orgID)
	if err != nil {
		return nil, err
	}

	children, err := s.childStore.FindByOrganizationInDateRange(ctx, orgID, rangeStart, rangeEnd, sectionID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children")
	}

	employees, err := s.employeeStore.FindByOrganizationInDateRange(ctx, orgID, rangeStart, rangeEnd, []string(nil), sectionID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch employees")
	}

	payPlans, err := s.loadPayPlans(ctx, employees)
	if err != nil {
		return nil, err
	}

	budgetItems, err := s.budgetItemStore.FindByOrganizationWithEntries(ctx, orgID)
	loadWarnings := make([]models.CalculationWarning, 0)
	if err != nil {
		// Non-fatal: proceed without budget items, but tell the caller
		// the breakdown is incomplete so the UI can render a banner
		// instead of silently showing an empty operating-costs slice.
		slog.Warn("failed to load budget items; expense breakdown will exclude operating costs",
			"org_id", orgID, "error", err)
		loadWarnings = append(loadWarnings, models.CalculationWarning{
			Code:    "budget_items_load_failed",
			Message: "could not load budget items; expense breakdown excludes operating costs",
		})
		budgetItems = nil
	}

	dataPoints, warnings := calculateFinancials(children, employees, payPlans, fundingPeriods, budgetItems, rangeStart, rangeEnd)
	warnings = append(loadWarnings, warnings...)

	// Merge actual funding from government funding bills
	if s.billStore != nil {
		billTotals, errTotals := s.billStore.FindFacilityTotalsByOrganizationInDateRange(ctx, orgID, rangeStart, rangeEnd)
		billByRowType, errRowType := s.billStore.FindBillTotalsByRowTypeInDateRange(ctx, orgID, rangeStart, rangeEnd)

		if errTotals != nil || errRowType != nil {
			// Non-fatal: proceed without actual funding overlays. A single
			// warning covers either failure since both fuel the same set of
			// frontend fields (ActualFunding / ActualFundingRegular /
			// ActualFundingCorrection) — the UI just needs to know the
			// numbers may be incomplete.
			slog.Warn("failed to load government funding bills; actual funding overlay will be incomplete",
				"org_id", orgID, "totals_error", errTotals, "row_type_error", errRowType)
			warnings = append(warnings, models.CalculationWarning{
				Code:    "funding_bills_load_failed",
				Message: "could not load actual government funding bills; reported actuals may be incomplete",
			})
		}

		// Pre-parse dates once for all data points
		dateKeys := make(map[string]time.Time, len(dataPoints))
		for i := range dataPoints {
			if t, parseErr := time.Parse("2006-01-02", dataPoints[i].Date); parseErr == nil {
				dateKeys[dataPoints[i].Date] = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			}
		}

		for i := range dataPoints {
			dp := &dataPoints[i]
			key, ok := dateKeys[dp.Date]
			if !ok {
				continue
			}
			if errTotals == nil {
				if total, found := billTotals[key]; found {
					total := total
					dp.ActualFunding = &total
				}
			}
			if errRowType == nil {
				if entry, found := billByRowType[key]; found {
					regular := entry.RegularTotal
					correction := entry.CorrectionTotal
					dp.ActualFundingRegular = &regular
					dp.ActualFundingCorrection = &correction
				}
			}
		}
	}

	return &models.FinancialResponse{DataPoints: dataPoints, Warnings: warnings}, nil
}

// GetOccupancy calculates monthly occupancy data points broken down by age group, care type, and supplements.
func (s *StatisticsService) GetOccupancy(ctx context.Context, orgID uint, from, to *time.Time, sectionID *uint) (*models.OccupancyResponse, error) {
	rangeStart, rangeEnd, err := snapAndValidateRange(from, to)
	if err != nil {
		return nil, err
	}

	fundingPeriods, err := s.loadOrgAndFunding(ctx, orgID)
	if err != nil {
		return nil, err
	}

	children, err := s.childStore.FindByOrganizationInDateRange(ctx, orgID, rangeStart, rangeEnd, sectionID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children")
	}

	return calculateOccupancy(children, fundingPeriods, rangeStart, rangeEnd), nil
}

// CalculateFunding calculates government funding for all children with active contracts on the given date
func (s *StatisticsService) CalculateFunding(ctx context.Context, orgID uint, date time.Time) (*models.ChildrenFundingResponse, error) {
	fundingPeriods, err := s.loadOrgAndFunding(ctx, orgID)
	if err != nil {
		return nil, err
	}

	children, err := s.childStore.FindByOrganizationWithActiveOn(ctx, orgID, date)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children")
	}

	return calculateFunding(children, fundingPeriods, date), nil
}

// GetAgeDistribution returns age distribution of children with active contracts on the given date
func (s *StatisticsService) GetAgeDistribution(ctx context.Context, orgID uint, date time.Time) (*models.AgeDistributionResponse, error) {
	children, err := s.childStore.FindByOrganizationWithActiveOn(ctx, orgID, date)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children")
	}
	return calculateAgeDistribution(children, date), nil
}

// GetContractPropertiesDistribution returns the distribution of contract properties
// for children with active contracts on the given date
func (s *StatisticsService) GetContractPropertiesDistribution(ctx context.Context, orgID uint, date time.Time) (*models.ContractPropertiesDistributionResponse, error) {
	fundingPeriods, err := s.loadOrgAndFunding(ctx, orgID)
	if err != nil {
		return nil, err
	}

	children, err := s.childStore.FindByOrganizationWithActiveOn(ctx, orgID, date)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children")
	}

	return calculateContractPropertiesDistribution(children, fundingPeriods, date), nil
}

// EstimateChildFunding calculates government funding for a hypothetical child.
func (s *StatisticsService) EstimateChildFunding(ctx context.Context, orgID uint, req *models.ChildFundingEstimateRequest) (*models.ChildFundingResponse, error) {
	date := models.Today()
	if req.Date != nil {
		date = *req.Date
	}
	date = time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)

	fundingPeriods, err := s.loadOrgAndFunding(ctx, orgID)
	if err != nil {
		return nil, err
	}

	period := findPeriodForDate(fundingPeriods, date)
	fundingAge := validation.FundingAgeOnDate(req.Birthdate, date)

	result := calculateChildFunding(fundingAge, req.Properties, period)
	result.Age = validation.CalculateAgeOnDate(req.Birthdate, date)

	return &result, nil
}

// EstimateEmployeeCost calculates monthly cost for a hypothetical employee.
func (s *StatisticsService) EstimateEmployeeCost(ctx context.Context, orgID uint, req *models.EmployeeCostEstimateRequest) (*models.EmployeeCostEstimateResponse, error) {
	date := models.Today()
	if req.Date != nil {
		date = *req.Date
	}
	date = time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)

	payPlan, err := s.payPlanStore.FindByIDWithPeriods(ctx, req.PayPlanID, &date)
	if err != nil {
		return nil, apperror.NotFound("pay plan")
	}
	if payPlan.OrganizationID != orgID {
		return nil, apperror.NotFound("pay plan")
	}

	period := findPayPlanPeriodForDate(payPlan.Periods, date)
	if period == nil {
		return nil, apperror.NotFound("no active pay plan period for the given date")
	}

	entryIdx := buildEntryIndex(period.Entries)
	entry := entryIdx[gradeStepKey{req.Grade, req.Step}]
	if entry == nil {
		return nil, apperror.NotFound("no pay plan entry for grade %s step %d", req.Grade, req.Step)
	}

	gross, contrib := employeeMonthlyCost(entry.MonthlyAmount, req.WeeklyHours, period.WeeklyHours, period.EmployerContributionRate)

	return &models.EmployeeCostEstimateResponse{
		Date:                     date.Format(models.DateFormat),
		StaffCategory:            req.StaffCategory,
		Grade:                    req.Grade,
		Step:                     req.Step,
		WeeklyHours:              req.WeeklyHours,
		PayPlanWeeklyHours:       period.WeeklyHours,
		FullTimeMonthlyAmount:    entry.MonthlyAmount,
		GrossSalary:              gross,
		EmployerContributionRate: period.EmployerContributionRate,
		EmployerCosts:            contrib,
		TotalMonthlyCost:         gross + contrib,
	}, nil
}
