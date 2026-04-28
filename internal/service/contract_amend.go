package service

import (
	"context"
	"errors"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// amendMode determines how a contract update should be handled.
type amendMode int

const (
	// amendModeInPlace means the contract started today or later — update in place.
	amendModeInPlace amendMode = iota
	// amendModeAmend means the contract started before today — close old + create new.
	amendModeAmend
)

// determineAmendMode decides whether to update in place or amend.
// Returns an error if the contract has already ended (To date is in the past).
func determineAmendMode(contractFrom time.Time, contractTo *time.Time) (amendMode, error) {
	today := models.Today()
	from := models.TruncateToDate(contractFrom)

	// Contract already ended → reject
	if contractTo != nil {
		to := models.TruncateToDate(*contractTo)
		if to.Before(today) {
			return 0, apperror.BadRequest("cannot update a contract that has already ended")
		}
	}

	// Contract starts today or in the future → update in place
	if !from.Before(today) {
		return amendModeInPlace, nil
	}

	// Contract started before today → amend (close old + create new)
	return amendModeAmend, nil
}

// contractOverlapError maps store.ErrPeriodOverlap to apperror.Conflict,
// and wraps everything else as internal.
func contractOverlapError(err error) error {
	if errors.Is(err, store.ErrPeriodOverlap) {
		return apperror.Conflict(err.Error())
	}
	return apperror.InternalWrap(err, "failed to validate contract")
}

// mapContractDeferredOverlap translates a Postgres exclusion-constraint
// violation (sqlstate 23P01) into apperror.Conflict. The exclusion constraint
// is the truthful gate against the SELECT-then-INSERT race in
// PeriodStore.ValidateNoOverlap; with DEFERRABLE INITIALLY DEFERRED it fires
// at COMMIT, so this must be called on the error returned by
// transactor.InTransaction (not on errors from inside the closure).
//
// All other errors pass through unchanged so the application-level
// pre-check that returns ErrPeriodOverlap (already mapped to Conflict by
// contractOverlapError) is not double-wrapped.
func mapContractDeferredOverlap(err error) error {
	if err == nil {
		return nil
	}
	if store.IsExclusionViolation(err) {
		return apperror.Conflict("contract dates overlap with an existing contract")
	}
	return err
}

// inPlaceContractUpdate validates a period and runs overlap validation + update
// inside a single transaction.
func inPlaceContractUpdate[T models.PeriodRecord](
	ctx context.Context,
	transactor store.Transactor,
	contracts store.PeriodStorer[T],
	ownerID uint,
	from time.Time, to *time.Time,
	contractID uint,
	updateFn func(ctx context.Context) error,
) error {
	if err := validation.ValidatePeriod(from, to); err != nil {
		return apperror.BadRequest(err.Error())
	}

	err := transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := contracts.ValidateNoOverlap(txCtx, ownerID, from, to, &contractID); err != nil {
			return contractOverlapError(err)
		}
		return updateFn(txCtx)
	})
	return mapContractDeferredOverlap(err)
}

// amendContractTx closes the old contract (To = yesterday) and creates a new
// one, with overlap validation, all inside a single transaction.
//
// `today` is taken from the caller rather than re-read from time.Now() so that
// the new contract's From and the old contract's To+1 cannot drift apart if the
// request crosses midnight UTC between the caller deciding "today" and this
// helper running. Yesterday is derived from the caller's today for the same
// reason.
func amendContractTx[T models.PeriodRecord](
	ctx context.Context,
	transactor store.Transactor,
	contracts store.PeriodStorer[T],
	ownerID uint,
	today time.Time,
	newFrom time.Time, newTo *time.Time,
	closeOldFn func(ctx context.Context, yesterday time.Time) error,
	createNewFn func(ctx context.Context) error,
) error {
	if err := validation.ValidatePeriod(newFrom, newTo); err != nil {
		return apperror.BadRequest(err.Error())
	}

	yesterday := today.AddDate(0, 0, -1)

	err := transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := closeOldFn(txCtx, yesterday); err != nil {
			return err
		}
		if err := contracts.ValidateNoOverlap(txCtx, ownerID, newFrom, newTo, nil); err != nil {
			return contractOverlapError(err)
		}
		return createNewFn(txCtx)
	})
	return mapContractDeferredOverlap(err)
}
