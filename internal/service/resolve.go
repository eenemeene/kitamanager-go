package service

import (
	"context"
	"errors"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// resolveSection looks up a section by name (with caching) and auto-creates it if missing.
func resolveSection(ctx context.Context, sectionStore store.SectionStorer, sectionName *string, orgID uint, cache map[string]uint) (uint, error) {
	if sectionName == nil || *sectionName == "" {
		sec, err := sectionStore.FindDefaultSection(ctx, orgID)
		if err != nil {
			return 0, apperror.InternalWrap(err, "no default section found for organization")
		}
		return sec.ID, nil
	}
	name := *sectionName
	if id, ok := cache[name]; ok {
		return id, nil
	}
	sec, err := sectionStore.FindByNameAndOrg(ctx, name, orgID)
	if err == nil {
		cache[name] = sec.ID
		return sec.ID, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return 0, apperror.InternalWrap(err, "failed to look up section")
	}
	newSec := &models.Section{
		OrganizationID: orgID,
		Name:           name,
	}
	if err := sectionStore.Create(ctx, newSec); err != nil {
		return 0, apperror.InternalWrap(err, "failed to auto-create section")
	}
	cache[name] = newSec.ID
	return newSec.ID, nil
}
