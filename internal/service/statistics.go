package service

import (
	"context"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
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
}

// NewStatisticsService creates a new statistics service
func NewStatisticsService(childStore store.ChildStorer, employeeStore store.EmployeeStorer, orgStore store.OrganizationStorer, fundingStore store.GovernmentFundingStorer, payPlanStore store.PayPlanStorer, budgetItemStore store.BudgetItemStorer, sectionStore store.SectionStorer) *StatisticsService {
	return &StatisticsService{
		childStore:      childStore,
		employeeStore:   employeeStore,
		orgStore:        orgStore,
		fundingStore:    fundingStore,
		payPlanStore:    payPlanStore,
		budgetItemStore: budgetItemStore,
		sectionStore:    sectionStore,
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

// loadPayPlans batch-fetches pay plans referenced by the given employees' contracts.
func (s *StatisticsService) loadPayPlans(ctx context.Context, employees []models.Employee) map[uint]*models.PayPlan {
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
	payPlanMap, err := s.payPlanStore.FindByIDsWithPeriods(ctx, payPlanIDs)
	if err != nil {
		return make(map[uint]*models.PayPlan) // non-fatal: proceed without pay plans
	}
	return payPlanMap
}

// GetStaffingHours calculates monthly staffing hours data points
func (s *StatisticsService) GetStaffingHours(ctx context.Context, orgID uint, from, to *time.Time, sectionID *uint) (*models.StaffingHoursResponse, error) {
	rangeStart, rangeEnd := snapDateRange(from, to)

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
	rangeStart, rangeEnd := snapDateRange(from, to)

	employees, err := s.employeeStore.FindByOrganizationInDateRange(ctx, orgID, rangeStart, rangeEnd, []string(nil), sectionID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch employees")
	}

	dates, rows := calculateEmployeeStaffingHours(employees, rangeStart, rangeEnd)
	return &models.EmployeeStaffingHoursResponse{Dates: dates, Employees: rows}, nil
}

// GetFinancials calculates monthly financial data points (income, expenses, balance)
func (s *StatisticsService) GetFinancials(ctx context.Context, orgID uint, from, to *time.Time) (*models.FinancialResponse, error) {
	rangeStart, rangeEnd := snapDateRange(from, to)

	fundingPeriods, err := s.loadOrgAndFunding(ctx, orgID)
	if err != nil {
		return nil, err
	}

	children, err := s.childStore.FindByOrganizationInDateRange(ctx, orgID, rangeStart, rangeEnd, nil)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children")
	}

	employees, err := s.employeeStore.FindByOrganizationInDateRange(ctx, orgID, rangeStart, rangeEnd, []string(nil), nil)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch employees")
	}

	payPlans := s.loadPayPlans(ctx, employees)

	budgetItems, err := s.budgetItemStore.FindByOrganizationWithEntries(ctx, orgID)
	if err != nil {
		budgetItems = nil // non-fatal: proceed without budget items
	}

	dataPoints := calculateFinancials(children, employees, payPlans, fundingPeriods, budgetItems, rangeStart, rangeEnd)
	return &models.FinancialResponse{DataPoints: dataPoints}, nil
}

// GetOccupancy calculates monthly occupancy data points broken down by age group, care type, and supplements.
func (s *StatisticsService) GetOccupancy(ctx context.Context, orgID uint, from, to *time.Time, sectionID *uint) (*models.OccupancyResponse, error) {
	rangeStart, rangeEnd := snapDateRange(from, to)

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
