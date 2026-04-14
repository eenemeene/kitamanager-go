package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/isbj"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// GovernmentFundingBillService handles government funding bill file processing.
type GovernmentFundingBillService struct {
	childStore        store.ChildStorer
	childVoucherStore store.ChildVoucherStorer
	billPeriodStore   store.GovernmentFundingBillPeriodStorer
	orgStore          store.OrganizationStorer
	fundingStore      store.GovernmentFundingStorer
}

// NewGovernmentFundingBillService creates a new GovernmentFundingBillService.
func NewGovernmentFundingBillService(
	childStore store.ChildStorer,
	childVoucherStore store.ChildVoucherStorer,
	billPeriodStore store.GovernmentFundingBillPeriodStorer,
	orgStore store.OrganizationStorer,
	fundingStore store.GovernmentFundingStorer,
) *GovernmentFundingBillService {
	return &GovernmentFundingBillService{
		childStore:        childStore,
		childVoucherStore: childVoucherStore,
		billPeriodStore:   billPeriodStore,
		orgStore:          orgStore,
		fundingStore:      fundingStore,
	}
}

// resolveVouchersToContracts matches voucher numbers to system children and their active contracts.
// Returns childIDMap (voucher_number → child_id) and contractByChildID (child_id → active contract).
func (s *GovernmentFundingBillService) resolveVouchersToContracts(ctx context.Context, orgID uint, voucherNumbers []string, activeOn time.Time) (map[string]uint, map[uint]models.ChildContract, error) {
	childIDMap := make(map[string]uint)
	contractByChildID := make(map[uint]models.ChildContract)
	if len(voucherNumbers) == 0 {
		return childIDMap, contractByChildID, nil
	}

	var err error
	childIDMap, err = s.childVoucherStore.FindChildIDsByVoucherNumbers(ctx, orgID, voucherNumbers)
	if err != nil {
		return nil, nil, err
	}
	if len(childIDMap) > 0 {
		childIDs := make([]uint, 0, len(childIDMap))
		seen := make(map[uint]bool)
		for _, cid := range childIDMap {
			if !seen[cid] {
				childIDs = append(childIDs, cid)
				seen[cid] = true
			}
		}
		contractByChildID, err = s.childVoucherStore.FindActiveContractsByChildIDsAndDate(ctx, orgID, childIDs, activeOn)
		if err != nil {
			return nil, nil, err
		}
	}
	return childIDMap, contractByChildID, nil
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
			rowType := models.RowTypeRegular
			if row.IsCorrection {
				rowType = models.RowTypeCorrection
			}
			for _, amt := range row.Amounts {
				billChild.Payments = append(billChild.Payments, models.GovernmentFundingBillPayment{
					Key:      amt.Key,
					Value:    amt.Value,
					Amount:   amt.Amount,
					RowIndex: rowIdx,
					RowType:  rowType,
				})
			}
		}
		period.Children = append(period.Children, billChild)
	}

	if err := s.billPeriodStore.Create(ctx, period); err != nil {
		return nil, fmt.Errorf("persisting bill period: %w", err)
	}

	// Auto-discover vouchers: for bill children whose voucher is unknown,
	// try to match by name + birth month/year and create child_voucher entries.
	s.autoDiscoverVouchers(ctx, orgID, period.From, converted)

	// Match vouchers and build response
	return s.buildResponse(ctx, orgID, period.ID, period.From, converted)
}

// ChildrenWithoutVouchers returns children with active contracts but no voucher entries.
func (s *GovernmentFundingBillService) ChildrenWithoutVouchers(ctx context.Context, orgID uint) ([]models.ChildResponse, error) {
	now := time.Now().UTC()
	children, err := s.childVoucherStore.FindChildrenWithoutVouchers(ctx, orgID, now)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children without vouchers")
	}

	result := make([]models.ChildResponse, len(children))
	for i, c := range children {
		result[i] = c.ToResponse()
	}
	return result, nil
}

// parseBillChildName splits a bill child name "LastName,FirstName" into first and last name.
func parseBillChildName(billName string) (firstName, lastName string) {
	parts := strings.SplitN(billName, ",", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[0])
	}
	return "", strings.TrimSpace(billName)
}

// parseBillBirthMonth parses "MM.YY" format into month and year.
func parseBillBirthMonth(billDate string) (time.Month, int, error) {
	parts := strings.SplitN(strings.TrimSpace(billDate), ".", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid birth date format: %q", billDate)
	}
	month, err := strconv.Atoi(parts[0])
	if err != nil || month < 1 || month > 12 {
		return 0, 0, fmt.Errorf("invalid month in %q", billDate)
	}
	year, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid year in %q", billDate)
	}
	if year < 100 {
		year += 2000
	}
	return time.Month(month), year, nil
}

// autoDiscoverVouchers attempts to match unrecognized bill vouchers to existing
// children by name + birth month/year. When exactly one child matches, a new
// child_voucher entry is created so subsequent lookups will find the match.
func (s *GovernmentFundingBillService) autoDiscoverVouchers(ctx context.Context, orgID uint, billDate time.Time, converted *isbj.ConvertedSettlement) {
	// Collect all voucher numbers from the bill
	voucherNumbers := make([]string, 0, len(converted.Children))
	for _, child := range converted.Children {
		voucherNumbers = append(voucherNumbers, child.VoucherNumber)
	}

	// Find which vouchers are already known
	known, err := s.childVoucherStore.FindChildIDsByVoucherNumbers(ctx, orgID, voucherNumbers)
	if err != nil {
		slog.Warn("auto-discover: failed to look up existing vouchers", "error", err)
		return
	}

	// For each unknown voucher, try to match by name + birth month/year
	for _, billChild := range converted.Children {
		if _, ok := known[billChild.VoucherNumber]; ok {
			continue // already known
		}

		firstName, lastName := parseBillChildName(billChild.ChildName)
		if firstName == "" || lastName == "" {
			continue
		}

		birthMonth, birthYear, err := parseBillBirthMonth(billChild.BirthDate)
		if err != nil {
			continue
		}

		matches, err := s.childVoucherStore.FindChildByNameAndBirthMonth(ctx, orgID, firstName, lastName, birthMonth, birthYear)
		if err != nil || len(matches) != 1 {
			continue // no match or ambiguous — skip
		}

		// Exactly one match — create child_voucher entry
		if err := s.childVoucherStore.CreateVoucher(ctx, &models.ChildVoucher{
			ChildID:       matches[0].ID,
			VoucherNumber: billChild.VoucherNumber,
			FirstSeen:     billDate,
		}); err != nil {
			slog.Warn("auto-discover: failed to create voucher",
				"child_id", matches[0].ID,
				"voucher", billChild.VoucherNumber,
				"error", err,
			)
		} else {
			slog.Info("auto-discover: linked voucher to child",
				"child_id", matches[0].ID,
				"child_name", firstName+" "+lastName,
				"voucher", billChild.VoucherNumber,
			)
		}
	}
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

	// Resolve vouchers to children and contracts
	voucherNumbers := make([]string, 0, len(period.Children))
	for _, child := range period.Children {
		voucherNumbers = append(voucherNumbers, child.VoucherNumber)
	}
	childIDMap, contractByChildID, err := s.resolveVouchersToContracts(ctx, orgID, voucherNumbers, period.From)
	if err != nil {
		return nil, err
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

		if childID, ok := childIDMap[child.VoucherNumber]; ok {
			if contract, cok := contractByChildID[childID]; cok {
				resp.ChildID = &contract.ChildID
				resp.ContractID = &contract.ID
				resp.Matched = true
				matchedCount++
			}
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
	return s.comparePeriod(ctx, period, orgID)
}

// CompareByDate finds the bill for a specific month and runs comparison.
func (s *GovernmentFundingBillService) CompareByDate(ctx context.Context, orgID uint, date time.Time) (*models.FundingComparisonResponse, error) {
	period, err := s.billPeriodStore.FindByOrgAndMonth(ctx, orgID, date)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("no bill found for the specified date")
		}
		return nil, apperror.InternalWrap(err, "failed to find bill by date")
	}
	return s.comparePeriod(ctx, period, orgID)
}

// CompareLatest finds the most recent bill and runs comparison.
func (s *GovernmentFundingBillService) CompareLatest(ctx context.Context, orgID uint) (*models.FundingComparisonResponse, error) {
	period, err := s.billPeriodStore.FindLatestByOrganization(ctx, orgID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("no bills found for this organization")
		}
		return nil, apperror.InternalWrap(err, "failed to find latest bill")
	}
	return s.comparePeriod(ctx, period, orgID)
}

// comparePeriod runs the full comparison for a pre-loaded bill period.
func (s *GovernmentFundingBillService) comparePeriod(ctx context.Context, period *models.GovernmentFundingBillPeriod, orgID uint) (*models.FundingComparisonResponse, error) {
	// 1. Get org state
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

	// 4. Resolve vouchers to children and contracts
	voucherNumbers := make([]string, 0, len(period.Children))
	for _, child := range period.Children {
		voucherNumbers = append(voucherNumbers, child.VoucherNumber)
	}
	childIDMap, contractByChildID, err := s.resolveVouchersToContracts(ctx, orgID, voucherNumbers, period.From)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to resolve vouchers to contracts")
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

	// Build activeChildren lookup by ID for birthdate resolution
	activeChildByID := make(map[uint]*models.Child, len(activeChildren))
	for i := range activeChildren {
		activeChildByID[activeChildren[i].ID] = &activeChildren[i]
	}

	// Track matched child IDs for calc-only detection
	matchedChildIDs := make(map[uint]bool)

	var fundingPeriods []models.GovernmentFundingPeriod
	if funding != nil {
		fundingPeriods = funding.Periods
	}

	for _, billChild := range period.Children {
		compChild := models.FundingComparisonChild{
			VoucherNumber: billChild.VoucherNumber,
			ChildName:     billChild.ChildName,
			BirthDate:     billChild.BirthDate,
		}

		// Resolve contract and birthdate for this bill child
		var contract *models.ChildContract
		var birthdate *time.Time
		if childID, ok := childIDMap[billChild.VoucherNumber]; ok {
			if c, cok := contractByChildID[childID]; cok {
				contract = &c
				compChild.ChildID = &c.ChildID
				matchedChildIDs[c.ChildID] = true
				if ac, ok := activeChildByID[c.ChildID]; ok {
					birthdate = &ac.Birthdate
				}
			}
		}

		// Compute comparison using shared pure function
		comp := computeChildComparison(childComparisonInput{
			BillPayments:   billChild.Payments,
			Contract:       contract,
			Birthdate:      birthdate,
			BillDate:       period.From,
			FundingPeriods: fundingPeriods,
			LabelMap:       labelMap,
		})

		compChild.BillTotal = comp.BillTotal
		compChild.CorrectionTotal = comp.CorrectionTotal
		compChild.CalcTotal = comp.CalcTotal
		compChild.Difference = comp.Difference
		compChild.Age = comp.Age
		compChild.Status = comp.Status
		compChild.Properties = comp.Properties

		// Aggregate response totals
		switch comp.Status {
		case "bill_only":
			response.BillOnlyCount++
			response.BillTotal += comp.BillTotal
		case "match":
			response.MatchCount++
			response.BillTotal += comp.BillTotal
			if comp.CalcTotal != nil {
				response.CalcTotal += *comp.CalcTotal
			}
		case "difference":
			response.DifferenceCount++
			response.BillTotal += comp.BillTotal
			if comp.CalcTotal != nil {
				response.CalcTotal += *comp.CalcTotal
			}
		default:
			response.BillTotal += comp.BillTotal
			if comp.CalcTotal != nil {
				response.CalcTotal += *comp.CalcTotal
			}
		}

		response.CorrectionTotal += comp.CorrectionTotal
		response.Children = append(response.Children, compChild)
	}

	// 7. Detect calc-only children
	// Fetch vouchers for all active children to check against bill vouchers
	activeChildIDs := make([]uint, 0, len(activeChildren))
	for _, ac := range activeChildren {
		if !matchedChildIDs[ac.ID] {
			activeChildIDs = append(activeChildIDs, ac.ID)
		}
	}
	activeChildVouchers, _ := s.childVoucherStore.FindVouchersByChildIDs(ctx, activeChildIDs)
	// Build voucher set per child
	vouchersByChildID := make(map[uint][]models.ChildVoucher)
	for _, v := range activeChildVouchers {
		vouchersByChildID[v.ChildID] = append(vouchersByChildID[v.ChildID], v)
	}

	for _, ac := range activeChildren {
		if matchedChildIDs[ac.ID] {
			continue
		}
		// Check if this child has a voucher that's already in the bill
		childVouchers := vouchersByChildID[ac.ID]
		allInBill := len(childVouchers) > 0
		for _, v := range childVouchers {
			if !billVoucherSet[v.VoucherNumber] {
				allInBill = false
				break
			}
		}
		if allInBill && len(childVouchers) > 0 {
			continue
		}

		if len(ac.Contracts) == 0 {
			continue
		}
		contract := ac.Contracts[0]

		childAge := validation.CalculateAgeOnDate(ac.Birthdate, period.From)
		calcAmounts, calcTotal := calcAmountsFromFunding(childAge, contract.Properties, fundingPeriod)

		voucherDisplay := ""
		if len(childVouchers) > 0 {
			voucherDisplay = childVouchers[0].VoucherNumber
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
		if len(childVouchers) > 0 {
			appearances, err := s.billPeriodStore.FindByOrganizationAndVoucherNumber(ctx, orgID, childVouchers[0].VoucherNumber)
			if err == nil {
				// Filter out the current bill
				filtered := make([]models.BillAppearance, 0, len(appearances))
				for _, a := range appearances {
					if a.BillID != period.ID {
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

// childComparisonInput holds inputs for computing a per-child bill vs contract comparison.
type childComparisonInput struct {
	BillPayments   []models.GovernmentFundingBillPayment
	Contract       *models.ChildContract // nil if no matching contract
	Birthdate      *time.Time            // nil if unknown
	BillDate       time.Time
	FundingPeriods []models.GovernmentFundingPeriod // all funding periods (empty if not loaded)
	LabelMap       map[string]string
}

// childComparisonResult holds the output of a per-child bill vs contract comparison.
type childComparisonResult struct {
	BillTotal       int
	CorrectionTotal int
	CalcTotal       *int
	Difference      *int
	Status          string // match|difference|bill_only
	Age             *int
	Properties      []models.FundingComparisonAmount
	NoFundingConfig bool // true when contract matched but no funding config/period available
}

// computeChildComparison computes the property-level comparison for a single bill child entry.
// This is a pure function with no database access, making it trivially testable.
func computeChildComparison(input childComparisonInput) childComparisonResult {
	billAmounts, billTotal := billPaymentsToAmountMap(input.BillPayments, models.RowTypeRegular)
	_, correctionTotal := billPaymentsToAmountMap(input.BillPayments, models.RowTypeCorrection)

	result := childComparisonResult{
		BillTotal:       billTotal,
		CorrectionTotal: correctionTotal,
	}

	if input.Contract == nil {
		result.Status = "bill_only"
		result.Properties = buildBillOnlyProperties(input.BillPayments, input.LabelMap)
		return result
	}

	// Contract matched — compute age if birthdate available
	if input.Birthdate != nil {
		age := validation.CalculateAgeOnDate(*input.Birthdate, input.BillDate)
		result.Age = &age
	}

	// Find applicable funding period and compute calculated amounts
	var calcAmounts map[string]int
	var calcTotal int

	fundingPeriod := findPeriodForDate(input.FundingPeriods, input.BillDate)
	if fundingPeriod != nil && result.Age != nil {
		calcAmounts, calcTotal = calcAmountsFromFunding(*result.Age, input.Contract.Properties, fundingPeriod)
	} else {
		calcAmounts = make(map[string]int)
		result.NoFundingConfig = len(input.FundingPeriods) == 0 || fundingPeriod == nil
	}
	result.CalcTotal = &calcTotal

	diff := billTotal - calcTotal
	result.Difference = &diff

	result.Properties = buildComparisonProperties(billAmounts, calcAmounts, input.LabelMap)

	if diff == 0 {
		result.Status = "match"
	} else {
		result.Status = "difference"
	}

	return result
}

// billPaymentsToAmountMap aggregates bill payments into a "key:value" → total amount map and computes the total.
// If rowType is non-empty, only payments with that RowType are included.
func billPaymentsToAmountMap(payments []models.GovernmentFundingBillPayment, rowType string) (map[string]int, int) {
	amounts := make(map[string]int, len(payments))
	total := 0
	for _, p := range payments {
		if rowType != "" && p.RowType != rowType {
			continue
		}
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

	// Detect property mismatches between bill and calc
	mismatchTypes := classifyMismatches(billAmounts, calcAmounts)

	props := make([]models.FundingComparisonAmount, 0, len(sortedKeys))
	for _, kv := range sortedKeys {
		parts := splitKeyValue(kv)
		prop := models.FundingComparisonAmount{
			Key:      parts[0],
			Value:    parts[1],
			Label:    labelMap[kv],
			Mismatch: mismatchTypes[kv],
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

// contractPropertyKeys lists the base keys that represent actual contract properties.
// Only these keys are considered for mismatch detection. Surcharges (parent, ndh, qm/mss, but)
// are excluded because they are facility-level or funding-config-driven, not contract properties.
//
// This is a hardcoded list because these keys are a fixed set defined by the Berlin funding
// model (RV-Tag). The keys come from the ContractProperties JSONB field on child contracts.
// A more dynamic approach could derive this from the funding config (e.g., properties with
// apply_to_all_contracts are surcharges), but that would require loading the funding config
// just to classify mismatches — unnecessary complexity since these keys won't change unless
// Berlin fundamentally restructures their funding model.
var contractPropertyKeys = map[string]bool{
	"care_type":   true,
	"integration": true,
}

// classifyMismatches classifies each key:value pair as missing, additional, different, or none.
// Only contract property keys (care_type, integration) are checked for mismatches.
//
// For each base key:
//   - If a key:value exists in calc but not bill → "missing" (expected but not billed)
//   - If a key:value exists in bill but not calc → "additional" (billed but not expected)
//   - If the same base key has DIFFERENT values on each side → "different" on both entries
//   - If the key:value exists on both sides → "" (no mismatch)
func classifyMismatches(billAmounts, calcAmounts map[string]int) map[string]models.MismatchType {
	result := make(map[string]models.MismatchType)

	// Collect values per base key for each side
	type keyPresence struct {
		inBill bool
		inCalc bool
	}
	// Track all key:value pairs and which side they appear on
	allPairs := make(map[string]*keyPresence)
	for kv := range billAmounts {
		if allPairs[kv] == nil {
			allPairs[kv] = &keyPresence{}
		}
		allPairs[kv].inBill = true
	}
	for kv := range calcAmounts {
		if allPairs[kv] == nil {
			allPairs[kv] = &keyPresence{}
		}
		allPairs[kv].inCalc = true
	}

	// Group by base key to detect "different" (same key, different values)
	// Only consider contract property keys for mismatch detection
	baseKeyValues := make(map[string][]string) // base_key → list of key:value strings
	for kv := range allPairs {
		parts := splitKeyValue(kv)
		if !contractPropertyKeys[parts[0]] {
			continue
		}
		baseKeyValues[parts[0]] = append(baseKeyValues[parts[0]], kv)
	}

	for _, kvList := range baseKeyValues {
		// Check if any pair in this base key group is one-sided
		hasOneSided := false
		for _, kv := range kvList {
			p := allPairs[kv]
			if p.inBill != p.inCalc {
				hasOneSided = true
				break
			}
		}
		if !hasOneSided {
			continue // all values for this base key exist on both sides — no mismatch
		}

		// Determine if this is a "different" case (both sides have the key but with different values)
		// or a pure "missing"/"additional" case (key only on one side)
		hasBillSide := false
		hasCalcSide := false
		for _, kv := range kvList {
			p := allPairs[kv]
			if p.inBill {
				hasBillSide = true
			}
			if p.inCalc {
				hasCalcSide = true
			}
		}

		if hasBillSide && hasCalcSide {
			// Same base key on both sides but with different values → "different"
			for _, kv := range kvList {
				p := allPairs[kv]
				if p.inBill != p.inCalc {
					result[kv] = models.MismatchDifferent
				}
			}
		} else {
			// Key only on one side
			for _, kv := range kvList {
				p := allPairs[kv]
				if p.inBill && !p.inCalc {
					result[kv] = models.MismatchAdditional
				} else if p.inCalc && !p.inBill {
					result[kv] = models.MismatchMissing
				}
			}
		}
	}

	return result
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
	// Resolve vouchers to children and contracts
	voucherNumbers := make([]string, 0, len(converted.Children))
	for _, child := range converted.Children {
		voucherNumbers = append(voucherNumbers, child.VoucherNumber)
	}
	childIDMap, contractByChildID, err := s.resolveVouchersToContracts(ctx, orgID, voucherNumbers, billDate)
	if err != nil {
		return nil, err
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

		if childID, ok := childIDMap[child.VoucherNumber]; ok {
			if contract, cok := contractByChildID[childID]; cok {
				resp.ChildID = &contract.ChildID
				resp.ContractID = &contract.ID
				resp.Matched = true
				matchedCount++
			}
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

	// 2. Collect all voucher numbers from child_vouchers
	childVouchers, err := s.childVoucherStore.FindVouchersByChildID(ctx, childID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch child vouchers")
	}
	voucherNumbers := make([]string, 0, len(childVouchers))
	for _, v := range childVouchers {
		voucherNumbers = append(voucherNumbers, v.VoucherNumber)
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

	// 6. For each bill entry, compute comparison using shared logic
	totalBilled := 0
	totalCalc := 0
	hasCalc := false

	var fundingPeriods []models.GovernmentFundingPeriod
	if funding != nil {
		fundingPeriods = funding.Periods
	}

	for _, entry := range billEntries {
		// Find the contract active on this bill date
		var activeContract *models.ChildContract
		for i := range child.Contracts {
			c := &child.Contracts[i]
			if c.IsActiveOn(entry.BillFrom) {
				activeContract = c
				break
			}
		}

		// Use "no_contract" status when contract is nil but we know this child exists
		birthdate := child.Birthdate
		comp := computeChildComparison(childComparisonInput{
			BillPayments:   entry.Child.Payments,
			Contract:       activeContract,
			Birthdate:      &birthdate,
			BillDate:       entry.BillFrom,
			FundingPeriods: fundingPeriods,
			LabelMap:       labelMap,
		})

		// Override statuses for billing history context:
		// - computeChildComparison returns "bill_only" for nil contract,
		//   but in billing history we know the child exists — it's "no_contract"
		// - NoFundingConfig flag means we have a contract but can't compute rates;
		//   in billing history, this means we can't produce CalcTotal/Difference
		if activeContract == nil {
			comp.Status = "no_contract"
		} else if comp.NoFundingConfig {
			comp.Status = "no_funding_config"
			comp.CalcTotal = nil
			comp.Difference = nil
			comp.Properties = buildBillOnlyProperties(entry.Child.Payments, labelMap)
		}

		entryResp := models.ChildBillingHistoryEntryResponse{
			BillID:          entry.BillPeriodID,
			BillFrom:        entry.BillFrom.Format(models.DateFormat),
			BillTo:          formatToDate(entry.BillTo),
			FacilityName:    entry.FacilityName,
			VoucherNumber:   entry.Child.VoucherNumber,
			ChildName:       entry.Child.ChildName,
			BirthDate:       entry.Child.BirthDate,
			BillTotal:       comp.BillTotal,
			CorrectionTotal: comp.CorrectionTotal,
			CalcTotal:       comp.CalcTotal,
			Difference:      comp.Difference,
			Age:             comp.Age,
			Status:          comp.Status,
			Properties:      comp.Properties,
		}
		if activeContract != nil {
			entryResp.ContractID = &activeContract.ID
		}

		totalBilled += comp.BillTotal
		if comp.CalcTotal != nil {
			totalCalc += *comp.CalcTotal
			hasCalc = true
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

	// Collect all unique full voucher numbers seen across bill entries
	seenVouchers := make(map[string]bool)
	allVouchers := make([]string, 0)
	for _, entry := range response.Entries {
		if !seenVouchers[entry.VoucherNumber] {
			seenVouchers[entry.VoucherNumber] = true
			allVouchers = append(allVouchers, entry.VoucherNumber)
		}
	}
	response.VoucherNumbers = allVouchers

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

	// 3. All vouchers for this org and their child mappings
	orgVouchers, err := s.childVoucherStore.FindVouchersByOrganization(ctx, orgID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch vouchers for organization")
	}

	// Build lookup: voucher number → childID
	childIDByVoucher := make(map[string]uint, len(orgVouchers))
	childIDs := make(map[uint]bool)
	for _, v := range orgVouchers {
		childIDByVoucher[v.VoucherNumber] = v.ChildID
		childIDs[v.ChildID] = true
	}

	// Load all contracts for children that have vouchers, grouped by child
	allChildIDs := make([]uint, 0, len(childIDs))
	for id := range childIDs {
		allChildIDs = append(allChildIDs, id)
	}
	// Fetch contracts for all children with vouchers
	contractsByChild := make(map[uint][]models.ChildContract)
	for _, cid := range allChildIDs {
		c, err := s.childStore.FindByIDAndOrg(ctx, cid, orgID)
		if err == nil {
			contractsByChild[c.ID] = c.Contracts
		}
	}

	// Build voucher → []ChildContract for active-on lookups
	contractsByVoucher := make(map[string][]models.ChildContract, len(orgVouchers))
	for _, v := range orgVouchers {
		if contracts, ok := contractsByChild[v.ChildID]; ok {
			contractsByVoucher[v.VoucherNumber] = contracts
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

	// 7. Compute contract months per child (how many months their voucher contracts cover)
	now := time.Now().UTC()
	contractMonthsByChild := make(map[uint]int)
	for childID, childContracts := range contractsByChild {
		months := 0
		for _, c := range childContracts {
			end := now
			if c.To != nil && c.To.Before(now) {
				end = *c.To
			}
			if end.Before(c.From) || c.From.After(now) {
				continue
			}
			// Count months: from start month to end month (capped at today) inclusive
			months += countMonths(c.From, end)
		}
		contractMonthsByChild[childID] = months
	}

	// 8. Aggregate per child: sum across all vouchers belonging to the same child
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
			ContractMonths:  contractMonthsByChild[childID],
		})
	}

	return &models.ChildrenBillingSummaryResponse{
		Children: children,
	}, nil
}

// countMonths returns the number of months between from and to (inclusive of both months).
func countMonths(from, to time.Time) int {
	if to.Before(from) {
		return 0
	}
	months := (to.Year()-from.Year())*12 + int(to.Month()) - int(from.Month()) + 1
	if months < 0 {
		return 0
	}
	return months
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
