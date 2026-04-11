package service

import (
	"context"
	"fmt"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

// virtualIDBase is the starting ID for virtual (overlay-added) entities.
// Chosen to be high enough to never collide with real DB IDs.
const virtualIDBase uint = 1_000_000

// GetForecast runs all statistics calculations with overlay modifications applied.
func (s *StatisticsService) GetForecast(ctx context.Context, orgID uint, req *models.ForecastRequest) (*models.ForecastResponse, error) {
	if err := s.validateOverlay(ctx, req, orgID); err != nil {
		return nil, err
	}

	rangeStart, rangeEnd := snapDateRange(req.From, req.To)

	ds, err := s.loadDataSet(ctx, orgID, rangeStart, rangeEnd, req.SectionID)
	if err != nil {
		return nil, err
	}

	applyOverlay(ds, req, req.SectionID)

	// Load any pay plans referenced by overlay employees that aren't already in the DataSet.
	// We must NOT reload all pay plans — that would wipe overlay-added periods.
	s.loadMissingPayPlans(ctx, ds)

	pedEmployees := ds.PedagogicalEmployees()

	dates, rows := calculateEmployeeStaffingHours(ds.Employees, rangeStart, rangeEnd)

	return &models.ForecastResponse{
		Financials: &models.FinancialResponse{
			DataPoints: calculateFinancials(ds.Children, ds.Employees, ds.PayPlans, ds.FundingPeriods, ds.BudgetItems, rangeStart, rangeEnd),
		},
		StaffingHours: &models.StaffingHoursResponse{
			DataPoints: calculateStaffingHours(ds.Children, pedEmployees, ds.FundingPeriods, rangeStart, rangeEnd),
		},
		Occupancy:             calculateOccupancy(ds.Children, ds.FundingPeriods, rangeStart, rangeEnd),
		EmployeeStaffingHours: &models.EmployeeStaffingHoursResponse{Dates: dates, Employees: rows},
	}, nil
}

// validateOverlay checks overlay fields and that all referenced IDs belong to the organization.
func (s *StatisticsService) validateOverlay(ctx context.Context, req *models.ForecastRequest, orgID uint) error {
	// Validate overlay children fields
	if err := validateOverlayChildren(req.AddChildren); err != nil {
		return err
	}
	if err := validateOverlayChildContracts(req.AddChildContracts); err != nil {
		return err
	}

	// Validate overlay employee fields
	if err := validateOverlayEmployees(req.AddEmployees); err != nil {
		return err
	}
	if err := validateOverlayEmployeeContracts(req.AddEmployeeContracts); err != nil {
		return err
	}

	// Validate section IDs belong to org
	sectionIDs := collectOverlaySectionIDs(req)
	for _, sid := range sectionIDs {
		if err := validateSectionOrg(ctx, s.sectionStore, sid, orgID); err != nil {
			return err
		}
	}

	// Validate pay plan IDs belong to org
	payPlanIDs := collectOverlayPayPlanIDs(req)
	for _, ppID := range payPlanIDs {
		pp, err := s.payPlanStore.FindByID(ctx, ppID)
		if err != nil {
			return apperror.BadRequest("pay plan not found")
		}
		if pp.OrganizationID != orgID {
			return apperror.BadRequest("pay plan does not belong to this organization")
		}
	}

	// Validate employee IDs to remove
	for _, eid := range req.RemoveEmployeeIDs {
		if _, err := s.employeeStore.FindByIDMinimalAndOrg(ctx, eid, orgID); err != nil {
			return apperror.BadRequest("employee not found in this organization")
		}
	}

	// Validate employee IDs for standalone contract additions
	for _, ac := range req.AddEmployeeContracts {
		if ac.EmployeeID == 0 {
			return apperror.BadRequest("standalone employee contract requires employee_id")
		}
		if _, err := s.employeeStore.FindByIDMinimalAndOrg(ctx, ac.EmployeeID, orgID); err != nil {
			return apperror.BadRequest("employee not found in this organization")
		}
	}

	// Validate child IDs to remove
	for _, cid := range req.RemoveChildIDs {
		if _, err := s.childStore.FindByIDMinimalAndOrg(ctx, cid, orgID); err != nil {
			return apperror.BadRequest("child not found in this organization")
		}
	}

	// Validate child IDs for standalone contract additions
	for _, ac := range req.AddChildContracts {
		if ac.ChildID == 0 {
			return apperror.BadRequest("standalone child contract requires child_id")
		}
		if _, err := s.childStore.FindByIDMinimalAndOrg(ctx, ac.ChildID, orgID); err != nil {
			return apperror.BadRequest("child not found in this organization")
		}
	}

	return nil
}

// validateOverlayChildren validates the calculation-critical fields on overlay children.
func validateOverlayChildren(children []models.Child) error {
	for i, c := range children {
		if c.Birthdate.IsZero() {
			return apperror.BadRequest(fmt.Sprintf("add_children[%d]: birthdate is required", i))
		}
		if len(c.Contracts) == 0 {
			return apperror.BadRequest(fmt.Sprintf("add_children[%d]: at least one contract is required", i))
		}
		for j, ct := range c.Contracts {
			if ct.From.IsZero() {
				return apperror.BadRequest(fmt.Sprintf("add_children[%d].contracts[%d]: from is required", i, j))
			}
			if ct.SectionID == 0 {
				return apperror.BadRequest(fmt.Sprintf("add_children[%d].contracts[%d]: section_id is required", i, j))
			}
		}
	}
	return nil
}

// validateOverlayChildContracts validates standalone child contract additions.
func validateOverlayChildContracts(contracts []models.ChildContract) error {
	for i, ct := range contracts {
		if ct.ChildID == 0 {
			return apperror.BadRequest(fmt.Sprintf("add_child_contracts[%d]: child_id is required", i))
		}
		if ct.From.IsZero() {
			return apperror.BadRequest(fmt.Sprintf("add_child_contracts[%d]: from is required", i))
		}
		if ct.SectionID == 0 {
			return apperror.BadRequest(fmt.Sprintf("add_child_contracts[%d]: section_id is required", i))
		}
	}
	return nil
}

// validateOverlayEmployees validates the calculation-critical fields on overlay employees.
func validateOverlayEmployees(employees []models.Employee) error {
	for i, e := range employees {
		if len(e.Contracts) == 0 {
			return apperror.BadRequest(fmt.Sprintf("add_employees[%d]: at least one contract is required", i))
		}
		for j, ct := range e.Contracts {
			if ct.From.IsZero() {
				return apperror.BadRequest(fmt.Sprintf("add_employees[%d].contracts[%d]: from is required", i, j))
			}
			if ct.SectionID == 0 {
				return apperror.BadRequest(fmt.Sprintf("add_employees[%d].contracts[%d]: section_id is required", i, j))
			}
			if ct.PayPlanID == 0 {
				return apperror.BadRequest(fmt.Sprintf("add_employees[%d].contracts[%d]: pay_plan_id is required", i, j))
			}
			if ct.Grade == "" {
				return apperror.BadRequest(fmt.Sprintf("add_employees[%d].contracts[%d]: grade is required", i, j))
			}
			if ct.Step < 1 {
				return apperror.BadRequest(fmt.Sprintf("add_employees[%d].contracts[%d]: step must be >= 1", i, j))
			}
			if ct.WeeklyHours <= 0 {
				return apperror.BadRequest(fmt.Sprintf("add_employees[%d].contracts[%d]: weekly_hours must be > 0", i, j))
			}
			if ct.StaffCategory == "" {
				return apperror.BadRequest(fmt.Sprintf("add_employees[%d].contracts[%d]: staff_category is required", i, j))
			}
		}
	}
	return nil
}

// validateOverlayEmployeeContracts validates standalone employee contract additions.
func validateOverlayEmployeeContracts(contracts []models.EmployeeContract) error {
	for i, ct := range contracts {
		if ct.EmployeeID == 0 {
			return apperror.BadRequest(fmt.Sprintf("add_employee_contracts[%d]: employee_id is required", i))
		}
		if ct.From.IsZero() {
			return apperror.BadRequest(fmt.Sprintf("add_employee_contracts[%d]: from is required", i))
		}
		if ct.SectionID == 0 {
			return apperror.BadRequest(fmt.Sprintf("add_employee_contracts[%d]: section_id is required", i))
		}
		if ct.PayPlanID == 0 {
			return apperror.BadRequest(fmt.Sprintf("add_employee_contracts[%d]: pay_plan_id is required", i))
		}
		if ct.Grade == "" {
			return apperror.BadRequest(fmt.Sprintf("add_employee_contracts[%d]: grade is required", i))
		}
		if ct.Step < 1 {
			return apperror.BadRequest(fmt.Sprintf("add_employee_contracts[%d]: step must be >= 1", i))
		}
		if ct.WeeklyHours <= 0 {
			return apperror.BadRequest(fmt.Sprintf("add_employee_contracts[%d]: weekly_hours must be > 0", i))
		}
		if ct.StaffCategory == "" {
			return apperror.BadRequest(fmt.Sprintf("add_employee_contracts[%d]: staff_category is required", i))
		}
	}
	return nil
}

// collectOverlaySectionIDs returns all unique section IDs referenced in overlay operations.
func collectOverlaySectionIDs(req *models.ForecastRequest) []uint {
	seen := make(map[uint]bool)
	var ids []uint
	add := func(id uint) {
		if id != 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}

	for i := range req.AddEmployees {
		for j := range req.AddEmployees[i].Contracts {
			add(req.AddEmployees[i].Contracts[j].SectionID)
		}
	}
	for i := range req.AddEmployeeContracts {
		add(req.AddEmployeeContracts[i].SectionID)
	}
	for i := range req.AddChildren {
		for j := range req.AddChildren[i].Contracts {
			add(req.AddChildren[i].Contracts[j].SectionID)
		}
	}
	for i := range req.AddChildContracts {
		add(req.AddChildContracts[i].SectionID)
	}
	return ids
}

// collectOverlayPayPlanIDs returns all unique pay plan IDs referenced in overlay operations.
func collectOverlayPayPlanIDs(req *models.ForecastRequest) []uint {
	seen := make(map[uint]bool)
	var ids []uint
	add := func(id uint) {
		if id != 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}

	for i := range req.AddEmployees {
		for j := range req.AddEmployees[i].Contracts {
			add(req.AddEmployees[i].Contracts[j].PayPlanID)
		}
	}
	for i := range req.AddEmployeeContracts {
		add(req.AddEmployeeContracts[i].PayPlanID)
	}
	return ids
}

// loadMissingPayPlans loads pay plans referenced by employees but not yet in the DataSet.
func (s *StatisticsService) loadMissingPayPlans(ctx context.Context, ds *DataSet) {
	var missingIDs []uint
	for i := range ds.Employees {
		for j := range ds.Employees[i].Contracts {
			ppID := ds.Employees[i].Contracts[j].PayPlanID
			if ppID != 0 {
				if _, exists := ds.PayPlans[ppID]; !exists {
					missingIDs = append(missingIDs, ppID)
				}
			}
		}
	}
	if len(missingIDs) == 0 {
		return
	}
	loaded, err := s.payPlanStore.FindByIDsWithPeriods(ctx, missingIDs)
	if err != nil {
		return // non-fatal
	}
	for id, pp := range loaded {
		ds.PayPlans[id] = pp
	}
}

// applyOverlay mutates the DataSet in-place according to the overlay request.
// Order: removes → add contracts to existing → add new virtual entities.
// If sectionID is non-nil, overlay additions are filtered to that section.
func applyOverlay(ds *DataSet, req *models.ForecastRequest, sectionID *uint) {
	// 1. Remove employees
	if len(req.RemoveEmployeeIDs) > 0 {
		removeSet := toUintSet(req.RemoveEmployeeIDs)
		ds.Employees = filterSlice(ds.Employees, func(e models.Employee) bool {
			return !removeSet[e.ID]
		})
	}

	// 2. Remove children
	if len(req.RemoveChildIDs) > 0 {
		removeSet := toUintSet(req.RemoveChildIDs)
		ds.Children = filterSlice(ds.Children, func(c models.Child) bool {
			return !removeSet[c.ID]
		})
	}

	// 3. Add contracts to existing employees
	for _, ac := range req.AddEmployeeContracts {
		if sectionID != nil && ac.SectionID != *sectionID {
			continue
		}
		for i := range ds.Employees {
			if ds.Employees[i].ID == ac.EmployeeID {
				ds.Employees[i].Contracts = append(ds.Employees[i].Contracts, ac)
				break
			}
		}
	}

	// 4. Add contracts to existing children
	for _, ac := range req.AddChildContracts {
		if sectionID != nil && ac.SectionID != *sectionID {
			continue
		}
		for i := range ds.Children {
			if ds.Children[i].ID == ac.ChildID {
				ds.Children[i].Contracts = append(ds.Children[i].Contracts, ac)
				break
			}
		}
	}

	// 5. Add new virtual employees
	for i := range req.AddEmployees {
		emp := req.AddEmployees[i]
		virtualID := virtualIDBase + uint(i) //nolint:gosec // index cannot overflow
		emp.ID = virtualID
		for j := range emp.Contracts {
			emp.Contracts[j].ID = virtualIDBase + uint(j) //nolint:gosec // index cannot overflow
			emp.Contracts[j].EmployeeID = virtualID
		}
		if sectionID != nil {
			emp.Contracts = filterSlice(emp.Contracts, func(c models.EmployeeContract) bool {
				return c.SectionID == *sectionID
			})
		}
		if len(emp.Contracts) > 0 {
			ds.Employees = append(ds.Employees, emp)
		}
	}

	// 6. Add new virtual children (IDs offset from employees to avoid collisions)
	childVirtualIDBase := virtualIDBase + uint(len(req.AddEmployees)) //nolint:gosec // length cannot overflow
	for i := range req.AddChildren {
		child := req.AddChildren[i]
		virtualID := childVirtualIDBase + uint(i) //nolint:gosec // index cannot overflow
		child.ID = virtualID
		for j := range child.Contracts {
			child.Contracts[j].ID = virtualIDBase + uint(j) //nolint:gosec // index cannot overflow
			child.Contracts[j].ChildID = virtualID
		}
		if sectionID != nil {
			child.Contracts = filterSlice(child.Contracts, func(c models.ChildContract) bool {
				return c.SectionID == *sectionID
			})
		}
		if len(child.Contracts) > 0 {
			ds.Children = append(ds.Children, child)
		}
	}
}

// --- Generic helpers ---

func toUintSet(ids []uint) map[uint]bool {
	set := make(map[uint]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

func filterSlice[T any](s []T, keep func(T) bool) []T {
	var result []T
	for _, item := range s {
		if keep(item) {
			result = append(result, item)
		}
	}
	return result
}
