package service

import (
	"context"
	"strings"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// Intent-based employee contract operations: correct, amend, end, move boundary.
// Mirrors the child side; the difference is the pay-plan coverage check, which
// has to be anchored at the date the contract actually starts.

// loadEmployeeContract fetches a contract and proves the caller may touch it.
func (s *EmployeeService) loadEmployeeContract(ctx context.Context, contractID, employeeID, orgID uint) (*models.Employee, *models.EmployeeContract, error) {
	employee, err := s.store.FindByIDMinimal(ctx, employeeID)
	if err != nil {
		return nil, nil, classifyStoreError(err, "employee")
	}
	if err := verifyOrgOwnership(employee, orgID, "employee"); err != nil {
		return nil, nil, err
	}

	contract, err := s.store.FindContractByID(ctx, contractID)
	if err != nil {
		return nil, nil, classifyStoreError(err, "contract")
	}
	if err := verifyRecordOwnership(contract, employeeID, "contract"); err != nil {
		return nil, nil, err
	}
	return employee, contract, nil
}

// validateEmployeeContractFields checks the employee-specific values a correct or
// amend request may carry. Only fields the request actually set are checked, so
// touching one field cannot fail on another's stored value.
func (s *EmployeeService) validateEmployeeContractFields(
	ctx context.Context, orgID uint,
	payPlanID models.Opt[uint], staffCategory models.Opt[string],
	weeklyHours models.Opt[float64], sectionID models.Opt[uint],
) error {
	if payPlanID.IsNull() {
		return apperror.BadRequest("payplan_id cannot be null: every contract has a pay plan")
	}
	if staffCategory.IsNull() {
		return apperror.BadRequest("staff_category cannot be null")
	}
	if weeklyHours.IsNull() {
		return apperror.BadRequest("weekly_hours cannot be null; send 0 for a contract with no hours")
	}
	if sectionID.IsNull() {
		return apperror.BadRequest("section_id cannot be null: every contract belongs to a section")
	}

	if id, ok := payPlanID.Get(); ok {
		payPlan, err := s.payPlanStore.FindByID(ctx, id)
		if err != nil {
			return apperror.BadRequest("payplan_id not found")
		}
		if payPlan.OrganizationID != orgID {
			return apperror.BadRequest("payplan does not belong to this organization")
		}
	}
	if category, ok := staffCategory.Get(); ok && !models.IsValidStaffCategory(category) {
		return apperror.BadRequest("staff_category must be one of: qualified, supplementary, non_pedagogical")
	}
	if hours, ok := weeklyHours.Get(); ok {
		// 0 is legitimate — parental leave keeps the contract with no hours — which
		// is why this is a range check and not a `required` binding.
		if err := validation.ValidateWeeklyHours(hours, "weekly_hours"); err != nil {
			return apperror.BadRequest(err.Error())
		}
	}
	if id, ok := sectionID.Get(); ok {
		if err := validateSectionOrg(ctx, s.sectionStore, id, orgID); err != nil {
			return err
		}
	}
	return nil
}

// CorrectContract fixes an employee contract's recorded facts in place. Omitted
// fields are left untouched; `to` and `properties` are cleared by an explicit
// null.
func (s *EmployeeService) CorrectContract(ctx context.Context, contractID, employeeID, orgID uint, req *models.EmployeeContractCorrectRequest) (*models.EmployeeContractResponse, error) {
	employee, contract, err := s.loadEmployeeContract(ctx, contractID, employeeID, orgID)
	if err != nil {
		return nil, err
	}

	if req.From.IsNull() {
		return nil, apperror.BadRequest("from cannot be null: every contract has a start date")
	}
	if req.Grade.IsNull() {
		return nil, apperror.BadRequest("grade cannot be null; send an empty string to clear it")
	}
	if req.Step.IsNull() {
		return nil, apperror.BadRequest("step cannot be null")
	}
	if err := s.validateEmployeeContractFields(ctx, orgID, req.PayPlanID, req.StaffCategory, req.WeeklyHours, req.SectionID); err != nil {
		return nil, err
	}

	if from, ok := req.From.Get(); ok {
		contract.From = from
	}
	if req.To.Set {
		contract.To = req.To.Value
	}
	if id, ok := req.SectionID.Get(); ok {
		contract.SectionID = id
		contract.Section = nil
	}
	if id, ok := req.PayPlanID.Get(); ok {
		contract.PayPlanID = id
		contract.PayPlan = nil
	}
	if category, ok := req.StaffCategory.Get(); ok {
		contract.StaffCategory = category
	}
	if grade, ok := req.Grade.Get(); ok {
		contract.Grade = strings.TrimSpace(grade)
	}
	if step, ok := req.Step.Get(); ok {
		contract.Step = step
	}
	if hours, ok := req.WeeklyHours.Get(); ok {
		contract.WeeklyHours = hours
	}
	if req.Properties.Set {
		if props, ok := req.Properties.Get(); ok {
			contract.Properties = props
		} else {
			contract.Properties = nil
		}
	}

	if err := validateContractDatesAfterBirthdate(contract.From, contract.To, employee.Birthdate); err != nil {
		return nil, err
	}

	// Only when the (pay plan, grade, step, from) tuple actually moved: a legacy
	// contract whose pay plan was edited later should not fail because someone
	// corrected its section.
	if req.PayPlanID.Set || req.Grade.Set || req.Step.Set || req.From.Set {
		if err := validatePayPlanGradeStepCovers(ctx, s.payPlanStore, contract.PayPlanID, contract.Grade, contract.Step, contract.From); err != nil {
			return nil, err
		}
	}

	if err := inPlaceContractUpdate(ctx, s.transactor, s.store.Contracts(), employeeID,
		contract.From, contract.To, contract.ID,
		func(txCtx context.Context) error { return s.store.UpdateContract(txCtx, contract) },
	); err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// AmendContract records a change to an employee's contract effective from a
// date — a raise, a change in hours, a move between sections. The addressed
// contract is closed the day before and a successor carrying the changes starts
// on that date; both are returned.
func (s *EmployeeService) AmendContract(ctx context.Context, contractID, employeeID, orgID uint, req *models.EmployeeContractAmendRequest) (*models.EmployeeContractAmendResponse, error) {
	employee, contract, err := s.loadEmployeeContract(ctx, contractID, employeeID, orgID)
	if err != nil {
		return nil, err
	}

	seam := models.TruncateToDate(req.EffectiveFrom)
	if err := checkAmendSeam(contract.From, contract.To, seam); err != nil {
		return nil, err
	}

	if req.Grade.IsNull() {
		return nil, apperror.BadRequest("grade cannot be null; send an empty string to clear it")
	}
	if req.Step.IsNull() {
		return nil, apperror.BadRequest("step cannot be null")
	}
	if err := s.validateEmployeeContractFields(ctx, orgID, req.PayPlanID, req.StaffCategory, req.WeeklyHours, req.SectionID); err != nil {
		return nil, err
	}

	successor := &models.EmployeeContract{
		EmployeeID: contract.EmployeeID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: seam, To: contract.To},
			SectionID:  contract.SectionID,
			Properties: contract.Properties,
		},
		StaffCategory: contract.StaffCategory,
		Grade:         contract.Grade,
		Step:          contract.Step,
		WeeklyHours:   contract.WeeklyHours,
		PayPlanID:     contract.PayPlanID,
	}

	if req.To.Set {
		successor.To = req.To.Value
	}
	if id, ok := req.SectionID.Get(); ok {
		successor.SectionID = id
	}
	if id, ok := req.PayPlanID.Get(); ok {
		successor.PayPlanID = id
	}
	if category, ok := req.StaffCategory.Get(); ok {
		successor.StaffCategory = category
	}
	if grade, ok := req.Grade.Get(); ok {
		successor.Grade = strings.TrimSpace(grade)
	}
	if step, ok := req.Step.Get(); ok {
		successor.Step = step
	}
	if hours, ok := req.WeeklyHours.Get(); ok {
		successor.WeeklyHours = hours
	}
	if req.Properties.Set {
		if props, ok := req.Properties.Get(); ok {
			successor.Properties = props
		} else {
			successor.Properties = nil
		}
	}

	if err := validateContractDatesAfterBirthdate(successor.From, successor.To, employee.Birthdate); err != nil {
		return nil, err
	}

	// Anchored at the seam, not today: the successor starts there, so that is the
	// date at which the pay plan has to cover its grade and step. The old path
	// checked at today, which accepted a backdated amendment to a grade the plan
	// only gained later and rejected one it had at the time.
	if err := validatePayPlanGradeStepCovers(ctx, s.payPlanStore, successor.PayPlanID, successor.Grade, successor.Step, successor.From); err != nil {
		return nil, err
	}

	if err := amendSeam(ctx, s.transactor, s.store.Contracts(), employeeID,
		seam, successor.To,
		func(txCtx context.Context, dayBefore time.Time) error {
			contract.To = &dayBefore
			return s.store.UpdateContract(txCtx, contract)
		},
		func(txCtx context.Context) error { return s.store.CreateContract(txCtx, successor) },
	); err != nil {
		return nil, err
	}

	return &models.EmployeeContractAmendResponse{
		Closed:  contract.ToResponse(),
		Created: successor.ToResponse(),
	}, nil
}

// EndContract sets or clears an employee contract's end date. `to` must be
// present; a null reopens the contract as ongoing.
func (s *EmployeeService) EndContract(ctx context.Context, contractID, employeeID, orgID uint, req *models.ContractEndRequest) (*models.EmployeeContractResponse, error) {
	employee, contract, err := s.loadEmployeeContract(ctx, contractID, employeeID, orgID)
	if err != nil {
		return nil, err
	}
	if !req.To.Set {
		return nil, apperror.BadRequest("to is required; send null to reopen the contract as ongoing")
	}

	contract.To = req.To.Value

	if err := validation.ValidatePeriod(contract.From, contract.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	if err := validateContractDatesAfterBirthdate(contract.From, contract.To, employee.Birthdate); err != nil {
		return nil, err
	}

	if err := inPlaceContractUpdate(ctx, s.transactor, s.store.Contracts(), employeeID,
		contract.From, contract.To, contract.ID,
		func(txCtx context.Context) error { return s.store.UpdateContract(txCtx, contract) },
	); err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// MoveContractBoundary moves the seam between two adjacent employee contracts.
func (s *EmployeeService) MoveContractBoundary(ctx context.Context, employeeID, orgID uint, req *models.ContractBoundaryMoveRequest) (*models.EmployeeContractBoundaryResponse, error) {
	if req.EarlierID == req.LaterID {
		return nil, apperror.BadRequest("earlier_id and later_id must name different contracts")
	}

	employee, earlier, err := s.loadEmployeeContract(ctx, req.EarlierID, employeeID, orgID)
	if err != nil {
		return nil, err
	}
	_, later, err := s.loadEmployeeContract(ctx, req.LaterID, employeeID, orgID)
	if err != nil {
		return nil, err
	}

	if err := checkAdjacent(earlier.From, earlier.To, later.From, later.To, req.At); err != nil {
		return nil, err
	}

	seam := models.TruncateToDate(req.At)
	dayBefore := seam.AddDate(0, 0, -1)
	earlier.To = &dayBefore
	later.From = seam

	if err := validateContractDatesAfterBirthdate(later.From, later.To, employee.Birthdate); err != nil {
		return nil, err
	}

	// The later contract now starts on a new date, so the pay plan has to cover
	// its grade and step there — a seam dragged back across a pay-plan period
	// boundary would otherwise produce a contract the plan cannot price.
	if err := validatePayPlanGradeStepCovers(ctx, s.payPlanStore, later.PayPlanID, later.Grade, later.Step, later.From); err != nil {
		return nil, err
	}

	if err := moveBoundaryTx(ctx, s.transactor, s.store.Contracts(), employeeID,
		periodRef{id: earlier.ID, from: earlier.From, to: earlier.To},
		periodRef{id: later.ID, from: later.From, to: later.To},
		func(txCtx context.Context) error {
			if err := s.store.UpdateContract(txCtx, earlier); err != nil {
				return err
			}
			return s.store.UpdateContract(txCtx, later)
		},
	); err != nil {
		return nil, err
	}

	return &models.EmployeeContractBoundaryResponse{
		Earlier: earlier.ToResponse(),
		Later:   later.ToResponse(),
	}, nil
}
