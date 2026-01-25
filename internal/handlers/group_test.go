package handlers

import (
	"net/http"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestGroupHandler_List(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService)

	createTestGroup(t, db, "Group 1")
	createTestGroup(t, db, "Group 2")

	r := setupTestRouter()
	r.GET("/groups", handler.List)

	w := performRequest(r, "GET", "/groups", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Group]
	parseResponse(t, w, &response)

	if len(response.Data) != 2 {
		t.Errorf("expected 2 groups, got %d", len(response.Data))
	}
}

func TestGroupHandler_Get(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService)

	group := createTestGroup(t, db, "Test Group")

	r := setupTestRouter()
	r.GET("/groups/:id", handler.Get)

	w := performRequest(r, "GET", "/groups/1", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result models.Group
	parseResponse(t, w, &result)

	if result.Name != group.Name {
		t.Errorf("expected name '%s', got '%s'", group.Name, result.Name)
	}
}

func TestGroupHandler_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService)

	r := setupTestRouter()
	r.GET("/groups/:id", handler.Get)

	w := performRequest(r, "GET", "/groups/999", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGroupHandler_Create(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService)

	// Create an organization first
	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/groups", handler.Create)

	body := CreateGroupRequest{
		Name:           "New Group",
		OrganizationID: org.ID,
		Active:         true,
	}

	w := performRequest(r, "POST", "/groups", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result models.GroupResponse
	parseResponse(t, w, &result)

	if result.Name != "New Group" {
		t.Errorf("expected name 'New Group', got '%s'", result.Name)
	}
	if result.CreatedBy != "test@example.com" {
		t.Errorf("expected created_by 'test@example.com', got '%s'", result.CreatedBy)
	}
	if result.OrganizationID != org.ID {
		t.Errorf("expected organization_id %d, got %d", org.ID, result.OrganizationID)
	}
}

func TestGroupHandler_Create_BadRequest(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService)

	r := setupTestRouter()
	r.POST("/groups", handler.Create)

	// Missing required name field
	body := map[string]interface{}{
		"active": true,
	}

	w := performRequest(r, "POST", "/groups", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGroupHandler_Update(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService)

	createTestGroup(t, db, "Original Name")

	r := setupTestRouter()
	r.PUT("/groups/:id", handler.Update)

	body := UpdateGroupRequest{
		Name: "Updated Name",
	}

	w := performRequest(r, "PUT", "/groups/1", body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.Group
	parseResponse(t, w, &result)

	if result.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", result.Name)
	}
}

func TestGroupHandler_Delete(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService)

	createTestGroup(t, db, "To Delete")

	r := setupTestRouter()
	r.DELETE("/groups/:id", handler.Delete)

	w := performRequest(r, "DELETE", "/groups/1", nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Verify group was deleted
	var groups []models.Group
	db.Find(&groups)
	if len(groups) != 0 {
		t.Error("expected group to be deleted")
	}
}

func TestGroupHandler_Create_BadRequest_MissingOrganization(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService)

	r := setupTestRouter()
	r.POST("/groups", handler.Create)

	// Missing required organization_id field
	body := map[string]interface{}{
		"name":   "Test Group",
		"active": true,
	}

	w := performRequest(r, "POST", "/groups", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGroupHandler_Get_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService)

	r := setupTestRouter()
	r.GET("/groups/:id", handler.Get)

	w := performRequest(r, "GET", "/groups/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGroupHandler_Update_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService)

	r := setupTestRouter()
	r.PUT("/groups/:id", handler.Update)

	body := UpdateGroupRequest{
		Name: "Updated Name",
	}

	w := performRequest(r, "PUT", "/groups/invalid", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGroupHandler_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService)

	r := setupTestRouter()
	r.PUT("/groups/:id", handler.Update)

	body := UpdateGroupRequest{
		Name: "Updated Name",
	}

	w := performRequest(r, "PUT", "/groups/999", body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGroupHandler_Update_ActiveFlag(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService)

	createTestGroup(t, db, "Test Group")

	r := setupTestRouter()
	r.PUT("/groups/:id", handler.Update)

	active := false
	body := UpdateGroupRequest{
		Active: &active,
	}

	w := performRequest(r, "PUT", "/groups/1", body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.Group
	parseResponse(t, w, &result)

	if result.Active != false {
		t.Errorf("expected active false, got %v", result.Active)
	}
}

func TestGroupHandler_Delete_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService)

	r := setupTestRouter()
	r.DELETE("/groups/:id", handler.Delete)

	w := performRequest(r, "DELETE", "/groups/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGroupHandler_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService)

	r := setupTestRouter()
	r.GET("/groups", handler.List)

	w := performRequest(r, "GET", "/groups", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.GroupResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 0 {
		t.Errorf("expected 0 groups, got %d", len(response.Data))
	}
}

// Validation edge case tests

func TestGroupHandler_Create_WhitespaceOnlyName(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/groups", handler.Create)

	body := CreateGroupRequest{
		Name:           "   ",
		OrganizationID: org.ID,
		Active:         true,
	}

	w := performRequest(r, "POST", "/groups", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for whitespace-only name, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestGroupHandler_Create_NameTooLong(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/groups", handler.Create)

	// Create a name longer than 255 characters
	longName := ""
	for i := 0; i < 256; i++ {
		longName += "a"
	}

	body := CreateGroupRequest{
		Name:           longName,
		OrganizationID: org.ID,
		Active:         true,
	}

	w := performRequest(r, "POST", "/groups", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for name too long, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}
