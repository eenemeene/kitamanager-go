package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func TestUserHandler_List(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	createTestUser(t, db, "User 1", "user1@example.com", "password")
	createTestUser(t, db, "User 2", "user2@example.com", "password")

	r := setupTestRouterWithUser(admin.ID)
	r.GET("/users", handler.List)

	w := performRequest(r, "GET", "/users", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.UserResponse]
	parseResponse(t, w, &response)

	// 3 users: superadmin + 2 test users
	if len(response.Data) != 3 {
		t.Errorf("expected 3 users, got %d", len(response.Data))
	}
}

func TestUserHandler_ListByOrganization(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	// Create two orgs
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create users
	user1 := createTestUser(t, db, "User 1", "user1@example.com", "password")
	user2 := createTestUser(t, db, "User 2", "user2@example.com", "password")
	user3 := createTestUser(t, db, "User 3", "user3@example.com", "password")

	// user1 and user2 in org1, user3 in org2
	createTestUserOrganization(t, db, user1.ID, org1.ID, models.RoleMember)
	createTestUserOrganization(t, db, user2.ID, org1.ID, models.RoleMember)
	createTestUserOrganization(t, db, user3.ID, org2.ID, models.RoleMember)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/users", handler.ListByOrganization)

	// List users in org1
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/users", org1.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.UserResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 2 {
		t.Errorf("expected 2 users in org1, got %d", len(response.Data))
	}

	// List users in org2
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/users", org2.ID), nil)

	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Errorf("expected 1 user in org2, got %d", len(response.Data))
	}
}

func TestUserHandler_ListByOrganization_Empty(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	org := createTestOrganization(t, db, "Empty Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/users", handler.ListByOrganization)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/users", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.UserResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 0 {
		t.Errorf("expected 0 users, got %d", len(response.Data))
	}
}

func TestUserHandler_Get(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	r := setupTestRouter()
	r.GET("/users/:userId", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/users/%d", user.ID), nil)

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
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.GET("/users/:userId", handler.Get)

	w := performRequest(r, "GET", "/users/999", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUserHandler_Create(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.POST("/users", handler.Create)

	body := models.UserCreateRequest{
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
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

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
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	user := createTestUser(t, db, "Original Name", "test@example.com", "password")

	r := setupTestRouter()
	r.PUT("/users/:userId", handler.Update)

	body := models.UserUpdateRequest{
		Name: "Updated Name",
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d", user.ID), body)

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
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "To Delete", "delete@example.com", "password")

	r := setupTestRouterWithUser(admin.ID)
	r.DELETE("/users/:userId", handler.Delete)

	w := performRequest(r, "DELETE", fmt.Sprintf("/users/%d", user.ID), nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Verify target user was deleted (admin should remain)
	var users []models.User
	db.Find(&users)
	if len(users) != 1 {
		t.Errorf("expected 1 user (admin) remaining, got %d", len(users))
	}
}

func TestUserHandler_AddToOrganization(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouterWithUser(admin.ID)
	r.POST("/users/:userId/organizations", handler.AddToOrganization)

	body := models.UserAddOrganizationRequest{
		OrganizationID: org.ID,
		Role:           models.RoleMember,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/users/%d/organizations", user.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result models.UserOrganizationResponse
	parseResponse(t, w, &result)

	if result.UserID != user.ID {
		t.Errorf("expected user_id %d, got %d", user.ID, result.UserID)
	}
	if result.OrganizationID != org.ID {
		t.Errorf("expected organization_id %d, got %d", org.ID, result.OrganizationID)
	}
	if result.Role != models.RoleMember {
		t.Errorf("expected role %v, got %v", models.RoleMember, result.Role)
	}
}

func TestUserHandler_AddToOrganization_WithRole(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouterWithUser(admin.ID)
	r.POST("/users/:userId/organizations", handler.AddToOrganization)

	body := models.UserAddOrganizationRequest{
		OrganizationID: org.ID,
		Role:           models.RoleAdmin,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/users/%d/organizations", user.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result models.UserOrganizationResponse
	parseResponse(t, w, &result)

	if result.Role != models.RoleAdmin {
		t.Errorf("expected role %v, got %v", models.RoleAdmin, result.Role)
	}
}

func TestUserHandler_AddToOrganization_DefaultRole(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouterWithUser(admin.ID)
	r.POST("/users/:userId/organizations", handler.AddToOrganization)

	// No role specified - should default to member
	body := models.UserAddOrganizationRequest{
		OrganizationID: org.ID,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/users/%d/organizations", user.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result models.UserOrganizationResponse
	parseResponse(t, w, &result)

	if result.Role != models.RoleMember {
		t.Errorf("expected default role %v, got %v", models.RoleMember, result.Role)
	}
}

func TestUserHandler_UpdateOrganizationRole(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	// Add user to org as member first
	createTestUserOrganization(t, db, user.ID, org.ID, models.RoleMember)

	r := setupTestRouterWithUser(admin.ID)
	r.PUT("/users/:userId/organizations/:orgId", handler.UpdateOrganizationRole)

	body := models.UserOrganizationRoleUpdateRequest{
		Role: models.RoleAdmin,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/organizations/%d", user.ID, org.ID), body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.UserOrganizationResponse
	parseResponse(t, w, &result)

	if result.Role != models.RoleAdmin {
		t.Errorf("expected role %v, got %v", models.RoleAdmin, result.Role)
	}
}

func TestUserHandler_UpdateOrganizationRole_InvalidRole(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	createTestUserOrganization(t, db, user.ID, org.ID, models.RoleMember)

	r := setupTestRouter()
	r.PUT("/users/:userId/organizations/:orgId", handler.UpdateOrganizationRole)

	body := map[string]interface{}{
		"role": "invalid_role",
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/organizations/%d", user.ID, org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid role, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_UpdateOrganizationRole_UserNotInOrg(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	// User is NOT added to org

	r := setupTestRouter()
	r.PUT("/users/:userId/organizations/:orgId", handler.UpdateOrganizationRole)

	body := models.UserOrganizationRoleUpdateRequest{
		Role: models.RoleAdmin,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/organizations/%d", user.ID, org.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for user not in org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUserHandler_UpdateOrganizationRole_InvalidUserID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.PUT("/users/:userId/organizations/:orgId", handler.UpdateOrganizationRole)

	body := models.UserOrganizationRoleUpdateRequest{
		Role: models.RoleAdmin,
	}

	w := performRequest(r, "PUT", "/users/invalid/organizations/1", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_UpdateOrganizationRole_InvalidOrgID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	r := setupTestRouter()
	r.PUT("/users/:userId/organizations/:orgId", handler.UpdateOrganizationRole)

	body := models.UserOrganizationRoleUpdateRequest{
		Role: models.RoleAdmin,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/organizations/invalid", user.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_RemoveFromOrganization(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")
	createTestUserOrganization(t, db, user.ID, org.ID, models.RoleMember)

	r := setupTestRouterWithUser(admin.ID)
	r.DELETE("/users/:userId/organizations/:orgId", handler.RemoveFromOrganization)

	w := performRequest(r, "DELETE", fmt.Sprintf("/users/%d/organizations/%d", user.ID, org.ID), nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
	}

	// Verify membership was removed
	var count int64
	db.Model(&models.UserOrganization{}).Where("user_id = ? AND organization_id = ?", user.ID, org.ID).Count(&count)
	if count != 0 {
		t.Error("expected user to be removed from organization")
	}
}

// Edge case tests

func TestUserHandler_Get_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.GET("/users/:userId", handler.Get)

	w := performRequest(r, "GET", "/users/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_Get_ZeroID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.GET("/users/:userId", handler.Get)

	w := performRequest(r, "GET", "/users/0", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for zero ID, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUserHandler_Create_EmptyEmail(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.POST("/users", handler.Create)

	body := models.UserCreateRequest{
		Name:     "Test User",
		Email:    "",
		Password: "password123",
		Active:   true,
	}

	w := performRequest(r, "POST", "/users", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for empty email, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_Create_EmptyPassword(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.POST("/users", handler.Create)

	body := models.UserCreateRequest{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "",
		Active:   true,
	}

	w := performRequest(r, "POST", "/users", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for empty password, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_Create_EmptyName(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.POST("/users", handler.Create)

	body := models.UserCreateRequest{
		Name:     "",
		Email:    "test@example.com",
		Password: "password123",
		Active:   true,
	}

	w := performRequest(r, "POST", "/users", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for empty name, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_Create_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	createTestUser(t, db, "Existing User", "existing@example.com", "password")

	r := setupTestRouter()
	r.POST("/users", handler.Create)

	body := models.UserCreateRequest{
		Name:     "New User",
		Email:    "existing@example.com", // Duplicate
		Password: "password123",
		Active:   true,
	}

	w := performRequest(r, "POST", "/users", body)

	// Should fail due to unique constraint
	if w.Code == http.StatusCreated {
		t.Errorf("expected duplicate email to fail, but got status %d", w.Code)
	}
}

func TestUserHandler_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.PUT("/users/:userId", handler.Update)

	body := models.UserUpdateRequest{
		Name: "Updated Name",
	}

	w := performRequest(r, "PUT", "/users/999", body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUserHandler_Update_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.PUT("/users/:userId", handler.Update)

	body := models.UserUpdateRequest{
		Name: "Updated Name",
	}

	w := performRequest(r, "PUT", "/users/invalid", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.DELETE("/users/:userId", handler.Delete)

	w := performRequest(r, "DELETE", "/users/999", nil)

	// Should return NoContent (idempotent) or NotFound
	if w.Code != http.StatusNoContent && w.Code != http.StatusNotFound {
		t.Errorf("expected status %d or %d, got %d", http.StatusNoContent, http.StatusNotFound, w.Code)
	}
}

func TestUserHandler_Delete_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.DELETE("/users/:userId", handler.Delete)

	w := performRequest(r, "DELETE", "/users/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.GET("/users", handler.List)

	w := performRequest(r, "GET", "/users", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.UserResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 0 {
		t.Errorf("expected empty list, got %d users", len(response.Data))
	}
}

func TestUserHandler_AddToOrganization_InvalidUserID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.POST("/users/:userId/organizations", handler.AddToOrganization)

	body := models.UserAddOrganizationRequest{
		OrganizationID: 1,
	}

	w := performRequest(r, "POST", "/users/invalid/organizations", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_AddToOrganization_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouterWithUser(admin.ID)
	r.POST("/users/:userId/organizations", handler.AddToOrganization)

	body := models.UserAddOrganizationRequest{
		OrganizationID: org.ID,
	}

	w := performRequest(r, "POST", "/users/999/organizations", body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUserHandler_AddToOrganization_OrgNotFound(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	r := setupTestRouterWithUser(admin.ID)
	r.POST("/users/:userId/organizations", handler.AddToOrganization)

	body := models.UserAddOrganizationRequest{
		OrganizationID: 999, // Non-existent
	}

	w := performRequest(r, "POST", fmt.Sprintf("/users/%d/organizations", user.ID), body)

	// Non-existent org triggers a FK constraint error at the store level (500)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d for non-existent org, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestUserHandler_RemoveFromOrganization_InvalidUserID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.DELETE("/users/:userId/organizations/:orgId", handler.RemoveFromOrganization)

	w := performRequest(r, "DELETE", "/users/invalid/organizations/1", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_RemoveFromOrganization_InvalidOrgID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	r := setupTestRouter()
	r.DELETE("/users/:userId/organizations/:orgId", handler.RemoveFromOrganization)

	w := performRequest(r, "DELETE", fmt.Sprintf("/users/%d/organizations/invalid", user.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_RemoveFromOrganization_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouterWithUser(admin.ID)
	r.DELETE("/users/:userId/organizations/:orgId", handler.RemoveFromOrganization)

	w := performRequest(r, "DELETE", fmt.Sprintf("/users/999/organizations/%d", org.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// Validation edge case tests

func TestUserHandler_Create_WhitespaceOnlyName(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.POST("/users", handler.Create)

	body := models.UserCreateRequest{
		Name:     "   ",
		Email:    "test@example.com",
		Password: "password123",
		Active:   true,
	}

	w := performRequest(r, "POST", "/users", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for whitespace-only name, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestUserHandler_Create_NameTooLong(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.POST("/users", handler.Create)

	// Create a name longer than 255 characters
	longName := string(make([]byte, 256))
	for i := range longName {
		longName = longName[:i] + "a" + longName[i+1:]
	}

	body := models.UserCreateRequest{
		Name:     longName,
		Email:    "test@example.com",
		Password: "password123",
		Active:   true,
	}

	w := performRequest(r, "POST", "/users", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for name too long, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_Create_PasswordTooShort(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.POST("/users", handler.Create)

	body := models.UserCreateRequest{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "1234567", // 7 chars, min is 8
		Active:   true,
	}

	w := performRequest(r, "POST", "/users", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for password too short, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_Create_PasswordTooLong(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.POST("/users", handler.Create)

	// Create a password longer than 72 characters
	longPassword := string(make([]byte, 73))
	for i := range longPassword {
		longPassword = longPassword[:i] + "a" + longPassword[i+1:]
	}

	body := models.UserCreateRequest{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: longPassword,
		Active:   true,
	}

	w := performRequest(r, "POST", "/users", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for password too long, got %d", http.StatusBadRequest, w.Code)
	}
}

// GetMemberships tests

func TestUserHandler_GetMemberships(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	// Add user to two orgs
	createTestUserOrganization(t, db, user.ID, org1.ID, models.RoleAdmin)
	createTestUserOrganization(t, db, user.ID, org2.ID, models.RoleMember)

	r := setupTestRouter()
	r.GET("/users/:userId/memberships", handler.GetMemberships)

	w := performRequest(r, "GET", fmt.Sprintf("/users/%d/memberships", user.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.UserMembershipsResponse
	parseResponse(t, w, &result)

	if len(result.Memberships) != 2 {
		t.Errorf("expected 2 memberships, got %d", len(result.Memberships))
	}
}

func TestUserHandler_GetMemberships_RolesCorrect(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	// Admin in org1, member in org2
	createTestUserOrganization(t, db, user.ID, org1.ID, models.RoleAdmin)
	createTestUserOrganization(t, db, user.ID, org2.ID, models.RoleMember)

	r := setupTestRouter()
	r.GET("/users/:userId/memberships", handler.GetMemberships)

	w := performRequest(r, "GET", fmt.Sprintf("/users/%d/memberships", user.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result models.UserMembershipsResponse
	parseResponse(t, w, &result)

	// Check that roles are correctly returned
	rolesByOrg := make(map[uint]models.Role)
	for _, m := range result.Memberships {
		rolesByOrg[m.OrganizationID] = m.Role
	}

	if rolesByOrg[org1.ID] != models.RoleAdmin {
		t.Errorf("expected role admin in org1, got %v", rolesByOrg[org1.ID])
	}
	if rolesByOrg[org2.ID] != models.RoleMember {
		t.Errorf("expected role member in org2, got %v", rolesByOrg[org2.ID])
	}
}

func TestUserHandler_GetMemberships_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.GET("/users/:userId/memberships", handler.GetMemberships)

	w := performRequest(r, "GET", "/users/999/memberships", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUserHandler_GetMemberships_Empty(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	r := setupTestRouter()
	r.GET("/users/:userId/memberships", handler.GetMemberships)

	w := performRequest(r, "GET", fmt.Sprintf("/users/%d/memberships", user.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result models.UserMembershipsResponse
	parseResponse(t, w, &result)

	if len(result.Memberships) != 0 {
		t.Errorf("expected 0 memberships, got %d", len(result.Memberships))
	}
}

func TestUserHandler_GetMemberships_InvalidUserID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.GET("/users/:userId/memberships", handler.GetMemberships)

	w := performRequest(r, "GET", "/users/invalid/memberships", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// SetSuperAdmin tests

func TestUserHandler_SetSuperAdmin_True(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	r := setupTestRouter()
	r.PUT("/users/:userId/superadmin", handler.SetSuperAdmin)

	body := models.UserSetSuperAdminRequest{
		IsSuperAdmin: true,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/superadmin", user.ID), body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.UserResponse
	parseResponse(t, w, &result)

	if !result.IsSuperAdmin {
		t.Error("expected IsSuperAdmin = true")
	}
}

func TestUserHandler_SetSuperAdmin_False(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	// Create two superadmins so we can demote one
	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	db.Model(&models.User{}).Where("id = ?", user.ID).Update("is_superadmin", true)

	r := setupTestRouterWithUser(admin.ID)
	r.PUT("/users/:userId/superadmin", handler.SetSuperAdmin)

	body := models.UserSetSuperAdminRequest{
		IsSuperAdmin: false,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/superadmin", user.ID), body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result models.UserResponse
	parseResponse(t, w, &result)

	if result.IsSuperAdmin {
		t.Error("expected IsSuperAdmin = false")
	}
}

func TestUserHandler_SetSuperAdmin_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.PUT("/users/:userId/superadmin", handler.SetSuperAdmin)

	body := models.UserSetSuperAdminRequest{
		IsSuperAdmin: true,
	}

	w := performRequest(r, "PUT", "/users/999/superadmin", body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUserHandler_SetSuperAdmin_InvalidUserID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	r := setupTestRouter()
	r.PUT("/users/:userId/superadmin", handler.SetSuperAdmin)

	body := models.UserSetSuperAdminRequest{
		IsSuperAdmin: true,
	}

	w := performRequest(r, "PUT", "/users/invalid/superadmin", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_SetSuperAdmin_MissingBody(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	r := setupTestRouter()
	r.PUT("/users/:userId/superadmin", handler.SetSuperAdmin)

	// Send empty body
	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/superadmin", user.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for missing body, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_List_Search(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	createTestUser(t, db, "Alice Smith", "alice@example.com", "password")
	createTestUser(t, db, "Bob Jones", "bob@example.com", "password")
	createTestUser(t, db, "Charlie Admin", "admin@company.com", "password")

	r := setupTestRouterWithUser(admin.ID)
	r.GET("/users", handler.List)

	// Search by name
	w := performRequest(r, "GET", "/users?search=alice", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.UserResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Errorf("expected 1 user matching 'alice', got %d", len(response.Data))
	}

	// Search by email - "admin" matches both superadmin and Charlie Admin
	w = performRequest(r, "GET", "/users?search=company.com", nil)
	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Errorf("expected 1 user matching 'company.com', got %d", len(response.Data))
	}

	// Empty search returns all (4: superadmin + 3 test users)
	w = performRequest(r, "GET", "/users", nil)
	parseResponse(t, w, &response)

	if len(response.Data) != 4 {
		t.Errorf("expected 4 users without search, got %d", len(response.Data))
	}
}

func TestUserHandler_ResetPassword_Success(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	targetUser := createTestUser(t, db, "Target User", "target@example.com", "oldpassword")

	userService := createUserService(db)
	auditService := createAuditService(db)
	tokenStore := store.NewTokenStore(db)
	handler := NewUserHandler(userService, nil, auditService, tokenStore)

	r := setupTestRouterWithUser(admin.ID)
	r.PUT("/users/:userId/password", handler.ResetPassword)

	body := models.UserPasswordResetRequest{
		NewPassword: "newpassword123",
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/password", targetUser.ID), body)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
	}
}

func TestUserHandler_ResetPassword_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)

	userService := createUserService(db)
	auditService := createAuditService(db)
	handler := NewUserHandler(userService, nil, auditService, nil)

	r := setupTestRouterWithUser(admin.ID)
	r.PUT("/users/:userId/password", handler.ResetPassword)

	body := models.UserPasswordResetRequest{
		NewPassword: "newpassword123",
	}

	w := performRequest(r, "PUT", "/users/99999/password", body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestUserHandler_ResetPassword_BadRequest_MissingPassword(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	targetUser := createTestUser(t, db, "Target User", "target@example.com", "oldpassword")

	userService := createUserService(db)
	auditService := createAuditService(db)
	handler := NewUserHandler(userService, nil, auditService, nil)

	r := setupTestRouterWithUser(admin.ID)
	r.PUT("/users/:userId/password", handler.ResetPassword)

	// Empty body
	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/password", targetUser.ID), map[string]string{})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestUserHandler_ResetPassword_PasswordTooShort(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	targetUser := createTestUser(t, db, "Target User", "target@example.com", "oldpassword")

	userService := createUserService(db)
	auditService := createAuditService(db)
	handler := NewUserHandler(userService, nil, auditService, nil)

	r := setupTestRouterWithUser(admin.ID)
	r.PUT("/users/:userId/password", handler.ResetPassword)

	body := models.UserPasswordResetRequest{
		NewPassword: "short",
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/password", targetUser.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for short password, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestUserHandler_ResetPassword_InvalidUserID(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)

	userService := createUserService(db)
	auditService := createAuditService(db)
	handler := NewUserHandler(userService, nil, auditService, nil)

	r := setupTestRouterWithUser(admin.ID)
	r.PUT("/users/:userId/password", handler.ResetPassword)

	body := models.UserPasswordResetRequest{
		NewPassword: "newpassword123",
	}

	w := performRequest(r, "PUT", "/users/abc/password", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid userId, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestUserHandler_ResetPassword_NonSuperAdminCannotResetSuperAdmin(t *testing.T) {
	db := setupTestDB(t)

	org := createTestOrganization(t, db, "Test Org")
	adminUser := createTestUser(t, db, "Admin User", "admin@example.com", "password")
	createTestUserOrganization(t, db, adminUser.ID, org.ID, models.RoleAdmin)

	superAdmin := createTestSuperAdmin(t, db)

	userService := createUserService(db)
	auditService := createAuditService(db)
	handler := NewUserHandler(userService, nil, auditService, nil)

	r := setupTestRouterWithUser(adminUser.ID)
	r.PUT("/users/:userId/password", handler.ResetPassword)

	body := models.UserPasswordResetRequest{
		NewPassword: "hacked12345",
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/password", superAdmin.ID), body)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d: %s", http.StatusForbidden, w.Code, w.Body.String())
	}
}

func TestUserHandler_ResetPassword_TokensRevoked(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	targetUser := createTestUser(t, db, "Target User", "target@example.com", string(hashedPassword))

	authHandler := createAuthHandler(db)
	userService := createUserService(db)
	auditService := createAuditService(db)
	tokenStore := store.NewTokenStore(db)
	userHandler := NewUserHandler(userService, nil, auditService, tokenStore)

	// Login as the target user to create tokens
	loginRouter := gin.New()
	loginRouter.POST("/login", authHandler.Login)

	loginBody := models.LoginRequest{
		Email:    "target@example.com",
		Password: "password123",
	}
	loginResp := performRequest(loginRouter, "POST", "/login", loginBody)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("login failed: %d: %s", loginResp.Code, loginResp.Body.String())
	}

	// Admin resets the target user's password
	r := setupTestRouterWithUser(admin.ID)
	r.PUT("/users/:userId/password", userHandler.ResetPassword)

	resetBody := models.UserPasswordResetRequest{
		NewPassword: "newpassword456",
	}
	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/password", targetUser.ID), resetBody)
	if w.Code != http.StatusNoContent {
		t.Fatalf("reset failed: %d: %s", w.Code, w.Body.String())
	}

	// Old refresh token should no longer work
	refreshRouter := gin.New()
	refreshRouter.POST("/refresh", authHandler.Refresh)

	refreshResp := performRequestWithCookies(refreshRouter, "POST", "/refresh", nil, loginResp.Result().Cookies())
	if refreshResp.Code != http.StatusUnauthorized {
		t.Errorf("expected old refresh token to fail with %d after password reset, got %d", http.StatusUnauthorized, refreshResp.Code)
	}
}
