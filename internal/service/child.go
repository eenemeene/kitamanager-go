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

// ChildService handles business logic for child operations
type ChildService struct {
	store        store.ChildStorer
	orgStore     store.OrganizationStorer
	fundingStore store.GovernmentFundingStorer
	payPlanStore store.PayPlanStorer
	transactor   store.Transactor
}

// NewChildService creates a new child service
func NewChildService(store store.ChildStorer, orgStore store.OrganizationStorer, fundingStore store.GovernmentFundingStorer, payPlanStore store.PayPlanStorer, transactor store.Transactor) *ChildService {
	return &ChildService{
		store:        store,
		orgStore:     orgStore,
		fundingStore: fundingStore,
		payPlanStore: payPlanStore,
		transactor:   transactor,
	}
}

// List returns a paginated list of children
func (s *ChildService) List(ctx context.Context, limit, offset int) ([]models.ChildResponse, int64, error) {
	children, total, err := s.store.FindAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch children")
	}

	return toResponseList(children, (*models.Child).ToResponse), total, nil
}

// ListByOrganization returns a paginated list of children for an organization
func (s *ChildService) ListByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.ChildResponse, int64, error) {
	return s.ListByOrganizationAndSection(ctx, orgID, models.ChildListFilter{}, limit, offset)
}

// ListByOrganizationAndSection returns a paginated list of children for an organization,
// optionally filtered by section, active contract date, contract-after date, and/or name search.
func (s *ChildService) ListByOrganizationAndSection(ctx context.Context, orgID uint, filter models.ChildListFilter, limit, offset int) ([]models.ChildResponse, int64, error) {
	if err := filter.Validate(); err != nil {
		return nil, 0, apperror.BadRequest(err.Error())
	}

	children, total, err := s.store.FindByOrganizationAndSection(ctx, orgID, filter.SectionID, filter.ActiveOn, filter.ContractAfter, filter.Search, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch children")
	}

	return toResponseList(children, (*models.Child).ToResponse), total, nil
}

// GetByID returns a child by ID, validating it belongs to the specified organization
func (s *ChildService) GetByID(ctx context.Context, id, orgID uint) (*models.ChildResponse, error) {
	child, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
		return nil, err
	}
	resp := child.ToResponse()
	return &resp, nil
}

// Create creates a new child
func (s *ChildService) Create(ctx context.Context, orgID uint, req *models.ChildCreateRequest) (*models.ChildResponse, error) {
	// Trim and validate input
	person, err := validation.ValidatePersonCreate(&validation.PersonCreateFields{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Gender:    req.Gender,
		Birthdate: req.Birthdate,
	})
	if err != nil {
		return nil, err
	}

	child := &models.Child{
		Person: models.Person{
			OrganizationID: orgID,
			SectionID:      req.SectionID,
			FirstName:      person.FirstName,
			LastName:       person.LastName,
			Gender:         person.Gender,
			Birthdate:      person.Birthdate,
		},
	}

	if err := s.store.Create(ctx, child); err != nil {
		return nil, apperror.InternalWrap(err, "failed to create child")
	}

	resp := child.ToResponse()
	return &resp, nil
}

// Update updates an existing child, validating it belongs to the specified organization
func (s *ChildService) Update(ctx context.Context, id, orgID uint, req *models.ChildUpdateRequest) (*models.ChildResponse, error) {
	child, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
		return nil, err
	}

	if req.FirstName != nil {
		trimmed, err := validation.ValidateAndTrimName(*req.FirstName, "first_name")
		if err != nil {
			return nil, err
		}
		child.FirstName = trimmed
	}
	if req.LastName != nil {
		trimmed, err := validation.ValidateAndTrimName(*req.LastName, "last_name")
		if err != nil {
			return nil, err
		}
		child.LastName = trimmed
	}
	if req.Gender != nil {
		if err := validation.ValidateGender(*req.Gender); err != nil {
			return nil, err
		}
		child.Gender = *req.Gender
	}
	if req.Birthdate != nil {
		bd, err := validation.ParseAndValidateBirthdate(*req.Birthdate)
		if err != nil {
			return nil, err
		}
		child.Birthdate = bd
	}

	sectionChanged := req.SectionID != nil && !sameSectionID(req.SectionID, child.SectionID)

	if req.SectionID != nil {
		child.SectionID = req.SectionID
		// Clear preloaded association so GORM Save doesn't override the foreign key
		child.Section = nil
	}

	if sectionChanged {
		// Wrap child update + contract transition in a single transaction
		if err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
			if err := s.store.Update(txCtx, child); err != nil {
				return apperror.InternalWrap(err, "failed to update child")
			}
			return s.handleChildSectionTransfer(txCtx, id, req.SectionID)
		}); err != nil {
			return nil, err
		}
	} else {
		if err := s.store.Update(ctx, child); err != nil {
			return nil, apperror.InternalWrap(err, "failed to update child")
		}
	}

	// Reload to get fresh associations (e.g., new Section after section_id change)
	child, err = s.store.FindByID(ctx, id)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to reload child after update")
	}

	resp := child.ToResponse()
	return &resp, nil
}

// handleChildSectionTransfer creates a contract transition when a child moves between sections.
func (s *ChildService) handleChildSectionTransfer(ctx context.Context, childID uint, newSectionID *uint) error {
	return handleSectionTransfer(
		ctx,
		s.store.Contracts(),
		s.store.UpdateContract,
		s.store.CreateContract,
		func(c *models.ChildContract) *models.BaseContract { return &c.BaseContract },
		childID,
		newSectionID,
		func(old *models.ChildContract, today time.Time) *models.ChildContract {
			return &models.ChildContract{
				ChildID: childID,
				BaseContract: models.BaseContract{
					Period: models.Period{
						From: today,
						To:   nil,
					},
					SectionID:  newSectionID,
					Properties: old.Properties,
				},
			}
		},
	)
}

// Delete deletes a child and its contracts, validating it belongs to the specified organization.
// The ownership check and deletion run in a single transaction.
func (s *ChildService) Delete(ctx context.Context, id, orgID uint) error {
	return s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		// Security: Validate child belongs to the specified organization (use minimal query - no preloads needed)
		child, err := s.store.FindByIDMinimal(txCtx, id)
		if err != nil {
			return apperror.NotFound("child")
		}
		if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
			return err
		}

		if err := s.store.Delete(txCtx, id); err != nil {
			return apperror.InternalWrap(err, "failed to delete child")
		}
		return nil
	})
}

// ListContracts returns paginated contract history for a child, validating it belongs to the specified organization
func (s *ChildService) ListContracts(ctx context.Context, childID, orgID uint, limit, offset int) ([]models.ChildContractResponse, int64, error) {
	// Verify child exists and belongs to org (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(ctx, childID)
	if err != nil {
		return nil, 0, apperror.NotFound("child")
	}
	if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
		return nil, 0, err
	}

	contracts, total, err := s.store.FindContractsByChildPaginated(ctx, childID, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch contracts")
	}

	return toResponseList(contracts, (*models.ChildContract).ToResponse), total, nil
}

// GetCurrentContract returns the current active contract for a child, validating it belongs to the specified organization
func (s *ChildService) GetCurrentContract(ctx context.Context, childID, orgID uint) (*models.ChildContractResponse, error) {
	// Security: Validate child belongs to the specified organization (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(ctx, childID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
		return nil, err
	}

	contract, err := s.store.Contracts().GetCurrentContract(ctx, childID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch contract")
	}
	if contract == nil {
		return nil, apperror.NotFound("active contract")
	}
	resp := contract.ToResponse()
	return &resp, nil
}

// GetContractByID returns a contract by ID, validating ownership
func (s *ChildService) GetContractByID(ctx context.Context, contractID, childID, orgID uint) (*models.ChildContractResponse, error) {
	// Security: Validate child belongs to the specified organization (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(ctx, childID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
		return nil, err
	}

	// Get contract
	contract, err := s.store.FindContractByID(ctx, contractID)
	if err != nil {
		return nil, apperror.NotFound("contract")
	}
	if err := verifyContractOwnership(contract, childID); err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// CreateContract creates a new contract for a child, validating it belongs to the specified organization.
// The overlap validation and contract creation run in a single transaction.
func (s *ChildService) CreateContract(ctx context.Context, childID, orgID uint, req *models.ChildContractCreateRequest) (*models.ChildContractResponse, error) {
	// Validate period
	if err := validation.ValidatePeriod(req.From, req.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	// Verify child exists and belongs to org (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(ctx, childID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
		return nil, err
	}

	// Validate contract dates are not before child's birthdate
	if req.From.Before(child.Birthdate) {
		return nil, apperror.BadRequest("contract start date cannot be before child's birthdate")
	}
	if req.To != nil && req.To.Before(child.Birthdate) {
		return nil, apperror.BadRequest("contract end date cannot be before child's birthdate")
	}

	// Auto-set section from child when not specified in request
	sectionID := req.SectionID
	if sectionID == nil {
		sectionID = child.SectionID
	}

	contract := &models.ChildContract{
		ChildID: childID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: req.From,
				To:   req.To,
			},
			SectionID:  sectionID,
			Properties: req.Properties,
		},
	}

	// Validate + create in a single transaction to prevent race conditions
	if err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.store.Contracts().ValidateNoOverlap(txCtx, childID, req.From, req.To, nil); err != nil {
			if errors.Is(err, store.ErrContractOverlap) {
				return apperror.Conflict(err.Error())
			}
			return apperror.InternalWrap(err, "failed to validate contract")
		}
		return s.store.CreateContract(txCtx, contract)
	}); err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// UpdateContract updates an existing contract, validating it belongs to a child in the specified organization.
func (s *ChildService) UpdateContract(ctx context.Context, contractID, childID, orgID uint, req *models.ChildContractUpdateRequest) (*models.ChildContractResponse, error) {
	// Security: Validate child belongs to the specified organization (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(ctx, childID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
		return nil, err
	}

	// Validate contract belongs to the child
	contract, err := s.store.FindContractByID(ctx, contractID)
	if err != nil {
		return nil, apperror.NotFound("contract")
	}
	if err := verifyContractOwnership(contract, childID); err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.From != nil {
		contract.From = *req.From
	}
	if req.To != nil {
		contract.To = req.To
	}
	// Properties can be replaced entirely
	if req.Properties != nil {
		contract.Properties = req.Properties
	}

	// Validate period
	if err := validation.ValidatePeriod(contract.From, contract.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	// Validate + update in a single transaction to prevent race conditions
	if err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.store.Contracts().ValidateNoOverlap(txCtx, childID, contract.From, contract.To, &contractID); err != nil {
			if errors.Is(err, store.ErrContractOverlap) {
				return apperror.Conflict(err.Error())
			}
			return apperror.InternalWrap(err, "failed to validate contract")
		}
		return s.store.UpdateContract(txCtx, contract)
	}); err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// DeleteContract deletes a contract, validating it belongs to a child in the specified organization
func (s *ChildService) DeleteContract(ctx context.Context, contractID, childID, orgID uint) error {
	// Security: Validate child belongs to the specified organization (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(ctx, childID)
	if err != nil {
		return apperror.NotFound("child")
	}
	if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
		return err
	}

	// Validate contract belongs to the child
	contract, err := s.store.FindContractByID(ctx, contractID)
	if err != nil {
		return apperror.NotFound("contract")
	}
	if err := verifyContractOwnership(contract, childID); err != nil {
		return err
	}

	if err := s.store.DeleteContract(ctx, contractID); err != nil {
		return apperror.InternalWrap(err, "failed to delete contract")
	}
	return nil
}

// CalculateFunding calculates government funding for all children with active contracts on the given date
func (s *ChildService) CalculateFunding(ctx context.Context, orgID uint, date time.Time) (*models.ChildrenFundingResponse, error) {
	// Get organization to determine state
	org, err := s.orgStore.FindByID(ctx, orgID)
	if err != nil {
		return nil, apperror.NotFound("organization")
	}

	// Get children with active contracts on this date
	children, err := s.store.FindByOrganizationWithActiveOn(ctx, orgID, date)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children")
	}

	response := &models.ChildrenFundingResponse{
		Date:     date,
		Children: make([]models.ChildFundingResponse, 0, len(children)),
	}

	// Look up org's pay plan to get weekly hours basis
	if payPlans, _, err := s.payPlanStore.GetByOrganization(ctx, orgID, 1, 0); err == nil && len(payPlans) > 0 {
		if period, err := s.payPlanStore.GetActivePeriod(ctx, payPlans[0].ID, date); err == nil && period != nil {
			response.WeeklyHoursBasis = period.WeeklyHours
		}
	}

	// Look up funding by organization's state (0 = all periods, needed to find matching period for date)
	funding, err := s.fundingStore.FindByStateWithDetails(ctx, org.State, 0, nil)
	if err != nil {
		// No funding defined for this state - return 0 funding for all children
		for _, child := range children {
			if len(child.Contracts) == 0 {
				continue
			}
			contract := child.Contracts[0] // Already filtered to contracts active on the date
			response.Children = append(response.Children, models.ChildFundingResponse{
				ChildID:             child.ID,
				ChildName:           child.FirstName + " " + child.LastName,
				Age:                 validation.CalculateAgeOnDate(child.Birthdate, date),
				Funding:             0,
				MatchedProperties:   []models.ChildFundingMatchedProp{},
				UnmatchedProperties: getAllContractKeyValues(contract.Properties),
			})
		}
		return response, nil
	}

	// Find the period covering this date
	period := s.findPeriodForDate(funding.Periods, date)

	for _, child := range children {
		if len(child.Contracts) == 0 {
			continue
		}
		contract := child.Contracts[0]
		childAge := validation.CalculateAgeOnDate(child.Birthdate, date)

		childFunding := s.calculateChildFunding(childAge, contract.Properties, period)
		childFunding.ChildID = child.ID
		childFunding.ChildName = child.FirstName + " " + child.LastName
		childFunding.Age = childAge

		response.Children = append(response.Children, childFunding)
	}

	return response, nil
}

// findPeriodForDate finds the funding period that covers the given date
func (s *ChildService) findPeriodForDate(periods []models.GovernmentFundingPeriod, date time.Time) *models.GovernmentFundingPeriod {
	for i := range periods {
		period := &periods[i]
		// Check if date is within period: from <= date AND (to is nil OR to >= date)
		if !period.From.After(date) && (period.To == nil || !period.To.Before(date)) {
			return period
		}
	}
	return nil
}

// calculateChildFunding calculates funding for a single child based on their age and contract properties.
// It matches contract properties against government funding properties using Key/Value matching.
func (s *ChildService) calculateChildFunding(age int, properties models.ContractProperties, period *models.GovernmentFundingPeriod) models.ChildFundingResponse {
	result := models.ChildFundingResponse{
		MatchedProperties:   []models.ChildFundingMatchedProp{},
		UnmatchedProperties: []models.ChildFundingMatchedProp{},
	}

	// Get all key-value pairs from contract properties
	contractKeyValues := getAllContractKeyValues(properties)

	// No period covering this date
	if period == nil {
		result.UnmatchedProperties = contractKeyValues
		return result
	}

	// Track which contract key-value pairs have been matched
	matchedSet := make(map[string]bool) // key:value -> matched

	totalFunding := 0
	totalRequirement := 0.0
	for _, fundingProp := range period.Properties {
		// Check if age matches
		if !fundingProp.MatchesAge(age) {
			continue
		}

		// Check if contract has this key:value
		if properties.HasValue(fundingProp.Key, fundingProp.Value) {
			totalFunding += fundingProp.Payment
			totalRequirement += fundingProp.Requirement
			kvKey := fundingProp.Key + ":" + fundingProp.Value
			if !matchedSet[kvKey] {
				matchedSet[kvKey] = true
				result.MatchedProperties = append(result.MatchedProperties, models.ChildFundingMatchedProp{
					Key:   fundingProp.Key,
					Value: fundingProp.Value,
				})
			}
		}
	}

	// Find unmatched contract properties
	for _, kv := range contractKeyValues {
		kvKey := kv.Key + ":" + kv.Value
		if !matchedSet[kvKey] {
			result.UnmatchedProperties = append(result.UnmatchedProperties, kv)
		}
	}

	result.Funding = totalFunding
	result.Requirement = totalRequirement
	return result
}

// getAllContractKeyValues extracts all key-value pairs from contract properties.
// For scalar properties, returns one entry. For array properties, returns one entry per value.
func getAllContractKeyValues(properties models.ContractProperties) []models.ChildFundingMatchedProp {
	if properties == nil {
		return []models.ChildFundingMatchedProp{}
	}

	result := []models.ChildFundingMatchedProp{}
	for key := range properties {
		values := properties.GetAllValues(key)
		for _, value := range values {
			result = append(result, models.ChildFundingMatchedProp{
				Key:   key,
				Value: value,
			})
		}
	}
	return result
}

// GetAgeDistribution returns age distribution of children with active contracts on the given date
func (s *ChildService) GetAgeDistribution(ctx context.Context, orgID uint, date time.Time) (*models.AgeDistributionResponse, error) {
	// Get children with active contracts on this date
	children, err := s.store.FindByOrganizationWithActiveOn(ctx, orgID, date)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children")
	}

	// Define age buckets: 0, 1, 2, 3, 4, 5, 6+
	buckets := []models.AgeDistributionBucket{
		{AgeLabel: "0", MinAge: 0, MaxAge: intPtr(0), Count: 0},
		{AgeLabel: "1", MinAge: 1, MaxAge: intPtr(1), Count: 0},
		{AgeLabel: "2", MinAge: 2, MaxAge: intPtr(2), Count: 0},
		{AgeLabel: "3", MinAge: 3, MaxAge: intPtr(3), Count: 0},
		{AgeLabel: "4", MinAge: 4, MaxAge: intPtr(4), Count: 0},
		{AgeLabel: "5", MinAge: 5, MaxAge: intPtr(5), Count: 0},
		{AgeLabel: "6+", MinAge: 6, MaxAge: nil, Count: 0}, // Open-ended
	}

	totalCount := 0
	for _, child := range children {
		age := validation.CalculateAgeOnDate(child.Birthdate, date)
		totalCount++

		// Find matching bucket
		for i := range buckets {
			bucket := &buckets[i]
			matches := false
			if bucket.MaxAge == nil {
				// Open-ended bucket (6+)
				matches = age >= bucket.MinAge
			} else {
				matches = age >= bucket.MinAge && age <= *bucket.MaxAge
			}

			if matches {
				bucket.Count++
				// Count by gender
				switch child.Gender {
				case string(models.GenderMale):
					bucket.MaleCount++
				case string(models.GenderFemale):
					bucket.FemaleCount++
				case string(models.GenderDiverse):
					bucket.DiverseCount++
				}
				break
			}
		}
	}

	return &models.AgeDistributionResponse{
		Date:         date.Format(models.DateFormat),
		TotalCount:   totalCount,
		Distribution: buckets,
	}, nil
}

// intPtr returns a pointer to an int
func intPtr(i int) *int {
	return &i
}

// GetContractCountByMonth returns children contract counts per month for the specified year range
func (s *ChildService) GetContractCountByMonth(ctx context.Context, orgID uint, minYear, maxYear int) (*models.ChildrenContractCountByMonthResponse, error) {
	numYears := maxYear - minYear + 1

	startDate := time.Date(minYear, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(maxYear, 12, 31, 0, 0, 0, 0, time.UTC)

	response := &models.ChildrenContractCountByMonthResponse{
		Period: models.ContractCountPeriod{
			Start: startDate.Format(models.DateFormat),
			End:   endDate.Format(models.DateFormat),
		},
		Years: make([]models.ContractCountByMonthYear, numYears),
	}

	for i := 0; i < numYears; i++ {
		response.Years[i] = models.ContractCountByMonthYear{
			Year:   minYear + i,
			Counts: make([]int, 12),
		}
	}

	// Single query: fetch all contracts overlapping with the date range
	contracts, err := s.store.FindContractsByOrganizationInDateRange(ctx, orgID, startDate, endDate)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch contracts for date range")
	}

	// Compute per-month counts in memory
	for yearIdx := 0; yearIdx < numYears; yearIdx++ {
		year := minYear + yearIdx
		for month := 1; month <= 12; month++ {
			sampleDate := time.Date(year, time.Month(month), 15, 0, 0, 0, 0, time.UTC)
			uniqueChildren := make(map[uint]struct{})
			for i := range contracts {
				c := &contracts[i]
				if !c.From.After(sampleDate) && (c.To == nil || !c.To.Before(sampleDate)) {
					uniqueChildren[c.ChildID] = struct{}{}
				}
			}
			response.Years[yearIdx].Counts[month-1] = len(uniqueChildren)
		}
	}

	return response, nil
}
