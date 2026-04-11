package store

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type GovernmentFundingBillPeriodStore struct {
	db *gorm.DB
}

func NewGovernmentFundingBillPeriodStore(db *gorm.DB) *GovernmentFundingBillPeriodStore {
	return &GovernmentFundingBillPeriodStore{db: db}
}

func (s *GovernmentFundingBillPeriodStore) Create(ctx context.Context, period *models.GovernmentFundingBillPeriod) error {
	return DBFromContext(ctx, s.db).Create(period).Error
}

func (s *GovernmentFundingBillPeriodStore) FindByID(ctx context.Context, id uint) (*models.GovernmentFundingBillPeriod, error) {
	var period models.GovernmentFundingBillPeriod
	if err := DBFromContext(ctx, s.db).
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).
		Preload("Children.Payments").
		First(&period, id).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &period, nil
}

func (s *GovernmentFundingBillPeriodStore) FindByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.GovernmentFundingBillPeriod, int64, error) {
	var periods []models.GovernmentFundingBillPeriod
	var total int64

	db := DBFromContext(ctx, s.db).Where("organization_id = ?", orgID)

	if err := db.Model(&models.GovernmentFundingBillPeriod{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Order("from_date DESC").Limit(limit).Offset(offset).Find(&periods).Error; err != nil {
		return nil, 0, err
	}

	return periods, total, nil
}

func (s *GovernmentFundingBillPeriodStore) FindByOrganizationAndVoucherNumber(ctx context.Context, orgID uint, voucherNumber string) ([]models.BillAppearance, error) {
	type row struct {
		BillID       uint      `gorm:"column:bill_id"`
		BillFrom     time.Time `gorm:"column:bill_from"`
		FacilityName string    `gorm:"column:facility_name"`
	}
	var rows []row
	err := DBFromContext(ctx, s.db).
		Raw(`SELECT DISTINCT p.id AS bill_id, p.from_date AS bill_from, p.facility_name
			FROM government_funding_bill_periods p
			JOIN government_funding_bill_children c ON c.period_id = p.id
			WHERE p.organization_id = ? AND c.voucher_number = ?
			ORDER BY p.from_date`, orgID, voucherNumber).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	results := make([]models.BillAppearance, len(rows))
	for i, r := range rows {
		results[i] = models.BillAppearance{
			BillID:       r.BillID,
			BillFrom:     r.BillFrom.Format(models.DateFormat),
			FacilityName: r.FacilityName,
		}
	}
	return results, nil
}

func (s *GovernmentFundingBillPeriodStore) FindFacilityTotalsByOrganizationInDateRange(ctx context.Context, orgID uint, from, to time.Time) (map[time.Time]int, error) {
	var results []struct {
		FromDate      time.Time `gorm:"column:from_date"`
		FacilityTotal int       `gorm:"column:facility_total"`
	}
	err := DBFromContext(ctx, s.db).
		Model(&models.GovernmentFundingBillPeriod{}).
		Select("from_date, facility_total").
		Where("organization_id = ? AND from_date >= ? AND from_date <= ?", orgID, from, to).
		Order("from_date ASC").
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	m := make(map[time.Time]int, len(results))
	for _, r := range results {
		// Normalize to first of month
		key := time.Date(r.FromDate.Year(), r.FromDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		m[key] += r.FacilityTotal // sum if multiple bills for same month
	}
	return m, nil
}

func (s *GovernmentFundingBillPeriodStore) Delete(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.GovernmentFundingBillPeriod{}, id).Error
}
