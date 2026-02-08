package store

import (
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func setupChildNoteTestDB(t *testing.T) *ChildNoteStore {
	t.Helper()
	db := setupTestDB(t)
	db.AutoMigrate(&models.ChildNote{})
	return NewChildNoteStore(db)
}

func createChildNoteTestChild(t *testing.T, store *ChildNoteStore, orgID uint) *models.Child {
	t.Helper()
	child := &models.Child{
		Person: models.Person{
			FirstName:      "Test",
			LastName:       "Child",
			OrganizationID: orgID,
			Birthdate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	if err := store.db.Create(child).Error; err != nil {
		t.Fatalf("failed to create test child: %v", err)
	}
	return child
}

func createTestNote(t *testing.T, s *ChildNoteStore, childID, orgID uint, category, title, content string) *models.ChildNote {
	t.Helper()
	note := &models.ChildNote{
		ChildID:        childID,
		OrganizationID: orgID,
		Category:       category,
		Title:          title,
		Content:        content,
		AuthorID:       1,
	}
	if err := s.Create(note); err != nil {
		t.Fatalf("failed to create test note: %v", err)
	}
	return note
}

func TestChildNoteStore_Create(t *testing.T) {
	s := setupChildNoteTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildNoteTestChild(t, s, org.ID)

	note := &models.ChildNote{
		ChildID:        child.ID,
		OrganizationID: org.ID,
		Category:       models.NoteCategoryObservation,
		Title:          "Fine motor skills",
		Content:        "Showed improvement in coloring activities.",
		AuthorID:       1,
	}

	if err := s.Create(note); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if note.ID == 0 {
		t.Error("expected ID to be set")
	}
}

func TestChildNoteStore_FindByID(t *testing.T) {
	s := setupChildNoteTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildNoteTestChild(t, s, org.ID)
	note := createTestNote(t, s, child.ID, org.ID, models.NoteCategoryObservation, "Test Title", "Test Content")

	found, err := s.FindByID(note.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found.ID != note.ID {
		t.Errorf("expected ID %d, got %d", note.ID, found.ID)
	}
	if found.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got '%s'", found.Title)
	}
	if found.Content != "Test Content" {
		t.Errorf("expected content 'Test Content', got '%s'", found.Content)
	}
	if found.Category != models.NoteCategoryObservation {
		t.Errorf("expected category 'observation', got '%s'", found.Category)
	}
}

func TestChildNoteStore_FindByID_NotFound(t *testing.T) {
	s := setupChildNoteTestDB(t)

	_, err := s.FindByID(999)
	if err == nil {
		t.Fatal("expected error for non-existent ID, got nil")
	}
}

func TestChildNoteStore_FindByChild(t *testing.T) {
	s := setupChildNoteTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildNoteTestChild(t, s, org.ID)

	createTestNote(t, s, child.ID, org.ID, models.NoteCategoryObservation, "Note 1", "Content 1")
	createTestNote(t, s, child.ID, org.ID, models.NoteCategoryDevelopment, "Note 2", "Content 2")
	createTestNote(t, s, child.ID, org.ID, models.NoteCategoryMedical, "Note 3", "Content 3")

	notes, total, err := s.FindByChild(child.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(notes) != 3 {
		t.Errorf("expected 3 notes, got %d", len(notes))
	}
}

func TestChildNoteStore_FindByChildAndCategory(t *testing.T) {
	s := setupChildNoteTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildNoteTestChild(t, s, org.ID)

	createTestNote(t, s, child.ID, org.ID, models.NoteCategoryObservation, "Obs 1", "Content 1")
	createTestNote(t, s, child.ID, org.ID, models.NoteCategoryDevelopment, "Dev 1", "Content 2")
	createTestNote(t, s, child.ID, org.ID, models.NoteCategoryObservation, "Obs 2", "Content 3")

	notes, total, err := s.FindByChildAndCategory(child.ID, models.NoteCategoryObservation, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(notes))
	}
}

func TestChildNoteStore_Update(t *testing.T) {
	s := setupChildNoteTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildNoteTestChild(t, s, org.ID)
	note := createTestNote(t, s, child.ID, org.ID, models.NoteCategoryObservation, "Original", "Original content")

	note.Title = "Updated Title"
	note.Content = "Updated content"
	note.Category = models.NoteCategoryDevelopment

	if err := s.Update(note); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := s.FindByID(note.ID)
	if found.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got '%s'", found.Title)
	}
	if found.Content != "Updated content" {
		t.Errorf("expected content 'Updated content', got '%s'", found.Content)
	}
	if found.Category != models.NoteCategoryDevelopment {
		t.Errorf("expected category 'development', got '%s'", found.Category)
	}
}

func TestChildNoteStore_Delete(t *testing.T) {
	s := setupChildNoteTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildNoteTestChild(t, s, org.ID)
	note := createTestNote(t, s, child.ID, org.ID, models.NoteCategoryObservation, "To Delete", "Content")

	if err := s.Delete(note.ID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err := s.FindByID(note.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestChildNoteStore_ChildIsolation(t *testing.T) {
	s := setupChildNoteTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child1 := createChildNoteTestChild(t, s, org.ID)
	child2 := createChildNoteTestChild(t, s, org.ID)

	createTestNote(t, s, child1.ID, org.ID, models.NoteCategoryObservation, "C1 Note 1", "Content")
	createTestNote(t, s, child1.ID, org.ID, models.NoteCategoryObservation, "C1 Note 2", "Content")
	createTestNote(t, s, child2.ID, org.ID, models.NoteCategoryObservation, "C2 Note 1", "Content")

	notes1, total1, _ := s.FindByChild(child1.ID, 10, 0)
	notes2, total2, _ := s.FindByChild(child2.ID, 10, 0)

	if total1 != 2 || len(notes1) != 2 {
		t.Errorf("expected 2 notes for child1, got total=%d, len=%d", total1, len(notes1))
	}
	if total2 != 1 || len(notes2) != 1 {
		t.Errorf("expected 1 note for child2, got total=%d, len=%d", total2, len(notes2))
	}
}

func TestChildNoteStore_Pagination(t *testing.T) {
	s := setupChildNoteTestDB(t)
	org := createTestOrganization(t, s.db, "Test Org")
	child := createChildNoteTestChild(t, s, org.ID)

	for i := 0; i < 5; i++ {
		createTestNote(t, s, child.ID, org.ID, models.NoteCategoryObservation, "Note", "Content")
	}

	// First page
	notes, total, err := s.FindByChild(child.ID, 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(notes) != 2 {
		t.Errorf("expected 2 notes on first page, got %d", len(notes))
	}

	// Last page
	notes, _, _ = s.FindByChild(child.ID, 2, 4)
	if len(notes) != 1 {
		t.Errorf("expected 1 note on last page, got %d", len(notes))
	}
}
