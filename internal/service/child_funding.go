package service

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// CalculateFunding calculates government funding for all children with active contracts on the given date
func (s *ChildService) CalculateFunding(ctx context.Context, orgID uint, date time.Time) (*models.ChildrenFundingResponse, error) {
	// Get organization to determine state
	org, err := s.orgStore.FindByID(ctx, orgID)
	if err != nil {
		return nil, classifyStoreError(err, "organization")
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
	period := findPeriodForDate(funding.Periods, date)

	// Set weekly hours basis from the funding period
	if period != nil {
		response.WeeklyHoursBasis = period.FullTimeWeeklyHours
	}

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

// findPeriodForDate finds the funding period that covers the given date (package-level for reuse).
// Logs a critical warning if multiple periods match (overlapping date ranges).
func findPeriodForDate(periods []models.GovernmentFundingPeriod, date time.Time) *models.GovernmentFundingPeriod {
	var matched *models.GovernmentFundingPeriod
	matchCount := 0
	for i := range periods {
		period := &periods[i]
		if period.IsActiveOn(date) {
			matchCount++
			if matched == nil {
				matched = period
			}
		}
	}
	if matchCount > 1 {
		slog.Error("overlapping funding periods detected",
			"date", date.Format("2006-01-02"),
			"matches", matchCount,
		)
	}
	return matched
}

// matchFundingProperties returns the funding properties that match a child's age and
// contract properties. This is the single source of truth for funding property matching.
func matchFundingProperties(age int, props models.ContractProperties, period *models.GovernmentFundingPeriod) []*models.GovernmentFundingProperty {
	if period == nil {
		return nil
	}
	var matched []*models.GovernmentFundingProperty
	for i := range period.Properties {
		fp := &period.Properties[i]
		if fp.MatchesAge(age) && props.HasValue(fp.Key, fp.Value) {
			matched = append(matched, fp)
		}
	}
	return matched
}

// sumChildFundingMatch returns total payment (cents) and requirement for a child
// by matching their contract properties against government funding properties.
func sumChildFundingMatch(age int, props models.ContractProperties, period *models.GovernmentFundingPeriod) (payment int, requirement float64) {
	for _, fp := range matchFundingProperties(age, props, period) {
		payment += fp.Payment
		requirement += fp.Requirement
	}
	return
}

// sumChildRequirement calculates the total requirement for a child based on their age and contract properties.
func sumChildRequirement(age int, props models.ContractProperties, period *models.GovernmentFundingPeriod) float64 {
	_, req := sumChildFundingMatch(age, props, period)
	return req
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

	// Single pass: accumulate totals and track matched properties
	matches := matchFundingProperties(age, properties, period)
	matchedSet := make(map[string]bool, len(matches))
	for _, fp := range matches {
		result.Funding += fp.Payment
		result.Requirement += fp.Requirement
		kvKey := fp.Key + ":" + fp.Value
		if !matchedSet[kvKey] {
			matchedSet[kvKey] = true
			result.MatchedProperties = append(result.MatchedProperties, models.ChildFundingMatchedProp{
				Key:   fp.Key,
				Value: fp.Value,
			})
		}
	}

	// Find unmatched contract properties
	for _, kv := range contractKeyValues {
		kvKey := kv.Key + ":" + kv.Value
		if !matchedSet[kvKey] {
			result.UnmatchedProperties = append(result.UnmatchedProperties, kv)
		}
	}

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

// GetContractPropertiesDistribution returns the distribution of contract properties
// for children with active contracts on the given date
func (s *ChildService) GetContractPropertiesDistribution(ctx context.Context, orgID uint, date time.Time) (*models.ContractPropertiesDistributionResponse, error) {
	// Get children with active contracts on this date
	children, err := s.store.FindByOrganizationWithActiveOn(ctx, orgID, date)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children")
	}

	// Build label map from funding properties: "key:value" → label
	labelMap := s.buildFundingLabelMap(ctx, orgID)

	// Aggregate: key -> value -> count
	distribution := make(map[string]map[string]int)
	totalChildren := len(children)

	for _, child := range children {
		if len(child.Contracts) == 0 {
			continue
		}
		for _, contract := range child.Contracts {
			if contract.Properties == nil {
				continue
			}
			for key := range contract.Properties {
				values := contract.Properties.GetAllValues(key)
				for _, value := range values {
					if distribution[key] == nil {
						distribution[key] = make(map[string]int)
					}
					distribution[key][value]++
				}
			}
		}
	}

	// Flatten to sorted slice
	properties := make([]models.ContractPropertyCount, 0)
	// Collect and sort keys
	keys := make([]string, 0, len(distribution))
	for key := range distribution {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		values := make([]string, 0, len(distribution[key]))
		for value := range distribution[key] {
			values = append(values, value)
		}
		sort.Strings(values)
		for _, value := range values {
			properties = append(properties, models.ContractPropertyCount{
				Key:   key,
				Value: value,
				Label: labelMap[key+":"+value],
				Count: distribution[key][value],
			})
		}
	}

	return &models.ContractPropertiesDistributionResponse{
		Date:          date.Format(models.DateFormat),
		TotalChildren: totalChildren,
		Properties:    properties,
	}, nil
}

// buildFundingLabelMap looks up funding configuration for the org and returns
// a map of "key:value" → label for all funding properties across all periods.
func (s *ChildService) buildFundingLabelMap(ctx context.Context, orgID uint) map[string]string {
	labelMap := make(map[string]string)

	org, err := s.orgStore.FindByID(ctx, orgID)
	if err != nil || org.State == "" {
		return labelMap
	}

	funding, err := s.fundingStore.FindByStateWithDetails(ctx, org.State, 0, nil)
	if err != nil {
		return labelMap
	}

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

// intPtr returns a pointer to an int
func intPtr(i int) *int {
	return &i
}
