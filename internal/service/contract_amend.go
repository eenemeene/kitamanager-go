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
		// apperror.ContractConflict carries ErrorCode "contract_overlap", so a
		// client can tell an overlapping period apart from a stale-version
		// conflict. Both are 409; only one is fixed by reloading and retrying.
		return apperror.ContractConflict(err.Error())
	}
	return apperror.InternalWrap(err, "failed to validate contract")
}

// contractVersionConflict maps a version-guarded store update that matched no
// rows onto a 409 the client can act on: reload the contract and reapply.
// Distinct ErrorCode from an overlap, because the remedy is different.
func contractVersionConflict(err error) error {
	if errors.Is(err, store.ErrVersionConflict) {
		return apperror.Conflict("this contract was changed by someone else — reload and try again")
	}
	return err
}

// mapContractDeferredOverlap translates two Postgres race outcomes into
// apperror.Conflict so the user sees a consistent 409 for any concurrent
// "your contract conflicts with someone else's" path:
//
//   - sqlstate 23P01 (exclusion-constraint violation) — the truthful gate
//     against the SELECT-then-INSERT race in PeriodStore.ValidateNoOverlap.
//     With DEFERRABLE INITIALLY DEFERRED it fires at COMMIT, so this must
//     be called on the error returned by transactor.InTransaction (not on
//     errors from inside the closure).
//   - sqlstate 40P01 (deadlock detected) — when N transactions race to
//     insert overlapping rows, PG can detect a lock cycle and pick a
//     victim before the EXCLUDE check fires. The victim's transaction is
//     rolled back with 40P01. Same user-visible meaning as 23P01: another
//     concurrent writer won. Without this mapping the loser surfaces as
//     a 5xx for what is logically the same conflict the next-friendlier
//     race ordering would have produced.
//
// All other errors pass through unchanged so the application-level
// pre-check that returns ErrPeriodOverlap (already mapped to Conflict by
// contractOverlapError) is not double-wrapped.
func mapContractDeferredOverlap(err error) error {
	if err == nil {
		return nil
	}
	if store.IsExclusionViolation(err) || store.IsDeadlock(err) {
		return apperror.ContractConflict("contract dates overlap with an existing contract")
	}
	// A version-guarded update inside the transaction surfaces here; give it the
	// reload-and-retry message rather than letting it fall through as a 500.
	if errors.Is(err, store.ErrVersionConflict) {
		return contractVersionConflict(err)
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
