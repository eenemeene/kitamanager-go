package service

import (
	"context"
	"strings"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// ChildNoteService handles business logic for child note operations
type ChildNoteService struct {
	store      store.ChildNoteStorer
	childStore store.ChildStorer
}

// NewChildNoteService creates a new child note service
func NewChildNoteService(store store.ChildNoteStorer, childStore store.ChildStorer) *ChildNoteService {
	return &ChildNoteService{
		store:      store,
		childStore: childStore,
	}
}

// List returns a paginated list of notes for a child, optionally filtered by category.
func (s *ChildNoteService) List(ctx context.Context, childID, orgID uint, category string, limit, offset int) ([]models.ChildNoteResponse, int64, error) {
	// Verify child belongs to org
	child, err := s.childStore.FindByIDMinimal(childID)
	if err != nil {
		return nil, 0, apperror.NotFound("child")
	}
	if child.OrganizationID != orgID {
		return nil, 0, apperror.NotFound("child")
	}

	var notes []models.ChildNote
	var total int64

	if category != "" {
		if !models.IsValidNoteCategory(category) {
			return nil, 0, apperror.BadRequest("invalid note category")
		}
		notes, total, err = s.store.FindByChildAndCategory(childID, category, limit, offset)
	} else {
		notes, total, err = s.store.FindByChild(childID, limit, offset)
	}

	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch child notes")
	}

	responses := make([]models.ChildNoteResponse, len(notes))
	for i, n := range notes {
		responses[i] = n.ToResponse()
	}
	return responses, total, nil
}

// GetByID returns a child note by ID, validating ownership.
func (s *ChildNoteService) GetByID(ctx context.Context, noteID, childID, orgID uint) (*models.ChildNoteResponse, error) {
	// Verify child belongs to org
	child, err := s.childStore.FindByIDMinimal(childID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if child.OrganizationID != orgID {
		return nil, apperror.NotFound("child")
	}

	note, err := s.store.FindByID(noteID)
	if err != nil {
		return nil, apperror.NotFound("child note")
	}
	if note.ChildID != childID {
		return nil, apperror.NotFound("child note")
	}

	resp := note.ToResponse()
	return &resp, nil
}

// Create creates a new child note.
func (s *ChildNoteService) Create(ctx context.Context, childID, orgID, authorID uint, req *models.ChildNoteCreateRequest) (*models.ChildNoteResponse, error) {
	// Verify child belongs to org
	child, err := s.childStore.FindByIDMinimal(childID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if child.OrganizationID != orgID {
		return nil, apperror.NotFound("child")
	}

	// Validate category
	if !models.IsValidNoteCategory(req.Category) {
		return nil, apperror.BadRequest("invalid note category, must be one of: observation, development, medical, incident, general, parent_note")
	}

	// Validate title
	req.Title = strings.TrimSpace(req.Title)
	if validation.IsWhitespaceOnly(req.Title) {
		return nil, apperror.BadRequest("title cannot be empty or whitespace only")
	}

	// Validate content
	req.Content = strings.TrimSpace(req.Content)
	if validation.IsWhitespaceOnly(req.Content) {
		return nil, apperror.BadRequest("content cannot be empty or whitespace only")
	}

	note := &models.ChildNote{
		ChildID:        childID,
		OrganizationID: orgID,
		Category:       req.Category,
		Title:          req.Title,
		Content:        req.Content,
		AuthorID:       authorID,
	}

	if err := s.store.Create(note); err != nil {
		return nil, apperror.Internal("failed to create child note")
	}

	resp := note.ToResponse()
	return &resp, nil
}

// Update updates an existing child note.
func (s *ChildNoteService) Update(ctx context.Context, noteID, childID, orgID uint, req *models.ChildNoteUpdateRequest) (*models.ChildNoteResponse, error) {
	// Verify child belongs to org
	child, err := s.childStore.FindByIDMinimal(childID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if child.OrganizationID != orgID {
		return nil, apperror.NotFound("child")
	}

	note, err := s.store.FindByID(noteID)
	if err != nil {
		return nil, apperror.NotFound("child note")
	}
	if note.ChildID != childID {
		return nil, apperror.NotFound("child note")
	}

	if req.Category != nil {
		if !models.IsValidNoteCategory(*req.Category) {
			return nil, apperror.BadRequest("invalid note category")
		}
		note.Category = *req.Category
	}
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if validation.IsWhitespaceOnly(trimmed) {
			return nil, apperror.BadRequest("title cannot be empty or whitespace only")
		}
		note.Title = trimmed
	}
	if req.Content != nil {
		trimmed := strings.TrimSpace(*req.Content)
		if validation.IsWhitespaceOnly(trimmed) {
			return nil, apperror.BadRequest("content cannot be empty or whitespace only")
		}
		note.Content = trimmed
	}

	if err := s.store.Update(note); err != nil {
		return nil, apperror.Internal("failed to update child note")
	}

	resp := note.ToResponse()
	return &resp, nil
}

// Delete deletes a child note.
func (s *ChildNoteService) Delete(ctx context.Context, noteID, childID, orgID uint) error {
	// Verify child belongs to org
	child, err := s.childStore.FindByIDMinimal(childID)
	if err != nil {
		return apperror.NotFound("child")
	}
	if child.OrganizationID != orgID {
		return apperror.NotFound("child")
	}

	note, err := s.store.FindByID(noteID)
	if err != nil {
		return apperror.NotFound("child note")
	}
	if note.ChildID != childID {
		return apperror.NotFound("child note")
	}

	if err := s.store.Delete(noteID); err != nil {
		return apperror.Internal("failed to delete child note")
	}
	return nil
}
