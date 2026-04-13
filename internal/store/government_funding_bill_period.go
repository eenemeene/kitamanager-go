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

func (s *GovernmentFundingBillPeriodStore) FindChildEntriesByOrgAndVoucherNumbers(ctx context.Context, orgID uint, voucherNumbers []string) ([]models.GovernmentFundingBillChildWithPeriod, error) {
	if len(voucherNumbers) == 0 {
		return nil, nil
	}

	// Find matching bill children with period metadata
	var children []models.GovernmentFundingBillChild
	err := DBFromContext(ctx, s.db).
		Preload("Payments").
		Joins("JOIN government_funding_bill_periods p ON p.id = government_funding_bill_children.period_id").
		Where("p.organization_id = ? AND government_funding_bill_children.voucher_number IN ?", orgID, voucherNumbers).
		Order("p.from_date ASC, government_funding_bill_children.id ASC").
		Find(&children).Error
	if err != nil {
		return nil, err
	}

	if len(children) == 0 {
		return nil, nil
	}

	// Collect period IDs to batch-load period metadata
	periodIDs := make(map[uint]bool, len(children))
	for _, c := range children {
		periodIDs[c.PeriodID] = true
	}
	ids := make([]uint, 0, len(periodIDs))
	for id := range periodIDs {
		ids = append(ids, id)
	}

	var periods []models.GovernmentFundingBillPeriod
	if err := DBFromContext(ctx, s.db).Where("id IN ?", ids).Find(&periods).Error; err != nil {
		return nil, err
	}
	periodMap := make(map[uint]*models.GovernmentFundingBillPeriod, len(periods))
	for i := range periods {
		periodMap[periods[i].ID] = &periods[i]
	}

	// Build result
	result := make([]models.GovernmentFundingBillChildWithPeriod, 0, len(children))
	for _, child := range children {
		p := periodMap[child.PeriodID]
		if p == nil {
			continue
		}
		result = append(result, models.GovernmentFundingBillChildWithPeriod{
			BillPeriodID: p.ID,
			BillFrom:     p.From,
			BillTo:       p.To,
			FacilityName: p.FacilityName,
			Child:        child,
		})
	}

	return result, nil
}

// FindBilledTotalsByOrg returns SQL-aggregated billed totals per voucher number for an org.
// Only includes regular payments (not corrections) for accurate comparison.
func (s *GovernmentFundingBillPeriodStore) FindBilledTotalsByOrg(ctx context.Context, orgID uint) ([]models.VoucherBilledTotal, error) {
	var results []models.VoucherBilledTotal
	err := DBFromContext(ctx, s.db).
		Raw(`SELECT c.voucher_number,
				SUM(pay.amount) AS total_billed,
				COUNT(DISTINCT p.id) AS bill_count
			FROM government_funding_bill_periods p
			JOIN government_funding_bill_children c ON c.period_id = p.id
			JOIN government_funding_bill_payments pay ON pay.child_id = c.id
			WHERE p.organization_id = ? AND pay.row_type = ?
			GROUP BY c.voucher_number`, orgID, models.RowTypeRegular).
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

// FindAllBillDatesAndVouchersByOrg returns lightweight voucher + bill date pairs for an org.
// Used for computing expected amounts without loading payment data.
func (s *GovernmentFundingBillPeriodStore) FindAllBillDatesAndVouchersByOrg(ctx context.Context, orgID uint) ([]models.BillDateVoucher, error) {
	var results []models.BillDateVoucher
	err := DBFromContext(ctx, s.db).
		Raw(`SELECT c.voucher_number, p.from_date AS bill_from
			FROM government_funding_bill_periods p
			JOIN government_funding_bill_children c ON c.period_id = p.id
			WHERE p.organization_id = ?
			ORDER BY c.voucher_number, p.from_date`, orgID).
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (s *GovernmentFundingBillPeriodStore) ExistsByOrgAndHash(ctx context.Context, orgID uint, fileHash string) (bool, error) {
	var count int64
	err := DBFromContext(ctx, s.db).
		Model(&models.GovernmentFundingBillPeriod{}).
		Where("organization_id = ? AND file_sha256 = ?", orgID, fileHash).
		Count(&count).Error
	return count > 0, err
}

func (s *GovernmentFundingBillPeriodStore) ExistsByOrgAndMonth(ctx context.Context, orgID uint, from time.Time) (bool, error) {
	var count int64
	err := DBFromContext(ctx, s.db).
		Model(&models.GovernmentFundingBillPeriod{}).
		Where("organization_id = ? AND from_date = ?", orgID, from).
		Count(&count).Error
	return count > 0, err
}

func (s *GovernmentFundingBillPeriodStore) Delete(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.GovernmentFundingBillPeriod{}, id).Error
}
