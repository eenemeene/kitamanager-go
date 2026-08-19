package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
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
	if err := validateContractDatesAfterBirthdate(req.From, req.To, child.Birthdate); err != nil {
		return nil, err
	}

	// Validate section belongs to the same organization
	if err := validateSectionOrg(ctx, s.sectionStore, req.SectionID, orgID); err != nil {
		return nil, err
	}

	// Merge auto-apply funding properties (e.g. parent meals) into contract
	defaults := s.getAutoApplyProperties(ctx, orgID, req.From)
	properties := req.Properties.MergeDefaults(defaults)

	contract := &models.ChildContract{
		ChildID: childID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: req.From,
				To:   req.To,
			},
			SectionID:  req.SectionID,
			Properties: properties,
		},
	}

	// Validate + create in a single transaction to prevent race conditions.
	// The application-level ValidateNoOverlap is a friendly pre-check; the DB
	// EXCLUDE constraint (DEFERRABLE INITIALLY DEFERRED) is the truthful gate
	// against concurrent inserts that both pass the pre-check, and surfaces as
	// a 23P01 at commit — mapContractDeferredOverlap turns that into 409.
	err = mapContractDeferredOverlap(s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.store.Contracts().ValidateNoOverlap(txCtx, childID, req.From, req.To, nil); err != nil {
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
// first-class item with a day count, and children genuinely do leave and come
// back. Stretching a neighbour to close the hole would invent contract dates
// nobody agreed to, and those dates drive the ISBJ bill.
//
// It is safe to leave because the date lookup is a containment test
// (from <= d AND (to IS NULL OR to >= d)), so a date inside the hole resolves to
// no contract rather than to whichever row happens to be nearest.
// TestChildService_DeleteContract_LeavesGap pins all of that.
func (s *ChildService) DeleteContract(ctx context.Context, contractID, childID, orgID uint, expectedVersion *int64) error {
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
