package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

// CalculateFunding calculates government funding for all children with active contracts on the given date
func (s *ChildService) CalculateFunding(ctx context.Context, orgID uint, date time.Time) (*models.ChildrenFundingResponse, error) {
	org, err := s.orgStore.FindByID(ctx, orgID)
	if err != nil {
		return nil, classifyStoreError(err, "organization")
	}

	children, err := s.store.FindByOrganizationWithActiveOn(ctx, orgID, date)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children")
	}

	funding, err := s.fundingStore.FindByStateWithDetails(ctx, org.State, 0, nil)
	var fundingPeriods []models.GovernmentFundingPeriod
	if err == nil {
		fundingPeriods = funding.Periods
	}

	return calculateFunding(children, fundingPeriods, date), nil
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
	children, err := s.store.FindByOrganizationWithActiveOn(ctx, orgID, date)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children")
	}
	return calculateAgeDistribution(children, date), nil
}

// GetContractPropertiesDistribution returns the distribution of contract properties
// for children with active contracts on the given date
func (s *ChildService) GetContractPropertiesDistribution(ctx context.Context, orgID uint, date time.Time) (*models.ContractPropertiesDistributionResponse, error) {
	children, err := s.store.FindByOrganizationWithActiveOn(ctx, orgID, date)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch children")
	}

	org, err := s.orgStore.FindByID(ctx, orgID)
	if err != nil {
		return calculateContractPropertiesDistribution(children, nil, date), nil
	}

	funding, err := s.fundingStore.FindByStateWithDetails(ctx, org.State, 0, nil)
	if err != nil {
		return calculateContractPropertiesDistribution(children, nil, date), nil
	}

	return calculateContractPropertiesDistribution(children, funding.Periods, date), nil
}
