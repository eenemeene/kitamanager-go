package importer

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"errors"
	"net/http"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// ImportResult contains statistics about what the import changed.
type ImportResult struct {
	FundingID         uint
	Created           bool // true if the funding was newly created
	PeriodsCreated    int
	PeriodsUpdated    int
	PropertiesCreated int
	PropertiesUpdated int
	PropertiesDeleted int
}

// GovernmentFundingImporter handles importing government funding data from YAML files.
type GovernmentFundingImporter struct {
	service    *service.GovernmentFundingService
	transactor store.Transactor
}

// NewGovernmentFundingImporter creates a new GovernmentFundingImporter.
func NewGovernmentFundingImporter(svc *service.GovernmentFundingService, transactor store.Transactor) *GovernmentFundingImporter {
	return &GovernmentFundingImporter{
		service:    svc,
		transactor: transactor,
	}
}

// ImportGovernmentFundingFromFile reads a YAML file and imports the government funding data.
func (i *GovernmentFundingImporter) ImportGovernmentFundingFromFile(ctx context.Context, filePath, state string) (*ImportResult, error) {
	if !models.IsValidState(state) {
		return nil, fmt.Errorf("invalid state: %s", state)
	}

	// #nosec G304 -- filePath is from trusted configuration, not user input
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return i.ImportGovernmentFunding(ctx, data, state)
}

// ImportGovernmentFunding imports government funding data from YAML bytes.
// If the funding already exists, it performs an incremental update (add/modify periods and properties).
// If it doesn't exist, it creates the funding with all periods and properties.
func (i *GovernmentFundingImporter) ImportGovernmentFunding(ctx context.Context, data []byte, state string) (*ImportResult, error) {
	if !models.IsValidState(state) {
		return nil, fmt.Errorf("invalid state: %s", state)
	}

	var yamlPeriods []YAMLGovernmentFundingPeriod
	if err := yaml.Unmarshal(data, &yamlPeriods); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Check if funding already exists
	existing, err := i.service.GetByState(ctx, state)
	if err != nil && !isNotFoundError(err) {
		return nil, fmt.Errorf("failed to check existing government funding: %w", err)
	}

	var result ImportResult
	if err := i.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if existing == nil {
			return i.freshImport(txCtx, state, yamlPeriods, &result)
		}
		return i.incrementalImport(txCtx, existing, yamlPeriods, &result)
	}); err != nil {
		return nil, err
	}

	slog.Info("Government funding import completed",
		"state", state,
		"funding_id", result.FundingID,
		"created", result.Created,
		"periods_created", result.PeriodsCreated,
		"periods_updated", result.PeriodsUpdated,
		"properties_created", result.PropertiesCreated,
		"properties_updated", result.PropertiesUpdated,
		"properties_deleted", result.PropertiesDeleted,
	)

	return &result, nil
}

// freshImport creates a new funding with all periods and properties.
func (i *GovernmentFundingImporter) freshImport(ctx context.Context, state string, yamlPeriods []YAMLGovernmentFundingPeriod, result *ImportResult) error {
	stateName := strings.ToUpper(state[:1]) + state[1:]
	resp, err := i.service.Create(ctx, &models.GovernmentFundingCreateRequest{
		Name:  stateName + " Kita-Förderung",
		State: state,
	})
	if err != nil {
		return fmt.Errorf("failed to create government funding: %w", err)
	}

	result.FundingID = resp.ID
	result.Created = true

	for _, yp := range yamlPeriods {
		if err := i.createPeriodWithProperties(ctx, resp.ID, yp, result); err != nil {
			return err
		}
	}

	return nil
}

// incrementalImport diffs YAML against existing data and applies changes.
func (i *GovernmentFundingImporter) incrementalImport(ctx context.Context, existing *models.GovernmentFunding, yamlPeriods []YAMLGovernmentFundingPeriod, result *ImportResult) error {
	// Load full details for diffing
	funding, err := i.service.GetByStateWithDetails(ctx, existing.State)
	if err != nil {
		return fmt.Errorf("failed to load existing funding details: %w", err)
	}

	result.FundingID = funding.ID

	// Build lookup: from date → existing period
	existingByFrom := make(map[string]*models.GovernmentFundingPeriod, len(funding.Periods))
	for idx := range funding.Periods {
		key := funding.Periods[idx].From.Format(models.DateFormat)
		existingByFrom[key] = &funding.Periods[idx]
	}

	for _, yp := range yamlPeriods {
		from, err := parseDate(yp.From)
		if err != nil {
			return fmt.Errorf("failed to parse from date %q: %w", yp.From, err)
		}
		fromKey := from.Format(models.DateFormat)

		existingPeriod, found := existingByFrom[fromKey]
		if !found {
			// New period
			if err := i.createPeriodWithProperties(ctx, funding.ID, yp, result); err != nil {
				return err
			}
			continue
		}

		// Existing period — check for updates
		if err := i.updatePeriodIfChanged(ctx, funding.ID, existingPeriod, yp, result); err != nil {
			return err
		}
	}

	return nil
}

// createPeriodWithProperties creates a period and all its properties via the service layer.
func (i *GovernmentFundingImporter) createPeriodWithProperties(ctx context.Context, fundingID uint, yp YAMLGovernmentFundingPeriod, result *ImportResult) error {
	from, err := parseDate(yp.From)
	if err != nil {
		return fmt.Errorf("failed to parse from date: %w", err)
	}

	to, err := parseToDate(yp.To)
	if err != nil {
		return fmt.Errorf("failed to parse to date: %w", err)
	}

	if yp.FullTimeWeeklyHours <= 0 {
		return fmt.Errorf("full_time_weekly_hours is required and must be > 0")
	}

	periodResp, err := i.service.CreatePeriod(ctx, fundingID, &models.GovernmentFundingPeriodCreateRequest{
		From:                from,
		To:                  to,
		FullTimeWeeklyHours: yp.FullTimeWeeklyHours,
		Comment:             strings.TrimSpace(yp.Comment),
	})
	if err != nil {
		return fmt.Errorf("failed to create period (from %s): %w", yp.From, err)
	}
	result.PeriodsCreated++

	for _, entry := range yp.Entries {
		for _, prop := range entry.Properties {
			if err := i.createProperty(ctx, fundingID, periodResp.ID, entry.Age, prop, result); err != nil {
				return err
			}
		}
	}

	return nil
}

// updatePeriodIfChanged compares YAML period with existing and updates if needed.
func (i *GovernmentFundingImporter) updatePeriodIfChanged(ctx context.Context, fundingID uint, existing *models.GovernmentFundingPeriod, yp YAMLGovernmentFundingPeriod, result *ImportResult) error {
	to, err := parseToDate(yp.To)
	if err != nil {
		return fmt.Errorf("failed to parse to date: %w", err)
	}
	comment := strings.TrimSpace(yp.Comment)

	// Check if period metadata changed
	periodChanged := !timePtrEqual(existing.To, to)

	if existing.FullTimeWeeklyHours != yp.FullTimeWeeklyHours {
		periodChanged = true
	}
	if existing.Comment != comment {
		periodChanged = true
	}

	if periodChanged {
		_, err := i.service.UpdatePeriod(ctx, existing.ID, fundingID, &models.GovernmentFundingPeriodUpdateRequest{
			To:                  to,
			FullTimeWeeklyHours: &yp.FullTimeWeeklyHours,
			Comment:             &comment,
		})
		if err != nil {
			return fmt.Errorf("failed to update period (from %s): %w", yp.From, err)
		}
		result.PeriodsUpdated++
	}

	// Diff properties
	return i.diffProperties(ctx, fundingID, existing, yp.Entries, result)
}

// diffProperties compares YAML properties against existing and applies changes.
func (i *GovernmentFundingImporter) diffProperties(ctx context.Context, fundingID uint, period *models.GovernmentFundingPeriod, entries []YAMLGovernmentFundingEntry, result *ImportResult) error {
	// Build lookup of existing properties by composite key
	existingByKey := make(map[string]*models.GovernmentFundingProperty, len(period.Properties))
	for idx := range period.Properties {
		p := &period.Properties[idx]
		key := propertyMatchKey(p.Key, p.Value, p.MinAge, p.MaxAge)
		existingByKey[key] = p
	}

	// Track which existing properties we see in YAML
	seen := make(map[string]bool, len(period.Properties))

	for _, entry := range entries {
		minAge := entry.Age[0]
		maxAge := entry.Age[1]

		for _, yProp := range entry.Properties {
			key := propertyMatchKey(
				strings.TrimSpace(yProp.Key),
				strings.TrimSpace(yProp.Value),
				&minAge, &maxAge,
			)
			seen[key] = true

			existing, found := existingByKey[key]
			if !found {
				// New property
				if err := i.createProperty(ctx, fundingID, period.ID, entry.Age, yProp, result); err != nil {
					return err
				}
				continue
			}

			// Check if property changed
			if err := i.updatePropertyIfChanged(ctx, fundingID, period.ID, existing, yProp, result); err != nil {
				return err
			}
		}
	}

	// Delete properties that exist in DB but not in YAML
	for key, prop := range existingByKey {
		if !seen[key] {
			if err := i.service.DeleteProperty(ctx, prop.ID, period.ID, fundingID); err != nil {
				return fmt.Errorf("failed to delete property %d: %w", prop.ID, err)
			}
			result.PropertiesDeleted++
		}
	}

	return nil
}

// createProperty creates a single property via the service layer.
func (i *GovernmentFundingImporter) createProperty(ctx context.Context, fundingID, periodID uint, age [2]int, yProp YAMLGovernmentFundingProperty, result *ImportResult) error {
	label := strings.TrimSpace(yProp.Label)
	if label == "" {
		label = formatLabel(yProp.Value)
	}

	minAge := age[0]
	maxAge := age[1]

	_, err := i.service.CreateProperty(ctx, fundingID, periodID, &models.GovernmentFundingPropertyCreateRequest{
		Key:                 strings.TrimSpace(yProp.Key),
		Value:               strings.TrimSpace(yProp.Value),
		Label:               label,
		Payment:             euroToCents(yProp.Payment),
		Requirement:         yProp.Requirement,
		MinAge:              &minAge,
		MaxAge:              &maxAge,
		Comment:             strings.TrimSpace(yProp.Comment),
		ApplyToAllContracts: yProp.ApplyToAllContracts,
	})
	if err != nil {
		return fmt.Errorf("failed to create property %s/%s: %w", yProp.Key, yProp.Value, err)
	}
	result.PropertiesCreated++
	return nil
}

// updatePropertyIfChanged compares YAML property with existing and updates if needed.
func (i *GovernmentFundingImporter) updatePropertyIfChanged(ctx context.Context, fundingID, periodID uint, existing *models.GovernmentFundingProperty, yProp YAMLGovernmentFundingProperty, result *ImportResult) error {
	label := strings.TrimSpace(yProp.Label)
	if label == "" {
		label = formatLabel(yProp.Value)
	}
	payment := euroToCents(yProp.Payment)
	comment := strings.TrimSpace(yProp.Comment)

	changed := existing.Label != label

	if existing.Payment != payment {
		changed = true
	}
	if existing.Requirement != yProp.Requirement {
		changed = true
	}
	if existing.Comment != comment {
		changed = true
	}
	if existing.ApplyToAllContracts != yProp.ApplyToAllContracts {
		changed = true
	}

	if !changed {
		return nil
	}

	_, err := i.service.UpdateProperty(ctx, existing.ID, periodID, fundingID, &models.GovernmentFundingPropertyUpdateRequest{
		Label:               &label,
		Payment:             &payment,
		Requirement:         &yProp.Requirement,
		Comment:             &comment,
		ApplyToAllContracts: &yProp.ApplyToAllContracts,
		MinAge:              existing.MinAge,
		MaxAge:              existing.MaxAge,
	})
	if err != nil {
		return fmt.Errorf("failed to update property %d: %w", existing.ID, err)
	}
	result.PropertiesUpdated++
	return nil
}

// propertyMatchKey builds a composite key for matching properties.
func propertyMatchKey(key, value string, minAge, maxAge *int) string {
	min, max := -1, -1
	if minAge != nil {
		min = *minAge
	}
	if maxAge != nil {
		max = *maxAge
	}
	return fmt.Sprintf("%s|%s|%d|%d", key, value, min, max)
}

// timePtrEqual compares two *time.Time values, handling nil.
func timePtrEqual(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(*b)
}

// parseToDate parses the "to" date, treating empty or far-future (2060-01-01) as nil (ongoing).
func parseToDate(s string) (*time.Time, error) {
	if s == "" || s == "2060-01-01" {
		return nil, nil
	}
	t, err := parseDate(s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// parseDate parses a date string in YYYY-MM-DD format.
func parseDate(s string) (time.Time, error) {
	return time.Parse(models.DateFormat, s)
}

// euroToCents converts a EUR amount to cents with overflow protection.
func euroToCents(eur float64) int {
	cents := math.Round(eur * 100)
	if cents > math.MaxInt32 || cents < math.MinInt32 {
		slog.Error("Currency value out of range", "eur", eur, "cents", cents)
		return 0
	}
	return int(cents)
}

// isNotFoundError checks if the error is a 404 not-found from the apperror package.
func isNotFoundError(err error) bool {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		return appErr.Code == http.StatusNotFound
	}
	return false
}

// formatLabel generates a human-readable label from a property value.
// It splits on underscores, slashes, and hyphens and capitalizes each word.
func formatLabel(value string) string {
	words := strings.FieldsFunc(value, func(r rune) bool {
		return r == '_' || r == '/' || r == '-'
	})
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
