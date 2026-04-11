package service

import (
	"context"
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

	payPlans := s.loadPayPlans(ctx, employees)

	budgetItems, err := s.budgetItemStore.FindByOrganizationWithEntries(ctx, orgID)
	if err != nil {
		budgetItems = nil // non-fatal: proceed without budget items
	}

	return &DataSet{
		Children:       children,
		Employees:      employees,
		FundingPeriods: fundingPeriods,
		PayPlans:       payPlans,
		BudgetItems:    budgetItems,
	}, nil
}
