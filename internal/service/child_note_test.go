package service

import (
	"context"
	"errors"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func setupChildNoteTest(t *testing.T) (*ChildNoteService, *models.Organization, *models.Child) {
	t.Helper()
	db := setupTestDB(t)
	db.AutoMigrate(&models.ChildNote{})

	childNoteStore := store.NewChildNoteStore(db)
	childStore := store.NewChildStore(db)
	svc := NewChildNoteService(childNoteStore, childStore)

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "Emma", "Schmidt", org.ID)

	return svc, org, child
}

func TestChildNoteService_Create(t *testing.T) {
	svc, org, child := setupChildNoteTest(t)
	ctx := context.Background()

	req := &models.ChildNoteCreateRequest{
		Category: models.NoteCategoryObservation,
		Title:    "Motor skill development",
		Content:  "Emma showed significant improvement in fine motor skills.",
	}

	resp, err := svc.Create(ctx, child.ID, org.ID, 1, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Category != models.NoteCategoryObservation {
		t.Errorf("expected category 'observation', got '%s'", resp.Category)
	}
	if resp.Title != "Motor skill development" {
		t.Errorf("expected title 'Motor skill development', got '%s'", resp.Title)
	}
	if resp.ChildID != child.ID {
		t.Errorf("expected ChildID %d, got %d", child.ID, resp.ChildID)
	}
	if resp.AuthorID != 1 {
		t.Errorf("expected AuthorID 1, got %d", resp.AuthorID)
	}
}

func TestChildNoteService_Create_InvalidCategory(t *testing.T) {
	svc, org, child := setupChildNoteTest(t)
	ctx := context.Background()

	req := &models.ChildNoteCreateRequest{
		Category: "invalid_category",
		Title:    "Test",
		Content:  "Content",
	}

	_, err := svc.Create(ctx, child.ID, org.ID, 1, req)
	if err == nil {
		t.Fatal("expected error for invalid category, got nil")
	}
}

func TestChildNoteService_Create_WhitespaceTitle(t *testing.T) {
	svc, org, child := setupChildNoteTest(t)
	ctx := context.Background()

	req := &models.ChildNoteCreateRequest{
		Category: models.NoteCategoryObservation,
		Title:    "   ",
		Content:  "Content",
	}

	_, err := svc.Create(ctx, child.ID, org.ID, 1, req)
	if err == nil {
		t.Fatal("expected error for whitespace-only title, got nil")
	}
}

func TestChildNoteService_Create_WhitespaceContent(t *testing.T) {
	svc, org, child := setupChildNoteTest(t)
	ctx := context.Background()

	req := &models.ChildNoteCreateRequest{
		Category: models.NoteCategoryObservation,
		Title:    "Title",
		Content:  "   ",
	}

	_, err := svc.Create(ctx, child.ID, org.ID, 1, req)
	if err == nil {
		t.Fatal("expected error for whitespace-only content, got nil")
	}
}

func TestChildNoteService_Create_ChildNotFound(t *testing.T) {
	svc, org, _ := setupChildNoteTest(t)
	ctx := context.Background()

	req := &models.ChildNoteCreateRequest{
		Category: models.NoteCategoryObservation,
		Title:    "Title",
		Content:  "Content",
	}

	_, err := svc.Create(ctx, 999, org.ID, 1, req)
	if err == nil {
		t.Fatal("expected error for non-existent child, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestChildNoteService_Create_WrongOrg(t *testing.T) {
	svc, _, child := setupChildNoteTest(t)
	ctx := context.Background()

	req := &models.ChildNoteCreateRequest{
		Category: models.NoteCategoryObservation,
		Title:    "Title",
		Content:  "Content",
	}

	_, err := svc.Create(ctx, child.ID, 999, 1, req)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
}

func TestChildNoteService_GetByID(t *testing.T) {
	svc, org, child := setupChildNoteTest(t)
	ctx := context.Background()

	created, _ := svc.Create(ctx, child.ID, org.ID, 1, &models.ChildNoteCreateRequest{
		Category: models.NoteCategoryObservation,
		Title:    "Test Note",
		Content:  "Content",
	})

	found, err := svc.GetByID(ctx, created.ID, child.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, found.ID)
	}
}

func TestChildNoteService_GetByID_NotFound(t *testing.T) {
	svc, org, child := setupChildNoteTest(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 999, child.ID, org.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestChildNoteService_GetByID_WrongChild(t *testing.T) {
	svc, org, child := setupChildNoteTest(t)
	ctx := context.Background()

	created, _ := svc.Create(ctx, child.ID, org.ID, 1, &models.ChildNoteCreateRequest{
		Category: models.NoteCategoryObservation,
		Title:    "Test",
		Content:  "Content",
	})

	// Try to get with wrong child ID
	_, err := svc.GetByID(ctx, created.ID, 999, org.ID)
	if err == nil {
		t.Fatal("expected error for wrong child, got nil")
	}
}

func TestChildNoteService_List(t *testing.T) {
	svc, org, child := setupChildNoteTest(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		svc.Create(ctx, child.ID, org.ID, 1, &models.ChildNoteCreateRequest{
			Category: models.NoteCategoryObservation,
			Title:    "Note",
			Content:  "Content",
		})
	}

	notes, total, err := svc.List(ctx, child.ID, org.ID, "", 10, 0)
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

func TestChildNoteService_ListByCategory(t *testing.T) {
	svc, org, child := setupChildNoteTest(t)
	ctx := context.Background()

	svc.Create(ctx, child.ID, org.ID, 1, &models.ChildNoteCreateRequest{
		Category: models.NoteCategoryObservation, Title: "Obs 1", Content: "C",
	})
	svc.Create(ctx, child.ID, org.ID, 1, &models.ChildNoteCreateRequest{
		Category: models.NoteCategoryMedical, Title: "Med 1", Content: "C",
	})
	svc.Create(ctx, child.ID, org.ID, 1, &models.ChildNoteCreateRequest{
		Category: models.NoteCategoryObservation, Title: "Obs 2", Content: "C",
	})

	notes, total, err := svc.List(ctx, child.ID, org.ID, models.NoteCategoryObservation, 10, 0)
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

func TestChildNoteService_ListByCategory_Invalid(t *testing.T) {
	svc, org, child := setupChildNoteTest(t)
	ctx := context.Background()

	_, _, err := svc.List(ctx, child.ID, org.ID, "invalid", 10, 0)
	if err == nil {
		t.Fatal("expected error for invalid category, got nil")
	}
}

func TestChildNoteService_Update(t *testing.T) {
	svc, org, child := setupChildNoteTest(t)
	ctx := context.Background()

	created, _ := svc.Create(ctx, child.ID, org.ID, 1, &models.ChildNoteCreateRequest{
		Category: models.NoteCategoryObservation,
		Title:    "Original",
		Content:  "Original content",
	})

	newTitle := "Updated Title"
	newContent := "Updated content"
	newCategory := models.NoteCategoryDevelopment
	updated, err := svc.Update(ctx, created.ID, child.ID, org.ID, &models.ChildNoteUpdateRequest{
		Title:    &newTitle,
		Content:  &newContent,
		Category: &newCategory,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got '%s'", updated.Title)
	}
	if updated.Content != "Updated content" {
		t.Errorf("expected content 'Updated content', got '%s'", updated.Content)
	}
	if updated.Category != models.NoteCategoryDevelopment {
		t.Errorf("expected category 'development', got '%s'", updated.Category)
	}
}

func TestChildNoteService_Update_InvalidCategory(t *testing.T) {
	svc, org, child := setupChildNoteTest(t)
	ctx := context.Background()

	created, _ := svc.Create(ctx, child.ID, org.ID, 1, &models.ChildNoteCreateRequest{
		Category: models.NoteCategoryObservation,
		Title:    "Test",
		Content:  "Content",
	})

	invalid := "invalid"
	_, err := svc.Update(ctx, created.ID, child.ID, org.ID, &models.ChildNoteUpdateRequest{
		Category: &invalid,
	})
	if err == nil {
		t.Fatal("expected error for invalid category, got nil")
	}
}

func TestChildNoteService_Delete(t *testing.T) {
	svc, org, child := setupChildNoteTest(t)
	ctx := context.Background()

	created, _ := svc.Create(ctx, child.ID, org.ID, 1, &models.ChildNoteCreateRequest{
		Category: models.NoteCategoryObservation,
		Title:    "To Delete",
		Content:  "Content",
	})

	err := svc.Delete(ctx, created.ID, child.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = svc.GetByID(ctx, created.ID, child.ID, org.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestChildNoteService_Delete_WrongOrg(t *testing.T) {
	svc, org, child := setupChildNoteTest(t)
	ctx := context.Background()

	created, _ := svc.Create(ctx, child.ID, org.ID, 1, &models.ChildNoteCreateRequest{
		Category: models.NoteCategoryObservation,
		Title:    "Test",
		Content:  "Content",
	})

	err := svc.Delete(ctx, created.ID, child.ID, 999)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
}
