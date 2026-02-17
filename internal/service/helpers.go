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

// classifyStoreError returns NotFound for store.ErrNotFound, InternalWrap for all other errors.
func classifyStoreError(err error, resourceName string) error {
	if errors.Is(err, store.ErrNotFound) {
		return apperror.NotFound(resourceName)
	}
	return apperror.InternalWrap(err, "failed to fetch "+resourceName)
}

// OrgOwned is implemented by entities that belong to an organization.
type OrgOwned interface {
	GetOrganizationID() uint
}

// verifyOrgOwnership checks that a looked-up entity belongs to the expected organization.
// Returns apperror.NotFound if entity is nil or belongs to a different org.
func verifyOrgOwnership(entity OrgOwned, orgID uint, resourceName string) error {
	if entity == nil || entity.GetOrganizationID() != orgID {
		return apperror.NotFound(resourceName)
	}
	return nil
}

// verifyRecordOwnership checks that a period record belongs to the expected owner.
// Returns apperror.NotFound if record is nil or belongs to a different owner.
func verifyRecordOwnership(record models.PeriodRecord, ownerID uint, resourceName string) error { //nolint:unparam // resourceName will vary as more record types adopt this
	if record == nil || record.GetOwnerID() != ownerID {
		return apperror.NotFound(resourceName)
	}
	return nil
}

// toResponseList converts a slice of items to a slice of responses using the given converter function.
func toResponseList[T any, R any](items []T, convert func(*T) R) []R {
	result := make([]R, len(items))
	for i := range items {
		result[i] = convert(&items[i])
	}
	return result
}

// validateRequiredName trims whitespace and validates that name is not empty.
func validateRequiredName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if validation.IsWhitespaceOnly(name) {
		return "", apperror.BadRequest("name cannot be empty or whitespace only")
	}
	return name, nil
}

// validateSectionOrg validates that a section exists and belongs to the given organization.
func validateSectionOrg(ctx context.Context, sectionStore store.SectionStorer, sectionID, orgID uint) error {
	section, err := sectionStore.FindByID(ctx, sectionID)
	if err != nil {
		return apperror.BadRequest("section not found")
	}
	if section.OrganizationID != orgID {
		return apperror.BadRequest("section does not belong to this organization")
	}
	return nil
}

// validateOptionalSectionOrg validates that a section exists and belongs to the given
// organization when sectionID is non-nil. Returns nil if sectionID is nil.
func validateOptionalSectionOrg(ctx context.Context, sectionStore store.SectionStorer, sectionID *uint, orgID uint) error {
	if sectionID == nil {
		return nil
	}
	return validateSectionOrg(ctx, sectionStore, *sectionID, orgID)
}

// periodsOverlap checks if two date ranges overlap.
// A period with nil To extends indefinitely into the future.
func periodsOverlap(from1 time.Time, to1 *time.Time, from2 time.Time, to2 *time.Time) bool {
	if to1 != nil && to1.Before(from2) {
		return false
	}
	if to2 != nil && to2.Before(from1) {
		return false
	}
	return true
}

// personUpdateFields describes the optional fields in a person update request.
type personUpdateFields struct {
	FirstName *string
	LastName  *string
	Gender    *string
	Birthdate *string
}

// applyPersonUpdates validates and applies person field updates to a Person model.
func applyPersonUpdates(person *models.Person, fields personUpdateFields) error {
	if fields.FirstName != nil {
		trimmed, err := validation.ValidateAndTrimName(*fields.FirstName, "first_name")
		if err != nil {
			return err
		}
		person.FirstName = trimmed
	}
	if fields.LastName != nil {
		trimmed, err := validation.ValidateAndTrimName(*fields.LastName, "last_name")
		if err != nil {
			return err
		}
		person.LastName = trimmed
	}
	if fields.Gender != nil {
		if err := validation.ValidateGender(*fields.Gender); err != nil {
			return err
		}
		person.Gender = *fields.Gender
	}
	if fields.Birthdate != nil {
		bd, err := validation.ParseAndValidateBirthdate(*fields.Birthdate)
		if err != nil {
			return err
		}
		person.Birthdate = bd
	}
	return nil
}
