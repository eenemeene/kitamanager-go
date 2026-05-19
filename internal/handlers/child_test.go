package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/testutil"
)

func TestChildHandler_List(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	child1 := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Child1", LastName: "Last", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child1)
	createActiveChildContract(t, db, child1.ID)
	child2 := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Child2", LastName: "Last", Gender: "male", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child2)
	createActiveChildContract(t, db, child2.ID)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if len(response.Data) != 2 {
		t.Errorf("expected 2 children, got %d", len(response.Data))
	}
}

func TestChildHandler_Get(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d", org.ID, child.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.Child
	parseResponse(t, w, &result)

	if result.FirstName != "Test" {
		t.Errorf("expected first name 'Test', got '%s'", result.FirstName)
	}
}

func TestChildHandler_Create(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children", handler.Create)

	body := models.ChildCreateRequest{
		FirstName: "New",
		LastName:  "Child",
		Gender:    "female",
		Birthdate: "2020-03-15",
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children", org.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result models.Child
	parseResponse(t, w, &result)

	if result.FirstName != "New" {
		t.Errorf("expected first name 'New', got '%s'", result.FirstName)
	}
	if result.OrganizationID != org.ID {
		t.Errorf("expected organization ID %d, got %d", org.ID, result.OrganizationID)
	}

	testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("child_create"),
		ResourceType: "child",
		ResourceID:   result.ID,
	})
}

func TestChildHandler_Update(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Original", LastName: "Child", Gender: "male", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:childId", handler.Update)

	newName := "Updated"
	body := models.ChildUpdateRequest{
		FirstName: &newName,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d", org.ID, child.ID), body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.Child
	parseResponse(t, w, &result)

	if result.FirstName != "Updated" {
		t.Errorf("expected first name 'Updated', got '%s'", result.FirstName)
	}

	row := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("child_update"),
		ResourceType: "child",
		ResourceID:   child.ID,
	})

	// H2: the audit row must carry the per-field diff. Pre-fix
	// Details only had `{"resource_name": "…"}` — the change to
	// first_name was invisible. Post-fix the Details JSON includes a
	// `changes` map keyed by field with {old, new} pairs.
	var details map[string]any
	if err := json.Unmarshal([]byte(row.Details), &details); err != nil {
		t.Fatalf("Details JSON parse: %v (raw=%q)", err, row.Details)
	}
	changes, _ := details["changes"].(map[string]any)
	if changes == nil {
		t.Fatalf("audit row Details missing `changes` map: %q", row.Details)
	}
	fn, ok := changes["first_name"].(map[string]any)
	if !ok {
		t.Fatalf("changes missing first_name: %+v", changes)
	}
	if fn["old"] != "Original" || fn["new"] != "Updated" {
		t.Errorf("first_name change = %+v, want {old: Original, new: Updated}", fn)
	}
	if _, present := changes["last_name"]; present {
		t.Errorf("changes should NOT include last_name (unchanged): %+v", changes)
	}
}

// TestChildHandler_Update_MultipleFieldsDiff covers the multi-field
// path. The diff must include every changed scalar and skip every
// unchanged one — pre-fix the entire Details payload was a static
// resource_name string so this distinction did not exist.
func TestChildHandler_Update_MultipleFieldsDiff(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Anna", LastName: "Old", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:childId", handler.Update)

	newLast := "New"
	newBirth := "2020-06-15"
	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d", org.ID, child.ID),
		models.ChildUpdateRequest{LastName: &newLast, Birthdate: &newBirth})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	row := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("child_update"),
		ResourceType: "child",
		ResourceID:   child.ID,
	})
	var details map[string]any
	if err := json.Unmarshal([]byte(row.Details), &details); err != nil {
		t.Fatalf("Details JSON parse: %v", err)
	}
	changes, _ := details["changes"].(map[string]any)
	if changes == nil {
		t.Fatalf("missing changes map: %q", row.Details)
	}
	if _, ok := changes["last_name"]; !ok {
		t.Errorf("changes missing last_name: %+v", changes)
	}
	if _, ok := changes["birthdate"]; !ok {
		t.Errorf("changes missing birthdate: %+v", changes)
	}
	if _, ok := changes["first_name"]; ok {
		t.Errorf("changes should NOT include first_name (unchanged): %+v", changes)
	}
	if _, ok := changes["gender"]; ok {
		t.Errorf("changes should NOT include gender (unchanged): %+v", changes)
	}
}

// TestChildHandler_Update_NoOpHasNoChangesMap proves the helper
// degrades cleanly when nothing actually changed: the audit row
// exists (we want "user touched X" still visible) but carries no
// `changes` map so dashboards filtering on `changes IS NOT NULL`
// don't surface no-op edits.
func TestChildHandler_Update_NoOpHasNoChangesMap(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Same", LastName: "Same", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:childId", handler.Update)

	// Submit identical values — nothing should change.
	sameFirst, sameLast := "Same", "Same"
	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d", org.ID, child.ID),
		models.ChildUpdateRequest{FirstName: &sameFirst, LastName: &sameLast})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	row := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("child_update"),
		ResourceType: "child",
		ResourceID:   child.ID,
	})
	var details map[string]any
	if err := json.Unmarshal([]byte(row.Details), &details); err != nil {
		t.Fatalf("Details JSON parse: %v", err)
	}
	if _, present := details["changes"]; present {
		t.Errorf("no-op update should not carry a `changes` map, got %+v", details)
	}
}

func TestChildHandler_Delete(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "ToDelete", LastName: "Child", Gender: "male", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:childId", handler.Delete)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/%d", org.ID, child.ID), nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	row := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditActionChildDelete,
		ResourceType: "child",
		ResourceID:   child.ID,
	})
	// End-to-end proof that the actor email captured by the auth middleware
	// flows through the handler, the audit helpers, and the audit service
	// into the persisted row. A regression here would reproduce the bug that
	// made the "User" column in the audit UI render as a dash.
	if row.UserEmail != "test@example.com" {
		t.Errorf("audit row UserEmail = %q, want test@example.com", row.UserEmail)
	}
}

func TestChildHandler_ListContracts(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId/contracts", handler.ListContracts)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/contracts", org.ID, child.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.ChildContractResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Errorf("expected 1 contract, got %d", len(response.Data))
	}
}

func TestChildHandler_GetCurrentRecord(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: nil},
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId/contracts/current", handler.GetCurrentRecord)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/contracts/current", org.ID, child.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var contract models.ChildContract
	parseResponse(t, w, &contract)

	if contract.Properties["care_type"] != "ganztag" {
		t.Errorf("expected care_type ganztag, got %v", contract.Properties)
	}
}

func TestChildHandler_GetCurrentRecord_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId/contracts/current", handler.GetCurrentRecord)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/contracts/current", org.ID, child.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_CreateContract(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/contracts", handler.CreateContract)

	body := models.ChildContractCreateRequest{
		SectionID:  sectionID,
		From:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:         nil,
		Properties: models.ContractProperties{"care_type": "ganztag", "supplements": []string{"ndh"}},
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/contracts", org.ID, child.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var contract models.ChildContract
	parseResponse(t, w, &contract)

	if contract.Properties["care_type"] != "ganztag" {
		t.Errorf("expected care_type ganztag, got %v", contract.Properties)
	}
}

func TestChildHandler_CreateContract_SameDay(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/contracts", handler.CreateContract)

	// Create a same-day contract (from == to)
	sameDay := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	body := models.ChildContractCreateRequest{
		SectionID:  sectionID,
		From:       sameDay,
		To:         &sameDay,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/contracts", org.ID, child.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d for same-day contract, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var contract models.ChildContract
	parseResponse(t, w, &contract)

	if !contract.From.Equal(sameDay) {
		t.Errorf("expected from %v, got %v", sameDay, contract.From)
	}
	if contract.To == nil || !contract.To.Equal(sameDay) {
		t.Errorf("expected to %v, got %v", sameDay, contract.To)
	}
}

func TestChildHandler_CreateContract_Overlap(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	// Create existing contract
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: nil},
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/contracts", handler.CreateContract)

	// Try to create overlapping contract
	body := models.ChildContractCreateRequest{
		SectionID:  sectionID,
		From:       time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		To:         nil,
		Properties: models.ContractProperties{"care_type": "halbtag"},
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/contracts", org.ID, child.ID), body)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestChildHandler_DeleteContract(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	db.Create(contract)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:childId/contracts/:contractId", handler.DeleteContract)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract.ID), nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

// Edge case tests

func TestChildHandler_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/999", org.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_Get_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/invalid", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_Get_ZeroID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/0", org.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for zero ID, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_Create_MissingRequiredFields(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children", handler.Create)

	body := map[string]any{}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for missing fields, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_Create_EmptyFirstName(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children", handler.Create)

	body := models.ChildCreateRequest{
		FirstName: "",
		LastName:  "Child",
		Gender:    "male",
		Birthdate: "2020-05-15",
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for empty first name, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_Create_EmptyLastName(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children", handler.Create)

	body := models.ChildCreateRequest{
		FirstName: "Test",
		LastName:  "",
		Gender:    "male",
		Birthdate: "2020-05-15",
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for empty last name, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:childId", handler.Update)

	newName := "Updated"
	body := models.ChildUpdateRequest{
		FirstName: &newName,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/999", org.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_Update_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:childId", handler.Update)

	newName := "Updated"
	body := models.ChildUpdateRequest{
		FirstName: &newName,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/invalid", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:childId", handler.Delete)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/999", org.ID), nil)

	if w.Code != http.StatusNoContent && w.Code != http.StatusNotFound {
		t.Errorf("expected status %d or %d, got %d", http.StatusNoContent, http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_Delete_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:childId", handler.Delete)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/invalid", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if len(response.Data) != 0 {
		t.Errorf("expected empty list, got %d children", len(response.Data))
	}
}

func TestChildHandler_ListContracts_ChildNotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId/contracts", handler.ListContracts)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/999/contracts", org.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_ListContracts_Empty(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId/contracts", handler.ListContracts)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/contracts", org.ID, child.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.ChildContractResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 0 {
		t.Errorf("expected empty list, got %d contracts", len(response.Data))
	}
}

func TestChildHandler_CreateContract_ChildNotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/contracts", handler.CreateContract)

	body := models.ChildContractCreateRequest{
		SectionID: sectionID,
		From:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/999/contracts", org.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_CreateContract_InvalidChildID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/contracts", handler.CreateContract)

	body := models.ChildContractCreateRequest{
		SectionID: sectionID,
		From:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/invalid/contracts", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_CreateContract_ContractBoundaryTouch(t *testing.T) {
	// Tests that adjacent contracts (A ends day before B starts) are allowed.
	// This is the correct way to transition between contracts.
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	// Create contract ending on Dec 31, 2024
	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &endDate},
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/contracts", handler.CreateContract)

	// Create contract starting Jan 1, 2025 (day after previous ends) - should succeed
	body := models.ChildContractCreateRequest{
		SectionID:  sectionID,
		From:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Properties: models.ContractProperties{"care_type": "halbtag"},
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/contracts", org.ID, child.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d for adjacent (non-overlapping) contract, got %d: %s",
			http.StatusCreated, w.Code, w.Body.String())
	}
}

func TestChildHandler_CreateContract_SameDayTransitionRejected(t *testing.T) {
	// Tests that "touching" contracts (A ends same day B starts) are rejected.
	// Both start and end dates are inclusive, so same-day transition would mean
	// both contracts are active on that day, which is not allowed.
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	// Create contract ending on Jan 31, 2025
	endDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: &endDate},
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/contracts", handler.CreateContract)

	// Try to create contract starting Jan 31, 2025 (same day as previous ends) - should fail
	body := models.ChildContractCreateRequest{
		SectionID:  sectionID,
		From:       time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC),
		Properties: models.ContractProperties{"care_type": "halbtag"},
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/contracts", org.ID, child.ID), body)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d for same-day transition (overlap), got %d: %s",
			http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestChildHandler_DeleteContract_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:childId/contracts/:contractId", handler.DeleteContract)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/%d/contracts/999", org.ID, child.ID), nil)

	if w.Code != http.StatusNoContent && w.Code != http.StatusNotFound {
		t.Errorf("expected status %d or %d, got %d", http.StatusNoContent, http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_DeleteContract_InvalidContractID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:childId/contracts/:contractId", handler.DeleteContract)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/%d/contracts/invalid", org.ID, child.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_GetCurrentRecord_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId/contracts/current", handler.GetCurrentRecord)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/invalid/contracts/current", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Validation edge case tests

func TestChildHandler_Create_FutureBirthdate(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children", handler.Create)

	body := models.ChildCreateRequest{
		FirstName: "Test",
		LastName:  "Child",
		Gender:    "male",
		Birthdate: "2099-01-01",
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for future birthdate, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_Create_WhitespaceOnlyFirstName(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children", handler.Create)

	body := models.ChildCreateRequest{
		FirstName: "   ",
		LastName:  "Child",
		Gender:    "male",
		Birthdate: "2020-05-15",
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for whitespace-only first name, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_Create_WhitespaceOnlyLastName(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children", handler.Create)

	body := models.ChildCreateRequest{
		FirstName: "Test",
		LastName:  "   ",
		Gender:    "male",
		Birthdate: "2020-05-15",
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for whitespace-only last name, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_CreateContract_FromAfterTo(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/contracts", handler.CreateContract)

	toDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	body := models.ChildContractCreateRequest{
		SectionID: sectionID,
		From:      time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		To:        &toDate,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/contracts", org.ID, child.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for from > to, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

// SECURITY TESTS: Cross-organization access attempts

func TestChildHandler_Get_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId", handler.Get)

	// Try to access org1's child via org2's URL
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d", org2.ID, child.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("SECURITY: expected status %d when accessing child from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_Update_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Original", LastName: "Child", Gender: "male", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:childId", handler.Update)

	newName := "Hacker"
	body := models.ChildUpdateRequest{
		FirstName: &newName,
	}

	// Try to update org1's child via org2's URL
	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d", org2.ID, child.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("SECURITY: expected status %d when updating child from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_Delete_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:childId", handler.Delete)

	// Try to delete org1's child via org2's URL
	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/%d", org2.ID, child.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("SECURITY: expected status %d when deleting child from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_ListContracts_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId/contracts", handler.ListContracts)

	// Try to list contracts for org1's child via org2's URL
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/contracts", org2.ID, child.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("SECURITY: expected status %d when listing contracts from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_CreateContract_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	sectionID := ensureTestSection(t, db, org1.ID)
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/contracts", handler.CreateContract)

	body := models.ChildContractCreateRequest{
		SectionID: sectionID,
		From:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Try to create contract for org1's child via org2's URL
	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/contracts", org2.ID, child.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("SECURITY: expected status %d when creating contract from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_DeleteContract_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	sectionID := ensureTestSection(t, db, org1.ID)
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	// Create contract for child
	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	db.Create(contract)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:childId/contracts/:contractId", handler.DeleteContract)

	// Try to delete contract for org1's child via org2's URL
	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org2.ID, child.ID, contract.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("SECURITY: expected status %d when deleting contract from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_GetCurrentRecord_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	sectionID := ensureTestSection(t, db, org1.ID)
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	// Create active contract
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId/contracts/current", handler.GetCurrentRecord)

	// Try to get current contract for org1's child via org2's URL
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/contracts/current", org2.ID, child.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("SECURITY: expected status %d when getting current contract from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

// CROSS-ORG SECTION TESTS: Ensure sections from one org can't be used in another

func TestChildHandler_CreateContract_SectionFromWrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	org2SectionID := ensureTestSection(t, db, org2.ID)

	// Create child in org1
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/contracts", handler.CreateContract)

	// Try to create contract with org2's section for org1's child
	body := models.ChildContractCreateRequest{
		SectionID: org2SectionID,
		From:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/contracts", org1.ID, child.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("SECURITY: expected status %d when using section from wrong org, got %d: %s",
			http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_UpdateContract_SectionFromWrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	org1SectionID := ensureTestSection(t, db, org1.ID)
	org2 := createTestOrganization(t, db, "Org 2")
	org2SectionID := ensureTestSection(t, db, org2.ID)

	// Create child in org1 with a valid contract
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: org1SectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	db.Create(contract)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:childId/contracts/:contractId", handler.UpdateContract)

	// Try to update contract to use org2's section
	body := models.ChildContractUpdateRequest{
		SectionID: &org2SectionID,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org1.ID, child.ID, contract.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("SECURITY: expected status %d when updating to section from wrong org, got %d: %s",
			http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_CreateContract_MissingSectionID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/contracts", handler.CreateContract)

	// Send request without section_id (required field)
	body := map[string]any{
		"from": "2025-01-01",
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/contracts", org.ID, child.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for missing section_id, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_CreateContract_NonExistentSection(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/contracts", handler.CreateContract)

	body := models.ChildContractCreateRequest{
		SectionID: 99999, // Non-existent section
		From:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/contracts", org.ID, child.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for non-existent section, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

// PAGINATION TESTS

func TestChildHandler_List_Pagination(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	// Create 15 children with active contracts
	for i := 1; i <= 15; i++ {
		child := &models.Child{
			Person: models.Person{
				OrganizationID: org.ID,
				FirstName:      fmt.Sprintf("Child%02d", i),
				LastName:       "Last",
				Birthdate:      time.Now(),
			},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	// Test page 1 with limit 5
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=1&limit=5", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if len(response.Data) != 5 {
		t.Errorf("expected 5 children on page 1, got %d", len(response.Data))
	}
	if response.Page != 1 {
		t.Errorf("expected page 1, got %d", response.Page)
	}
	if response.Limit != 5 {
		t.Errorf("expected limit 5, got %d", response.Limit)
	}
	if response.Total != 15 {
		t.Errorf("expected total 15, got %d", response.Total)
	}
	if response.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", response.TotalPages)
	}
}

func TestChildHandler_List_Pagination_SecondPage(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	// Create 15 children with active contracts
	for i := 1; i <= 15; i++ {
		child := &models.Child{
			Person: models.Person{
				OrganizationID: org.ID,
				FirstName:      fmt.Sprintf("Child%02d", i),
				LastName:       "Last",
				Birthdate:      time.Now(),
			},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	// Get page 1
	w1 := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=1&limit=5", org.ID), nil)
	var response1 models.PaginatedResponse[models.Child]
	parseResponse(t, w1, &response1)

	// Get page 2
	w2 := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=2&limit=5", org.ID), nil)

	if w2.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w2.Code)
	}

	var response2 models.PaginatedResponse[models.Child]
	parseResponse(t, w2, &response2)

	if len(response2.Data) != 5 {
		t.Errorf("expected 5 children on page 2, got %d", len(response2.Data))
	}
	if response2.Page != 2 {
		t.Errorf("expected page 2, got %d", response2.Page)
	}

	// Verify page 1 and page 2 have different children
	page1IDs := make(map[uint]bool)
	for _, child := range response1.Data {
		page1IDs[child.ID] = true
	}
	for _, child := range response2.Data {
		if page1IDs[child.ID] {
			t.Errorf("child ID %d appears on both page 1 and page 2", child.ID)
		}
	}
}

func TestChildHandler_List_Pagination_LastPage(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	// Create 12 children with active contracts (3 pages of 5, last page has 2)
	for i := 1; i <= 12; i++ {
		child := &models.Child{
			Person: models.Person{
				OrganizationID: org.ID,
				FirstName:      fmt.Sprintf("Child%02d", i),
				LastName:       "Last",
				Birthdate:      time.Now(),
			},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	// Get last page (page 3)
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=3&limit=5", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if len(response.Data) != 2 {
		t.Errorf("expected 2 children on last page, got %d", len(response.Data))
	}
	if response.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", response.TotalPages)
	}
}

func TestChildHandler_List_Pagination_BeyondLastPage(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	// Create 5 children with active contracts
	for i := 1; i <= 5; i++ {
		child := &models.Child{
			Person: models.Person{
				OrganizationID: org.ID,
				FirstName:      fmt.Sprintf("Child%02d", i),
				LastName:       "Last",
				Birthdate:      time.Now(),
			},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	// Request page 10 (beyond available data)
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=10&limit=5", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if len(response.Data) != 0 {
		t.Errorf("expected 0 children beyond last page, got %d", len(response.Data))
	}
	if response.Total != 5 {
		t.Errorf("expected total 5, got %d", response.Total)
	}
}

func TestChildHandler_List_Pagination_DefaultValues(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	// Create 25 children with active contracts
	for i := 1; i <= 25; i++ {
		child := &models.Child{
			Person: models.Person{
				OrganizationID: org.ID,
				FirstName:      fmt.Sprintf("Child%02d", i),
				LastName:       "Last",
				Birthdate:      time.Now(),
			},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	// Request without pagination params (should use defaults: page=1, limit=20)
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if len(response.Data) != 20 {
		t.Errorf("expected 20 children with default limit, got %d", len(response.Data))
	}
	if response.Page != 1 {
		t.Errorf("expected default page 1, got %d", response.Page)
	}
	if response.Limit != 20 {
		t.Errorf("expected default limit 20, got %d", response.Limit)
	}
}

func TestChildHandler_List_Pagination_InvalidNegativePage(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=-1&limit=10", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for negative page, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_List_Pagination_InvalidNegativeLimit(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=1&limit=-5", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for negative limit, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_List_Pagination_LimitExceedsMax(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=1&limit=101", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for limit > 100, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_List_Pagination_MaxLimit(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	// Create 5 children with active contracts
	for i := 1; i <= 5; i++ {
		child := &models.Child{
			Person: models.Person{
				OrganizationID: org.ID,
				FirstName:      fmt.Sprintf("Child%02d", i),
				LastName:       "Last",
				Birthdate:      time.Now(),
			},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	// Request with max limit (100)
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=1&limit=100", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for limit=100, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if response.Limit != 100 {
		t.Errorf("expected limit 100, got %d", response.Limit)
	}
}

// =========================================
// Search Tests
// =========================================

func TestChildHandler_List_Search(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	// Create children with distinct names
	for _, name := range []struct{ first, last string }{
		{"Emma", "Schmidt"},
		{"Emilia", "Fischer"},
		{"Liam", "Mueller"},
	} {
		child := &models.Child{
			Person: models.Person{OrganizationID: org.ID, FirstName: name.first, LastName: name.last, Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	// Search by first name prefix
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?search=em", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if response.Total != 2 {
		t.Errorf("expected total 2 for search 'em', got %d", response.Total)
	}
	if len(response.Data) != 2 {
		t.Errorf("expected 2 children, got %d", len(response.Data))
	}

	// Search by last name
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?search=mueller", org.ID), nil)
	parseResponse(t, w, &response)
	if response.Total != 1 {
		t.Errorf("expected total 1 for search 'mueller', got %d", response.Total)
	}

	// Search with no results
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?search=zzz", org.ID), nil)
	parseResponse(t, w, &response)
	if response.Total != 0 {
		t.Errorf("expected total 0 for search 'zzz', got %d", response.Total)
	}
}

func TestChildHandler_List_SearchWithPagination(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	// Create 5 matching children and 2 non-matching
	for i := 1; i <= 5; i++ {
		child := &models.Child{
			Person: models.Person{OrganizationID: org.ID, FirstName: fmt.Sprintf("Emma%d", i), LastName: "Schmidt", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}
	for _, name := range []string{"Liam", "Noah"} {
		child := &models.Child{
			Person: models.Person{OrganizationID: org.ID, FirstName: name, LastName: "Other", Gender: "male", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	// Page 1: search + pagination
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?search=emma&page=1&limit=2", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if response.Total != 5 {
		t.Errorf("expected total 5, got %d", response.Total)
	}
	if len(response.Data) != 2 {
		t.Errorf("expected 2 children on page 1, got %d", len(response.Data))
	}
	if response.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", response.TotalPages)
	}

	// Page 3: last page should have 1 result
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?search=emma&page=3&limit=2", org.ID), nil)
	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Errorf("expected 1 child on page 3, got %d", len(response.Data))
	}
}

// =========================================
// GetContract Tests
// =========================================

func TestChildHandler_GetContract(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	db.Create(contract)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId/contracts/:contractId", handler.GetContract)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.ChildContractResponse
	parseResponse(t, w, &result)

	if result.ChildID != child.ID {
		t.Errorf("expected child_id %d, got %d", child.ID, result.ChildID)
	}
}

func TestChildHandler_GetContract_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId/contracts/:contractId", handler.GetContract)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/contracts/999", org.ID, child.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_GetContract_InvalidContractID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId/contracts/:contractId", handler.GetContract)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/contracts/abc", org.ID, child.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_GetContract_WrongChild(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child1 := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Child", LastName: "One", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child1)

	child2 := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Child", LastName: "Two", Gender: "male", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child2)

	// Create contract for child1
	contract := &models.ChildContract{
		ChildID: child1.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	db.Create(contract)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId/contracts/:contractId", handler.GetContract)

	// Try to get child1's contract via child2's URL
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child2.ID, contract.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for wrong child access, got %d", http.StatusNotFound, w.Code)
	}
}

// =========================================
// UpdateContract Tests
// =========================================

func TestChildHandler_UpdateContract(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	// Use today so contract qualifies for in-place update
	today := models.Today()
	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: today},
		},
	}
	db.Create(contract)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:childId/contracts/:contractId", handler.UpdateContract)

	newFrom := today.AddDate(0, 2, 0)
	body := models.ChildContractUpdateRequest{
		From: &newFrom,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract.ID), body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.ChildContractResponse
	parseResponse(t, w, &result)

	if !result.From.Equal(newFrom) {
		t.Errorf("expected from %v, got %v", newFrom, result.From)
	}
}

func TestChildHandler_UpdateContract_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:childId/contracts/:contractId", handler.UpdateContract)

	newFrom := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	body := models.ChildContractUpdateRequest{
		From: &newFrom,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d/contracts/999", org.ID, child.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_UpdateContract_Overlap(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	// Use future dates so contracts qualify for in-place update
	today := models.Today()
	endDate1 := today.AddDate(0, 6, 0)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: today, To: &endDate1},
		},
	})

	contract2 := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: today.AddDate(0, 7, 0)},
		},
	}
	db.Create(contract2)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:childId/contracts/:contractId", handler.UpdateContract)

	// Update contract2's From to overlap with contract1
	newFrom := today.AddDate(0, 3, 0)
	body := models.ChildContractUpdateRequest{
		From: &newFrom,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract2.ID), body)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d for overlap, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestChildHandler_UpdateContract_InvalidBody(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	db.Create(contract)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:childId/contracts/:contractId", handler.UpdateContract)

	body := map[string]any{
		"from": "not-a-date",
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_ExportYAML_NoFilters(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	// Create two children: one with active contract, one with expired
	child1 := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Active", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child1)
	createActiveChildContract(t, db, child1.ID)

	child2 := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Expired", LastName: "Child", Gender: "male", Birthdate: time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child2)
	sectionID := ensureTestSection(t, db, org.ID)
	db.Create(&models.ChildContract{
		ChildID:      child2.ID,
		BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), To: timePtr(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC))}, SectionID: sectionID},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/export/yaml", handler.ExportYAML)

	// No filters → should export ALL children
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/export/yaml", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/x-yaml" {
		t.Errorf("expected Content-Type application/x-yaml, got %s", ct)
	}

	var data models.ChildImportExportData
	if err := yaml.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatalf("failed to parse YAML response: %v", err)
	}

	if len(data.Children) != 2 {
		t.Errorf("expected 2 children without filters, got %d", len(data.Children))
	}
}

func TestChildHandler_ExportYAML_WithActiveOnFilter(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	child1 := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Active", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child1)
	createActiveChildContract(t, db, child1.ID)

	child2 := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Expired", LastName: "Child", Gender: "male", Birthdate: time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child2)
	sectionID := ensureTestSection(t, db, org.ID)
	db.Create(&models.ChildContract{
		ChildID:      child2.ID,
		BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), To: timePtr(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC))}, SectionID: sectionID},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/export/yaml", handler.ExportYAML)

	// Filter by active_on=today → only active child.
	// Use models.Today() (Berlin tz) instead of time.Now() so the test
	// is stable around the UTC-local boundary; the service computes
	// "today" via models.Today() and a UTC-formatted date can land on
	// the wrong calendar day late in the evening (M4).
	today := models.Today().Format("2006-01-02")
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/export/yaml?active_on=%s", org.ID, today), nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var data models.ChildImportExportData
	if err := yaml.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatalf("failed to parse YAML response: %v", err)
	}

	if len(data.Children) != 1 {
		t.Errorf("expected 1 child with active_on filter, got %d", len(data.Children))
	}
	if len(data.Children) == 1 && data.Children[0].FirstName != "Active" {
		t.Errorf("expected Active child, got %s", data.Children[0].FirstName)
	}
}

func TestChildHandler_ExportYAML_WithSearchFilter(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	child1 := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Alice", LastName: "Smith", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child1)
	createActiveChildContract(t, db, child1.ID)

	child2 := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Bob", LastName: "Jones", Gender: "male", Birthdate: time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child2)
	createActiveChildContract(t, db, child2.ID)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/export/yaml", handler.ExportYAML)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/export/yaml?search=Alice", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var data models.ChildImportExportData
	if err := yaml.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatalf("failed to parse YAML response: %v", err)
	}

	if len(data.Children) != 1 {
		t.Errorf("expected 1 child with search=Alice, got %d", len(data.Children))
	}
}

// TestChildHandler_Import_AuditDetailsCaptured closes review finding
// M1: pre-fix the import audit row was `auditCreate(..., 0, "YAML
// import")` — id=0, no row count, no filename, no list of created ids.
// Post-fix the row records record_count, filename, and the new entity
// ids so investigators can pivot from a single child back to the
// import event that created it.
func TestChildHandler_Import_AuditDetailsCaptured(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	// Default section is required so contracts can resolve their section_name.
	_ = ensureTestSection(t, db, org.ID)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/import", handler.Import)

	payload := models.ChildImportExportData{
		Children: []models.ChildResponse{
			{
				FirstName: "Imp",
				LastName:  "One",
				Gender:    "female",
				Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ChildContractResponse{
					{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
			{
				FirstName: "Imp",
				LastName:  "Two",
				Gender:    "male",
				Birthdate: time.Date(2021, 6, 15, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ChildContractResponse{
					{From: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
		},
	}
	yamlBytes, err := yaml.Marshal(payload)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "kinder-2024.yaml")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(yamlBytes)); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	writer.Close()

	req, _ := http.NewRequest("POST", fmt.Sprintf("/organizations/%d/children/import", org.ID), body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var imported []models.ChildResponse
	if err := json.Unmarshal(w.Body.Bytes(), &imported); err != nil {
		t.Fatalf("response decode: %v", err)
	}
	if len(imported) != 2 {
		t.Fatalf("expected 2 imported children, got %d", len(imported))
	}

	row := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("child_import"),
		ResourceType: "child",
	})
	if row.OrganizationID == nil || *row.OrganizationID != org.ID {
		t.Errorf("audit row OrganizationID = %v, want %d", row.OrganizationID, org.ID)
	}

	var details map[string]any
	if err := json.Unmarshal([]byte(row.Details), &details); err != nil {
		t.Fatalf("Details JSON parse: %v (raw=%q)", err, row.Details)
	}
	if got, _ := details["record_count"].(float64); int(got) != 2 {
		t.Errorf("Details.record_count = %v, want 2", details["record_count"])
	}
	if got, _ := details["filename"].(string); got != "kinder-2024.yaml" {
		t.Errorf("Details.filename = %q, want kinder-2024.yaml", got)
	}
	ids, _ := details["ids"].([]any)
	if len(ids) != 2 {
		t.Fatalf("Details.ids length = %d, want 2", len(ids))
	}
	gotIDs := make([]uint, len(ids))
	for i, v := range ids {
		gotIDs[i] = uint(v.(float64))
	}
	if !slices.Contains(gotIDs, imported[0].ID) || !slices.Contains(gotIDs, imported[1].ID) {
		t.Errorf("Details.ids = %v, want all of %d and %d", gotIDs, imported[0].ID, imported[1].ID)
	}
}
