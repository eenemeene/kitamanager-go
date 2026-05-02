package store

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type ChildVoucherStore struct {
	db *gorm.DB
}

func NewChildVoucherStore(db *gorm.DB) *ChildVoucherStore {
	return &ChildVoucherStore{db: db}
}

// FindChildIDsByVoucherNumbers looks up child_vouchers by exact voucher number
// and returns a map of voucher_number → child_id. Only children in the given org are returned.
func (s *ChildVoucherStore) FindChildIDsByVoucherNumbers(ctx context.Context, orgID uint, voucherNumbers []string) (map[string]uint, error) {
	if len(voucherNumbers) == 0 {
		return make(map[string]uint), nil
	}

	type row struct {
		VoucherNumber string `gorm:"column:voucher_number"`
		ChildID       uint   `gorm:"column:child_id"`
	}
	var rows []row
	err := DBFromContext(ctx, s.db).
		Raw(`SELECT cv.voucher_number, cv.child_id
			FROM child_vouchers cv
			JOIN children c ON c.id = cv.child_id
			WHERE c.organization_id = ? AND cv.voucher_number IN ?`,
			orgID, voucherNumbers).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]uint, len(rows))
	for _, r := range rows {
		result[r.VoucherNumber] = r.ChildID
	}
	return result, nil
}

// FindVouchersByChildID returns all vouchers for a child.
func (s *ChildVoucherStore) FindVouchersByChildID(ctx context.Context, childID uint) ([]models.ChildVoucher, error) {
	var vouchers []models.ChildVoucher
	err := DBFromContext(ctx, s.db).
		Where("child_id = ?", childID).
		Order("first_seen ASC").
		Find(&vouchers).Error
	return vouchers, err
}

// FindVouchersByChildIDs returns all vouchers for multiple children.
func (s *ChildVoucherStore) FindVouchersByChildIDs(ctx context.Context, childIDs []uint) ([]models.ChildVoucher, error) {
	if len(childIDs) == 0 {
		return nil, nil
	}
	var vouchers []models.ChildVoucher
	err := DBFromContext(ctx, s.db).
		Where("child_id IN ?", childIDs).
		Find(&vouchers).Error
	return vouchers, err
}

// FindVouchersByOrganization returns all child_vouchers for children in an org.
func (s *ChildVoucherStore) FindVouchersByOrganization(ctx context.Context, orgID uint) ([]models.ChildVoucher, error) {
	var vouchers []models.ChildVoucher
	err := DBFromContext(ctx, s.db).
		Joins("JOIN children c ON c.id = child_vouchers.child_id").
		Where("c.organization_id = ?", orgID).
		Find(&vouchers).Error
	return vouchers, err
}

// CreateVoucher creates a new child_voucher entry. Uses ON CONFLICT DO NOTHING
// so duplicate voucher numbers are silently ignored. Used by the
// auto-discovery path during ISBJ bill upload — see autoDiscoverVouchers.
// Do NOT use from the user-driven AssignVoucher path: it would silently
// swallow the case where the voucher is already on a different child,
// returning 200 to the user while doing nothing. Use CreateVoucherStrict
// in that path so the service can surface a 409 Conflict.
func (s *ChildVoucherStore) CreateVoucher(ctx context.Context, voucher *models.ChildVoucher) error {
	return DBFromContext(ctx, s.db).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(voucher).Error
}

// CreateVoucherStrict creates a new child_voucher entry WITHOUT ON CONFLICT
// handling, wrapped in a SAVEPOINT so a duplicate-key violation does not
// poison the surrounding transaction. A duplicate voucher_number surfaces as
// store.ErrDuplicateKey (translated from PG 23505); after that error fires
// the surrounding transaction is still alive (the savepoint absorbs the
// abort) so the caller can SELECT to disambiguate idempotent self-reassign
// from a real cross-child conflict.
//
// MUST be called from inside a transactor.InTransaction closure — the
// savepoint requires a live transaction. A naked call outside a tx will
// still work on the bare DB (savepoint becomes a no-op via implicit-tx
// rules in PG), but disambiguation via the surrounding tx won't survive.
func (s *ChildVoucherStore) CreateVoucherStrict(ctx context.Context, voucher *models.ChildVoucher) error {
	db := DBFromContext(ctx, s.db)

	// Savepoint name is a constant — PG nests savepoints by name, so
	// callers nesting two CreateVoucherStrict in one tx are fine: the
	// inner SAVEPOINT shadows the outer until released. The outer will
	// have been released at the end of the inner's lifecycle.
	const sp = "child_voucher_strict_insert"
	if err := db.Exec("SAVEPOINT " + sp).Error; err != nil {
		return err
	}

	if err := db.Create(voucher).Error; err != nil {
		// Roll back the failed INSERT but keep the surrounding tx alive
		// so the caller can SELECT to disambiguate. Best-effort: if
		// ROLLBACK TO SAVEPOINT itself fails (DB pool shenanigans),
		// surface the original error so the caller doesn't lose context.
		_ = db.Exec("ROLLBACK TO SAVEPOINT " + sp).Error
		if IsDuplicateKeyError(err) {
			return ErrDuplicateKey
		}
		return err
	}

	// Release the savepoint on success — keeps PG's snapshot tracking
	// tidy on long-running transactions.
	_ = db.Exec("RELEASE SAVEPOINT " + sp).Error
	return nil
}

// FindByVoucherNumber returns the (single) child_voucher row matching the
// given voucher_number. Returns ErrNotFound when no row exists. The
// uniqueness constraint is global, so at most one row matches.
func (s *ChildVoucherStore) FindByVoucherNumber(ctx context.Context, voucherNumber string) (*models.ChildVoucher, error) {
	var v models.ChildVoucher
	if err := DBFromContext(ctx, s.db).
		Where("voucher_number = ?", voucherNumber).
		First(&v).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &v, nil
}

// DeleteVouchersByChild removes all vouchers for a child.
func (s *ChildVoucherStore) DeleteVouchersByChild(ctx context.Context, childID uint) error {
	return DBFromContext(ctx, s.db).
		Where("child_id = ?", childID).
		Delete(&models.ChildVoucher{}).Error
}

// FindActiveContractsByChildIDsAndDate returns the active contract for each child on the given date.
// Returns a map of child_id → ChildContract (at most one per child).
func (s *ChildVoucherStore) FindActiveContractsByChildIDsAndDate(ctx context.Context, orgID uint, childIDs []uint, date time.Time) (map[uint]models.ChildContract, error) {
	if len(childIDs) == 0 {
		return make(map[uint]models.ChildContract), nil
	}

	var contracts []models.ChildContract
	err := DBFromContext(ctx, s.db).
		Joins("JOIN children ON children.id = child_contracts.child_id").
		Where("children.organization_id = ? AND child_contracts.child_id IN ?", orgID, childIDs).
		Scopes(PeriodActiveOn("child_contracts.from_date", "child_contracts.to_date", date)).
		Find(&contracts).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uint]models.ChildContract, len(contracts))
	for _, c := range contracts {
		result[c.ChildID] = c
	}
	return result, nil
}

// FindChildrenWithoutVouchers returns children with active contracts but no voucher entries.
func (s *ChildVoucherStore) FindChildrenWithoutVouchers(ctx context.Context, orgID uint, activeOn time.Time) ([]models.Child, error) {
	var children []models.Child
	err := DBFromContext(ctx, s.db).
		Joins("JOIN child_contracts cc ON cc.child_id = children.id").
		Where("children.organization_id = ?", orgID).
		Where("cc.from_date <= ? AND (cc.to_date IS NULL OR cc.to_date >= ?)", activeOn, activeOn).
		Where("NOT EXISTS (SELECT 1 FROM child_vouchers cv WHERE cv.child_id = children.id)").
		Group("children.id").
		Order("children.last_name, children.first_name").
		Find(&children).Error
	return children, err
}

// FindChildByNameAndBirthMonth finds children by case-insensitive name and birth month/year.
func (s *ChildVoucherStore) FindChildByNameAndBirthMonth(ctx context.Context, orgID uint, firstName, lastName string, birthMonth time.Month, birthYear int) ([]models.Child, error) {
	var children []models.Child
	err := DBFromContext(ctx, s.db).
		Where("organization_id = ? AND LOWER(first_name) = LOWER(?) AND LOWER(last_name) = LOWER(?) AND EXTRACT(MONTH FROM birthdate) = ? AND EXTRACT(YEAR FROM birthdate) = ?",
			orgID, firstName, lastName, int(birthMonth), birthYear).
		Find(&children).Error
	return children, err
}
