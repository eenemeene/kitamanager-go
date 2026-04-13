package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"slices"
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
			if slices.Contains(isbj.SurchargeKeys, p.Key) {
				surchargeMap[p.Key] += p.Amount
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
		billAmounts, billTotal := billPaymentsToAmountMap(billChild.Payments)

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
			var calcAmounts map[string]int
			var calcTotal int
			if childAge != nil {
				calcAmounts, calcTotal = calcAmountsFromFunding(*childAge, contract.Properties, fundingPeriod)
			} else {
				calcAmounts = make(map[string]int)
			}

			// Build property-level comparison
			compChild.Properties = buildComparisonProperties(billAmounts, calcAmounts, labelMap)
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
		calcAmounts, calcTotal := calcAmountsFromFunding(childAge, contract.Properties, fundingPeriod)

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

// billPaymentsToAmountMap aggregates bill payments into a "key:value" → total amount map and computes the total.
func billPaymentsToAmountMap(payments []models.GovernmentFundingBillPayment) (map[string]int, int) {
	amounts := make(map[string]int, len(payments))
	total := 0
	for _, p := range payments {
		amounts[p.Key+":"+p.Value] += p.Amount
		total += p.Amount
	}
	return amounts, total
}

// calcAmountsFromFunding computes calculated amounts from matched funding properties.
func calcAmountsFromFunding(age int, props models.ContractProperties, period *models.GovernmentFundingPeriod) (map[string]int, int) {
	amounts := make(map[string]int)
	total := 0
	for _, fp := range matchFundingProperties(age, props, period) {
		key := fp.Key + ":" + fp.Value
		amounts[key] += fp.Payment
		total += fp.Payment
	}
	return amounts, total
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

// ChildBillingHistory returns the complete billing history for a child across all uploaded bills.
func (s *GovernmentFundingBillService) ChildBillingHistory(ctx context.Context, childID, orgID uint) (*models.ChildBillingHistoryResponse, error) {
	// 1. Fetch child with contracts
	child, err := s.childStore.FindByIDAndOrg(ctx, childID, orgID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("child")
		}
		return nil, apperror.InternalWrap(err, "failed to fetch child")
	}

	// 2. Collect all voucher numbers across all contracts
	voucherNumbers := make([]string, 0)
	voucherSet := make(map[string]bool)
	for _, contract := range child.Contracts {
		if contract.VoucherNumber != nil && !voucherSet[*contract.VoucherNumber] {
			voucherNumbers = append(voucherNumbers, *contract.VoucherNumber)
			voucherSet[*contract.VoucherNumber] = true
		}
	}

	response := &models.ChildBillingHistoryResponse{
		ChildID:        child.ID,
		ChildName:      child.FirstName + " " + child.LastName,
		Birthdate:      child.Birthdate.Format(models.DateFormat),
		VoucherNumbers: voucherNumbers,
		Entries:        []models.ChildBillingHistoryEntryResponse{},
	}

	// 3. Early exit if no voucher numbers
	if len(voucherNumbers) == 0 {
		return response, nil
	}

	// 4. Fetch all bill entries for these voucher numbers
	billEntries, err := s.billPeriodStore.FindChildEntriesByOrgAndVoucherNumbers(ctx, orgID, voucherNumbers)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch bill entries")
	}

	if len(billEntries) == 0 {
		return response, nil
	}

	// 5. Fetch funding config for the org's state
	org, err := s.orgStore.FindByID(ctx, orgID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch organization")
	}

	var funding *models.GovernmentFunding
	var labelMap map[string]string
	funding, fundingErr := s.fundingStore.FindByStateWithDetails(ctx, org.State, 0, nil)
	if fundingErr == nil {
		labelMap = buildLabelMap(funding)
	}
	if labelMap == nil {
		labelMap = make(map[string]string)
	}

	// 6. For each bill entry, compute comparison
	totalBilled := 0
	totalCalc := 0
	hasCalc := false

	for _, entry := range billEntries {
		billAmounts, billTotal := billPaymentsToAmountMap(entry.Child.Payments)
		totalBilled += billTotal

		entryResp := models.ChildBillingHistoryEntryResponse{
			BillID:        entry.BillPeriodID,
			BillFrom:      entry.BillFrom.Format(models.DateFormat),
			BillTo:        formatToDate(entry.BillTo),
			FacilityName:  entry.FacilityName,
			VoucherNumber: entry.Child.VoucherNumber,
			ChildName:     entry.Child.ChildName,
			BirthDate:     entry.Child.BirthDate,
			BillTotal:     billTotal,
		}

		// Find the contract active on this bill date with matching voucher
		var activeContract *models.ChildContract
		for i := range child.Contracts {
			c := &child.Contracts[i]
			if c.VoucherNumber == nil || *c.VoucherNumber != entry.Child.VoucherNumber {
				continue
			}
			if c.IsActiveOn(entry.BillFrom) {
				activeContract = c
				break
			}
		}

		if activeContract == nil {
			// Bill entry exists but no matching contract
			entryResp.Status = "no_contract"
			entryResp.Properties = buildBillOnlyProperties(entry.Child.Payments, labelMap)
		} else {
			entryResp.ContractID = &activeContract.ID
			age := validation.CalculateAgeOnDate(child.Birthdate, entry.BillFrom)
			entryResp.Age = &age

			if funding == nil {
				// No funding config available
				entryResp.Status = "no_funding_config"
				entryResp.Properties = buildBillOnlyProperties(entry.Child.Payments, labelMap)
			} else {
				fundingPeriod := findPeriodForDate(funding.Periods, entry.BillFrom)
				if fundingPeriod == nil {
					entryResp.Status = "no_funding_config"
					entryResp.Properties = buildBillOnlyProperties(entry.Child.Payments, labelMap)
				} else {
					calcAmounts, calcTotal := calcAmountsFromFunding(age, activeContract.Properties, fundingPeriod)
					entryResp.CalcTotal = &calcTotal
					diff := billTotal - calcTotal
					entryResp.Difference = &diff
					entryResp.Properties = buildComparisonProperties(billAmounts, calcAmounts, labelMap)
					totalCalc += calcTotal
					hasCalc = true

					if diff == 0 {
						entryResp.Status = "match"
					} else {
						entryResp.Status = "difference"
					}
				}
			}
		}

		response.Entries = append(response.Entries, entryResp)
	}

	// Compute running difference: cumulative sum of computable differences
	running := 0
	for i := range response.Entries {
		if response.Entries[i].Difference != nil {
			running += *response.Entries[i].Difference
		}
		response.Entries[i].RunningDifference = running
	}

	response.TotalBilled = totalBilled
	if hasCalc {
		response.TotalCalculated = totalCalc
		response.TotalDifference = totalBilled - totalCalc
	}

	return response, nil
}

// ChildrenBillingSummary returns billing summaries for all children in an org.
// Uses SQL aggregation for billed totals and batch Go computation for expected amounts.
func (s *GovernmentFundingBillService) ChildrenBillingSummary(ctx context.Context, orgID uint) (*models.ChildrenBillingSummaryResponse, error) {
	// 1. SQL-aggregated billed totals per voucher number
	billedTotals, err := s.billPeriodStore.FindBilledTotalsByOrg(ctx, orgID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch billed totals")
	}

	billedByVoucher := make(map[string]models.VoucherBilledTotal, len(billedTotals))
	for _, bt := range billedTotals {
		billedByVoucher[bt.VoucherNumber] = bt
	}

	// 2. Lightweight bill date + voucher pairs for computing expected amounts
	billDateVouchers, err := s.billPeriodStore.FindAllBillDatesAndVouchersByOrg(ctx, orgID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch bill date vouchers")
	}

	// 3. All contracts with voucher numbers for this org
	contracts, err := s.childStore.FindContractsByOrganizationWithVouchers(ctx, orgID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch contracts with vouchers")
	}

	// Build lookup: voucher number → []ChildContract (a voucher belongs to one child,
	// but we keep a slice because we need to check IsActiveOn for each bill date)
	contractsByVoucher := make(map[string][]models.ChildContract, len(contracts))
	childIDByVoucher := make(map[string]uint, len(contracts))
	childIDs := make(map[uint]bool)
	for _, c := range contracts {
		if c.VoucherNumber != nil {
			v := *c.VoucherNumber
			contractsByVoucher[v] = append(contractsByVoucher[v], c)
			childIDByVoucher[v] = c.ChildID
			childIDs[c.ChildID] = true
		}
	}

	// 4. Load child birthdates for age calculation
	childBirthdates := make(map[uint]time.Time)
	if len(childIDs) > 0 {
		ids := make([]uint, 0, len(childIDs))
		for id := range childIDs {
			ids = append(ids, id)
		}
		for _, id := range ids {
			child, err := s.childStore.FindByIDMinimal(ctx, id)
			if err == nil {
				childBirthdates[child.ID] = child.Birthdate
			}
		}
	}

	// 5. Load funding config
	org, err := s.orgStore.FindByID(ctx, orgID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch organization")
	}

	var funding *models.GovernmentFunding
	funding, _ = s.fundingStore.FindByStateWithDetails(ctx, org.State, 0, nil)

	// 6. Compute expected totals per voucher from bill date pairs
	calcByVoucher := make(map[string]int)
	for _, bdv := range billDateVouchers {
		contracts := contractsByVoucher[bdv.VoucherNumber]
		if len(contracts) == 0 {
			continue // no contract for this voucher
		}

		// Find contract active on this bill date
		var activeContract *models.ChildContract
		for i := range contracts {
			if contracts[i].IsActiveOn(bdv.BillFrom) {
				activeContract = &contracts[i]
				break
			}
		}
		if activeContract == nil {
			continue // no active contract on this date
		}

		childID := activeContract.ChildID
		birthdate, hasBirthdate := childBirthdates[childID]
		if !hasBirthdate || funding == nil {
			continue
		}

		fundingPeriod := findPeriodForDate(funding.Periods, bdv.BillFrom)
		if fundingPeriod == nil {
			continue
		}

		age := validation.CalculateAgeOnDate(birthdate, bdv.BillFrom)
		_, calcTotal := calcAmountsFromFunding(age, activeContract.Properties, fundingPeriod)
		calcByVoucher[bdv.VoucherNumber] += calcTotal
	}

	// 7. Aggregate per child: sum across all vouchers belonging to the same child
	type childAccum struct {
		totalBilled     int
		totalCalculated int
		billCount       int
	}
	perChild := make(map[uint]*childAccum)

	// Add billed totals (from SQL aggregation)
	for voucher, bt := range billedByVoucher {
		childID, ok := childIDByVoucher[voucher]
		if !ok {
			continue // bill voucher not matched to any child contract
		}
		acc := perChild[childID]
		if acc == nil {
			acc = &childAccum{}
			perChild[childID] = acc
		}
		acc.totalBilled += bt.TotalBilled
		acc.billCount += bt.BillCount
	}

	// Add calculated totals
	for voucher, calcTotal := range calcByVoucher {
		childID, ok := childIDByVoucher[voucher]
		if !ok {
			continue
		}
		acc := perChild[childID]
		if acc == nil {
			acc = &childAccum{}
			perChild[childID] = acc
		}
		acc.totalCalculated += calcTotal
	}

	// Build response
	children := make([]models.ChildBillingSummaryEntry, 0, len(perChild))
	for childID, acc := range perChild {
		children = append(children, models.ChildBillingSummaryEntry{
			ChildID:         childID,
			TotalBilled:     acc.totalBilled,
			TotalCalculated: acc.totalCalculated,
			TotalDifference: acc.totalBilled - acc.totalCalculated,
			BillCount:       acc.billCount,
		})
	}

	return &models.ChildrenBillingSummaryResponse{
		Children: children,
	}, nil
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
