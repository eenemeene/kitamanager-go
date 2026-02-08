package models

import (
	"testing"
	"time"
)

func TestIsValidNoteCategory(t *testing.T) {
	tests := []struct {
		category string
		valid    bool
	}{
		{NoteCategoryObservation, true},
		{NoteCategoryDevelopment, true},
		{NoteCategoryMedical, true},
		{NoteCategoryIncident, true},
		{NoteCategoryGeneral, true},
		{NoteCategoryParentNote, true},
		{"invalid", false},
		{"", false},
		{"OBSERVATION", false},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			got := IsValidNoteCategory(tt.category)
			if got != tt.valid {
				t.Errorf("IsValidNoteCategory(%q) = %v, want %v", tt.category, got, tt.valid)
			}
		})
	}
}

func TestChildNote_ToResponse(t *testing.T) {
	now := time.Now()
	note := &ChildNote{
		ID:             1,
		ChildID:        2,
		OrganizationID: 3,
		Category:       NoteCategoryObservation,
		Title:          "Motor skills",
		Content:        "Improved coloring ability.",
		AuthorID:       4,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	resp := note.ToResponse()

	if resp.ID != 1 {
		t.Errorf("expected ID 1, got %d", resp.ID)
	}
	if resp.ChildID != 2 {
		t.Errorf("expected ChildID 2, got %d", resp.ChildID)
	}
	if resp.OrganizationID != 3 {
		t.Errorf("expected OrganizationID 3, got %d", resp.OrganizationID)
	}
	if resp.Category != NoteCategoryObservation {
		t.Errorf("expected category 'observation', got '%s'", resp.Category)
	}
	if resp.Title != "Motor skills" {
		t.Errorf("expected title 'Motor skills', got '%s'", resp.Title)
	}
	if resp.Content != "Improved coloring ability." {
		t.Errorf("expected content, got '%s'", resp.Content)
	}
	if resp.AuthorID != 4 {
		t.Errorf("expected AuthorID 4, got %d", resp.AuthorID)
	}
}
