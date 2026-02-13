package store

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type ChildStore struct {
	db            *gorm.DB
	contractStore *PeriodStore[models.ChildContract]
}

func NewChildStore(db *gorm.DB) *ChildStore {
	return &ChildStore{
		db:            db,
		contractStore: NewPeriodStore[models.ChildContract](db, "child_id"),
	}
}

func (s *ChildStore) FindAll(ctx context.Context, limit, offset int) ([]models.Child, int64, error) {
	var children []models.Child
	var total int64

	if err := DBFromContext(ctx, s.db).Model(&models.Child{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := DBFromContext(ctx, s.db).Limit(limit).Offset(offset).Find(&children).Error; err != nil {
		return nil, 0, err
	}

	return children, total, nil
}

func (s *ChildStore) FindByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.Child, int64, error) {
	return s.FindByOrganizationAndSection(ctx, orgID, nil, nil, nil, "", limit, offset)
}

// applyListFilters adds WHERE/JOIN clauses for the child list filters.
// Returns the modified query and whether DISTINCT is needed (due to JOINs).
func (s *ChildStore) applyListFilters(query *gorm.DB, orgID uint, sectionID *uint, activeOn *time.Time, contractAfter *time.Time, search string) (*gorm.DB, bool) {
	needsDistinct := false
	query = query.Where("children.organization_id = ?", orgID)
	if sectionID != nil {
		query = query.Where("children.section_id = ?", *sectionID)
	}
	if search != "" {
		query = query.Scopes(PersonNameSearch("children", search))
	}
	if activeOn != nil {
		query = query.
			Joins("JOIN child_contracts ON child_contracts.child_id = children.id").
			Scopes(PeriodActiveOn("child_contracts.from_date", "child_contracts.to_date", *activeOn))
		needsDistinct = true
	}
	if contractAfter != nil {
		query = query.
			Joins("JOIN child_contracts ON child_contracts.child_id = children.id").
			Where("child_contracts.from_date > ?", *contractAfter)
		needsDistinct = true
	}
	return query, needsDistinct
}

func (s *ChildStore) FindByOrganizationAndSection(ctx context.Context, orgID uint, sectionID *uint, activeOn *time.Time, contractAfter *time.Time, search string, limit, offset int) ([]models.Child, int64, error) {
	var children []models.Child
	var total int64

	countQuery, needsDistinct := s.applyListFilters(
		DBFromContext(ctx, s.db).Model(&models.Child{}),
		orgID, sectionID, activeOn, contractAfter, search,
	)
	if needsDistinct {
		countQuery = countQuery.Distinct("children.id")
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dataQuery, needsDistinct := s.applyListFilters(
		DBFromContext(ctx, s.db).Preload("Contracts").Preload("Section"),
		orgID, sectionID, activeOn, contractAfter, search,
	)
	if needsDistinct {
		dataQuery = dataQuery.Distinct()
	}
	if err := dataQuery.Limit(limit).Offset(offset).Find(&children).Error; err != nil {
		return nil, 0, err
	}

	return children, total, nil
}

// Contracts returns the contract store for children
func (s *ChildStore) Contracts() ContractStorer[models.ChildContract] {
	return s.contractStore
}

func (s *ChildStore) FindByID(ctx context.Context, id uint) (*models.Child, error) {
	var child models.Child
	if err := DBFromContext(ctx, s.db).Preload("Organization").Preload("Section").Preload("Contracts.Section").Preload("Contracts").First(&child, id).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &child, nil
}

// FindByIDMinimal returns a child without preloading relationships.
// Useful for existence checks and org validation where relationships aren't needed.
func (s *ChildStore) FindByIDMinimal(ctx context.Context, id uint) (*models.Child, error) {
	var child models.Child
	if err := DBFromContext(ctx, s.db).First(&child, id).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &child, nil
}

func (s *ChildStore) Create(ctx context.Context, child *models.Child) error {
	return DBFromContext(ctx, s.db).Create(child).Error
}

func (s *ChildStore) Update(ctx context.Context, child *models.Child) error {
	return DBFromContext(ctx, s.db).Save(child).Error
}

func (s *ChildStore) Delete(ctx context.Context, id uint) error {
	db := DBFromContext(ctx, s.db)
	// Delete all contracts
	if err := db.Where("child_id = ?", id).Delete(&models.ChildContract{}).Error; err != nil {
		return err
	}
	return db.Delete(&models.Child{}, id).Error
}

func (s *ChildStore) CreateContract(ctx context.Context, contract *models.ChildContract) error {
	return DBFromContext(ctx, s.db).Create(contract).Error
}

func (s *ChildStore) FindContractByID(ctx context.Context, id uint) (*models.ChildContract, error) {
	var contract models.ChildContract
	if err := DBFromContext(ctx, s.db).Preload("Section").First(&contract, id).Error; err != nil {
		return nil, err
	}
	return &contract, nil
}

// FindContractsByChildPaginated returns paginated contracts for a child with Section preloaded.
func (s *ChildStore) FindContractsByChildPaginated(ctx context.Context, childID uint, limit, offset int) ([]models.ChildContract, int64, error) {
	var contracts []models.ChildContract
	var total int64

	db := DBFromContext(ctx, s.db)
	if err := db.Model(&models.ChildContract{}).Where("child_id = ?", childID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Preload("Section").Where("child_id = ?", childID).
		Order("from_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&contracts).Error
	return contracts, total, err
}

func (s *ChildStore) UpdateContract(ctx context.Context, contract *models.ChildContract) error {
	return DBFromContext(ctx, s.db).Save(contract).Error
}

func (s *ChildStore) DeleteContract(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.ChildContract{}, id).Error
}

// FindByOrganizationWithActiveOn returns children that have an active contract on the given date.
// A contract is active if: from_date <= date AND (to_date IS NULL OR to_date >= date)
func (s *ChildStore) FindByOrganizationWithActiveOn(ctx context.Context, orgID uint, date time.Time) ([]models.Child, error) {
	var children []models.Child

	// Find children with contracts active on the given date
	if err := DBFromContext(ctx, s.db).
		Preload("Contracts", "from_date <= ? AND (to_date IS NULL OR to_date >= ?)", date, date).
		Joins("JOIN child_contracts ON child_contracts.child_id = children.id").
		Where("children.organization_id = ?", orgID).
		Scopes(PeriodActiveOn("child_contracts.from_date", "child_contracts.to_date", date)).
		Distinct().
		Find(&children).Error; err != nil {
		return nil, err
	}

	return children, nil
}

// FindContractsByOrganizationInDateRange returns all child contracts for an organization
// where the contract overlaps with the given date range.
// This is useful for batch processing to avoid N+1 queries.
func (s *ChildStore) FindContractsByOrganizationInDateRange(ctx context.Context, orgID uint, rangeStart, rangeEnd time.Time) ([]models.ChildContract, error) {
	var contracts []models.ChildContract
	if err := DBFromContext(ctx, s.db).
		Joins("JOIN children ON children.id = child_contracts.child_id").
		Where("children.organization_id = ?", orgID).
		Where("child_contracts.from_date <= ?", rangeEnd).
		Where("child_contracts.to_date IS NULL OR child_contracts.to_date >= ?", rangeStart).
		Find(&contracts).Error; err != nil {
		return nil, err
	}
	return contracts, nil
}

// CountByOrganizationWithActiveOn counts children with active contracts on the given date.
// A contract is active if: from_date <= date AND (to_date IS NULL OR to_date >= date)
func (s *ChildStore) CountByOrganizationWithActiveOn(ctx context.Context, orgID uint, date time.Time) (int64, error) {
	var count int64
	if err := DBFromContext(ctx, s.db).Model(&models.Child{}).
		Joins("JOIN child_contracts ON child_contracts.child_id = children.id").
		Where("children.organization_id = ?", orgID).
		Scopes(PeriodActiveOn("child_contracts.from_date", "child_contracts.to_date", date)).
		Distinct("children.id").
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
