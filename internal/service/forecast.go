package service

import (
	"context"
	"fmt"
	"maps"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

// virtualIDBase is the FLOOR for virtual (overlay-added) entity IDs. The
// allocator below takes max(this, max-real-ID-in-DataSet) + 1 as its
// starting point, so:
//
//   - Old assumption "any id >= 1_000_000 is a virtual entity" still
//     holds for fresh deployments (front-end / log-grep filters relying
//     on it keep working).
//   - A long-lived org whose auto-incrementing sequence has crossed
//     1_000_000 no longer collides — the allocator starts above the
//     real max, deterministically.
//
// Picking 1_000_000 keeps the boundary visible in logs and avoids the
// top-half-of-uint trick, which would make virtual IDs look like
// uint-overflow garbage.
const virtualIDBase uint = 1_000_000

// overlayIDAllocator hands out unique uint IDs for virtual entities and
// their contracts. A single counter spans employees, employee contracts,
// children, and child contracts so we never have to reason about
// "could two contract kinds share an ID space" (they would, but the
// downstream code is type-disambiguated, so collision-free is simpler
// than collision-safe).
//
// The starting point is `max(virtualIDBase, max(real IDs in DataSet)+1)`
// — see virtualIDBase doc-comment for why both inputs matter.
type overlayIDAllocator struct {
	next uint
}

func newOverlayIDAllocator(ds *DataSet) *overlayIDAllocator {
	next := virtualIDBase
	bump := func(id uint) {
		if id >= next {
			next = id + 1
		}
	}
	for i := range ds.Employees {
		bump(ds.Employees[i].ID)
		for j := range ds.Employees[i].Contracts {
			bump(ds.Employees[i].Contracts[j].ID)
		}
	}
	for i := range ds.Children {
		bump(ds.Children[i].ID)
		for j := range ds.Children[i].Contracts {
			bump(ds.Children[i].Contracts[j].ID)
		}
	}
	return &overlayIDAllocator{next: next}
}

// nextID returns a fresh ID and advances the counter. Per-call so each
// virtual contract — even within the same overlay-added employee — gets
// its own ID. Without this, two overlay employees with one contract
// each previously both ended up with contract.ID == virtualIDBase, an
// upstream bug that mostly survived because nothing yet keys per
// contract.ID; left in place it would silently corrupt any future
// `map[contractID]Foo`.
func (a *overlayIDAllocator) nextID() uint {
	id := a.next
	a.next++
	return id
}

// GetForecast runs all statistics calculations with overlay modifications applied.
func (s *StatisticsService) GetForecast(ctx context.Context, orgID uint, req *models.ForecastRequest) (*models.ForecastResponse, error) {
	if err := s.validateOverlay(ctx, req, orgID); err != nil {
		return nil, err
	}

	rangeStart, rangeEnd, err := snapAndValidateRange(req.From, req.To)
	if err != nil {
		return nil, err
	}

	ds, err := s.loadDataSet(ctx, orgID, rangeStart, rangeEnd, req.SectionID)
	if err != nil {
		return nil, err
	}

	applyOverlay(ds, req)

	// Load any pay plans referenced by overlay employees that aren't already in the DataSet.
	// We must NOT reload all pay plans — that would wipe overlay-added periods.
	if err := s.loadMissingPayPlans(ctx, ds); err != nil {
		return nil, err
	}

	pedEmployees := ds.PedagogicalEmployees()

	dates, rows := calculateEmployeeStaffingHours(ds.Employees, rangeStart, rangeEnd)
	dataPoints, warnings := calculateFinancials(ds.Children, ds.Employees, ds.PayPlans, ds.FundingPeriods, ds.BudgetItems, rangeStart, rangeEnd)

	return &models.ForecastResponse{
		Financials: &models.FinancialResponse{
			DataPoints: dataPoints,
			Warnings:   warnings,
		},
		StaffingHours: &models.StaffingHoursResponse{
			DataPoints: calculateStaffingHours(ds.Children, pedEmployees, ds.FundingPeriods, rangeStart, rangeEnd),
		},
		Occupancy:             calculateOccupancy(ds.Children, ds.FundingPeriods, rangeStart, rangeEnd),
		EmployeeStaffingHours: &models.EmployeeStaffingHoursResponse{Dates: dates, Employees: rows},
		Warnings:              warnings,
	}, nil
}

// validateOverlay checks overlay fields and that all referenced IDs belong to the organization.
func (s *StatisticsService) validateOverlay(ctx context.Context, req *models.ForecastRequest, orgID uint) error {
	// When the request scopes to a specific section, every overlay add
	// MUST target that section. The previous behavior silently filtered
	// mismatches in applyOverlay; users who submitted "add 5 employees
	// in section A" with `section_id: B` got back "0 employees added"
	// with no error and no clue why. Catch the contradiction at the
	// boundary so the response carries a precise field path.
	if req.SectionID != nil {
		if err := validateOverlaySectionMatches(req, *req.SectionID); err != nil {
			return err
		}
	}

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

	// First pass: cheap, no-DB validation. Catches missing fields and
	// (for standalone contracts) the parent-id-zero footgun before we
	// burn round trips on requests that can never succeed.
	for i, ac := range req.AddEmployeeContracts {
		if ac.EmployeeID == 0 {
			return apperror.BadRequest(fmt.Sprintf("add_employee_contracts[%d]: employee_id is required", i))
		}
	}
	for i, ac := range req.AddChildContracts {
		if ac.ChildID == 0 {
			return apperror.BadRequest(fmt.Sprintf("add_child_contracts[%d]: child_id is required", i))
		}
	}

	// Second pass: batch all org-membership checks. The previous code
	// did ~N+1 round trips (one per section, pay plan, removed employee,
	// removed child, standalone contract reference) — for a 50-employee
	// + 50-child overlay that's 200+ DB calls before any calculation
	// runs. Each batch returns a presence map; absence from the map
	// means either the row doesn't exist OR it belongs to another org —
	// validateOverlay treats both as the same not-found error to avoid
	// leaking cross-org existence.
	if err := s.validateOverlaySectionsExist(ctx, req, orgID); err != nil {
		return err
	}
	if err := s.validateOverlayPayPlansBelongToOrg(ctx, req, orgID); err != nil {
		return err
	}
	if err := s.validateOverlayEmployeesExist(ctx, req, orgID); err != nil {
		return err
	}
	if err := s.validateOverlayChildrenExist(ctx, req, orgID); err != nil {
		return err
	}

	return nil
}

// validateOverlaySectionsExist batches one query for every section
// referenced anywhere in the overlay.
func (s *StatisticsService) validateOverlaySectionsExist(ctx context.Context, req *models.ForecastRequest, orgID uint) error {
	ids := collectOverlaySectionIDs(req)
	if len(ids) == 0 {
		return nil
	}
	found, err := s.sectionStore.FindByIDsAndOrg(ctx, ids, orgID)
	if err != nil {
		return apperror.InternalWrap(err, "failed to validate overlay sections")
	}
	for _, id := range ids {
		if _, ok := found[id]; !ok {
			return apperror.BadRequest(fmt.Sprintf("section %d not found in this organization", id))
		}
	}
	return nil
}

// validateOverlayPayPlansBelongToOrg uses the existing
// FindByIDsWithPeriods batch (same one the calculator uses to load
// rates), then checks org-membership inline. Avoids a second query
// just to verify ownership.
func (s *StatisticsService) validateOverlayPayPlansBelongToOrg(ctx context.Context, req *models.ForecastRequest, orgID uint) error {
	ids := collectOverlayPayPlanIDs(req)
	if len(ids) == 0 {
		return nil
	}
	loaded, err := s.payPlanStore.FindByIDsWithPeriods(ctx, ids)
	if err != nil {
		return apperror.InternalWrap(err, "failed to validate overlay pay plans")
	}
	for _, id := range ids {
		pp, ok := loaded[id]
		if !ok {
			return apperror.BadRequest(fmt.Sprintf("pay plan %d not found", id))
		}
		if pp.OrganizationID != orgID {
			return apperror.BadRequest(fmt.Sprintf("pay plan %d does not belong to this organization", id))
		}
	}
	return nil
}

// validateOverlayEmployeesExist batches both RemoveEmployeeIDs and
// AddEmployeeContracts.EmployeeID into a single query.
func (s *StatisticsService) validateOverlayEmployeesExist(ctx context.Context, req *models.ForecastRequest, orgID uint) error {
	seen := make(map[uint]bool)
	var ids []uint
	add := func(id uint) {
		if id != 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, eid := range req.RemoveEmployeeIDs {
		add(eid)
	}
	for _, ac := range req.AddEmployeeContracts {
		add(ac.EmployeeID)
	}
	if len(ids) == 0 {
		return nil
	}
	found, err := s.employeeStore.FindByIDsAndOrg(ctx, ids, orgID)
	if err != nil {
		return apperror.InternalWrap(err, "failed to validate overlay employees")
	}
	for _, id := range ids {
		if _, ok := found[id]; !ok {
			return apperror.BadRequest(fmt.Sprintf("employee %d not found in this organization", id))
		}
	}
	return nil
}

// validateOverlayChildrenExist is the child-side counterpart.
func (s *StatisticsService) validateOverlayChildrenExist(ctx context.Context, req *models.ForecastRequest, orgID uint) error {
	seen := make(map[uint]bool)
	var ids []uint
	add := func(id uint) {
		if id != 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, cid := range req.RemoveChildIDs {
		add(cid)
	}
	for _, ac := range req.AddChildContracts {
		add(ac.ChildID)
	}
	if len(ids) == 0 {
		return nil
	}
	found, err := s.childStore.FindByIDsAndOrg(ctx, ids, orgID)
	if err != nil {
		return apperror.InternalWrap(err, "failed to validate overlay children")
	}
	for _, id := range ids {
		if _, ok := found[id]; !ok {
			return apperror.BadRequest(fmt.Sprintf("child %d not found in this organization", id))
		}
	}
	return nil
}

// validateOverlaySectionMatches enforces that every overlay add targets
// the request's section_id when one is set. Iterates all four add paths
// (whole entities and standalone contracts, employees and children) so
// the error message points at the exact path the caller can fix.
func validateOverlaySectionMatches(req *models.ForecastRequest, want uint) error {
	for i := range req.AddEmployees {
		for j, ct := range req.AddEmployees[i].Contracts {
			if ct.SectionID != want {
				return apperror.BadRequest(fmt.Sprintf(
					"add_employees[%d].contracts[%d]: section_id %d does not match request section_id %d",
					i, j, ct.SectionID, want))
			}
		}
	}
	for i, ct := range req.AddEmployeeContracts {
		if ct.SectionID != want {
			return apperror.BadRequest(fmt.Sprintf(
				"add_employee_contracts[%d]: section_id %d does not match request section_id %d",
				i, ct.SectionID, want))
		}
	}
	for i := range req.AddChildren {
		for j, ct := range req.AddChildren[i].Contracts {
			if ct.SectionID != want {
				return apperror.BadRequest(fmt.Sprintf(
					"add_children[%d].contracts[%d]: section_id %d does not match request section_id %d",
					i, j, ct.SectionID, want))
			}
		}
	}
	for i, ct := range req.AddChildContracts {
		if ct.SectionID != want {
			return apperror.BadRequest(fmt.Sprintf(
				"add_child_contracts[%d]: section_id %d does not match request section_id %d",
				i, ct.SectionID, want))
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

// loadMissingPayPlans loads pay plans referenced by employees (typically
// overlay-added virtual employees) that aren't already in the DataSet.
//
// A store error here would cause calculateFinancials to silently exclude
// those employees' salaries — the same forecast-correctness footgun that
// loadPayPlans had. Propagate. Per-row "pay plan id has no row in DB"
// becomes a CalculationWarning at calc time, not an error here.
func (s *StatisticsService) loadMissingPayPlans(ctx context.Context, ds *DataSet) error {
	var missingIDs []uint
	seen := make(map[uint]bool)
	for i := range ds.Employees {
		for j := range ds.Employees[i].Contracts {
			ppID := ds.Employees[i].Contracts[j].PayPlanID
			if ppID == 0 || seen[ppID] {
				continue
			}
			seen[ppID] = true
			if _, exists := ds.PayPlans[ppID]; !exists {
				missingIDs = append(missingIDs, ppID)
			}
		}
	}
	if len(missingIDs) == 0 {
		return nil
	}
	loaded, err := s.payPlanStore.FindByIDsWithPeriods(ctx, missingIDs)
	if err != nil {
		return apperror.InternalWrap(err, "failed to load overlay pay plans")
	}
	maps.Copy(ds.PayPlans, loaded)
	return nil
}

// applyOverlay mutates the DataSet in-place according to the overlay
// request. Order: removes → add contracts to existing → add new virtual
// entities.
//
// Section filtering is NOT applied here — validateOverlay rejects any
// overlay add whose SectionID disagrees with req.SectionID, so by the
// time we reach this function the only legal sections are present.
// Keeping the filter out of applyOverlay means "0 results" can no
// longer be a silent symptom of a section/section mismatch.
func applyOverlay(ds *DataSet, req *models.ForecastRequest) {
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
		for i := range ds.Employees {
			if ds.Employees[i].ID == ac.EmployeeID {
				ds.Employees[i].Contracts = append(ds.Employees[i].Contracts, ac)
				break
			}
		}
	}

	// 4. Add contracts to existing children
	for _, ac := range req.AddChildContracts {
		for i := range ds.Children {
			if ds.Children[i].ID == ac.ChildID {
				ds.Children[i].Contracts = append(ds.Children[i].Contracts, ac)
				break
			}
		}
	}

	// 5+6. Allocate virtual IDs for new entities and their contracts. A
	// single allocator handles both kinds; see overlayIDAllocator for
	// why every contract — including multiple under one new employee —
	// gets its own ID rather than the index-within-entity that the
	// previous code used (which collided across entities).
	alloc := newOverlayIDAllocator(ds)

	// 5. Add new virtual employees
	for i := range req.AddEmployees {
		emp := req.AddEmployees[i]
		emp.ID = alloc.nextID()
		for j := range emp.Contracts {
			emp.Contracts[j].ID = alloc.nextID()
			emp.Contracts[j].EmployeeID = emp.ID
		}
		if len(emp.Contracts) > 0 {
			ds.Employees = append(ds.Employees, emp)
		}
	}

	// 6. Add new virtual children
	for i := range req.AddChildren {
		child := req.AddChildren[i]
		child.ID = alloc.nextID()
		for j := range child.Contracts {
			child.Contracts[j].ID = alloc.nextID()
			child.Contracts[j].ChildID = child.ID
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
