package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func TestUserHandler_List(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	handler := NewUserHandler(userStore, groupStore)

	createTestUser(t, db, "User 1", "user1@example.com", "password")
	createTestUser(t, db, "User 2", "user2@example.com", "password")

	r := setupTestRouter()
	r.GET("/users", handler.List)

	w := performRequest(r, "GET", "/users", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var users []models.UserResponse
	parseResponse(t, w, &users)

	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestUserHandler_Get(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	handler := NewUserHandler(userStore, groupStore)

	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	r := setupTestRouter()
	r.GET("/users/:id", handler.Get)

	w := performRequest(r, "GET", "/users/1", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result models.UserResponse
	parseResponse(t, w, &result)

	if result.Name != user.Name {
		t.Errorf("expected name '%s', got '%s'", user.Name, result.Name)
	}
}

func TestUserHandler_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	handler := NewUserHandler(userStore, groupStore)

	r := setupTestRouter()
	r.GET("/users/:id", handler.Get)

	w := performRequest(r, "GET", "/users/999", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUserHandler_Create(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	handler := NewUserHandler(userStore, groupStore)

	r := setupTestRouter()
	r.POST("/users", handler.Create)

	body := models.UserCreate{
		Name:     "New User",
		Email:    "new@example.com",
		Password: "password123",
		Active:   true,
	}

	w := performRequest(r, "POST", "/users", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result models.UserResponse
	parseResponse(t, w, &result)

	if result.Name != "New User" {
		t.Errorf("expected name 'New User', got '%s'", result.Name)
	}
	if result.CreatedBy != "test@example.com" {
		t.Errorf("expected created_by 'test@example.com', got '%s'", result.CreatedBy)
	}
}

func TestUserHandler_Create_BadRequest(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	handler := NewUserHandler(userStore, groupStore)

	r := setupTestRouter()
	r.POST("/users", handler.Create)

	// Missing required fields
	body := map[string]interface{}{
		"active": true,
	}

	w := performRequest(r, "POST", "/users", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_Update(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	handler := NewUserHandler(userStore, groupStore)

	createTestUser(t, db, "Original Name", "test@example.com", "password")

	r := setupTestRouter()
	r.PUT("/users/:id", handler.Update)

	body := models.UserUpdate{
		Name: "Updated Name",
	}

	w := performRequest(r, "PUT", "/users/1", body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.UserResponse
	parseResponse(t, w, &result)

	if result.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", result.Name)
	}
}

func TestUserHandler_Delete(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	handler := NewUserHandler(userStore, groupStore)

	createTestUser(t, db, "To Delete", "delete@example.com", "password")

	r := setupTestRouter()
	r.DELETE("/users/:id", handler.Delete)

	w := performRequest(r, "DELETE", "/users/1", nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Verify user was deleted
	users, _ := userStore.FindAll()
	if len(users) != 0 {
		t.Error("expected user to be deleted")
	}
}

func TestUserHandler_AddToGroup(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	handler := NewUserHandler(userStore, groupStore)

	// Create org, group, and user
	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "Test Group", org.ID)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	// User must be in the organization before being added to a group
	_ = userStore.AddToOrganization(user.ID, org.ID)

	r := setupTestRouter()
	r.POST("/users/:id/groups", handler.AddToGroup)

	body := AddToGroupRequest{
		GroupID: group.ID,
	}

	w := performRequest(r, "POST", "/users/1/groups", body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify user was added to group
	foundUser, _ := userStore.FindByID(user.ID)
	if len(foundUser.Groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(foundUser.Groups))
	}
}

func TestUserHandler_AddToGroup_NotInOrganization(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	handler := NewUserHandler(userStore, groupStore)

	// Create group with its own org, and user without org membership
	group := createTestGroup(t, db, "Test Group")
	createTestUser(t, db, "Test User", "test@example.com", "password")

	r := setupTestRouter()
	r.POST("/users/:id/groups", handler.AddToGroup)

	body := AddToGroupRequest{
		GroupID: group.ID,
	}

	w := performRequest(r, "POST", "/users/1/groups", body)

	// Should fail because user is not in the group's organization
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d: %s", http.StatusForbidden, w.Code, w.Body.String())
	}
}

func TestUserHandler_RemoveFromGroup(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	handler := NewUserHandler(userStore, groupStore)

	// Create org, group, and user
	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "Test Group", org.ID)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	// Add user to org and group
	_ = userStore.AddToOrganization(user.ID, org.ID)
	_ = userStore.AddToGroup(user.ID, group.ID)

	r := setupTestRouter()
	r.DELETE("/users/:id/groups/:gid", handler.RemoveFromGroup)

	w := performRequest(r, "DELETE", fmt.Sprintf("/users/%d/groups/%d", user.ID, group.ID), nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
	}

	// Verify user was removed from group
	foundUser, _ := userStore.FindByID(user.ID)
	if len(foundUser.Groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(foundUser.Groups))
	}
}

func TestUserHandler_AddToOrganization(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	handler := NewUserHandler(userStore, groupStore)

	createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/users/:id/organizations", handler.AddToOrganization)

	body := AddToOrganizationRequest{
		OrganizationID: org.ID,
	}

	w := performRequest(r, "POST", "/users/1/organizations", body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify user was added to organization
	user, _ := userStore.FindByID(1)
	if len(user.Organizations) != 1 {
		t.Errorf("expected 1 organization, got %d", len(user.Organizations))
	}
}

func TestUserHandler_RemoveFromOrganization(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	handler := NewUserHandler(userStore, groupStore)

	createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")
	_ = userStore.AddToOrganization(1, org.ID)

	r := setupTestRouter()
	r.DELETE("/users/:id/organizations/:oid", handler.RemoveFromOrganization)

	w := performRequest(r, "DELETE", "/users/1/organizations/1", nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
	}

	// Verify user was removed from organization
	user, _ := userStore.FindByID(1)
	if len(user.Organizations) != 0 {
		t.Errorf("expected 0 organizations, got %d", len(user.Organizations))
	}
}
