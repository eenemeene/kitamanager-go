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
	if err := validation.ValidateWeeklyHours(req.WeeklyHours, "weekly_hours"); err != nil {
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

	// Validate pay plan exists and belongs to same organization
	payPlan, err := s.payPlanStore.FindByID(ctx, req.PayPlanID)
	if err != nil {
		return nil, apperror.BadRequest("payplan_id not found")
	}
	if payPlan.OrganizationID != orgID {
		return nil, apperror.BadRequest("payplan does not belong to this organization")
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
		WeeklyHours:   req.WeeklyHours,
		PayPlanID:     req.PayPlanID,
	}

	// Validate + create in a single transaction to prevent race conditions
	if err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.store.Contracts().ValidateNoOverlap(txCtx, employeeID, req.From, req.To, nil); err != nil {
			if errors.Is(err, store.ErrPeriodOverlap) {
				return apperror.Conflict(err.Error())
			}
			return apperror.InternalWrap(err, "failed to validate contract")
		}
		return s.store.CreateContract(txCtx, contract)
	}); err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// BatchUpdateContracts atomically updates multiple contracts for an employee.
// Unlike single UpdateContract, batch updates are always in-place (no amend mode).
func (s *EmployeeService) BatchUpdateContracts(ctx context.Context, employeeID, orgID uint, req *models.EmployeeContractBatchUpdateRequest) ([]models.EmployeeContractResponse, error) {
	// Verify employee exists and belongs to org
	employee, err := s.store.FindByIDMinimal(ctx, employeeID)
	if err != nil {
		return nil, classifyStoreError(err, "employee")
	}
	if err := verifyOrgOwnership(employee, orgID, "employee"); err != nil {
		return nil, err
	}

	// Check for duplicate contract IDs in the batch
	seen := make(map[uint]bool, len(req.Updates))
	for _, entry := range req.Updates {
		if seen[entry.ID] {
			return nil, apperror.BadRequest("duplicate contract ID in batch")
		}
		seen[entry.ID] = true
	}

	// Validate all fields upfront (before transaction)
	for _, entry := range req.Updates {
		if err := validateOptionalSectionOrg(ctx, s.sectionStore, entry.SectionID, orgID); err != nil {
			return nil, err
		}
		if entry.PayPlanID != nil {
			payPlan, err := s.payPlanStore.FindByID(ctx, *entry.PayPlanID)
			if err != nil {
				return nil, apperror.BadRequest("payplan_id not found")
			}
			if payPlan.OrganizationID != orgID {
				return nil, apperror.BadRequest("payplan does not belong to this organization")
			}
		}
		if entry.StaffCategory != nil {
			if !models.IsValidStaffCategory(*entry.StaffCategory) {
				return nil, apperror.BadRequest("staff_category must be one of: qualified, supplementary, non_pedagogical")
			}
		}
		if entry.WeeklyHours != nil {
			if err := validation.ValidateWeeklyHours(*entry.WeeklyHours, "weekly_hours"); err != nil {
				return nil, apperror.BadRequest(err.Error())
			}
		}
	}

	responses := make([]models.EmployeeContractResponse, len(req.Updates))

	if err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		contracts := make([]*models.EmployeeContract, len(req.Updates))

		// Phase 1: fetch, verify, apply fields, validate period, save
		for i, entry := range req.Updates {
			contract, err := s.store.FindContractByID(txCtx, entry.ID)
			if err != nil {
				return classifyStoreError(err, "contract")
			}
			if err := verifyRecordOwnership(contract, employeeID, "contract"); err != nil {
				return err
			}

			applyEmployeeContractFields(contract, &entry.EmployeeContractUpdateRequest)

			if err := validation.ValidatePeriod(contract.From, contract.To); err != nil {
				return apperror.BadRequest(err.Error())
			}

			if err := s.store.UpdateContract(txCtx, contract); err != nil {
				return apperror.InternalWrap(err, "failed to update contract")
			}

			contracts[i] = contract
		}

		// Phase 2: validate no overlaps against final state (all saves done)
		for _, contract := range contracts {
			if err := s.store.Contracts().ValidateNoOverlap(txCtx, employeeID, contract.From, contract.To, &contract.ID); err != nil {
				return contractOverlapError(err)
			}
		}

		for i, contract := range contracts {
			responses[i] = contract.ToResponse()
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return responses, nil
}

// DeleteContract deletes a contract, validating it belongs to an employee in the specified organization
func (s *EmployeeService) DeleteContract(ctx context.Context, contractID, employeeID, orgID uint) error {
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

	if err := s.store.DeleteContract(ctx, contractID); err != nil {
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

// UpdateContract updates an existing contract with amend semantics.
// - Contract started today or in the future -> update in place
// - Contract started before today -> end old contract (yesterday), create new contract (today) with changes
// - Contract already ended -> reject with 400
func (s *EmployeeService) UpdateContract(ctx context.Context, contractID, employeeID, orgID uint, req *models.EmployeeContractUpdateRequest) (*models.EmployeeContractResponse, error) {
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

	// Determine amend mode
	mode, err := determineAmendMode(contract.From, contract.To)
	if err != nil {
		return nil, err
	}

	// Validate pay plan if provided (applies to both modes)
	if req.PayPlanID != nil {
		payPlan, err := s.payPlanStore.FindByID(ctx, *req.PayPlanID)
		if err != nil {
			return nil, apperror.BadRequest("payplan_id not found")
		}
		if payPlan.OrganizationID != orgID {
			return nil, apperror.BadRequest("payplan does not belong to this organization")
		}
	}

	// Validate staff category if provided
	if req.StaffCategory != nil {
		if !models.IsValidStaffCategory(*req.StaffCategory) {
			return nil, apperror.BadRequest("staff_category must be one of: qualified, supplementary, non_pedagogical")
		}
	}

	// Validate weekly hours if provided
	if req.WeeklyHours != nil {
		if err := validation.ValidateWeeklyHours(*req.WeeklyHours, "weekly_hours"); err != nil {
			return nil, apperror.BadRequest(err.Error())
		}
	}

	// Validate section if provided (applies to both modes)
	if err := validateOptionalSectionOrg(ctx, s.sectionStore, req.SectionID, orgID); err != nil {
		return nil, err
	}

	switch mode {
	case amendModeInPlace:
		return s.updateContractInPlace(ctx, contract, employeeID, req)
	case amendModeAmend:
		return s.amendContract(ctx, contract, employeeID, req)
	default:
		return nil, apperror.Internal("unexpected amend mode")
	}
}

// applyEmployeeContractFields applies update request fields to an employee contract.
func applyEmployeeContractFields(contract *models.EmployeeContract, req *models.EmployeeContractUpdateRequest) {
	if req.PayPlanID != nil {
		contract.PayPlanID = *req.PayPlanID
	}
	if req.StaffCategory != nil {
		contract.StaffCategory = *req.StaffCategory
	}
	if req.WeeklyHours != nil {
		contract.WeeklyHours = *req.WeeklyHours
	}
	if req.Grade != nil {
		contract.Grade = strings.TrimSpace(*req.Grade)
	}
	if req.Step != nil {
		contract.Step = *req.Step
	}
	if req.SectionID != nil {
		contract.SectionID = *req.SectionID
		contract.Section = nil
	}
	// Always assign nullable fields so the frontend can clear them by sending null.
	contract.Properties = req.Properties
	if req.From != nil {
		contract.From = *req.From
	}
	contract.To = req.To
}

// updateContractInPlace applies changes directly to the existing employee contract.
func (s *EmployeeService) updateContractInPlace(ctx context.Context, contract *models.EmployeeContract, employeeID uint, req *models.EmployeeContractUpdateRequest) (*models.EmployeeContractResponse, error) {
	applyEmployeeContractFields(contract, req)

	if err := inPlaceContractUpdate(ctx, s.transactor, s.store.Contracts(), employeeID,
		contract.From, contract.To, contract.ID,
		func(txCtx context.Context) error { return s.store.UpdateContract(txCtx, contract) },
	); err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// amendContract closes the old employee contract and creates a new one with changes applied.
func (s *EmployeeService) amendContract(ctx context.Context, contract *models.EmployeeContract, employeeID uint, req *models.EmployeeContractUpdateRequest) (*models.EmployeeContractResponse, error) {
	today := models.TruncateToDate(time.Now())

	// Clone contract with current values, new contract starts today
	newContract := &models.EmployeeContract{
		EmployeeID: contract.EmployeeID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: today,
				To:   contract.To, // inherit original To
			},
			SectionID:  contract.SectionID,
			Properties: contract.Properties,
		},
		StaffCategory: contract.StaffCategory,
		Grade:         contract.Grade,
		Step:          contract.Step,
		WeeklyHours:   contract.WeeklyHours,
		PayPlanID:     contract.PayPlanID,
	}

	// Apply changes (From is ignored -- always today; To is applied if provided)
	if req.SectionID != nil {
		newContract.SectionID = *req.SectionID
	}
	if req.Properties != nil {
		newContract.Properties = req.Properties
	}
	if req.To != nil {
		newContract.To = req.To
	}
	if req.PayPlanID != nil {
		newContract.PayPlanID = *req.PayPlanID
	}
	if req.StaffCategory != nil {
		newContract.StaffCategory = *req.StaffCategory
	}
	if req.WeeklyHours != nil {
		newContract.WeeklyHours = *req.WeeklyHours
	}
	if req.Grade != nil {
		newContract.Grade = strings.TrimSpace(*req.Grade)
	}
	if req.Step != nil {
		newContract.Step = *req.Step
	}

	if err := amendContractTx(ctx, s.transactor, s.store.Contracts(), employeeID,
		newContract.From, newContract.To,
		func(txCtx context.Context, yesterday time.Time) error {
			contract.To = &yesterday
			return s.store.UpdateContract(txCtx, contract)
		},
		func(txCtx context.Context) error { return s.store.CreateContract(txCtx, newContract) },
	); err != nil {
		return nil, err
	}

	resp := newContract.ToResponse()
	return &resp, nil
}
