package service

import (
	"strings"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
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

// PersonOwned is implemented by contracts that belong to a person (child/employee).
type PersonOwned interface {
	GetPersonID() uint
}

// verifyContractOwnership checks that a contract belongs to the expected person.
// Returns apperror.NotFound if contract is nil or belongs to a different person.
func verifyContractOwnership(contract PersonOwned, personID uint) error {
	if contract == nil || contract.GetPersonID() != personID {
		return apperror.NotFound("contract")
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
