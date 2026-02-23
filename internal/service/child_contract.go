package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// ListContracts returns paginated contract history for a child, validating it belongs to the specified organization
func (s *ChildService) ListContracts(ctx context.Context, childID, orgID uint, limit, offset int) ([]models.ChildContractResponse, int64, error) {
	// Verify child exists and belongs to org (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(ctx, childID)
	if err != nil {
		return nil, 0, classifyStoreError(err, "child")
	}
	if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
		return nil, 0, err
	}

	contracts, total, err := s.store.FindContractsByChildPaginated(ctx, childID, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch contracts")
	}

	return toResponseList(contracts, (*models.ChildContract).ToResponse), total, nil
}

// GetCurrentRecord returns the current active contract for a child, validating it belongs to the specified organization
func (s *ChildService) GetCurrentRecord(ctx context.Context, childID, orgID uint) (*models.ChildContractResponse, error) {
	// Security: Validate child belongs to the specified organization (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(ctx, childID)
	if err != nil {
		return nil, classifyStoreError(err, "child")
	}
	if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
		return nil, err
	}

	contract, err := s.store.Contracts().GetCurrentRecord(ctx, childID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch contract")
	}
	if contract == nil {
		return nil, apperror.NotFound("active contract")
	}
	resp := contract.ToResponse()
	return &resp, nil
}

// GetContractByID returns a contract by ID, validating ownership
func (s *ChildService) GetContractByID(ctx context.Context, contractID, childID, orgID uint) (*models.ChildContractResponse, error) {
	// Security: Validate child belongs to the specified organization (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(ctx, childID)
	if err != nil {
		return nil, classifyStoreError(err, "child")
	}
	if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
		return nil, err
	}

	// Get contract
	contract, err := s.store.FindContractByID(ctx, contractID)
	if err != nil {
		return nil, classifyStoreError(err, "contract")
	}
	if err := verifyRecordOwnership(contract, childID, "contract"); err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// CreateContract creates a new contract for a child, validating it belongs to the specified organization.
// The overlap validation and contract creation run in a single transaction.
func (s *ChildService) CreateContract(ctx context.Context, childID, orgID uint, req *models.ChildContractCreateRequest) (*models.ChildContractResponse, error) {
	// Validate period
	if err := validation.ValidatePeriod(req.From, req.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	// Verify child exists and belongs to org (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(ctx, childID)
	if err != nil {
		return nil, classifyStoreError(err, "child")
	}
	if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
		return nil, err
	}

	// Validate contract dates are not before child's birthdate
	if req.From.Before(child.Birthdate) {
		return nil, apperror.BadRequest("contract start date cannot be before child's birthdate")
	}
	if req.To != nil && req.To.Before(child.Birthdate) {
		return nil, apperror.BadRequest("contract end date cannot be before child's birthdate")
	}

	// Validate section belongs to the same organization
	if err := validateSectionOrg(ctx, s.sectionStore, req.SectionID, orgID); err != nil {
		return nil, err
	}

	// Merge auto-apply funding properties (e.g. parent meals) into contract
	defaults := s.getAutoApplyProperties(ctx, orgID, req.From)
	properties := req.Properties.MergeDefaults(defaults)

	contract := &models.ChildContract{
		ChildID:       childID,
		VoucherNumber: req.VoucherNumber,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: req.From,
				To:   req.To,
			},
			SectionID:  req.SectionID,
			Properties: properties,
		},
	}

	// Validate + create in a single transaction to prevent race conditions
	if err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.store.Contracts().ValidateNoOverlap(txCtx, childID, req.From, req.To, nil); err != nil {
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

// UpdateContract updates an existing contract with amend semantics.
// - Contract started today or in the future -> update in place
// - Contract started before today -> end old contract (yesterday), create new contract (today) with changes
// - Contract already ended -> reject with 400
func (s *ChildService) UpdateContract(ctx context.Context, contractID, childID, orgID uint, req *models.ChildContractUpdateRequest) (*models.ChildContractResponse, error) {
	// Security: Validate child belongs to the specified organization (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(ctx, childID)
	if err != nil {
		return nil, classifyStoreError(err, "child")
	}
	if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
		return nil, err
	}

	// Validate contract belongs to the child
	contract, err := s.store.FindContractByID(ctx, contractID)
	if err != nil {
		return nil, classifyStoreError(err, "contract")
	}
	if err := verifyRecordOwnership(contract, childID, "contract"); err != nil {
		return nil, err
	}

	// Determine amend mode
	mode, err := determineAmendMode(contract.From, contract.To)
	if err != nil {
		return nil, err
	}

	// Validate section if provided (applies to both modes)
	if err := validateOptionalSectionOrg(ctx, s.sectionStore, req.SectionID, orgID); err != nil {
		return nil, err
	}

	switch mode {
	case amendModeInPlace:
		return s.updateContractInPlace(ctx, contract, childID, orgID, req)
	case amendModeAmend:
		return s.amendContract(ctx, contract, childID, orgID, req)
	default:
		return nil, apperror.Internal("unexpected amend mode")
	}
}

// updateContractInPlace applies changes directly to the existing contract.
func (s *ChildService) updateContractInPlace(ctx context.Context, contract *models.ChildContract, childID, orgID uint, req *models.ChildContractUpdateRequest) (*models.ChildContractResponse, error) {
	if req.From != nil {
		contract.From = *req.From
	}
	// Always assign nullable fields so the frontend can clear them by sending null.
	contract.To = req.To
	if req.SectionID != nil {
		contract.SectionID = *req.SectionID
		contract.Section = nil
	}
	// Merge auto-apply funding properties into updated contract
	defaults := s.getAutoApplyProperties(ctx, orgID, contract.From)
	contract.Properties = req.Properties.MergeDefaults(defaults)
	contract.VoucherNumber = req.VoucherNumber

	if err := validation.ValidatePeriod(contract.From, contract.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	contractID := contract.ID
	if err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.store.Contracts().ValidateNoOverlap(txCtx, childID, contract.From, contract.To, &contractID); err != nil {
			if errors.Is(err, store.ErrPeriodOverlap) {
				return apperror.Conflict(err.Error())
			}
			return apperror.InternalWrap(err, "failed to validate contract")
		}
		return s.store.UpdateContract(txCtx, contract)
	}); err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// amendContract closes the old contract and creates a new one with changes applied.
func (s *ChildService) amendContract(ctx context.Context, contract *models.ChildContract, childID, orgID uint, req *models.ChildContractUpdateRequest) (*models.ChildContractResponse, error) {
	today := models.TruncateToDate(time.Now())
	yesterday := today.AddDate(0, 0, -1)

	// Clone contract with current values, new contract starts today
	newContract := &models.ChildContract{
		ChildID:       contract.ChildID,
		VoucherNumber: contract.VoucherNumber,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: today,
				To:   contract.To, // inherit original To
			},
			SectionID:  contract.SectionID,
			Properties: contract.Properties,
		},
	}

	// Apply changes (From is ignored -- always today; To is applied if provided)
	if req.SectionID != nil {
		newContract.SectionID = *req.SectionID
	}
	if req.Properties != nil {
		newContract.Properties = req.Properties
	}
	if req.VoucherNumber != nil {
		newContract.VoucherNumber = req.VoucherNumber
	}
	if req.To != nil {
		newContract.To = req.To
	}

	// Merge auto-apply funding properties into amended contract
	defaults := s.getAutoApplyProperties(ctx, orgID, newContract.From)
	newContract.Properties = newContract.Properties.MergeDefaults(defaults)

	if err := validation.ValidatePeriod(newContract.From, newContract.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	// Transaction: close old contract + validate overlap + create new
	if err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		// Close old contract (end = yesterday)
		contract.To = &yesterday
		if err := s.store.UpdateContract(txCtx, contract); err != nil {
			return apperror.InternalWrap(err, "failed to close old contract")
		}

		// Validate new contract doesn't overlap with any other contracts
		if err := s.store.Contracts().ValidateNoOverlap(txCtx, childID, newContract.From, newContract.To, nil); err != nil {
			if errors.Is(err, store.ErrPeriodOverlap) {
				return apperror.Conflict(err.Error())
			}
			return apperror.InternalWrap(err, "failed to validate contract")
		}

		// Create new contract
		return s.store.CreateContract(txCtx, newContract)
	}); err != nil {
		return nil, err
	}

	resp := newContract.ToResponse()
	return &resp, nil
}

// DeleteContract deletes a contract, validating it belongs to a child in the specified organization
func (s *ChildService) DeleteContract(ctx context.Context, contractID, childID, orgID uint) error {
	// Security: Validate child belongs to the specified organization (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(ctx, childID)
	if err != nil {
		return classifyStoreError(err, "child")
	}
	if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
		return err
	}

	// Validate contract belongs to the child
	contract, err := s.store.FindContractByID(ctx, contractID)
	if err != nil {
		return classifyStoreError(err, "contract")
	}
	if err := verifyRecordOwnership(contract, childID, "contract"); err != nil {
		return err
	}

	if err := s.store.DeleteContract(ctx, contractID); err != nil {
		return apperror.InternalWrap(err, "failed to delete contract")
	}
	return nil
}

// getAutoApplyProperties returns properties marked with ApplyToAllContracts from the
// government funding period active on the given date for the organization's state.
// These are merged into every child contract so that universal funding items (e.g. meals)
// are always included in funding calculations without manual selection.
func (s *ChildService) getAutoApplyProperties(ctx context.Context, orgID uint, date time.Time) models.ContractProperties {
	org, err := s.orgStore.FindByID(ctx, orgID)
	if err != nil || org.State == "" {
		return nil
	}

	funding, err := s.fundingStore.FindByStateWithDetails(ctx, org.State, 0, nil)
	if err != nil {
		return nil
	}

	period := findPeriodForDate(funding.Periods, date)
	if period == nil {
		return nil
	}

	defaults := make(models.ContractProperties)
	for _, prop := range period.Properties {
		if prop.ApplyToAllContracts {
			defaults[prop.Key] = prop.Value
		}
	}
	if len(defaults) == 0 {
		return nil
	}

	slog.Debug("auto-apply properties", "orgID", orgID, "date", date, "defaults", defaults)
	return defaults
}
