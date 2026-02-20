package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/isbj"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// GovernmentFundingBillService handles government funding bill file processing.
type GovernmentFundingBillService struct {
	childStore      store.ChildStorer
	billPeriodStore store.GovernmentFundingBillPeriodStorer
}

// NewGovernmentFundingBillService creates a new GovernmentFundingBillService.
func NewGovernmentFundingBillService(childStore store.ChildStorer, billPeriodStore store.GovernmentFundingBillPeriodStorer) *GovernmentFundingBillService {
	return &GovernmentFundingBillService{
		childStore:      childStore,
		billPeriodStore: billPeriodStore,
	}
}

// ProcessISBJ parses an ISBJ Excel file, persists the bill period, and returns enriched data.
func (s *GovernmentFundingBillService) ProcessISBJ(ctx context.Context, orgID uint, reader io.Reader, fileName string, fileHash string, userID uint) (*models.GovernmentFundingBillResponse, error) {
	output, err := isbj.ParseFromReader(reader)
	if err != nil {
		return nil, err
	}

	converted, err := isbj.Convert(output)
	if err != nil {
		return nil, err
	}

	// Build GORM model for persistence
	lastDay := lastDayOfMonth(output.BillingMonth)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID:    orgID,
		Period:            models.Period{From: output.BillingMonth, To: &lastDay},
		FileName:          fileName,
		FileSha256:        fileHash,
		FacilityName:      converted.FacilityName,
		FacilityTotal:     converted.FacilityTotal,
		ContractBooking:   converted.ContractBooking,
		CorrectionBooking: converted.CorrectionBooking,
		CreatedBy:         userID,
	}

	for _, child := range converted.Children {
		billChild := models.GovernmentFundingBillChild{
			VoucherNumber: child.VoucherNumber,
			ChildName:     child.ChildName,
			BirthDate:     child.BirthDate,
			District:      child.District,
		}
		for _, amt := range child.Amounts {
			billChild.Payments = append(billChild.Payments, models.GovernmentFundingBillPayment{
				Key:    amt.Key,
				Value:  amt.Value,
				Amount: amt.Amount,
			})
		}
		period.Children = append(period.Children, billChild)
	}

	if err := s.billPeriodStore.Create(ctx, period); err != nil {
		return nil, fmt.Errorf("persisting bill period: %w", err)
	}

	// Match vouchers and build response
	return s.buildResponse(ctx, orgID, period.ID, converted)
}

// List returns a paginated list of bill periods for an organization.
func (s *GovernmentFundingBillService) List(ctx context.Context, orgID uint, limit, offset int) ([]models.GovernmentFundingBillPeriodListResponse, int64, error) {
	periods, total, err := s.billPeriodStore.FindByOrganization(ctx, orgID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	result := make([]models.GovernmentFundingBillPeriodListResponse, len(periods))
	for i, p := range periods {
		result[i] = models.GovernmentFundingBillPeriodListResponse{
			ID:                p.ID,
			From:              p.From.Format(models.DateFormat),
			To:                formatToDate(p.To),
			FileName:          p.FileName,
			FacilityName:      p.FacilityName,
			FacilityTotal:     p.FacilityTotal,
			ContractBooking:   p.ContractBooking,
			CorrectionBooking: p.CorrectionBooking,
			ChildrenCount:     len(p.Children), // not preloaded in list, will be 0
			CreatedAt:         p.CreatedAt,
		}
	}
	return result, total, nil
}

// GetByID returns a single bill period with enriched children.
func (s *GovernmentFundingBillService) GetByID(ctx context.Context, id, orgID uint) (*models.GovernmentFundingBillPeriodResponse, error) {
	period, err := s.billPeriodStore.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("bill period")
		}
		return nil, apperror.InternalWrap(err, "failed to fetch bill period")
	}
	if period.OrganizationID != orgID {
		return nil, apperror.NotFound("bill period")
	}

	// Collect voucher numbers for matching
	voucherNumbers := make([]string, 0, len(period.Children))
	for _, child := range period.Children {
		voucherNumbers = append(voucherNumbers, child.VoucherNumber)
	}

	contractMap := make(map[string]models.ChildContract)
	if len(voucherNumbers) > 0 {
		contracts, err := s.childStore.FindContractsByVoucherNumbers(ctx, orgID, voucherNumbers)
		if err != nil {
			return nil, err
		}
		for _, c := range contracts {
			if c.VoucherNumber != nil {
				contractMap[*c.VoucherNumber] = c
			}
		}
	}

	// Build enriched children + aggregate surcharges
	matchedCount := 0
	surchargeMap := map[string]int{}
	children := make([]models.GovernmentFundingBillChildResponse, 0, len(period.Children))
	for _, child := range period.Children {
		totalAmount := 0
		amounts := make([]models.GovernmentFundingBillAmount, 0, len(child.Payments))
		for _, p := range child.Payments {
			amounts = append(amounts, models.GovernmentFundingBillAmount{
				Key:    p.Key,
				Value:  p.Value,
				Amount: p.Amount,
			})
			totalAmount += p.Amount

			// Aggregate surcharges (keys defined by ISBJ format)
			for _, sk := range isbj.SurchargeKeys {
				if p.Key == sk {
					surchargeMap[p.Key] += p.Amount
					break
				}
			}
		}

		resp := models.GovernmentFundingBillChildResponse{
			VoucherNumber: child.VoucherNumber,
			ChildName:     child.ChildName,
			BirthDate:     child.BirthDate,
			District:      child.District,
			TotalAmount:   totalAmount,
			Amounts:       amounts,
		}

		if contract, ok := contractMap[child.VoucherNumber]; ok {
			resp.ChildID = &contract.ChildID
			resp.ContractID = &contract.ID
			resp.Matched = true
			matchedCount++
		}

		children = append(children, resp)
	}

	surcharges := make([]models.GovernmentFundingBillAmount, 0, len(isbj.SurchargeKeys))
	for _, sk := range isbj.SurchargeKeys {
		surcharges = append(surcharges, models.GovernmentFundingBillAmount{
			Key: sk, Value: sk, Amount: surchargeMap[sk],
		})
	}

	childrenCount := len(period.Children)
	return &models.GovernmentFundingBillPeriodResponse{
		ID:                period.ID,
		OrganizationID:    period.OrganizationID,
		From:              period.From.Format(models.DateFormat),
		To:                formatToDate(period.To),
		FileName:          period.FileName,
		FileSha256:        period.FileSha256,
		FacilityName:      period.FacilityName,
		FacilityTotal:     period.FacilityTotal,
		ContractBooking:   period.ContractBooking,
		CorrectionBooking: period.CorrectionBooking,
		ChildrenCount:     childrenCount,
		MatchedCount:      matchedCount,
		UnmatchedCount:    childrenCount - matchedCount,
		Surcharges:        surcharges,
		Children:          children,
		CreatedBy:         period.CreatedBy,
		CreatedAt:         period.CreatedAt,
	}, nil
}

// Delete removes a bill period after verifying organization ownership.
func (s *GovernmentFundingBillService) Delete(ctx context.Context, id, orgID uint) (*models.GovernmentFundingBillPeriod, error) {
	period, err := s.billPeriodStore.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("bill period")
		}
		return nil, apperror.InternalWrap(err, "failed to fetch bill period")
	}
	if period.OrganizationID != orgID {
		return nil, apperror.NotFound("bill period")
	}
	if err := s.billPeriodStore.Delete(ctx, id); err != nil {
		return nil, err
	}
	return period, nil
}

// ComputeFileHash computes the SHA-256 hash of the given reader content.
func ComputeFileHash(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("computing file hash: %w", err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (s *GovernmentFundingBillService) buildResponse(ctx context.Context, orgID, periodID uint, converted *isbj.ConvertedSettlement) (*models.GovernmentFundingBillResponse, error) {
	// Collect voucher numbers for matching
	voucherNumbers := make([]string, 0, len(converted.Children))
	for _, child := range converted.Children {
		voucherNumbers = append(voucherNumbers, child.VoucherNumber)
	}

	// Look up contracts by voucher number
	contractMap := make(map[string]models.ChildContract)
	if len(voucherNumbers) > 0 {
		contracts, err := s.childStore.FindContractsByVoucherNumbers(ctx, orgID, voucherNumbers)
		if err != nil {
			return nil, err
		}
		for _, c := range contracts {
			if c.VoucherNumber != nil {
				contractMap[*c.VoucherNumber] = c
			}
		}
	}

	// Build response
	matchedCount := 0
	children := make([]models.GovernmentFundingBillChildResponse, 0, len(converted.Children))
	for _, child := range converted.Children {
		resp := models.GovernmentFundingBillChildResponse{
			VoucherNumber: child.VoucherNumber,
			ChildName:     child.ChildName,
			BirthDate:     child.BirthDate,
			District:      child.District,
			TotalAmount:   child.TotalAmount,
			Amounts:       convertBillAmounts(child.Amounts),
		}

		if contract, ok := contractMap[child.VoucherNumber]; ok {
			resp.ChildID = &contract.ChildID
			resp.ContractID = &contract.ID
			resp.Matched = true
			matchedCount++
		}

		children = append(children, resp)
	}

	return &models.GovernmentFundingBillResponse{
		ID:                periodID,
		FacilityName:      converted.FacilityName,
		FacilityTotal:     converted.FacilityTotal,
		ContractBooking:   converted.ContractBooking,
		CorrectionBooking: converted.CorrectionBooking,
		ChildrenCount:     converted.ChildrenCount,
		MatchedCount:      matchedCount,
		UnmatchedCount:    converted.ChildrenCount - matchedCount,
		Surcharges:        convertBillAmounts(converted.Surcharges),
		Children:          children,
	}, nil
}

func convertBillAmounts(amounts []isbj.SettlementAmount) []models.GovernmentFundingBillAmount {
	result := make([]models.GovernmentFundingBillAmount, len(amounts))
	for i, a := range amounts {
		result[i] = models.GovernmentFundingBillAmount{
			Key:    a.Key,
			Value:  a.Value,
			Amount: a.Amount,
		}
	}
	return result
}

func lastDayOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, time.UTC)
}

func formatToDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(models.DateFormat)
}
