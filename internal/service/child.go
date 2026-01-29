package service

import (
	"context"
	"errors"
	"strings"
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
}

// NewChildService creates a new child service
func NewChildService(store store.ChildStorer, orgStore store.OrganizationStorer, fundingStore store.GovernmentFundingStorer) *ChildService {
	return &ChildService{
		store:        store,
		orgStore:     orgStore,
		fundingStore: fundingStore,
	}
}

// List returns a paginated list of children
func (s *ChildService) List(ctx context.Context, limit, offset int) ([]models.ChildResponse, int64, error) {
	children, total, err := s.store.FindAll(limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch children")
	}

	responses := make([]models.ChildResponse, len(children))
	for i, c := range children {
		responses[i] = c.ToResponse()
	}
	return responses, total, nil
}

// ListByOrganization returns a paginated list of children for an organization
func (s *ChildService) ListByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.ChildResponse, int64, error) {
	children, total, err := s.store.FindByOrganization(orgID, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch children")
	}

	responses := make([]models.ChildResponse, len(children))
	for i, c := range children {
		responses[i] = c.ToResponse()
	}
	return responses, total, nil
}

// GetByID returns a child by ID, validating it belongs to the specified organization
func (s *ChildService) GetByID(ctx context.Context, id, orgID uint) (*models.ChildResponse, error) {
	child, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	// Security: Validate child belongs to the specified organization
	if child.OrganizationID != orgID {
		return nil, apperror.NotFound("child")
	}
	resp := child.ToResponse()
	return &resp, nil
}

// Create creates a new child
func (s *ChildService) Create(ctx context.Context, orgID uint, req *models.ChildCreateRequest) (*models.ChildResponse, error) {
	// Trim and validate input
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)

	if validation.IsWhitespaceOnly(req.FirstName) {
		return nil, apperror.BadRequest("first_name cannot be empty or whitespace only")
	}
	if validation.IsWhitespaceOnly(req.LastName) {
		return nil, apperror.BadRequest("last_name cannot be empty or whitespace only")
	}
	if !models.IsValidGender(req.Gender) {
		return nil, apperror.BadRequest("gender must be one of: male, female, diverse")
	}
	if err := validation.ValidateBirthdate(req.Birthdate); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	child := &models.Child{
		Person: models.Person{
			OrganizationID: orgID,
			FirstName:      req.FirstName,
			LastName:       req.LastName,
			Gender:         req.Gender,
			Birthdate:      req.Birthdate,
		},
	}

	if err := s.store.Create(child); err != nil {
		return nil, apperror.Internal("failed to create child")
	}

	resp := child.ToResponse()
	return &resp, nil
}

// Update updates an existing child, validating it belongs to the specified organization
func (s *ChildService) Update(ctx context.Context, id, orgID uint, req *models.ChildUpdateRequest) (*models.ChildResponse, error) {
	child, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	// Security: Validate child belongs to the specified organization
	if child.OrganizationID != orgID {
		return nil, apperror.NotFound("child")
	}

	if req.FirstName != nil {
		trimmed := strings.TrimSpace(*req.FirstName)
		if validation.IsWhitespaceOnly(trimmed) {
			return nil, apperror.BadRequest("first_name cannot be empty or whitespace only")
		}
		child.FirstName = trimmed
	}
	if req.LastName != nil {
		trimmed := strings.TrimSpace(*req.LastName)
		if validation.IsWhitespaceOnly(trimmed) {
			return nil, apperror.BadRequest("last_name cannot be empty or whitespace only")
		}
		child.LastName = trimmed
	}
	if req.Gender != nil {
		if !models.IsValidGender(*req.Gender) {
			return nil, apperror.BadRequest("gender must be one of: male, female, diverse")
		}
		child.Gender = *req.Gender
	}
	if req.Birthdate != nil {
		if err := validation.ValidateBirthdate(*req.Birthdate); err != nil {
			return nil, apperror.BadRequest(err.Error())
		}
		child.Birthdate = *req.Birthdate
	}

	if err := s.store.Update(child); err != nil {
		return nil, apperror.Internal("failed to update child")
	}

	resp := child.ToResponse()
	return &resp, nil
}

// Delete deletes a child, validating it belongs to the specified organization
func (s *ChildService) Delete(ctx context.Context, id, orgID uint) error {
	// Security: Validate child belongs to the specified organization (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(id)
	if err != nil {
		return apperror.NotFound("child")
	}
	if child.OrganizationID != orgID {
		return apperror.NotFound("child")
	}

	if err := s.store.Delete(id); err != nil {
		return apperror.Internal("failed to delete child")
	}
	return nil
}

// ListContracts returns paginated contract history for a child, validating it belongs to the specified organization
func (s *ChildService) ListContracts(ctx context.Context, childID, orgID uint, limit, offset int) ([]models.ChildContractResponse, int64, error) {
	// Verify child exists and belongs to org (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(childID)
	if err != nil {
		return nil, 0, apperror.NotFound("child")
	}
	// Security: Validate child belongs to the specified organization
	if child.OrganizationID != orgID {
		return nil, 0, apperror.NotFound("child")
	}

	contracts, total, err := s.store.Contracts().GetHistoryPaginated(childID, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch contracts")
	}

	responses := make([]models.ChildContractResponse, len(contracts))
	for i, c := range contracts {
		responses[i] = c.ToResponse()
	}
	return responses, total, nil
}

// GetCurrentContract returns the current active contract for a child, validating it belongs to the specified organization
func (s *ChildService) GetCurrentContract(ctx context.Context, childID, orgID uint) (*models.ChildContractResponse, error) {
	// Security: Validate child belongs to the specified organization (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(childID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if child.OrganizationID != orgID {
		return nil, apperror.NotFound("child")
	}

	contract, err := s.store.Contracts().GetCurrentContract(childID)
	if err != nil {
		return nil, apperror.Internal("failed to fetch contract")
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
	child, err := s.store.FindByIDMinimal(childID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if child.OrganizationID != orgID {
		return nil, apperror.NotFound("child")
	}

	// Get contract
	contract, err := s.store.FindContractByID(contractID)
	if err != nil {
		return nil, apperror.NotFound("contract")
	}
	if contract.ChildID != childID {
		return nil, apperror.NotFound("contract")
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// CreateContract creates a new contract for a child, validating it belongs to the specified organization
func (s *ChildService) CreateContract(ctx context.Context, childID, orgID uint, req *models.ChildContractCreateRequest) (*models.ChildContractResponse, error) {
	// Validate period
	if err := validation.ValidatePeriod(req.From, req.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	// Verify child exists and belongs to org (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(childID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	// Security: Validate child belongs to the specified organization
	if child.OrganizationID != orgID {
		return nil, apperror.NotFound("child")
	}

	// Validate no overlap
	if err := s.store.Contracts().ValidateNoOverlap(childID, req.From, req.To, nil); err != nil {
		if errors.Is(err, store.ErrContractOverlap) {
			return nil, apperror.Conflict(err.Error())
		}
		return nil, apperror.Internal("failed to validate contract")
	}

	contract := &models.ChildContract{
		ChildID: childID,
		Period: models.Period{
			From: req.From,
			To:   req.To,
		},
		Attributes: req.Attributes,
	}

	if err := s.store.CreateContract(contract); err != nil {
		return nil, apperror.Internal("failed to create contract")
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// UpdateContract updates an existing contract, validating it belongs to a child in the specified organization
func (s *ChildService) UpdateContract(ctx context.Context, contractID, childID, orgID uint, req *models.ChildContractUpdateRequest) (*models.ChildContractResponse, error) {
	// Security: Validate child belongs to the specified organization (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(childID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if child.OrganizationID != orgID {
		return nil, apperror.NotFound("child")
	}

	// Validate contract belongs to the child
	contract, err := s.store.FindContractByID(contractID)
	if err != nil {
		return nil, apperror.NotFound("contract")
	}
	if contract.ChildID != childID {
		return nil, apperror.NotFound("contract")
	}

	// Update fields if provided
	if req.From != nil {
		contract.From = *req.From
	}
	if req.To != nil {
		contract.To = req.To
	}
	if req.Attributes != nil {
		contract.Attributes = req.Attributes
	}

	// Validate period
	if err := validation.ValidatePeriod(contract.From, contract.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	// Validate no overlap (excluding this contract)
	if err := s.store.Contracts().ValidateNoOverlap(childID, contract.From, contract.To, &contractID); err != nil {
		if errors.Is(err, store.ErrContractOverlap) {
			return nil, apperror.Conflict(err.Error())
		}
		return nil, apperror.Internal("failed to validate contract")
	}

	if err := s.store.UpdateContract(contract); err != nil {
		return nil, apperror.Internal("failed to update contract")
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// DeleteContract deletes a contract, validating it belongs to a child in the specified organization
func (s *ChildService) DeleteContract(ctx context.Context, contractID, childID, orgID uint) error {
	// Security: Validate child belongs to the specified organization (use minimal query - no preloads needed)
	child, err := s.store.FindByIDMinimal(childID)
	if err != nil {
		return apperror.NotFound("child")
	}
	if child.OrganizationID != orgID {
		return apperror.NotFound("child")
	}

	// Validate contract belongs to the child
	contract, err := s.store.FindContractByID(contractID)
	if err != nil {
		return apperror.NotFound("contract")
	}
	if contract.ChildID != childID {
		return apperror.NotFound("contract")
	}

	if err := s.store.DeleteContract(contractID); err != nil {
		return apperror.Internal("failed to delete contract")
	}
	return nil
}

// CalculateFunding calculates government funding for all children with active contracts on the given date
func (s *ChildService) CalculateFunding(ctx context.Context, orgID uint, date time.Time) (*models.ChildrenFundingResponse, error) {
	// Get organization to determine state
	org, err := s.orgStore.FindByID(orgID)
	if err != nil {
		return nil, apperror.NotFound("organization")
	}

	// Get children with active contracts on this date
	children, err := s.store.FindByOrganizationWithContractOn(orgID, date)
	if err != nil {
		return nil, apperror.Internal("failed to fetch children")
	}

	response := &models.ChildrenFundingResponse{
		Date:     date,
		Children: make([]models.ChildFundingResponse, 0, len(children)),
	}

	// Look up funding by organization's state (0 = all periods, needed to find matching period for date)
	funding, err := s.fundingStore.FindByStateWithDetails(org.State, 0)
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
				MatchedAttributes:   []string{},
				UnmatchedAttributes: uniqueStrings(contract.Attributes),
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

		childFunding := s.calculateChildFunding(childAge, contract.Attributes, period)
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

// calculateChildFunding calculates funding for a single child based on their age and contract attributes
func (s *ChildService) calculateChildFunding(age int, attributes []string, period *models.GovernmentFundingPeriod) models.ChildFundingResponse {
	result := models.ChildFundingResponse{
		MatchedAttributes:   []string{},
		UnmatchedAttributes: []string{},
	}

	uniqueAttrs := uniqueStrings(attributes)

	// No period covering this date
	if period == nil {
		result.UnmatchedAttributes = uniqueAttrs
		return result
	}

	// Build a map of attribute names to matching properties (filtered by age)
	// A property matches if its name matches the attribute AND the age is within range
	propertyMap := make(map[string]int)
	for _, prop := range period.Properties {
		if prop.MatchesAge(age) {
			// If multiple properties with same name match, sum their payments
			propertyMap[prop.Name] += prop.Payment
		}
	}

	// Match attributes to properties
	totalFunding := 0
	for _, attr := range uniqueAttrs {
		if payment, exists := propertyMap[attr]; exists {
			totalFunding += payment
			result.MatchedAttributes = append(result.MatchedAttributes, attr)
		} else {
			result.UnmatchedAttributes = append(result.UnmatchedAttributes, attr)
		}
	}

	result.Funding = totalFunding
	return result
}

// uniqueStrings returns a slice with duplicate strings removed, preserving order
func uniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(input))
	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// GetAgeDistribution returns age distribution of children with active contracts on the given date
func (s *ChildService) GetAgeDistribution(ctx context.Context, orgID uint, date time.Time) (*models.AgeDistributionResponse, error) {
	// Get children with active contracts on this date
	children, err := s.store.FindByOrganizationWithContractOn(orgID, date)
	if err != nil {
		return nil, apperror.Internal("failed to fetch children")
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
		Date:         date.Format("2006-01-02"),
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
	// Calculate number of years in range
	numYears := maxYear - minYear + 1

	startDate := time.Date(minYear, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(maxYear, 12, 31, 0, 0, 0, 0, time.UTC)

	response := &models.ChildrenContractCountByMonthResponse{
		Period: models.ContractCountPeriod{
			Start: startDate.Format("2006-01-02"),
			End:   endDate.Format("2006-01-02"),
		},
		Years: make([]models.ContractCountByMonthYear, numYears),
	}

	// Initialize yearly data
	for i := 0; i < numYears; i++ {
		response.Years[i] = models.ContractCountByMonthYear{
			Year:   minYear + i,
			Counts: make([]int, 12),
		}
	}

	// Loop through each month and count children with active contracts
	for yearIdx := 0; yearIdx < numYears; yearIdx++ {
		year := minYear + yearIdx

		for month := 1; month <= 12; month++ {
			// Use 15th of the month as sample date
			sampleDate := time.Date(year, time.Month(month), 15, 0, 0, 0, 0, time.UTC)
			count, err := s.store.CountByOrganizationWithContractOn(orgID, sampleDate)
			if err != nil {
				return nil, apperror.Internal("failed to count children for month")
			}
			response.Years[yearIdx].Counts[month-1] = int(count)
		}
	}

	return response, nil
}
