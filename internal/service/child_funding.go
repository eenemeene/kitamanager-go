package service

import (
	"log/slog"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

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
