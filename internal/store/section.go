package store

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type SectionStore struct {
	db *gorm.DB
}

func NewSectionStore(db *gorm.DB) *SectionStore {
	return &SectionStore{db: db}
}

func (s *SectionStore) FindByID(ctx context.Context, id uint) (*models.Section, error) {
	var section models.Section
	if err := DBFromContext(ctx, s.db).Preload("Organization").First(&section, id).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &section, nil
}

// FindByIDsAndOrg returns the subset of `ids` that exist AND belong to
// orgID, keyed by id for O(1) presence checks at the call site. Used by
// the forecast validator to check N section references in one round
// trip instead of N. Missing/wrong-org IDs are simply absent from the
// result map — the caller turns absence into the appropriate error.
func (s *SectionStore) FindByIDsAndOrg(ctx context.Context, ids []uint, orgID uint) (map[uint]*models.Section, error) {
	out := make(map[uint]*models.Section, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []models.Section
	if err := DBFromContext(ctx, s.db).
		Where("id IN ? AND organization_id = ?", ids, orgID).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		out[rows[i].ID] = &rows[i]
	}
	return out, nil
}

func (s *SectionStore) FindByOrganizationPaginated(ctx context.Context, orgID uint, search string, limit, offset int) ([]models.Section, int64, error) {
	var sections []models.Section
	var total int64

	query := DBFromContext(ctx, s.db).Model(&models.Section{}).Where("organization_id = ?", orgID)
	if search != "" {
		query = query.Scopes(NameSearch("sections", "name", search))
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dataQuery := DBFromContext(ctx, s.db).Preload("Organization").Where("organization_id = ?", orgID)
	if search != "" {
		dataQuery = dataQuery.Scopes(NameSearch("sections", "name", search))
	}

	if err := dataQuery.Order("COALESCE(min_age_months, 999) ASC, name ASC").Limit(limit).Offset(offset).Find(&sections).Error; err != nil {
		return nil, 0, err
	}

	return sections, total, nil
}

func (s *SectionStore) FindDefaultSection(ctx context.Context, orgID uint) (*models.Section, error) {
	var section models.Section
	if err := DBFromContext(ctx, s.db).Where("organization_id = ? AND is_default = ?", orgID, true).First(&section).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &section, nil
}

func (s *SectionStore) FindByNameAndOrg(ctx context.Context, name string, orgID uint) (*models.Section, error) {
	var section models.Section
	if err := DBFromContext(ctx, s.db).Where("organization_id = ? AND name = ?", orgID, name).First(&section).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &section, nil
}

func (s *SectionStore) Create(ctx context.Context, section *models.Section) error {
	return DBFromContext(ctx, s.db).Create(section).Error
}

func (s *SectionStore) Update(ctx context.Context, section *models.Section) error {
	return DBFromContext(ctx, s.db).Save(section).Error
}

// PromoteToDefault flips is_default so that exactly the given section
// is the org's default. Implemented as TWO statements inside a single
// transaction (the caller wraps the call) rather than one UPDATE with
// a CASE expression — Postgres checks NOT DEFERRABLE unique
// constraints immediately per row, so a single UPDATE that sets the
// NEW default first and the OLD default second would transiently
// violate the partial-unique index from migration 000019. Splitting
// into "clear all firsts, then set the new one" guarantees the
// invariant at every statement boundary.
//
// Caller must wrap in a transaction (Service does).
func (s *SectionStore) PromoteToDefault(ctx context.Context, id, orgID uint) error {
	db := DBFromContext(ctx, s.db)
	// Step 1: clear any existing default in this org. is_default rows
	// for soft-deleted sections are also cleared (no harm — they're
	// invisible everywhere else).
	if err := db.Model(&models.Section{}).
		Where("organization_id = ? AND is_default = ?", orgID, true).
		Update("is_default", false).Error; err != nil {
		return err
	}
	// Step 2: promote the chosen section. Use UpdateColumn to
	// bypass GORM's auto-fill of UpdatedAt — promotion shouldn't
	// touch other audit fields.
	res := db.Model(&models.Section{}).
		Where("id = ? AND organization_id = ?", id, orgID).
		Update("is_default", true)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		// Section vanished between the caller's existence check and
		// this update — surface as a not-found-style error so the
		// caller's friendly message stays consistent.
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *SectionStore) Delete(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.Section{}, id).Error
}

// HasActiveChildren reports whether any child_contract that is active
// on `asOf` references this section. Two semantic improvements over
// the previous HasChildren:
//
//   - Time-filtered. The previous query counted EVERY contract,
//     including ENDED ones from years ago. That left orgs with
//     "zombie" sections that could never be deleted.
//   - EXISTS-shaped. `LIMIT 1` short-circuits as soon as one row is
//     found, so even a section with thousands of historical
//     assignments answers in milliseconds rather than scanning the
//     full match set.
//
// The active predicate is `from_date <= asOf AND (to_date IS NULL OR
// to_date >= asOf)` — the same shape PeriodStore.HasActiveRecord
// uses elsewhere in the codebase.
func (s *SectionStore) HasActiveChildren(ctx context.Context, id uint, asOf time.Time) (bool, error) {
	var oneID uint
	err := DBFromContext(ctx, s.db).
		Model(&models.ChildContract{}).
		Select("id").
		Where("section_id = ? AND from_date <= ? AND (to_date IS NULL OR to_date >= ?)",
			id, asOf, asOf).
		Limit(1).
		Scan(&oneID).Error
	if err != nil {
		return false, err
	}
	return oneID != 0, nil
}

// HasActiveEmployees is the employee-side counterpart. See
// HasActiveChildren for the rationale.
func (s *SectionStore) HasActiveEmployees(ctx context.Context, id uint, asOf time.Time) (bool, error) {
	var oneID uint
	err := DBFromContext(ctx, s.db).
		Model(&models.EmployeeContract{}).
		Select("id").
		Where("section_id = ? AND from_date <= ? AND (to_date IS NULL OR to_date >= ?)",
			id, asOf, asOf).
		Limit(1).
		Scan(&oneID).Error
	if err != nil {
		return false, err
	}
	return oneID != 0, nil
}

// CountActiveChildren returns how many child_contracts active on
// `asOf` reference this section. Used by the delete-rejection path
// to put a number in the error message ("cannot delete: 5
// currently-assigned children"). Slower than HasActiveChildren
// because it can't short-circuit, so reserved for the
// already-rejected case where the user is about to see an error
// anyway.
func (s *SectionStore) CountActiveChildren(ctx context.Context, id uint, asOf time.Time) (int64, error) {
	var n int64
	err := DBFromContext(ctx, s.db).
		Model(&models.ChildContract{}).
		Where("section_id = ? AND from_date <= ? AND (to_date IS NULL OR to_date >= ?)",
			id, asOf, asOf).
		Count(&n).Error
	return n, err
}

// CountActiveEmployees is the employee-side counterpart of
// CountActiveChildren.
func (s *SectionStore) CountActiveEmployees(ctx context.Context, id uint, asOf time.Time) (int64, error) {
	var n int64
	err := DBFromContext(ctx, s.db).
		Model(&models.EmployeeContract{}).
		Where("section_id = ? AND from_date <= ? AND (to_date IS NULL OR to_date >= ?)",
			id, asOf, asOf).
		Count(&n).Error
	return n, err
}
