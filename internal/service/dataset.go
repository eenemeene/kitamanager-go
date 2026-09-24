package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

// DataSet holds all data needed for statistics calculations.
// Loaded once per request, shared across multiple calculation functions.
type DataSet struct {
	Children       []models.Child
	Employees      []models.Employee
	FundingPeriods []models.GovernmentFundingPeriod
	PayPlans       map[uint]*models.PayPlan
	BudgetItems    []models.BudgetItem
	// LoadWarnings are non-fatal data-loading problems the caller must surface,
	// so a figure computed from incomplete inputs is not reported as if it were
	// complete.
	LoadWarnings []models.CalculationWarning
}

// PedagogicalEmployees returns employees filtered to only pedagogical contracts.
// Employees with no pedagogical contracts are excluded entirely.
// This clones each employee to avoid mutating the original DataSet.
func (ds *DataSet) PedagogicalEmployees() []models.Employee {
	var result []models.Employee
	for i := range ds.Employees {
		emp := ds.Employees[i]
		var pedContracts []models.EmployeeContract
		for j := range emp.Contracts {
			cat := emp.Contracts[j].StaffCategory
			if cat == string(models.StaffCategoryQualified) || cat == string(models.StaffCategorySupplementary) {
				pedContracts = append(pedContracts, emp.Contracts[j])
			}
		}
		if len(pedContracts) > 0 {
			emp.Contracts = pedContracts
			result = append(result, emp)
		}
	}
	return result
}

// loadDataSet fetches all data needed for statistics calculations from the stores.
func (s *StatisticsService) loadDataSet(ctx context.Context, orgID uint, rangeStart, rangeEnd time.Time, sectionID *uint) (*DataSet, error) {
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

	// Non-fatal, but not silent. GetFinancials reports this as
	// budget_items_load_failed so the user knows operating costs are missing
	// from the figures; the forecast dropped them on the floor and showed a
	// balance with no expenses and no banner, which is the more misleading of
	// the two places to hide it.
	var warnings []models.CalculationWarning
	budgetItems, err := s.budgetItemStore.FindByOrganizationWithEntries(ctx, orgID)
	if err != nil {
		slog.Warn("failed to load budget items for forecast; expense breakdown will exclude operating costs",
			"org_id", orgID, "error", err)
		warnings = append(warnings, models.CalculationWarning{
			Code:    "budget_items_load_failed",
			Message: "could not load budget items; expense breakdown excludes operating costs",
		})
		budgetItems = nil
	}
	if sectionID != nil {
		budgetItems = sectionAttributableBudgetItems(budgetItems)
	}

	return &DataSet{
		Children:       children,
		Employees:      employees,
		FundingPeriods: fundingPeriods,
		PayPlans:       payPlans,
		BudgetItems:    budgetItems,
		LoadWarnings:   warnings,
	}, nil
}

// sectionAttributableBudgetItems keeps only the budget items that can honestly
// be charged to a single Bereich.
//
// A per-child item is attributable: calculateFinancials multiplies it by the
// child count, and under a section scope that count is the section's, so
// "Elternbeitrag 90 EUR x Nest's 7 children" is genuinely Nest's income.
//
// A fixed item is not. Rent, the garden, insurance belong to the house. The
// section-scoped calculation used to charge each of them in FULL to every
// section, so four sections reported four times the organization's operating
// costs between them. Splitting the cost instead would need an allocation key
// (per child, per hour, per head) — a management-accounting decision this
// product has not made, and one that belongs in a designed Bereichs-Umlage
// feature rather than in a filter.
//
// Dropping them is safe for what the forecast is actually for. The forecast's
// value is the DELTA between a baseline run and a scenario run
// (forecast-optimize-tab.tsx compares exactly that way), and a fixed cost is
// identical in both runs, so it cancels. What the exclusion removes is an
// absolute balance that silently carried the whole house's costs under one
// section's heading.
//
// GetFinancials takes no section filter at all for the same reason; see its
// doc-comment.
func sectionAttributableBudgetItems(items []models.BudgetItem) []models.BudgetItem {
	if len(items) == 0 {
		return items
	}
	attributable := make([]models.BudgetItem, 0, len(items))
	for i := range items {
		if items[i].PerChild {
			attributable = append(attributable, items[i])
		}
	}
	return attributable
}
