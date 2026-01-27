package importer

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// ErrGovernmentFundingExists is returned when attempting to import a government funding that already exists.
var ErrGovernmentFundingExists = errors.New("government funding already exists")

// GovernmentFundingImporter handles importing government funding data from YAML files.
type GovernmentFundingImporter struct {
	db           *gorm.DB
	fundingStore *store.GovernmentFundingStore
}

// NewGovernmentFundingImporter creates a new GovernmentFundingImporter.
func NewGovernmentFundingImporter(db *gorm.DB, fundingStore *store.GovernmentFundingStore) *GovernmentFundingImporter {
	return &GovernmentFundingImporter{
		db:           db,
		fundingStore: fundingStore,
	}
}

// ImportGovernmentFundingFromFile reads a YAML file and imports the government funding data.
// If a government funding with the given name already exists, it returns ErrGovernmentFundingExists
// and the existing government funding's ID.
func (i *GovernmentFundingImporter) ImportGovernmentFundingFromFile(filePath, fundingName string) (uint, error) {
	// Check if government funding already exists
	existingFunding, err := i.fundingStore.FindByName(fundingName)
	if err == nil && existingFunding != nil {
		slog.Info("Government funding already exists, skipping import", "name", fundingName, "id", existingFunding.ID)
		return existingFunding.ID, ErrGovernmentFundingExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, fmt.Errorf("failed to check existing government funding: %w", err)
	}

	// Read YAML file
	// #nosec G304 -- filePath is from trusted configuration, not user input
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse YAML
	var periods []YAMLGovernmentFundingPeriod
	if err := yaml.Unmarshal(data, &periods); err != nil {
		return 0, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Import in a transaction
	var fundingID uint
	err = i.db.Transaction(func(tx *gorm.DB) error {
		// Create government funding
		funding := &models.GovernmentFunding{
			Name: fundingName,
		}
		if err := tx.Create(funding).Error; err != nil {
			return fmt.Errorf("failed to create government funding: %w", err)
		}
		fundingID = funding.ID

		// Import periods
		for _, yamlPeriod := range periods {
			period, err := i.importPeriod(tx, fundingID, yamlPeriod)
			if err != nil {
				return fmt.Errorf("failed to import period: %w", err)
			}

			// Import properties from entries (flatten age-based entries to properties with age ranges)
			for _, yamlEntry := range yamlPeriod.Entries {
				if err := i.importPropertiesFromEntry(tx, period.ID, yamlEntry); err != nil {
					return fmt.Errorf("failed to import properties: %w", err)
				}
			}
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	slog.Info("Government funding imported successfully", "name", fundingName, "id", fundingID, "periods", len(periods))
	return fundingID, nil
}

func (i *GovernmentFundingImporter) importPeriod(tx *gorm.DB, fundingID uint, yamlPeriod YAMLGovernmentFundingPeriod) (*models.GovernmentFundingPeriod, error) {
	from, err := parseDate(yamlPeriod.From)
	if err != nil {
		return nil, fmt.Errorf("failed to parse from date: %w", err)
	}

	var to *time.Time
	// Empty string or far-future date (2060-01-01) indicates ongoing period
	if yamlPeriod.To != "" && yamlPeriod.To != "2060-01-01" {
		toDate, err := parseDate(yamlPeriod.To)
		if err != nil {
			return nil, fmt.Errorf("failed to parse to date: %w", err)
		}
		to = &toDate
	}

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: fundingID,
		From:                from,
		To:                  to,
		Comment:             strings.TrimSpace(yamlPeriod.Comment),
	}

	if err := tx.Create(period).Error; err != nil {
		return nil, err
	}

	return period, nil
}

// importPropertiesFromEntry converts a YAML entry (with age range) to properties with age filters.
// Both YAML and code use inclusive age ranges: [0,1] means ages 0 AND 1.
func (i *GovernmentFundingImporter) importPropertiesFromEntry(tx *gorm.DB, periodID uint, yamlEntry YAMLGovernmentFundingEntry) error {
	minAge := yamlEntry.Age[0]
	maxAge := yamlEntry.Age[1]

	// Import properties with the age range from the entry
	for name, yamlProp := range yamlEntry.Properties {
		property := &models.GovernmentFundingProperty{
			PeriodID:    periodID,
			Name:        name,
			Payment:     euroToCents(yamlProp.Payment),
			Requirement: yamlProp.Requirement,
			MinAge:      &minAge,
			MaxAge:      &maxAge,
			Comment:     yamlProp.Comment,
		}

		if err := tx.Create(property).Error; err != nil {
			return err
		}
	}

	return nil
}

// parseDate parses a date string in YYYY-MM-DD format.
func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

// euroToCents converts a EUR amount to cents.
func euroToCents(eur float64) int {
	return int(math.Round(eur * 100))
}
