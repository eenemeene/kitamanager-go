package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/isbj"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// GovernmentFundingBillService handles government funding bill file processing.
type GovernmentFundingBillService struct {
	childStore      store.ChildStorer
	billPeriodStore store.GovernmentFundingBillPeriodStorer
	orgStore        store.OrganizationStorer
	fundingStore    store.GovernmentFundingStorer
}

// NewGovernmentFundingBillService creates a new GovernmentFundingBillService.
func NewGovernmentFundingBillService(
	childStore store.ChildStorer,
	billPeriodStore store.GovernmentFundingBillPeriodStorer,
	orgStore store.OrganizationStorer,
	fundingStore store.GovernmentFundingStorer,
) *GovernmentFundingBillService {
	return &GovernmentFundingBillService{
		childStore:      childStore,
		billPeriodStore: billPeriodStore,
		orgStore:        orgStore,
		fundingStore:    fundingStore,
	}
}

// ProcessISBJ parses an ISBJ Excel file, persists the bill period, and returns enriched data.
func (s *GovernmentFundingBillService) ProcessISBJ(ctx context.Context, orgID uint, reader io.Reader, fileName string, fileHash string, userID uint) (*models.GovernmentFundingBillResponse, error) {
	// Check for duplicate file (same SHA-256 hash for this org)
	hashExists, err := s.billPeriodStore.ExistsByOrgAndHash(ctx, orgID, fileHash)
	if err != nil {
		return nil, fmt.Errorf("checking duplicate hash: %w", err)
	}
	if hashExists {
		return nil, &apperror.AppError{
			Err:       apperror.ErrConflict,
			Message:   "a bill with the same file has already been uploaded for this organization",
			Code:      409,
			ErrorCode: apperror.CodeDuplicateBillHash,
		}
	}

	output, err := isbj.ParseFromReader(reader)
	if err != nil {
		return nil, err
	}

	// Check for duplicate month (same billing month for this org)
	monthExists, err := s.billPeriodStore.ExistsByOrgAndMonth(ctx, orgID, output.BillingMonth)
	if err != nil {
		return nil, fmt.Errorf("checking duplicate month: %w", err)
	}
	if monthExists {
		return nil, &apperror.AppError{
			Err:       apperror.ErrConflict,
			Message:   fmt.Sprintf("a bill for %s already exists for this organization; delete the existing bill first", output.BillingMonth.Format("2006-01")),
			Code:      409,
			ErrorCode: apperror.CodeDuplicateBillMonth,
		}
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
		for rowIdx, row := range child.Rows {
			for _, amt := range row.Amounts {
				billChild.Payments = append(billChild.Payments, models.GovernmentFundingBillPayment{
					Key:      amt.Key,
					Value:    amt.Value,
					Amount:   amt.Amount,
					RowIndex: rowIdx,
				})
			}
		}
		period.Children = append(period.Children, billChild)
	}

	if err := s.billPeriodStore.Create(ctx, period); err != nil {
		return nil, fmt.Errorf("persisting bill period: %w", err)
	}

	// Match vouchers and build response
	return s.buildResponse(ctx, orgID, period.ID, period.From, converted)
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
		contracts, err := s.childStore.FindContractsByVoucherNumbers(ctx, orgID, voucherNumbers, period.From)
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

		// Group payments by RowIndex
		rowMap := map[int][]models.GovernmentFundingBillPayment{}
		for _, p := range child.Payments {
			rowMap[p.RowIndex] = append(rowMap[p.RowIndex], p)
			totalAmount += p.Amount

			// Aggregate surcharges (keys defined by ISBJ format)
			for _, sk := range isbj.SurchargeKeys {
				if p.Key == sk {
					surchargeMap[p.Key] += p.Amount
					break
				}
			}
		}

		// Build sorted rows
		maxIdx := 0
		for idx := range rowMap {
			if idx > maxIdx {
				maxIdx = idx
			}
		}
		rows := make([]models.GovernmentFundingBillRowResponse, 0, len(rowMap))
		for i := 0; i <= maxIdx; i++ {
			payments, ok := rowMap[i]
			if !ok {
				continue
			}
			rowTotal := 0
			amounts := make([]models.GovernmentFundingBillAmount, 0, len(payments))
			for _, p := range payments {
				amounts = append(amounts, models.GovernmentFundingBillAmount{
					Key:    p.Key,
					Value:  p.Value,
					Amount: p.Amount,
				})
				rowTotal += p.Amount
			}
			rows = append(rows, models.GovernmentFundingBillRowResponse{
				TotalRowAmount: rowTotal,
				Amounts:        amounts,
			})
		}

		resp := models.GovernmentFundingBillChildResponse{
			VoucherNumber: child.VoucherNumber,
			ChildName:     child.ChildName,
			BirthDate:     child.BirthDate,
			District:      child.District,
			TotalAmount:   totalAmount,
			Rows:          rows,
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

// Compare compares an uploaded ISBJ bill against calculated funding rates per child and property.
func (s *GovernmentFundingBillService) Compare(ctx context.Context, billID, orgID uint) (*models.FundingComparisonResponse, error) {
	// 1. Fetch bill period
	period, err := s.billPeriodStore.FindByID(ctx, billID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("bill period")
		}
		return nil, apperror.InternalWrap(err, "failed to fetch bill period")
	}
	if period.OrganizationID != orgID {
		return nil, apperror.NotFound("bill period")
	}

	// 2. Get org state
	org, err := s.orgStore.FindByID(ctx, orgID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch organization")
	}

	// 3. Get funding config and find period covering bill date
	var fundingPeriod *models.GovernmentFundingPeriod
	var labelMap map[string]string
	funding, fundingErr := s.fundingStore.FindByStateWithDetails(ctx, org.State, 0, nil)
	if fundingErr == nil {
		fundingPeriod = findPeriodForDate(funding.Periods, period.From)
		labelMap = buildLabelMap(funding)
	}
	if labelMap == nil {
		labelMap = make(map[string]string)
	}

	// 4. Match vouchers — same logic as GetByID
	voucherNumbers := make([]string, 0, len(period.Children))
	for _, child := range period.Children {
		voucherNumbers = append(voucherNumbers, child.VoucherNumber)
	}

	contractMap := make(map[string]models.ChildContract)
	if len(voucherNumbers) > 0 {
		contracts, err := s.childStore.FindContractsByVoucherNumbers(ctx, orgID, voucherNumbers, period.From)
		if err != nil {
			return nil, apperror.InternalWrap(err, "failed to fetch contracts")
		}
		for _, c := range contracts {
			if c.VoucherNumber != nil {
				contractMap[*c.VoucherNumber] = c
			}
		}
	}

	// 5. Get children with active contracts for calc-only detection
	activeChildren, err := s.childStore.FindByOrganizationWithActiveOn(ctx, orgID, period.From)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch active children")
	}

	// Build set of vouchers present in the bill
	billVoucherSet := make(map[string]bool, len(period.Children))
	for _, child := range period.Children {
		billVoucherSet[child.VoucherNumber] = true
	}

	// 6. Build comparison per bill child
	response := &models.FundingComparisonResponse{
		BillID:       period.ID,
		BillFrom:     period.From.Format(models.DateFormat),
		BillTo:       formatToDate(period.To),
		FacilityName: period.FacilityName,
		Children:     make([]models.FundingComparisonChild, 0, len(period.Children)),
	}

	// Track matched child IDs for calc-only detection
	matchedChildIDs := make(map[uint]bool)

	for _, billChild := range period.Children {
		billTotal := 0
		billAmounts := make(map[string]int) // "key:value" → amount
		for _, p := range billChild.Payments {
			billTotal += p.Amount
			billAmounts[p.Key+":"+p.Value] += p.Amount
		}

		compChild := models.FundingComparisonChild{
			VoucherNumber: billChild.VoucherNumber,
			ChildName:     billChild.ChildName,
			BirthDate:     billChild.BirthDate,
			BillTotal:     billTotal,
		}

		contract, matched := contractMap[billChild.VoucherNumber]
		if !matched {
			// bill_only
			compChild.Status = "bill_only"
			compChild.Properties = buildBillOnlyProperties(billChild.Payments, labelMap)
			response.BillOnlyCount++
			response.BillTotal += billTotal
		} else {
			// Matched: find child in active children for birthdate and age calculation
			compChild.ChildID = &contract.ChildID
			matchedChildIDs[contract.ChildID] = true

			var childAge *int
			for _, ac := range activeChildren {
				if ac.ID == contract.ChildID {
					age := validation.CalculateAgeOnDate(ac.Birthdate, period.From)
					childAge = &age
					break
				}
			}
			compChild.Age = childAge

			// Calculate funding amounts
			calcAmounts := make(map[string]int) // "key:value" → amount
			if childAge != nil {
				for _, fp := range matchFundingProperties(*childAge, contract.Properties, fundingPeriod) {
					calcAmounts[fp.Key+":"+fp.Value] += fp.Payment
				}
			}

			// Build property-level comparison
			compChild.Properties = buildComparisonProperties(billAmounts, calcAmounts, labelMap)

			// Compute totals
			calcTotal := 0
			for _, amt := range calcAmounts {
				calcTotal += amt
			}
			compChild.CalcTotal = &calcTotal

			diff := billTotal - calcTotal
			compChild.Difference = &diff

			if diff == 0 {
				compChild.Status = "match"
				response.MatchCount++
			} else {
				compChild.Status = "difference"
				response.DifferenceCount++
			}

			// Aggregate totals (only matched children)
			response.BillTotal += billTotal
			response.CalcTotal += calcTotal
		}

		response.Children = append(response.Children, compChild)
	}

	// 7. Detect calc-only children
	for _, ac := range activeChildren {
		if matchedChildIDs[ac.ID] {
			continue
		}
		// Check if this child has a voucher that's already in the bill
		if len(ac.Contracts) == 0 {
			continue
		}
		contract := ac.Contracts[0]
		if contract.VoucherNumber != nil && billVoucherSet[*contract.VoucherNumber] {
			continue
		}

		childAge := validation.CalculateAgeOnDate(ac.Birthdate, period.From)

		calcAmounts := make(map[string]int)
		for _, fp := range matchFundingProperties(childAge, contract.Properties, fundingPeriod) {
			calcAmounts[fp.Key+":"+fp.Value] += fp.Payment
		}

		calcTotal := 0
		for _, amt := range calcAmounts {
			calcTotal += amt
		}

		voucherDisplay := ""
		if contract.VoucherNumber != nil {
			voucherDisplay = *contract.VoucherNumber
		}

		compChild := models.FundingComparisonChild{
			VoucherNumber: voucherDisplay,
			ChildName:     ac.LastName + ", " + ac.FirstName,
			ChildID:       &ac.ID,
			Age:           &childAge,
			CalcTotal:     &calcTotal,
			Status:        "calc_only",
			Properties:    buildCalcOnlyProperties(calcAmounts, labelMap),
		}

		// Enrich with contract dates
		contractFrom := contract.From.Format(models.DateFormat)
		compChild.ContractFrom = &contractFrom
		if contract.To != nil {
			contractTo := contract.To.Format(models.DateFormat)
			compChild.ContractTo = &contractTo
		}

		// Look up bill appearances by voucher number
		if contract.VoucherNumber != nil {
			appearances, err := s.billPeriodStore.FindByOrganizationAndVoucherNumber(ctx, orgID, *contract.VoucherNumber)
			if err == nil {
				// Filter out the current bill
				filtered := make([]models.BillAppearance, 0, len(appearances))
				for _, a := range appearances {
					if a.BillID != billID {
						filtered = append(filtered, a)
					}
				}
				compChild.BillAppearances = filtered
			}
		}

		response.Children = append(response.Children, compChild)
		response.CalcOnlyCount++
		response.CalcTotal += calcTotal
	}

	response.ChildrenCount = len(response.Children)
	response.Difference = response.BillTotal - response.CalcTotal

	return response, nil
}

// buildLabelMap builds a map of "key:value" → label from all funding periods.
func buildLabelMap(funding *models.GovernmentFunding) map[string]string {
	labelMap := make(map[string]string)
	for _, period := range funding.Periods {
		for _, prop := range period.Properties {
			if prop.Label != "" {
				key := prop.Key + ":" + prop.Value
				if _, exists := labelMap[key]; !exists {
					labelMap[key] = prop.Label
				}
			}
		}
	}
	return labelMap
}

// buildComparisonProperties builds the property-level comparison from bill and calculated amounts.
func buildComparisonProperties(billAmounts, calcAmounts map[string]int, labelMap map[string]string) []models.FundingComparisonAmount {
	allKeys := make(map[string]bool)
	for k := range billAmounts {
		allKeys[k] = true
	}
	for k := range calcAmounts {
		allKeys[k] = true
	}

	sortedKeys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	props := make([]models.FundingComparisonAmount, 0, len(sortedKeys))
	for _, kv := range sortedKeys {
		parts := splitKeyValue(kv)
		prop := models.FundingComparisonAmount{
			Key:   parts[0],
			Value: parts[1],
			Label: labelMap[kv],
		}

		if amt, ok := billAmounts[kv]; ok {
			prop.BillAmount = &amt
		}
		if amt, ok := calcAmounts[kv]; ok {
			prop.CalcAmount = &amt
		}

		billVal := 0
		calcVal := 0
		if prop.BillAmount != nil {
			billVal = *prop.BillAmount
		}
		if prop.CalcAmount != nil {
			calcVal = *prop.CalcAmount
		}
		prop.Difference = billVal - calcVal

		props = append(props, prop)
	}
	return props
}

// buildBillOnlyProperties builds properties for a bill-only child (no calculated counterpart).
func buildBillOnlyProperties(payments []models.GovernmentFundingBillPayment, labelMap map[string]string) []models.FundingComparisonAmount {
	props := make([]models.FundingComparisonAmount, 0, len(payments))
	for _, p := range payments {
		amt := p.Amount
		props = append(props, models.FundingComparisonAmount{
			Key:        p.Key,
			Value:      p.Value,
			Label:      labelMap[p.Key+":"+p.Value],
			BillAmount: &amt,
			Difference: amt,
		})
	}
	return props
}

// buildCalcOnlyProperties builds properties for a calc-only child (not in bill).
func buildCalcOnlyProperties(calcAmounts map[string]int, labelMap map[string]string) []models.FundingComparisonAmount {
	sortedKeys := make([]string, 0, len(calcAmounts))
	for k := range calcAmounts {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	props := make([]models.FundingComparisonAmount, 0, len(calcAmounts))
	for _, kv := range sortedKeys {
		amt := calcAmounts[kv]
		parts := splitKeyValue(kv)
		a := amt
		props = append(props, models.FundingComparisonAmount{
			Key:        parts[0],
			Value:      parts[1],
			Label:      labelMap[kv],
			CalcAmount: &a,
			Difference: -a,
		})
	}
	return props
}

// splitKeyValue splits a "key:value" string into its parts.
func splitKeyValue(kv string) [2]string {
	for i, c := range kv {
		if c == ':' {
			return [2]string{kv[:i], kv[i+1:]}
		}
	}
	return [2]string{kv, ""}
}

func (s *GovernmentFundingBillService) buildResponse(ctx context.Context, orgID, periodID uint, billDate time.Time, converted *isbj.ConvertedSettlement) (*models.GovernmentFundingBillResponse, error) {
	// Collect voucher numbers for matching
	voucherNumbers := make([]string, 0, len(converted.Children))
	for _, child := range converted.Children {
		voucherNumbers = append(voucherNumbers, child.VoucherNumber)
	}

	// Look up contracts by voucher number
	contractMap := make(map[string]models.ChildContract)
	if len(voucherNumbers) > 0 {
		contracts, err := s.childStore.FindContractsByVoucherNumbers(ctx, orgID, voucherNumbers, billDate)
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
		rows := make([]models.GovernmentFundingBillRowResponse, 0, len(child.Rows))
		for _, row := range child.Rows {
			rows = append(rows, models.GovernmentFundingBillRowResponse{
				TotalRowAmount: row.TotalRowAmount,
				Amounts:        convertBillAmounts(row.Amounts),
			})
		}
		resp := models.GovernmentFundingBillChildResponse{
			VoucherNumber: child.VoucherNumber,
			ChildName:     child.ChildName,
			BirthDate:     child.BirthDate,
			District:      child.District,
			TotalAmount:   child.TotalAmount,
			Rows:          rows,
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
