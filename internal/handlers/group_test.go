package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestGroupHandler_List(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService, nil)

	org := createTestOrganization(t, db, "Test Org")
	createTestGroupWithOrg(t, db, "Group 1", org.ID)
	createTestGroupWithOrg(t, db, "Group 2", org.ID)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/groups", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/groups", org.ID), nil)

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
	handler := NewGroupHandler(groupService, nil)

	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "Test Group", org.ID)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/groups/:groupId", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/groups/%d", org.ID, group.ID), nil)

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
	handler := NewGroupHandler(groupService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/groups/:groupId", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/groups/999", org.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGroupHandler_Get_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService, nil)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	group := createTestGroupWithOrg(t, db, "Test Group", org1.ID)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/groups/:groupId", handler.Get)

	// Try to get group from org1 using org2's URL
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/groups/%d", org2.ID, group.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d when accessing group from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGroupHandler_Create(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/groups", handler.Create)

	body := GroupCreateRequest{
		Name:   "New Group",
		Active: true,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/groups", org.ID), body)

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
	handler := NewGroupHandler(groupService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/groups", handler.Create)

	// Missing required name field
	body := map[string]interface{}{
		"active": true,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/groups", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGroupHandler_Update(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService, nil)

	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "Original Name", org.ID)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/groups/:groupId", handler.Update)

	body := GroupUpdateRequest{
		Name: "Updated Name",
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/groups/%d", org.ID, group.ID), body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.Group
	parseResponse(t, w, &result)

	if result.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", result.Name)
	}
}

func TestGroupHandler_Update_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService, nil)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	group := createTestGroupWithOrg(t, db, "Test Group", org1.ID)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/groups/:groupId", handler.Update)

	body := GroupUpdateRequest{
		Name: "Updated Name",
	}

	// Try to update group from org1 using org2's URL
	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/groups/%d", org2.ID, group.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d when updating group from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGroupHandler_Delete(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "To Delete", org.ID)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/groups/:groupId", handler.Delete)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/groups/%d", org.ID, group.ID), nil)

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

func TestGroupHandler_Delete_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	group := createTestGroupWithOrg(t, db, "Test Group", org1.ID)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/groups/:groupId", handler.Delete)

	// Try to delete group from org1 using org2's URL
	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/groups/%d", org2.ID, group.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d when deleting group from wrong org, got %d", http.StatusNotFound, w.Code)
	}

	// Verify group was NOT deleted
	var groups []models.Group
	db.Find(&groups)
	if len(groups) != 1 {
		t.Error("expected group to still exist")
	}
}

func TestGroupHandler_Get_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/groups/:groupId", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/groups/invalid", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGroupHandler_Update_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/groups/:groupId", handler.Update)

	body := GroupUpdateRequest{
		Name: "Updated Name",
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/groups/invalid", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGroupHandler_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/groups/:groupId", handler.Update)

	body := GroupUpdateRequest{
		Name: "Updated Name",
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/groups/999", org.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGroupHandler_Update_ActiveFlag(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService, nil)

	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "Test Group", org.ID)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/groups/:groupId", handler.Update)

	active := false
	body := GroupUpdateRequest{
		Active: &active,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/groups/%d", org.ID, group.ID), body)

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
	handler := NewGroupHandler(groupService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/groups/:groupId", handler.Delete)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/groups/invalid", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGroupHandler_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/groups", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/groups", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.GroupResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 0 {
		t.Errorf("expected 0 groups, got %d", len(response.Data))
	}
}

func TestGroupHandler_List_OnlyShowsOrgGroups(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService, nil)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	createTestGroupWithOrg(t, db, "Group 1 Org1", org1.ID)
	createTestGroupWithOrg(t, db, "Group 2 Org1", org1.ID)
	createTestGroupWithOrg(t, db, "Group 1 Org2", org2.ID)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/groups", handler.List)

	// List groups for org1
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/groups", org1.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.GroupResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 2 {
		t.Errorf("expected 2 groups for org1, got %d", len(response.Data))
	}

	// List groups for org2
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/groups", org2.ID), nil)

	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Errorf("expected 1 group for org2, got %d", len(response.Data))
	}
}

// Validation edge case tests

func TestGroupHandler_Create_WhitespaceOnlyName(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/groups", handler.Create)

	body := GroupCreateRequest{
		Name:   "   ",
		Active: true,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/groups", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for whitespace-only name, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestGroupHandler_Create_NameTooLong(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/groups", handler.Create)

	// Create a name longer than 255 characters
	longName := ""
	for i := 0; i < 256; i++ {
		longName += "a"
	}

	body := GroupCreateRequest{
		Name:   longName,
		Active: true,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/groups", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for name too long, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestGroupHandler_List_Search(t *testing.T) {
	db := setupTestDB(t)
	groupService := createGroupService(db)
	handler := NewGroupHandler(groupService, nil)

	org := createTestOrganization(t, db, "Test Org")
	createTestGroupWithOrg(t, db, "Administrators", org.ID)
	createTestGroupWithOrg(t, db, "Admin Staff", org.ID)
	createTestGroupWithOrg(t, db, "Members", org.ID)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/groups", handler.List)

	// Search for "admin" (case-insensitive)
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/groups?search=admin", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.GroupResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 2 {
		t.Errorf("expected 2 groups matching 'admin', got %d", len(response.Data))
	}

	// Empty search returns all
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/groups", org.ID), nil)
	parseResponse(t, w, &response)

	if len(response.Data) != 3 {
		t.Errorf("expected 3 groups without search, got %d", len(response.Data))
	}
}
