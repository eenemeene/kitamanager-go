package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestPersonCreate_ValidChild(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org := createTestOrganization(t, db, "Test Org")

	resp, err := svc.Create(context.Background(), org.ID, &models.ChildCreateRequest{
		FirstName: "Emma",
		LastName:  "Schmidt",
		Gender:    "female",
		Birthdate: "2020-03-10",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.FirstName != "Emma" {
		t.Errorf("FirstName = %q, want %q", resp.FirstName, "Emma")
	}
	if resp.LastName != "Schmidt" {
		t.Errorf("LastName = %q, want %q", resp.LastName, "Schmidt")
	}
	if resp.Gender != "female" {
		t.Errorf("Gender = %q, want %q", resp.Gender, "female")
	}
	if resp.OrganizationID != org.ID {
		t.Errorf("OrganizationID = %d, want %d", resp.OrganizationID, org.ID)
	}
}

func TestPersonCreate_InvalidGender(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org := createTestOrganization(t, db, "Test Org")

	_, err := svc.Create(context.Background(), org.ID, &models.ChildCreateRequest{
		FirstName: "Emma",
		LastName:  "Schmidt",
		Gender:    "invalid",
		Birthdate: "2020-03-10",
	})
	if err == nil {
		t.Fatal("expected error for invalid gender, got nil")
	}
	assertHTTPStatus(t, err, http.StatusBadRequest)
}

func TestPersonCreate_InvalidBirthdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org := createTestOrganization(t, db, "Test Org")

	_, err := svc.Create(context.Background(), org.ID, &models.ChildCreateRequest{
		FirstName: "Emma",
		LastName:  "Schmidt",
		Gender:    "female",
		Birthdate: "not-a-date",
	})
	if err == nil {
		t.Fatal("expected error for invalid birthdate, got nil")
	}
	assertHTTPStatus(t, err, http.StatusBadRequest)
}

func TestPersonCreate_EmptyFirstName(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org := createTestOrganization(t, db, "Test Org")

	_, err := svc.Create(context.Background(), org.ID, &models.ChildCreateRequest{
		FirstName: "",
		LastName:  "Schmidt",
		Gender:    "female",
		Birthdate: "2020-03-10",
	})
	if err == nil {
		t.Fatal("expected error for empty first name, got nil")
	}
	assertHTTPStatus(t, err, http.StatusBadRequest)
}

func TestPersonCreate_WhitespaceOnlyName(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org := createTestOrganization(t, db, "Test Org")

	_, err := svc.Create(context.Background(), org.ID, &models.ChildCreateRequest{
		FirstName: "   ",
		LastName:  "Schmidt",
		Gender:    "female",
		Birthdate: "2020-03-10",
	})
	if err == nil {
		t.Fatal("expected error for whitespace-only first name, got nil")
	}
}

func TestPersonCreate_TrimsWhitespace(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org := createTestOrganization(t, db, "Test Org")

	resp, err := svc.Create(context.Background(), org.ID, &models.ChildCreateRequest{
		FirstName: "  Emma  ",
		LastName:  "  Schmidt  ",
		Gender:    "female",
		Birthdate: "2020-03-10",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.FirstName != "Emma" {
		t.Errorf("FirstName = %q, want %q (trimmed)", resp.FirstName, "Emma")
	}
	if resp.LastName != "Schmidt" {
		t.Errorf("LastName = %q, want %q (trimmed)", resp.LastName, "Schmidt")
	}
}

func TestPersonUpdate_PartialUpdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org := createTestOrganization(t, db, "Test Org")

	child := createTestChild(t, db, "Emma", "Schmidt", org.ID)

	newFirst := "Anna"
	resp, err := svc.Update(context.Background(), child.ID, org.ID, &models.ChildUpdateRequest{
		FirstName: &newFirst,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.FirstName != "Anna" {
		t.Errorf("FirstName = %q, want %q", resp.FirstName, "Anna")
	}
	if resp.LastName != "Schmidt" {
		t.Errorf("LastName = %q, want %q (should not change)", resp.LastName, "Schmidt")
	}
}

func TestPersonUpdate_InvalidGender(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org := createTestOrganization(t, db, "Test Org")

	child := createTestChild(t, db, "Emma", "Schmidt", org.ID)

	badGender := "invalid"
	_, err := svc.Update(context.Background(), child.ID, org.ID, &models.ChildUpdateRequest{
		Gender: &badGender,
	})
	if err == nil {
		t.Fatal("expected error for invalid gender, got nil")
	}
	assertHTTPStatus(t, err, http.StatusBadRequest)
}

func TestPersonUpdate_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	child := createTestChild(t, db, "Emma", "Schmidt", org1.ID)

	newName := "Anna"
	_, err := svc.Update(context.Background(), child.ID, org2.ID, &models.ChildUpdateRequest{
		FirstName: &newName,
	})
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
	assertHTTPStatus(t, err, http.StatusNotFound)
}

func TestPersonDelete_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org := createTestOrganization(t, db, "Test Org")

	child := createTestChild(t, db, "Emma", "Schmidt", org.ID)

	err := svc.Delete(context.Background(), child.ID, org.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify deleted
	_, err = svc.GetByID(context.Background(), child.ID, org.ID)
	if err == nil {
		t.Fatal("expected not found error after deletion, got nil")
	}
	assertHTTPStatus(t, err, http.StatusNotFound)
}

func TestPersonDelete_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	child := createTestChild(t, db, "Emma", "Schmidt", org1.ID)

	err := svc.Delete(context.Background(), child.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
	assertHTTPStatus(t, err, http.StatusNotFound)
}

func TestPersonDelete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org := createTestOrganization(t, db, "Test Org")

	err := svc.Delete(context.Background(), 99999, org.ID)
	if err == nil {
		t.Fatal("expected error for non-existent child, got nil")
	}
	assertHTTPStatus(t, err, http.StatusNotFound)
}

func TestPersonList_Pagination(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org := createTestOrganization(t, db, "Test Org")

	createTestChild(t, db, "Anna", "A", org.ID)
	createTestChild(t, db, "Berta", "B", org.ID)
	createTestChild(t, db, "Clara", "C", org.ID)

	// First page
	children, total, err := svc.ListByOrganization(context.Background(), org.ID, 2, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(children) != 2 {
		t.Errorf("len(children) = %d, want 2", len(children))
	}

	// Second page
	children, total, err = svc.ListByOrganization(context.Background(), org.ID, 2, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(children) != 1 {
		t.Errorf("len(children) = %d, want 1", len(children))
	}
}

func TestPersonGetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org := createTestOrganization(t, db, "Test Org")

	_, err := svc.GetByID(context.Background(), 99999, org.ID)
	if err == nil {
		t.Fatal("expected not found error, got nil")
	}
	assertHTTPStatus(t, err, http.StatusNotFound)
}

func TestPersonImportUpsert_CreateNew(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org := createTestOrganization(t, db, "Test Org")

	data := &models.ChildImportExportData{
		Children: []models.ChildResponse{
			{
				FirstName: "Imported",
				LastName:  "Child",
				Gender:    "female",
				Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	_, err := svc.Import(context.Background(), org.ID, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify child was created
	children, total, err := svc.ListByOrganizationAndSection(context.Background(), org.ID, models.ChildListFilter{Search: "Imported"}, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error listing: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(children) != 1 {
		t.Fatalf("len(children) = %d, want 1", len(children))
	}
	if children[0].FirstName != "Imported" {
		t.Errorf("FirstName = %q, want %q", children[0].FirstName, "Imported")
	}
}

func TestPersonImportUpsert_UpdateExisting(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org := createTestOrganization(t, db, "Test Org")

	// Create an existing child
	child := createTestChild(t, db, "Emma", "Schmidt", org.ID)

	// Import with same name+birthdate but different gender
	data := &models.ChildImportExportData{
		Children: []models.ChildResponse{
			{
				FirstName: child.FirstName,
				LastName:  child.LastName,
				Gender:    "diverse",
				Birthdate: child.Birthdate,
			},
		},
	}

	_, err := svc.Import(context.Background(), org.ID, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify gender was updated
	resp, err := svc.GetByID(context.Background(), child.ID, org.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Gender != "diverse" {
		t.Errorf("Gender = %q, want %q (should be updated)", resp.Gender, "diverse")
	}
}

func TestPersonImportUpsert_MissingName(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org := createTestOrganization(t, db, "Test Org")

	data := &models.ChildImportExportData{
		Children: []models.ChildResponse{
			{
				FirstName: "",
				LastName:  "Schmidt",
				Gender:    "female",
				Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	_, err := svc.Import(context.Background(), org.ID, data)
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
	assertHTTPStatus(t, err, http.StatusBadRequest)
}

func TestPersonImportUpsert_MissingBirthdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	org := createTestOrganization(t, db, "Test Org")

	data := &models.ChildImportExportData{
		Children: []models.ChildResponse{
			{
				FirstName: "Emma",
				LastName:  "Schmidt",
				Gender:    "female",
				// Zero-value birthdate
			},
		},
	}

	_, err := svc.Import(context.Background(), org.ID, data)
	if err == nil {
		t.Fatal("expected error for missing birthdate, got nil")
	}
	assertHTTPStatus(t, err, http.StatusBadRequest)
}

// assertHTTPStatus checks that err is an AppError with the expected HTTP status code.
func assertHTTPStatus(t *testing.T, err error, wantCode int) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Errorf("expected AppError, got %T: %v", err, err)
		return
	}
	if appErr.Code != wantCode {
		t.Errorf("HTTP status = %d, want %d (error: %s)", appErr.Code, wantCode, appErr.Message)
	}
}
