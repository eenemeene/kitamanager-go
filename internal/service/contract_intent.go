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

// Transaction shapes shared by the intent-based contract operations. The
// child/employee services differ only in which fields they carry, so the
// interesting part — what has to be true at COMMIT — lives here once.
//
// Note that from_date/to_date are Postgres DATE columns: any time-of-day is
// dropped on write. Comparisons here truncate for the same reason, so a request
// carrying a midday timestamp cannot compare differently to what gets stored.

// checkVersion compares a client's If-Match expectation against the contract as
// just loaded, and is the reason optimistic concurrency actually protects
// anything: the version column makes a stale write *fail*, but only once someone
// says which version the write was based on.
//
// Deliberately compared here rather than in the handler. The handler already
// fetches the contract for the audit diff, but comparing there would leave a
// window in which another writer bumps the version between that read and the
// service's own load — the store's `WHERE version = ?` guard would then match the
// newer row and cheerfully overwrite it. Compared against the loaded value, the
// two checks compose: a mismatch here is a 412, and a change landing after the
// load makes the guarded UPDATE match no rows — see mapVersionRace, which also
// reports 412 once a precondition was stated.
//
// A nil expectation means the caller sent If-Match: * , or is not an HTTP caller
// at all (the YAML importer builds these requests in Go).
func checkVersion(expected *int64, current int64, what string) error {
	if expected == nil || *expected == current {
		return nil
	}
	return apperror.PreconditionFailed(
		"%s was changed by someone else (you have version %d, current is %d) — reload and reapply your change",
		what, *expected, current)
}

// mapVersionRace reports a version-guarded write that matched no rows, choosing
// the status by whether the client actually stated a precondition.
//
// Same underlying event — the row moved on between the load and the write — but
// 412 means "the If-Match you sent no longer holds", and answering 412 to a
// request that sent no precondition would be nonsense. Those callers (the old
// PUT paths, the YAML importer) get 409.
func mapVersionRace(err error, hadPrecondition bool) error {
	if !errors.Is(err, store.ErrVersionConflict) {
		return err
	}
	if hadPrecondition {
		return apperror.PreconditionFailed("this contract was changed by someone else since you read it — reload and reapply your change")
	}
	return apperror.Conflict("this contract was changed by someone else — reload and try again")
}

// checkAmendSeam validates that a contract can be amended at the given seam,
// i.e. closed the day before `seam` with a successor starting on `seam`.
//
// Two things have to hold, and neither is obvious from the dates alone:
//
//   - the seam must fall strictly after the contract starts. Amending a contract
//     from its own first day would close it with to = from-1, an empty period.
//     That request is a correction, not an amendment.
//   - the contract must still be running at the seam. If it already ended
//     earlier, closing it at seam-1 would silently *extend* a finished contract
//     to cover months it never covered — and for a child contract those months
//     are billed.
func checkAmendSeam(contractFrom time.Time, contractTo *time.Time, seam time.Time) error {
	from := models.TruncateToDate(contractFrom)
	at := models.TruncateToDate(seam)

	if !at.After(from) {
		return apperror.BadRequest("effective_from must be after the contract's start date; correct the contract in place instead")
	}
	if contractTo != nil {
		to := models.TruncateToDate(*contractTo)
		if to.Before(at) {
			return apperror.BadRequest("the contract already ended before effective_from; amending it would extend it")
		}
	}
	return nil
}

// amendSeam closes the addressed contract the day before `seam` and creates its
// successor starting on `seam`, in one transaction.
//
// `seam` is passed in rather than read from the clock so the predecessor's `to`
// and the successor's `from` cannot drift apart across midnight, and so a
// backdated amendment is one call instead of an amend followed by a drag.
func amendSeam[T models.PeriodRecord](
	ctx context.Context,
	transactor store.Transactor,
	contracts store.PeriodStorer[T],
	ownerID uint,
	seam time.Time,
	newTo *time.Time,
	closeOldFn func(ctx context.Context, dayBefore time.Time) error,
	createNewFn func(ctx context.Context) error,
) error {
	return amendContractTx(ctx, transactor, contracts, ownerID, seam, seam, newTo, closeOldFn, createNewFn)
}

// moveBoundaryTx writes both sides of a moved seam and validates the resulting
// timeline, in one transaction.
//
// The two writes have to happen before the overlap check because a seam move is
// transiently invalid: once the later contract's `from` moves back it overlaps
// the earlier one until the earlier one's `to` follows. That is exactly what the
// DEFERRABLE INITIALLY DEFERRED exclusion constraint (migration 000022) is for —
// only the state at COMMIT has to be legal — so the commit-time 23P01 is mapped
// to a 409 rather than surfacing as a 5xx.
func moveBoundaryTx[T models.PeriodRecord](
	ctx context.Context,
	transactor store.Transactor,
	contracts store.PeriodStorer[T],
	ownerID uint,
	earlier, later periodRef,
	saveFn func(ctx context.Context) error,
) error {
	for _, side := range []periodRef{earlier, later} {
		if err := validation.ValidatePeriod(side.from, side.to); err != nil {
			return apperror.BadRequest(err.Error())
		}
	}

	err := transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := saveFn(txCtx); err != nil {
			return err
		}
		// Against the rest of the timeline: the pair now abuts, so each side is
		// checked with itself excluded and the other included.
		for _, side := range []periodRef{earlier, later} {
			if err := contracts.ValidateNoOverlap(txCtx, ownerID, side.from, side.to, &side.id); err != nil {
				return contractOverlapError(err)
			}
		}
		return nil
	})
	return mapContractDeferredOverlap(err)
}

// periodRef is the post-change period of one contract, used to validate a seam
// move without the helper needing to know the contract's concrete type.
type periodRef struct {
	id   uint
	from time.Time
	to   *time.Time
}

// checkAdjacent verifies the two contracts named in a boundary move really share
// a seam, and that moving it to `at` leaves both sides at least one day long.
//
// Adjacency is required rather than assumed: with a gap between the contracts
// there are two independent boundaries, not one seam, and moving "the" boundary
// would silently swallow the gap. The UI agrees — its timeline renders a
// draggable handle only between adjacent contracts and a static marker for a gap.
func checkAdjacent(earlierFrom time.Time, earlierTo *time.Time, laterFrom time.Time, laterTo *time.Time, at time.Time) error {
	eFrom := models.TruncateToDate(earlierFrom)
	lFrom := models.TruncateToDate(laterFrom)
	seam := models.TruncateToDate(at)

	if !eFrom.Before(lFrom) {
		return apperror.BadRequest("earlier_id must name the contract that starts first")
	}
	if earlierTo == nil {
		// Unreachable while the exclusion constraint holds — an open-ended earlier
		// contract would already overlap the later one — but a clear 400 beats a
		// nil dereference if the constraint is ever relaxed.
		return apperror.BadRequest("the earlier contract has no end date, so the two do not share a boundary")
	}
	if eTo := models.TruncateToDate(*earlierTo); !eTo.AddDate(0, 0, 1).Equal(lFrom) {
		return apperror.BadRequest("the two contracts are not adjacent, so they share no boundary; set each end date instead")
	}

	if !seam.After(eFrom) {
		return apperror.BadRequest("the boundary must leave the earlier contract at least one day")
	}
	if laterTo != nil {
		if lTo := models.TruncateToDate(*laterTo); seam.After(lTo) {
			return apperror.BadRequest("the boundary must leave the later contract at least one day")
		}
	}
	return nil
}
