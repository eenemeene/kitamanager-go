package service

import (
	"context"
	"time"

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

	// Reload pay plans to include any new PayPlanIDs introduced by overlay employees
	ds.PayPlans = s.loadPayPlans(ctx, ds.Employees)

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

// validateOverlay checks that all referenced IDs in the overlay belong to the given organization.
func (s *StatisticsService) validateOverlay(ctx context.Context, req *models.ForecastRequest, orgID uint) error {
	// Validate section IDs
	sectionIDs := collectOverlaySectionIDs(req)
	for _, sid := range sectionIDs {
		if err := validateSectionOrg(ctx, s.sectionStore, sid, orgID); err != nil {
			return err
		}
	}

	// Validate pay plan IDs
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
			continue
		}
		if _, err := s.employeeStore.FindByIDMinimalAndOrg(ctx, ac.EmployeeID, orgID); err != nil {
			return apperror.BadRequest("employee not found in this organization")
		}
	}

	// Validate employee contract IDs to end
	for _, ec := range req.EndEmployeeContracts {
		contract, err := s.employeeStore.FindContractByID(ctx, ec.ContractID)
		if err != nil {
			return apperror.BadRequest("employee contract not found")
		}
		if _, err := s.employeeStore.FindByIDMinimalAndOrg(ctx, contract.EmployeeID, orgID); err != nil {
			return apperror.BadRequest("employee contract does not belong to this organization")
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
			continue
		}
		if _, err := s.childStore.FindByIDMinimalAndOrg(ctx, ac.ChildID, orgID); err != nil {
			return apperror.BadRequest("child not found in this organization")
		}
	}

	// Validate child contract IDs to end
	for _, ec := range req.EndChildContracts {
		contract, err := s.childStore.FindContractByID(ctx, ec.ContractID)
		if err != nil {
			return apperror.BadRequest("child contract not found")
		}
		if _, err := s.childStore.FindByIDMinimalAndOrg(ctx, contract.ChildID, orgID); err != nil {
			return apperror.BadRequest("child contract does not belong to this organization")
		}
	}

	// Validate budget item IDs to remove
	for _, biID := range req.RemoveBudgetItemIDs {
		bi, err := s.budgetItemStore.FindByID(ctx, biID)
		if err != nil {
			return apperror.BadRequest("budget item not found")
		}
		if bi.OrganizationID != orgID {
			return apperror.BadRequest("budget item does not belong to this organization")
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
	for i := range req.AddPayPlanPeriods {
		add(req.AddPayPlanPeriods[i].PayPlanID)
	}
	return ids
}

// applyOverlay mutates the DataSet in-place according to the overlay request.
// Order: removes → end dates → add contracts → add entities → pay plan/funding/budget overlays.
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

	// 3. End employee contracts
	for _, ec := range req.EndEmployeeContracts {
		endEmployeeContract(ds.Employees, ec.ContractID, ec.EndDate)
	}

	// 4. End child contracts
	for _, ec := range req.EndChildContracts {
		endChildContract(ds.Children, ec.ContractID, ec.EndDate)
	}

	// 5. Add contracts to existing employees
	for _, ac := range req.AddEmployeeContracts {
		if ac.EmployeeID == 0 {
			continue // belongs to AddEmployee, not standalone
		}
		if sectionID != nil && ac.SectionID != *sectionID {
			continue
		}
		addContractToEmployee(ds.Employees, ac)
	}

	// 6. Add contracts to existing children
	for _, ac := range req.AddChildContracts {
		if ac.ChildID == 0 {
			continue
		}
		if sectionID != nil && ac.SectionID != *sectionID {
			continue
		}
		addContractToChild(ds.Children, ac)
	}

	// 7. Add new virtual employees
	for i, ae := range req.AddEmployees {
		virtualID := virtualIDBase + uint(i) //nolint:gosec // index cannot overflow
		contracts := toEmployeeContracts(ae.Contracts, virtualID, sectionID)
		if len(contracts) == 0 {
			continue // all contracts filtered out by section
		}
		emp := models.Employee{
			Person: models.Person{
				ID:        virtualID,
				FirstName: ae.FirstName,
				LastName:  ae.LastName,
				Gender:    ae.Gender,
				Birthdate: ae.Birthdate,
			},
			Contracts: contracts,
		}
		ds.Employees = append(ds.Employees, emp)
	}

	// 8. Add new virtual children (IDs offset from employees to avoid collisions)
	childVirtualIDBase := virtualIDBase + uint(len(req.AddEmployees)) //nolint:gosec // length cannot overflow
	for i, ac := range req.AddChildren {
		virtualID := childVirtualIDBase + uint(i) //nolint:gosec // index cannot overflow
		contracts := toChildContracts(ac.Contracts, virtualID, sectionID)
		if len(contracts) == 0 {
			continue
		}
		child := models.Child{
			Person: models.Person{
				ID:        virtualID,
				FirstName: ac.FirstName,
				LastName:  ac.LastName,
				Gender:    ac.Gender,
				Birthdate: ac.Birthdate,
			},
			Contracts: contracts,
		}
		ds.Children = append(ds.Children, child)
	}

	// 9. Add pay plan periods
	for _, pp := range req.AddPayPlanPeriods {
		if existing, ok := ds.PayPlans[pp.PayPlanID]; ok {
			existing.Periods = append(existing.Periods, toPayPlanPeriod(pp))
		}
		// If pay plan not in DataSet, it will be loaded by the pay plan reload after applyOverlay
	}

	// 10. Add funding periods
	for _, fp := range req.AddFundingPeriods {
		ds.FundingPeriods = append(ds.FundingPeriods, toFundingPeriod(fp))
	}

	// 11. Remove budget items
	if len(req.RemoveBudgetItemIDs) > 0 {
		removeSet := toUintSet(req.RemoveBudgetItemIDs)
		ds.BudgetItems = filterSlice(ds.BudgetItems, func(b models.BudgetItem) bool {
			return !removeSet[b.ID]
		})
	}

	// 12. Add virtual budget items
	budgetVirtualIDBase := virtualIDBase + uint(len(req.AddEmployees)) + uint(len(req.AddChildren)) //nolint:gosec // lengths cannot overflow
	for i, bi := range req.AddBudgetItems {
		ds.BudgetItems = append(ds.BudgetItems, toBudgetItem(bi, budgetVirtualIDBase+uint(i))) //nolint:gosec // index cannot overflow
	}
}

// --- Conversion helpers ---

func toEmployeeContracts(fcs []models.ForecastAddEmployeeContract, employeeID uint, sectionID *uint) []models.EmployeeContract {
	var contracts []models.EmployeeContract
	for i, fc := range fcs {
		if sectionID != nil && fc.SectionID != *sectionID {
			continue
		}
		contracts = append(contracts, models.EmployeeContract{
			ID:         virtualIDBase + uint(i), //nolint:gosec // index cannot overflow
			EmployeeID: employeeID,
			BaseContract: models.BaseContract{
				Period:    models.Period{From: fc.From, To: fc.To},
				SectionID: fc.SectionID,
			},
			StaffCategory: fc.StaffCategory,
			Grade:         fc.Grade,
			Step:          fc.Step,
			WeeklyHours:   fc.WeeklyHours,
			PayPlanID:     fc.PayPlanID,
		})
	}
	return contracts
}

func toChildContracts(fcs []models.ForecastAddChildContract, childID uint, sectionID *uint) []models.ChildContract {
	var contracts []models.ChildContract
	for i, fc := range fcs {
		if sectionID != nil && fc.SectionID != *sectionID {
			continue
		}
		contracts = append(contracts, models.ChildContract{
			ID:      virtualIDBase + uint(i), //nolint:gosec // index cannot overflow
			ChildID: childID,
			BaseContract: models.BaseContract{
				Period:     models.Period{From: fc.From, To: fc.To},
				SectionID:  fc.SectionID,
				Properties: fc.Properties,
			},
		})
	}
	return contracts
}

func toPayPlanPeriod(fp models.ForecastAddPayPlanPeriod) models.PayPlanPeriod {
	entries := make([]models.PayPlanEntry, len(fp.Entries))
	for i, e := range fp.Entries {
		entries[i] = models.PayPlanEntry{
			Grade:         e.Grade,
			Step:          e.Step,
			MonthlyAmount: e.MonthlyAmount,
		}
	}
	return models.PayPlanPeriod{
		PayPlanID:                fp.PayPlanID,
		Period:                   models.Period{From: fp.From, To: fp.To},
		WeeklyHours:              fp.WeeklyHours,
		EmployerContributionRate: fp.EmployerContributionRate,
		Entries:                  entries,
	}
}

func toFundingPeriod(fp models.ForecastAddFundingPeriod) models.GovernmentFundingPeriod {
	props := make([]models.GovernmentFundingProperty, len(fp.Properties))
	for i, p := range fp.Properties {
		props[i] = models.GovernmentFundingProperty{
			Key:                 p.Key,
			Value:               p.Value,
			Label:               p.Label,
			Payment:             p.Payment,
			Requirement:         p.Requirement,
			MinAge:              p.MinAge,
			MaxAge:              p.MaxAge,
			ApplyToAllContracts: p.ApplyToAllContracts,
		}
	}
	return models.GovernmentFundingPeriod{
		Period:              models.Period{From: fp.From, To: fp.To},
		FullTimeWeeklyHours: fp.FullTimeWeeklyHours,
		Properties:          props,
	}
}

func toBudgetItem(bi models.ForecastAddBudgetItem, virtualID uint) models.BudgetItem {
	entries := make([]models.BudgetItemEntry, len(bi.Entries))
	for i, e := range bi.Entries {
		entries[i] = models.BudgetItemEntry{
			Period:      models.Period{From: e.From, To: e.To},
			AmountCents: e.AmountCents,
		}
	}
	return models.BudgetItem{
		ID:       virtualID,
		Name:     bi.Name,
		Category: bi.Category,
		PerChild: bi.PerChild,
		Entries:  entries,
	}
}

// --- Mutation helpers ---

func endEmployeeContract(employees []models.Employee, contractID uint, endDate time.Time) {
	for i := range employees {
		for j := range employees[i].Contracts {
			if employees[i].Contracts[j].ID == contractID {
				employees[i].Contracts[j].To = &endDate
				return
			}
		}
	}
}

func endChildContract(children []models.Child, contractID uint, endDate time.Time) {
	for i := range children {
		for j := range children[i].Contracts {
			if children[i].Contracts[j].ID == contractID {
				children[i].Contracts[j].To = &endDate
				return
			}
		}
	}
}

func addContractToEmployee(employees []models.Employee, ac models.ForecastAddEmployeeContract) {
	for i := range employees {
		if employees[i].ID == ac.EmployeeID {
			employees[i].Contracts = append(employees[i].Contracts, models.EmployeeContract{
				EmployeeID: ac.EmployeeID,
				BaseContract: models.BaseContract{
					Period:    models.Period{From: ac.From, To: ac.To},
					SectionID: ac.SectionID,
				},
				StaffCategory: ac.StaffCategory,
				Grade:         ac.Grade,
				Step:          ac.Step,
				WeeklyHours:   ac.WeeklyHours,
				PayPlanID:     ac.PayPlanID,
			})
			return
		}
	}
}

func addContractToChild(children []models.Child, ac models.ForecastAddChildContract) {
	for i := range children {
		if children[i].ID == ac.ChildID {
			children[i].Contracts = append(children[i].Contracts, models.ChildContract{
				ChildID: ac.ChildID,
				BaseContract: models.BaseContract{
					Period:     models.Period{From: ac.From, To: ac.To},
					SectionID:  ac.SectionID,
					Properties: ac.Properties,
				},
			})
			return
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
