package service

import (
	"strings"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

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
