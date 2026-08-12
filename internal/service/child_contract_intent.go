package service

import (
	"context"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// Intent-based child contract operations: correct, amend, end, move boundary.
//
// Unlike UpdateContract these never infer what the caller meant from stored
// dates — determineAmendMode has no counterpart here. A correction is a
// correction whether the contract started last year or next week, and an
// amendment states its own effective date.

// loadChildContract fetches a contract and proves the caller may touch it:
// the child must belong to the organization in the URL and the contract must
// belong to that child. Every intent operation needs all three, and the child
// is returned because its birthdate bounds the contract's dates.
func (s *ChildService) loadChildContract(ctx context.Context, contractID, childID, orgID uint) (*models.Child, *models.ChildContract, error) {
	child, err := s.store.FindByIDMinimal(ctx, childID)
	if err != nil {
		return nil, nil, classifyStoreError(err, "child")
	}
	if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
		return nil, nil, err
	}

	contract, err := s.store.FindContractByID(ctx, contractID)
	if err != nil {
		return nil, nil, classifyStoreError(err, "contract")
	}
	if err := verifyRecordOwnership(contract, childID, "contract"); err != nil {
		return nil, nil, err
	}
	return child, contract, nil
}

// CorrectContract fixes a child contract's recorded facts in place, including on
// contracts that started or ended in the past — correcting history is the point.
// The per-field audit diff is what makes that safe to offer.
//
// Omitted fields are left untouched. That is the guarantee the old PUT could not
// make: there, omitting `to` or `properties` cleared them.
func (s *ChildService) CorrectContract(ctx context.Context, contractID, childID, orgID uint, req *models.ChildContractCorrectRequest) (*models.ChildContractResponse, error) {
	child, contract, err := s.loadChildContract(ctx, contractID, childID, orgID)
	if err != nil {
		return nil, err
	}

	if req.From.IsNull() {
		return nil, apperror.BadRequest("from cannot be null: every contract has a start date")
	}
	if req.SectionID.IsNull() {
		return nil, apperror.BadRequest("section_id cannot be null: every contract belongs to a section")
	}

	if sectionID, ok := req.SectionID.Get(); ok {
		if err := validateSectionOrg(ctx, s.sectionStore, sectionID, orgID); err != nil {
			return nil, err
		}
		contract.SectionID = sectionID
		contract.Section = nil
	}
	if from, ok := req.From.Get(); ok {
		contract.From = from
	}
	if req.To.Set {
		contract.To = req.To.Value
	}
	if req.Properties.Set {
		// An explicit null clears the map; an object replaces it wholesale, which
		// is what the funding editor sends. Auto-apply defaults are re-merged
		// below only because the request touched properties or the start date.
		if props, ok := req.Properties.Get(); ok {
			contract.Properties = props
		} else {
			contract.Properties = nil
		}
	}

	// Auto-applied funding properties (e.g. parent meals) are anchored at the
	// contract's start, so they are re-merged when properties or `from` move —
	// and deliberately not when neither did, because silently adding funding keys
	// to a contract whose section was corrected would break the promise that an
	// omitted field is untouched.
	if req.Properties.Set || req.From.Set {
		contract.Properties = contract.Properties.MergeDefaults(s.getAutoApplyProperties(ctx, orgID, contract.From))
	}

	if err := validateContractDatesAfterBirthdate(contract.From, contract.To, child.Birthdate); err != nil {
		return nil, err
	}

	if err := inPlaceContractUpdate(ctx, s.transactor, s.store.Contracts(), childID,
		contract.From, contract.To, contract.ID,
		func(txCtx context.Context) error { return s.store.UpdateContract(txCtx, contract) },
	); err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// AmendContract records a change to a child's contract effective from a date:
// the addressed contract is closed the day before, and a successor carrying the
// changes starts on that date. Both are returned.
//
// Fields the request omits inherit from the contract being amended, which is the
// common case — a new Bescheid usually changes the care type and nothing else.
func (s *ChildService) AmendContract(ctx context.Context, contractID, childID, orgID uint, req *models.ChildContractAmendRequest) (*models.ChildContractAmendResponse, error) {
	child, contract, err := s.loadChildContract(ctx, contractID, childID, orgID)
	if err != nil {
		return nil, err
	}

	seam := models.TruncateToDate(req.EffectiveFrom)
	if err := checkAmendSeam(contract.From, contract.To, seam); err != nil {
		return nil, err
	}

	if req.SectionID.IsNull() {
		return nil, apperror.BadRequest("section_id cannot be null: every contract belongs to a section")
	}

	successor := &models.ChildContract{
		ChildID: contract.ChildID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: seam, To: contract.To},
			SectionID:  contract.SectionID,
			Properties: contract.Properties,
		},
	}

	if sectionID, ok := req.SectionID.Get(); ok {
		if err := validateSectionOrg(ctx, s.sectionStore, sectionID, orgID); err != nil {
			return nil, err
		}
		successor.SectionID = sectionID
	}
	if req.To.Set {
		successor.To = req.To.Value
	}
	if req.Properties.Set {
		if props, ok := req.Properties.Get(); ok {
			successor.Properties = props
		} else {
			successor.Properties = nil
		}
	}

	// Anchored at the seam, not at today: a change backdated across a funding
	// period boundary has to pick up the properties that applied back then.
	successor.Properties = successor.Properties.MergeDefaults(s.getAutoApplyProperties(ctx, orgID, seam))

	if err := validateContractDatesAfterBirthdate(successor.From, successor.To, child.Birthdate); err != nil {
		return nil, err
	}

	// Snapshot for the response: the predecessor object is mutated in place by
	// the transaction below, so the pair reported back reflects the final state.
	if err := amendSeam(ctx, s.transactor, s.store.Contracts(), childID,
		seam, successor.To,
		func(txCtx context.Context, dayBefore time.Time) error {
			contract.To = &dayBefore
			return s.store.UpdateContract(txCtx, contract)
		},
		func(txCtx context.Context) error { return s.store.CreateContract(txCtx, successor) },
	); err != nil {
		return nil, err
	}

	return &models.ChildContractAmendResponse{
		Closed:  contract.ToResponse(),
		Created: successor.ToResponse(),
	}, nil
}

// EndContract sets or clears a child contract's end date — a departure, or
// undoing one. `to` must be present: a null reopens the contract as ongoing,
// which is a decision, not an omission.
func (s *ChildService) EndContract(ctx context.Context, contractID, childID, orgID uint, req *models.ContractEndRequest) (*models.ChildContractResponse, error) {
	child, contract, err := s.loadChildContract(ctx, contractID, childID, orgID)
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
	if err := validateContractDatesAfterBirthdate(contract.From, contract.To, child.Birthdate); err != nil {
		return nil, err
	}

	// Reopening can collide with a successor that already exists — that is a 409,
	// not a silent overwrite.
	if err := inPlaceContractUpdate(ctx, s.transactor, s.store.Contracts(), childID,
		contract.From, contract.To, contract.ID,
		func(txCtx context.Context) error { return s.store.UpdateContract(txCtx, contract) },
	); err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// MoveContractBoundary moves the seam between two adjacent child contracts: the
// later one starts on `at`, the earlier one ends the day before.
//
// One date, both sides derived here. The client used to compute four dates and
// send them as a batch, and got it wrong twice — clearing the neighbour's `to`,
// then wiping its properties along with the care type that funds it.
func (s *ChildService) MoveContractBoundary(ctx context.Context, childID, orgID uint, req *models.ContractBoundaryMoveRequest) (*models.ChildContractBoundaryResponse, error) {
	if req.EarlierID == req.LaterID {
		return nil, apperror.BadRequest("earlier_id and later_id must name different contracts")
	}

	child, earlier, err := s.loadChildContract(ctx, req.EarlierID, childID, orgID)
	if err != nil {
		return nil, err
	}
	_, later, err := s.loadChildContract(ctx, req.LaterID, childID, orgID)
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

	if err := validateContractDatesAfterBirthdate(later.From, later.To, child.Birthdate); err != nil {
		return nil, err
	}

	if err := moveBoundaryTx(ctx, s.transactor, s.store.Contracts(), childID,
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

	return &models.ChildContractBoundaryResponse{
		Earlier: earlier.ToResponse(),
		Later:   later.ToResponse(),
	}, nil
}
