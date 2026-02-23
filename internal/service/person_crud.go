package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// personList fetches all entities with pagination.
func personList[T any, R any](
	ctx context.Context,
	findAll func(ctx context.Context, limit, offset int) ([]T, int64, error),
	toResponse func(*T) R,
	resourceName string,
	limit, offset int,
) ([]R, int64, error) {
	items, total, err := findAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch "+resourceName)
	}
	return toResponseList(items, toResponse), total, nil
}

// personGetByID fetches an entity by ID scoped to the given organization.
func personGetByID[T any, R any](
	ctx context.Context,
	findByIDAndOrg func(ctx context.Context, id, orgID uint) (*T, error),
	toResponse func(*T) R,
	id, orgID uint,
	resourceName string,
) (*R, error) {
	entity, err := findByIDAndOrg(ctx, id, orgID)
	if err != nil {
		return nil, classifyStoreError(err, resourceName)
	}
	resp := toResponse(entity)
	return &resp, nil
}

// personCreate validates person fields and creates entity.
func personCreate[T any, R any](
	ctx context.Context,
	fields *validation.PersonCreateFields,
	buildEntity func(person models.Person) *T,
	createFn func(ctx context.Context, entity *T) error,
	toResponse func(*T) R,
	orgID uint,
	resourceName string,
) (*R, error) {
	person, err := validation.ValidatePersonCreate(fields)
	if err != nil {
		return nil, err
	}

	entity := buildEntity(models.Person{
		OrganizationID: orgID,
		FirstName:      person.FirstName,
		LastName:       person.LastName,
		Gender:         person.Gender,
		Birthdate:      person.Birthdate,
	})

	if err := createFn(ctx, entity); err != nil {
		return nil, apperror.InternalWrap(err, "failed to create "+resourceName)
	}

	resp := toResponse(entity)
	return &resp, nil
}

// personUpdate validates and applies person field updates, scoped to the given organization.
// The update and reload are wrapped in a transaction to ensure consistent reads.
func personUpdate[T any, R any](
	ctx context.Context,
	transactor store.Transactor,
	findByIDAndOrg func(ctx context.Context, id, orgID uint) (*T, error),
	getPerson func(*T) *models.Person,
	updateFn func(ctx context.Context, entity *T) error,
	toResponse func(*T) R,
	id, orgID uint,
	fields personUpdateFields,
	resourceName string,
) (*R, error) {
	entity, err := findByIDAndOrg(ctx, id, orgID)
	if err != nil {
		return nil, classifyStoreError(err, resourceName)
	}

	if err := applyPersonUpdates(getPerson(entity), fields); err != nil {
		return nil, err
	}

	var resp R
	if err := transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := updateFn(txCtx, entity); err != nil {
			return apperror.InternalWrap(err, "failed to update "+resourceName)
		}

		// Reload to get fresh associations within the same transaction
		reloaded, err := findByIDAndOrg(txCtx, id, orgID)
		if err != nil {
			return apperror.InternalWrap(err, "failed to reload "+resourceName+" after update")
		}
		resp = toResponse(reloaded)
		return nil
	}); err != nil {
		return nil, err
	}

	return &resp, nil
}

// importPersonItem describes one person in an import batch for validation/upsert.
type importPersonItem struct {
	Index     int // 1-based position for error messages
	FirstName string
	LastName  string
	Gender    string
	Birthdate time.Time
}

// personImportUpsert validates person fields, then looks up or creates the entity
// (matched by name+birthdate+org). On match, gender is updated and old contracts
// are deleted. Returns the entity's ID.
func personImportUpsert[T any](
	ctx context.Context,
	item importPersonItem,
	resourceName string,
	findByNameBirthdateAndOrg func(ctx context.Context, firstName, lastName string, birthdate time.Time, orgID uint) (*T, error),
	getPerson func(*T) *models.Person,
	getID func(*T) uint,
	updateFn func(ctx context.Context, entity *T) error,
	deleteContracts func(ctx context.Context, entityID uint) error,
	buildNew func(person models.Person) *T,
	createFn func(ctx context.Context, entity *T) error,
	orgID uint,
) (uint, error) {
	if item.FirstName == "" || item.LastName == "" {
		return 0, apperror.BadRequest(fmt.Sprintf("%s %d: first_name and last_name are required", resourceName, item.Index))
	}
	if item.Birthdate.IsZero() {
		return 0, apperror.BadRequest(fmt.Sprintf("%s %d (%s %s): birthdate is required", resourceName, item.Index, item.FirstName, item.LastName))
	}

	existing, err := findByNameBirthdateAndOrg(ctx, item.FirstName, item.LastName, item.Birthdate, orgID)
	if err == nil {
		getPerson(existing).Gender = item.Gender
		if err := updateFn(ctx, existing); err != nil {
			return 0, apperror.InternalWrap(err, fmt.Sprintf("failed to update %s %s %s", resourceName, item.FirstName, item.LastName))
		}
		if err := deleteContracts(ctx, getID(existing)); err != nil {
			return 0, apperror.InternalWrap(err, "failed to clear existing contracts")
		}
		return getID(existing), nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return 0, apperror.InternalWrap(err, "failed to look up "+resourceName)
	}

	entity := buildNew(models.Person{
		OrganizationID: orgID,
		FirstName:      item.FirstName,
		LastName:       item.LastName,
		Gender:         item.Gender,
		Birthdate:      item.Birthdate,
	})
	if err := createFn(ctx, entity); err != nil {
		return 0, apperror.InternalWrap(err, fmt.Sprintf("failed to create %s %s %s", resourceName, item.FirstName, item.LastName))
	}
	return getID(entity), nil
}

// personDelete validates org ownership at DB level and deletes within a transaction.
func personDelete[T any](
	ctx context.Context,
	transactor store.Transactor,
	findByIDAndOrg func(ctx context.Context, id, orgID uint) (*T, error),
	deleteFn func(ctx context.Context, id uint) error,
	id, orgID uint,
	resourceName string,
) error {
	return transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if _, err := findByIDAndOrg(txCtx, id, orgID); err != nil {
			return classifyStoreError(err, resourceName)
		}
		if err := deleteFn(txCtx, id); err != nil {
			return apperror.InternalWrap(err, "failed to delete "+resourceName)
		}
		return nil
	})
}
