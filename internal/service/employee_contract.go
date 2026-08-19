package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// validatePayPlanGradeStepCovers checks the pay plan has an active period at
// startDate AND that the period contains an entry for (grade, step). Catches
// misconfiguration at contract create/update time — without this, a typo like
// "S88a" or a grade that's missing from the latest period only surfaces later
// as a silent CalculateSalary NotFound or a vanished step-promotion row.
//
// An "unpinned" contract — one with no grade or step yet — is allowed:
// CalculateSalary won't be called for it. Only when both grade is non-empty
// and step >= 1 do we insist on the entry existing.
//
// Only the period covering startDate is checked; future periods that the
// admin hasn't imported yet are tolerated.
func validatePayPlanGradeStepCovers(ctx context.Context, payPlanStore store.PayPlanStorer, payPlanID uint, grade string, step int, startDate time.Time) error {
	grade = strings.TrimSpace(grade)
	if grade == "" || step < 1 {
		return nil
	}
	period, err := payPlanStore.FindActivePeriod(ctx, payPlanID, startDate)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return apperror.BadRequest("pay plan has no active period covering the contract's start date")
		}
		return apperror.InternalWrap(err, "failed to fetch pay plan period")
	}
	if _, err := payPlanStore.FindEntry(ctx, period.ID, grade, step); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return apperror.BadRequest("pay plan has no entry for the given (grade, step) at the contract's start date")
		}
		return apperror.InternalWrap(err, "failed to fetch pay plan entry")
	}
	return nil
}

// ListContracts returns paginated contract history for an employee, validating it belongs to the specified organization
func (s *EmployeeService) ListContracts(ctx context.Context, employeeID, orgID uint, limit, offset int) ([]models.EmployeeContractResponse, int64, error) {
	// Verify employee exists and belongs to org (use minimal query - no preloads needed)
	employee, err := s.store.FindByIDMinimal(ctx, employeeID)
	if err != nil {
		return nil, 0, classifyStoreError(err, "employee")
	}
	if err := verifyOrgOwnership(employee, orgID, "employee"); err != nil {
		return nil, 0, err
	}

	contracts, total, err := s.store.FindContractsByEmployeePaginated(ctx, employeeID, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch contracts")
	}

	return toResponseList(contracts, (*models.EmployeeContract).ToResponse), total, nil
}

// GetCurrentRecord returns the current active contract for an employee, validating it belongs to the specified organization
func (s *EmployeeService) GetCurrentRecord(ctx context.Context, employeeID, orgID uint) (*models.EmployeeContractResponse, error) {
	// Security: Validate employee belongs to the specified organization (use minimal query - no preloads needed)
	employee, err := s.store.FindByIDMinimal(ctx, employeeID)
	if err != nil {
		return nil, classifyStoreError(err, "employee")
	}
	if err := verifyOrgOwnership(employee, orgID, "employee"); err != nil {
		return nil, err
	}

	contract, err := s.store.Contracts().GetCurrentRecord(ctx, employeeID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch contract")
	}
	if contract == nil {
		return nil, apperror.NotFound("active contract")
	}
	resp := contract.ToResponse()
	return &resp, nil
}

// CreateContract creates a new contract for an employee, validating it belongs to the specified organization
func (s *EmployeeService) CreateContract(ctx context.Context, employeeID, orgID uint, req *models.EmployeeContractCreateRequest) (*models.EmployeeContractResponse, error) {
	// Validate staff category
	if !models.IsValidStaffCategory(req.StaffCategory) {
		return nil, apperror.BadRequest("staff_category must be one of: qualified, supplementary, non_pedagogical")
	}
	if err := validation.ValidatePeriod(req.From, req.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	// Over HTTP the binding tag already rejects an absent field (`required` on a
	// pointer means non-nil), but this method is also called directly by the YAML
	// importer, which never goes through binding.
	if req.WeeklyHours == nil {
		return nil, apperror.BadRequest("weekly_hours is required; send 0 for a contract with no hours")
	}
	if err := validation.ValidateWeeklyHours(*req.WeeklyHours, "weekly_hours"); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	req.Grade = strings.TrimSpace(req.Grade)

	// Verify employee exists and belongs to org (use minimal query - no preloads needed)
	employee, err := s.store.FindByIDMinimal(ctx, employeeID)
	if err != nil {
		return nil, classifyStoreError(err, "employee")
	}
	if err := verifyOrgOwnership(employee, orgID, "employee"); err != nil {
		return nil, err
	}

	// Symmetric to ChildService.CreateContract: contract dates must not predate
	// the employee's birthdate. Without this, seniority calculations downstream
	// can produce centuries-of-tenure nonsense.
	if err := validateContractDatesAfterBirthdate(req.From, req.To, employee.Birthdate); err != nil {
		return nil, err
	}

	// Validate pay plan exists and belongs to same organization
	payPlan, err := s.payPlanStore.FindByID(ctx, req.PayPlanID)
	if err != nil {
		return nil, apperror.BadRequest("payplan_id not found")
	}
	if payPlan.OrganizationID != orgID {
		return nil, apperror.BadRequest("payplan does not belong to this organization")
	}

	// Catch typos / misconfigured (grade, step) at contract creation rather
	// than letting CalculateSalary silently NotFound months later.
	if err := validatePayPlanGradeStepCovers(ctx, s.payPlanStore, req.PayPlanID, req.Grade, req.Step, req.From); err != nil {
		return nil, err
	}

	// Validate section belongs to the same organization
	if err := validateSectionOrg(ctx, s.sectionStore, req.SectionID, orgID); err != nil {
		return nil, err
	}

	contract := &models.EmployeeContract{
		EmployeeID: employeeID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: req.From,
				To:   req.To,
			},
			SectionID:  req.SectionID,
			Properties: req.Properties,
		},
		StaffCategory: req.StaffCategory,
		Grade:         req.Grade,
		Step:          req.Step,
		WeeklyHours:   *req.WeeklyHours,
		PayPlanID:     req.PayPlanID,
	}

	// Validate + create in a single transaction to prevent race conditions.
	// The application-level ValidateNoOverlap is a friendly pre-check; the DB
	// EXCLUDE constraint (DEFERRABLE INITIALLY DEFERRED) is the truthful gate
	// against concurrent inserts that both pass the pre-check, and surfaces as
	// a 23P01 at commit — mapContractDeferredOverlap turns that into 409.
	err = mapContractDeferredOverlap(s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.store.Contracts().ValidateNoOverlap(txCtx, employeeID, req.From, req.To, nil); err != nil {
			return contractOverlapError(err)
		}
		return s.store.CreateContract(txCtx, contract)
	}))
	if err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// DeleteContract removes one contract, leaving a hole in the timeline where it
// was. That is deliberate, and the redesign plan's open question about it is
// settled here: nothing repairs the gap.
//
// A gap is a modelled state rather than damage. The timeline renders it as a
// first-class item with a day count, and staff genuinely do leave and come
// back. Stretching a neighbour to close the hole would invent contract dates
// nobody agreed to, and those dates drive the ISBJ bill.
//
// It is safe to leave because the date lookup is a containment test
// (from <= d AND (to IS NULL OR to >= d)), so a date inside the hole resolves to
// no contract rather than to whichever row happens to be nearest.
// TestChildService_DeleteContract_LeavesGap (the child equivalent) pins all of that.
func (s *EmployeeService) DeleteContract(ctx context.Context, contractID, employeeID, orgID uint, expectedVersion *int64) error {
	// Security: Validate employee belongs to the specified organization (use minimal query - no preloads needed)
	employee, err := s.store.FindByIDMinimal(ctx, employeeID)
	if err != nil {
		return classifyStoreError(err, "employee")
	}
	if err := verifyOrgOwnership(employee, orgID, "employee"); err != nil {
		return err
	}

	// Validate contract belongs to the employee
	contract, err := s.store.FindContractByID(ctx, contractID)
	if err != nil {
		return classifyStoreError(err, "contract")
	}
	if err := verifyRecordOwnership(contract, employeeID, "contract"); err != nil {
		return err
	}

	if err := checkVersion(expectedVersion, contract.Version, "this contract"); err != nil {
		return err
	}

	// The guard closes the window between the check above and the delete itself:
	// if the row moved on in between, it matches nothing rather than destroying
	// an edit the caller never saw.
	if err := s.store.DeleteContract(ctx, contractID, expectedVersion); err != nil {
		if mapped := mapVersionRace(err, expectedVersion != nil); mapped != err {
			return mapped
		}
		return apperror.InternalWrap(err, "failed to delete contract")
	}
	return nil
}

// GetContractByID returns a contract by ID, validating ownership
func (s *EmployeeService) GetContractByID(ctx context.Context, contractID, employeeID, orgID uint) (*models.EmployeeContractResponse, error) {
	// Security: Validate employee belongs to the specified organization (use minimal query - no preloads needed)
	employee, err := s.store.FindByIDMinimal(ctx, employeeID)
	if err != nil {
		return nil, classifyStoreError(err, "employee")
	}
	if err := verifyOrgOwnership(employee, orgID, "employee"); err != nil {
		return nil, err
	}

	// Get contract
	contract, err := s.store.FindContractByID(ctx, contractID)
	if err != nil {
		return nil, classifyStoreError(err, "contract")
	}
	if err := verifyRecordOwnership(contract, employeeID, "contract"); err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}
