package models

import (
	"time"
)

// ChildNote represents a documentation/observation note for a child.
type ChildNote struct {
	ID             uint          `gorm:"primaryKey" json:"id" example:"1"`
	ChildID        uint          `gorm:"not null;index" json:"child_id" example:"1"`
	Child          *Child        `gorm:"foreignKey:ChildID;constraint:OnDelete:CASCADE" json:"child,omitempty"`
	OrganizationID uint          `gorm:"not null;index" json:"organization_id" example:"1"`
	Organization   *Organization `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"organization,omitempty"`
	Category       string        `gorm:"size:50;not null" json:"category" example:"observation"`
	Title          string        `gorm:"size:255;not null" json:"title" example:"Motor skill development"`
	Content        string        `gorm:"type:text;not null" json:"content" example:"Emma showed significant improvement in fine motor skills today."`
	AuthorID       uint          `gorm:"not null" json:"author_id" example:"1"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// Note categories
const (
	NoteCategoryObservation = "observation"
	NoteCategoryDevelopment = "development"
	NoteCategoryMedical     = "medical"
	NoteCategoryIncident    = "incident"
	NoteCategoryGeneral     = "general"
	NoteCategoryParentNote  = "parent_note"
)

// IsValidNoteCategory checks if a category string is valid.
func IsValidNoteCategory(category string) bool {
	switch category {
	case NoteCategoryObservation, NoteCategoryDevelopment, NoteCategoryMedical,
		NoteCategoryIncident, NoteCategoryGeneral, NoteCategoryParentNote:
		return true
	}
	return false
}

// ChildNoteCreateRequest represents the request body for creating a child note.
type ChildNoteCreateRequest struct {
	Category string `json:"category" binding:"required" example:"observation"`
	Title    string `json:"title" binding:"required,max=255" example:"Motor skill development"`
	Content  string `json:"content" binding:"required" example:"Emma showed significant improvement in fine motor skills today."`
}

// ChildNoteUpdateRequest represents the request body for updating a child note.
type ChildNoteUpdateRequest struct {
	Category *string `json:"category" example:"development"`
	Title    *string `json:"title" binding:"omitempty,max=255" example:"Updated title"`
	Content  *string `json:"content" example:"Updated content"`
}

// ChildNoteResponse represents the child note response.
type ChildNoteResponse struct {
	ID             uint      `json:"id" example:"1"`
	ChildID        uint      `json:"child_id" example:"1"`
	OrganizationID uint      `json:"organization_id" example:"1"`
	Category       string    `json:"category" example:"observation"`
	Title          string    `json:"title" example:"Motor skill development"`
	Content        string    `json:"content" example:"Emma showed significant improvement in fine motor skills today."`
	AuthorID       uint      `json:"author_id" example:"1"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ToResponse converts a ChildNote to a ChildNoteResponse.
func (n *ChildNote) ToResponse() ChildNoteResponse {
	return ChildNoteResponse{
		ID:             n.ID,
		ChildID:        n.ChildID,
		OrganizationID: n.OrganizationID,
		Category:       n.Category,
		Title:          n.Title,
		Content:        n.Content,
		AuthorID:       n.AuthorID,
		CreatedAt:      n.CreatedAt,
		UpdatedAt:      n.UpdatedAt,
	}
}
