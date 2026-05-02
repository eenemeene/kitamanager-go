package handlers

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
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
	body := map[string]any{
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

	body := map[string]any{
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

	// Self-view returns every membership.
	r := setupTestRouterWithUser(user.ID)
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

	r := setupTestRouterWithUser(user.ID)
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

	// Self-view returns empty list (not NotFound).
	r := setupTestRouterWithUser(user.ID)
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

// Scoping tests — H10. Without scoping, any user with users:read in any org
// could enumerate every other user's full organization graph.

func TestUserHandler_GetMemberships_CrossTenantRequesterGetsNotFound(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	orgA := createTestOrganization(t, db, "Org A")
	orgB := createTestOrganization(t, db, "Org B")
	requester := createTestUser(t, db, "Requester", "requester@example.com", "password")
	target := createTestUser(t, db, "Target", "target@example.com", "password")

	// Requester is only in Org A. Target is only in Org B. Zero overlap.
	createTestUserOrganization(t, db, requester.ID, orgA.ID, models.RoleAdmin)
	createTestUserOrganization(t, db, target.ID, orgB.ID, models.RoleAdmin)

	r := setupTestRouterWithUser(requester.ID)
	r.GET("/users/:userId/memberships", handler.GetMemberships)

	w := performRequest(r, "GET", fmt.Sprintf("/users/%d/memberships", target.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected %d (no overlap must not leak target's existence), got %d: %s",
			http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestUserHandler_GetMemberships_PartialOverlapReturnsOnlySharedOrgs(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	orgShared := createTestOrganization(t, db, "Shared Org")
	orgPrivate := createTestOrganization(t, db, "Target's Private Org")
	requester := createTestUser(t, db, "Requester", "requester@example.com", "password")
	target := createTestUser(t, db, "Target", "target@example.com", "password")

	// Requester and target both in orgShared. Target additionally in orgPrivate.
	createTestUserOrganization(t, db, requester.ID, orgShared.ID, models.RoleMember)
	createTestUserOrganization(t, db, target.ID, orgShared.ID, models.RoleManager)
	createTestUserOrganization(t, db, target.ID, orgPrivate.ID, models.RoleAdmin)

	r := setupTestRouterWithUser(requester.ID)
	r.GET("/users/:userId/memberships", handler.GetMemberships)

	w := performRequest(r, "GET", fmt.Sprintf("/users/%d/memberships", target.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.UserMembershipsResponse
	parseResponse(t, w, &result)
	if len(result.Memberships) != 1 {
		t.Fatalf("expected exactly 1 membership (the shared org), got %d", len(result.Memberships))
	}
	if result.Memberships[0].OrganizationID != orgShared.ID {
		t.Errorf("expected shared org %d, got %d", orgShared.ID, result.Memberships[0].OrganizationID)
	}
	if result.Memberships[0].Role != models.RoleManager {
		t.Errorf("expected manager role, got %v", result.Memberships[0].Role)
	}
}

func TestUserHandler_GetMemberships_SuperAdminSeesEverything(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	superAdmin := createTestSuperAdmin(t, db)
	orgA := createTestOrganization(t, db, "Org A")
	orgB := createTestOrganization(t, db, "Org B")
	target := createTestUser(t, db, "Target", "target@example.com", "password")

	// Superadmin is in no orgs but must still see target's full graph.
	createTestUserOrganization(t, db, target.ID, orgA.ID, models.RoleAdmin)
	createTestUserOrganization(t, db, target.ID, orgB.ID, models.RoleMember)

	r := setupTestRouterWithUser(superAdmin.ID)
	r.GET("/users/:userId/memberships", handler.GetMemberships)

	w := performRequest(r, "GET", fmt.Sprintf("/users/%d/memberships", target.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.UserMembershipsResponse
	parseResponse(t, w, &result)
	if len(result.Memberships) != 2 {
		t.Errorf("superadmin must see both memberships, got %d", len(result.Memberships))
	}
}

// SetSuperAdmin tests
//
// Every PUT /users/:userId/superadmin call carries a step-up authentication
// factor in `actor_password` — see review finding H1. The route is
// superadmin-only at the router level; without step-up a stolen superadmin
// session could mint a confederate. The tests below cover the full matrix:
// happy paths (grant, demote with multiple superadmins), step-up failure
// (wrong / missing actor_password), pre-existing guards (last superadmin,
// not-found target), and forensic audit emission.

const setSuperAdminActorPw = "adminpw"

// setupSetSuperAdminTest wires the handler with a real audit service and
// returns the router (with the admin actor in context), the admin actor,
// and the audit service so individual tests can assert audit emission.
func setupSetSuperAdminTest(t *testing.T) (*gin.Engine, *models.User, *service.AuditService, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)
	auditSvc := createAuditService(db)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, auditSvc, nil)

	admin := createTestSuperAdmin(t, db)
	hashedAdminPw(t, db, admin.ID, setSuperAdminActorPw)

	r := setupTestRouterWithUser(admin.ID)
	r.PUT("/users/:userId/superadmin", handler.SetSuperAdmin)
	return r, admin, auditSvc, db
}

// drainAudit forces the async audit worker to flush so tests can read
// rows synchronously without sleeping. Calling Shutdown closes the
// channel and waits for the worker to drain.
func drainAudit(svc *service.AuditService) {
	svc.Shutdown()
}

func TestUserHandler_SetSuperAdmin_GrantSucceedsWithCorrectActorPassword(t *testing.T) {
	r, _, auditSvc, db := setupSetSuperAdminTest(t)
	user := createTestUser(t, db, "Target", "target@example.com", "password")

	body := models.UserSetSuperAdminRequest{
		IsSuperAdmin:  true,
		ActorPassword: setSuperAdminActorPw,
	}
	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/superadmin", user.ID), body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result models.UserResponse
	parseResponse(t, w, &result)
	if !result.IsSuperAdmin {
		t.Error("expected IsSuperAdmin=true on response")
	}

	// Verify the DB was actually updated and a success audit row landed.
	var fresh models.User
	if err := db.First(&fresh, user.ID).Error; err != nil {
		t.Fatalf("re-fetch user: %v", err)
	}
	if !fresh.IsSuperAdmin {
		t.Error("DB row not updated to is_superadmin=true")
	}

	drainAudit(auditSvc)
	var grants []models.AuditLog
	if err := db.Where("action = ?", models.AuditActionSuperAdminGrant).Find(&grants).Error; err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if len(grants) != 1 {
		t.Fatalf("expected 1 superadmin_grant audit row, got %d", len(grants))
	}
}

func TestUserHandler_SetSuperAdmin_RevokeSucceedsWithCorrectActorPassword(t *testing.T) {
	r, _, _, db := setupSetSuperAdminTest(t)
	// Create another superadmin so we can demote them without violating the
	// last-superadmin guard.
	target := createTestUser(t, db, "Other Admin", "other-admin@example.com", "password")
	db.Model(&models.User{}).Where("id = ?", target.ID).Update("is_superadmin", true)

	body := models.UserSetSuperAdminRequest{
		IsSuperAdmin:  false,
		ActorPassword: setSuperAdminActorPw,
	}
	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/superadmin", target.ID), body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var fresh models.User
	if err := db.First(&fresh, target.ID).Error; err != nil {
		t.Fatalf("re-fetch user: %v", err)
	}
	if fresh.IsSuperAdmin {
		t.Error("DB row not updated to is_superadmin=false")
	}
}

func TestUserHandler_SetSuperAdmin_RejectsWrongActorPassword(t *testing.T) {
	r, _, auditSvc, db := setupSetSuperAdminTest(t)
	user := createTestUser(t, db, "Target", "target@example.com", "password")

	body := models.UserSetSuperAdminRequest{
		IsSuperAdmin:  true,
		ActorPassword: "WRONG",
	}
	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/superadmin", user.ID), body)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}

	// CRITICAL: the target user MUST NOT have been promoted.
	var fresh models.User
	if err := db.First(&fresh, user.ID).Error; err != nil {
		t.Fatalf("re-fetch user: %v", err)
	}
	if fresh.IsSuperAdmin {
		t.Fatal("target was promoted to superadmin despite wrong actor_password — H1 regression")
	}

	// Forensic audit row MUST exist with action=superadmin_change_failed.
	drainAudit(auditSvc)
	var failures []models.AuditLog
	if err := db.Where("action = ?", models.AuditActionSuperAdminChangeFailed).Find(&failures).Error; err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if len(failures) != 1 {
		t.Fatalf("expected 1 superadmin_change_failed audit row, got %d", len(failures))
	}
	if failures[0].ResourceID == nil || *failures[0].ResourceID != user.ID {
		t.Errorf("audit row should reference target user_id %d, got %v", user.ID, failures[0].ResourceID)
	}
	if failures[0].Success {
		t.Error("audit row Success should be false on step-up failure")
	}
}

func TestUserHandler_SetSuperAdmin_RejectsMissingActorPassword(t *testing.T) {
	r, _, _, db := setupSetSuperAdminTest(t)
	user := createTestUser(t, db, "Target", "target@example.com", "password")

	// Send body with IsSuperAdmin only — actor_password is required by the
	// binding tag, so this must surface as 400 from bindJSON, not as a
	// successful mutation.
	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/superadmin", user.ID),
		map[string]any{"is_superadmin": true})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing actor_password, got %d: %s", w.Code, w.Body.String())
	}
	var fresh models.User
	if err := db.First(&fresh, user.ID).Error; err != nil {
		t.Fatalf("re-fetch user: %v", err)
	}
	if fresh.IsSuperAdmin {
		t.Fatal("target was promoted despite missing actor_password — H1 regression")
	}
}

func TestUserHandler_SetSuperAdmin_RejectsEmptyActorPassword(t *testing.T) {
	r, _, _, db := setupSetSuperAdminTest(t)
	user := createTestUser(t, db, "Target", "target@example.com", "password")

	// Explicit empty string vs. missing field — both must be rejected.
	// `binding:"required"` treats empty string as missing for strings.
	body := models.UserSetSuperAdminRequest{
		IsSuperAdmin:  true,
		ActorPassword: "",
	}
	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/superadmin", user.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty actor_password, got %d", w.Code)
	}
	var fresh models.User
	_ = db.First(&fresh, user.ID).Error
	if fresh.IsSuperAdmin {
		t.Fatal("target was promoted despite empty actor_password — H1 regression")
	}
}

func TestUserHandler_SetSuperAdmin_DemoteLastSuperAdminBlockedEvenWithCorrectStepUp(t *testing.T) {
	// Even though the actor passes step-up, the last-superadmin guard inside
	// the service must still fire. This proves the step-up check is layered
	// on top of pre-existing safety checks, not a replacement for them.
	r, admin, _, db := setupSetSuperAdminTest(t)

	body := models.UserSetSuperAdminRequest{
		IsSuperAdmin:  false,
		ActorPassword: setSuperAdminActorPw,
	}
	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/superadmin", admin.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (cannot demote last superadmin), got %d: %s", w.Code, w.Body.String())
	}
	var fresh models.User
	if err := db.First(&fresh, admin.ID).Error; err != nil {
		t.Fatalf("re-fetch admin: %v", err)
	}
	if !fresh.IsSuperAdmin {
		t.Fatal("last superadmin was demoted")
	}
}

func TestUserHandler_SetSuperAdmin_TargetUserNotFound(t *testing.T) {
	r, _, _, _ := setupSetSuperAdminTest(t)

	// Body is well-formed (valid actor_password) so the request reaches
	// the GetByID(target) lookup, which returns 404 for an unknown id.
	body := models.UserSetSuperAdminRequest{
		IsSuperAdmin:  true,
		ActorPassword: setSuperAdminActorPw,
	}
	w := performRequest(r, "PUT", "/users/999999/superadmin", body)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandler_SetSuperAdmin_InvalidUserIDPath(t *testing.T) {
	r, _, _, _ := setupSetSuperAdminTest(t)

	body := models.UserSetSuperAdminRequest{
		IsSuperAdmin:  true,
		ActorPassword: setSuperAdminActorPw,
	}
	w := performRequest(r, "PUT", "/users/invalid/superadmin", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-numeric userId, got %d", w.Code)
	}
}

func TestUserHandler_SetSuperAdmin_EmptyBody(t *testing.T) {
	r, _, _, db := setupSetSuperAdminTest(t)
	user := createTestUser(t, db, "Target", "target@example.com", "password")

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/superadmin", user.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d", w.Code)
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
	hashedAdminPw(t, db, admin.ID, "adminpw")
	targetUser := createTestUser(t, db, "Target User", "target@example.com", "oldpassword")

	userService := createUserService(db)
	auditService := createAuditService(db)
	sessionStore := store.NewSessionStore(db)
	handler := NewUserHandler(userService, nil, auditService, sessionStore)

	r := setupTestRouterWithUser(admin.ID)
	r.PUT("/users/:userId/password", handler.ResetPassword)

	body := models.UserPasswordResetRequest{
		ActorPassword: "adminpw",
		NewPassword:   "newpassword123",
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/password", targetUser.ID), body)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
	}
}

func TestUserHandler_ResetPassword_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	hashedAdminPw(t, db, admin.ID, "adminpw")

	userService := createUserService(db)
	auditService := createAuditService(db)
	handler := NewUserHandler(userService, nil, auditService, nil)

	r := setupTestRouterWithUser(admin.ID)
	r.PUT("/users/:userId/password", handler.ResetPassword)

	body := models.UserPasswordResetRequest{
		ActorPassword: "adminpw",
		NewPassword:   "newpassword123",
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
	hashedAdminPw(t, db, admin.ID, "adminpw")
	targetUser := createTestUser(t, db, "Target User", "target@example.com", "oldpassword")

	userService := createUserService(db)
	auditService := createAuditService(db)
	handler := NewUserHandler(userService, nil, auditService, nil)

	r := setupTestRouterWithUser(admin.ID)
	r.PUT("/users/:userId/password", handler.ResetPassword)

	body := models.UserPasswordResetRequest{
		ActorPassword: "adminpw",
		NewPassword:   "short",
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/password", targetUser.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for short password, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestUserHandler_ResetPassword_InvalidUserID(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	hashedAdminPw(t, db, admin.ID, "adminpw")

	userService := createUserService(db)
	auditService := createAuditService(db)
	handler := NewUserHandler(userService, nil, auditService, nil)

	r := setupTestRouterWithUser(admin.ID)
	r.PUT("/users/:userId/password", handler.ResetPassword)

	body := models.UserPasswordResetRequest{
		ActorPassword: "adminpw",
		NewPassword:   "newpassword123",
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
	hashedAdminPw(t, db, adminUser.ID, "adminpw")
	createTestUserOrganization(t, db, adminUser.ID, org.ID, models.RoleAdmin)

	superAdmin := createTestSuperAdmin(t, db)

	userService := createUserService(db)
	auditService := createAuditService(db)
	handler := NewUserHandler(userService, nil, auditService, nil)

	r := setupTestRouterWithUser(adminUser.ID)
	r.PUT("/users/:userId/password", handler.ResetPassword)

	body := models.UserPasswordResetRequest{
		ActorPassword: "adminpw",
		NewPassword:   "hacked12345",
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/password", superAdmin.ID), body)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d: %s", http.StatusForbidden, w.Code, w.Body.String())
	}
}

// M1 — admin-initiated password reset must require the actor's own current
// password (step-up). Without this, a compromised admin session can silently
// rotate a peer user's password.
func TestUserHandler_ResetPassword_RejectsWrongActorPassword(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	hashedAdminPw(t, db, admin.ID, "correctpw")
	targetUser := createTestUser(t, db, "Target", "target@example.com", "oldpassword")

	userService := createUserService(db)
	auditService := createAuditService(db)
	handler := NewUserHandler(userService, nil, auditService, nil)

	r := setupTestRouterWithUser(admin.ID)
	r.PUT("/users/:userId/password", handler.ResetPassword)

	body := models.UserPasswordResetRequest{
		ActorPassword: "WRONG",
		NewPassword:   "newpassword123",
	}
	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/password", targetUser.ID), body)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected %d with wrong actor password, got %d: %s",
			http.StatusUnauthorized, w.Code, w.Body.String())
	}

	// The target user's password MUST NOT have been changed.
	var after models.User
	_ = db.First(&after, targetUser.ID).Error
	if after.Password != "oldpassword" {
		t.Errorf("target password changed despite wrong actor_password (%q != %q)",
			after.Password, "oldpassword")
	}
}

func TestUserHandler_ResetPassword_SessionsRevoked(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	hashedAdminPw(t, db, admin.ID, "adminpw")

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	targetUser := createTestUser(t, db, "Target User", "target@example.com", string(hashedPassword))

	authHandler := createAuthHandler(db)
	userService := createUserService(db)
	auditService := createAuditService(db)
	sessionStore := store.NewSessionStore(db)
	userHandler := NewUserHandler(userService, nil, auditService, sessionStore)

	// Target user logs in — this creates a session row we expect to be wiped.
	loginRouter := gin.New()
	loginRouter.POST("/login", authHandler.Login)
	loginResp := performRequest(loginRouter, "POST", "/login", models.LoginRequest{
		Email:    "target@example.com",
		Password: "password123",
	})
	if loginResp.Code != http.StatusOK {
		t.Fatalf("login failed: %d: %s", loginResp.Code, loginResp.Body.String())
	}

	var sessionCount int64
	db.Model(&models.Session{}).Where("user_id = ?", targetUser.ID).Count(&sessionCount)
	if sessionCount != 1 {
		t.Fatalf("precondition: expected 1 session, got %d", sessionCount)
	}

	// Admin resets the target user's password.
	r := setupTestRouterWithUser(admin.ID)
	r.PUT("/users/:userId/password", userHandler.ResetPassword)
	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/password", targetUser.ID), models.UserPasswordResetRequest{
		ActorPassword: "adminpw",
		NewPassword:   "newpassword456",
	})
	if w.Code != http.StatusNoContent {
		t.Fatalf("reset failed: %d: %s", w.Code, w.Body.String())
	}

	// Target user's sessions must be gone.
	db.Model(&models.Session{}).Where("user_id = ?", targetUser.ID).Count(&sessionCount)
	if sessionCount != 0 {
		t.Errorf("expected target user's sessions to be revoked, %d remain", sessionCount)
	}
}

// ----------------------------------------------------------------------
// Purge endpoint — DSGVO Art. 17 erasure (closes audit finding O-H-3).
// ----------------------------------------------------------------------

func TestUserHandler_Purge_Success(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	auditStore := store.NewAuditStore(db)
	auditService := service.NewAuditService(auditStore)
	t.Cleanup(auditService.Shutdown)
	handler := NewUserHandler(userService, userOrgService, auditService, nil)

	admin := createTestSuperAdmin(t, db)
	target := createTestUser(t, db, "Doomed", "doomed@example.com", "password")

	r := setupTestRouterWithUser(admin.ID)
	r.DELETE("/users/:userId/purge", handler.Purge)

	w := performRequest(r, "DELETE", fmt.Sprintf("/users/%d/purge", target.ID), nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", w.Code, w.Body.String())
	}

	// Row physically gone — even an unscoped lookup returns zero rows.
	var count int64
	db.Unscoped().Model(&models.User{}).Where("id = ?", target.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected user purged, %d row(s) remain", count)
	}
}

func TestUserHandler_Purge_AlreadyTombstoned_StillSucceeds(t *testing.T) {
	// Art. 17 requests sometimes arrive AFTER the user has already
	// soft-deleted themselves. Purge must still vaporise the row.
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	auditStore := store.NewAuditStore(db)
	auditService := service.NewAuditService(auditStore)
	t.Cleanup(auditService.Shutdown)
	handler := NewUserHandler(userService, userOrgService, auditService, nil)

	admin := createTestSuperAdmin(t, db)
	target := createTestUser(t, db, "Tomb", "tomb@example.com", "password")
	// Soft-delete first (sets deleted_at).
	if err := db.Delete(target).Error; err != nil {
		t.Fatalf("soft-delete: %v", err)
	}

	r := setupTestRouterWithUser(admin.ID)
	r.DELETE("/users/:userId/purge", handler.Purge)

	w := performRequest(r, "DELETE", fmt.Sprintf("/users/%d/purge", target.ID), nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on already-tombstoned, got %d body=%s", w.Code, w.Body.String())
	}
	var count int64
	db.Unscoped().Model(&models.User{}).Where("id = ?", target.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected hard-delete to remove the row, %d remain", count)
	}
}

func TestUserHandler_Purge_SelfTarget_BadRequest(t *testing.T) {
	// "Cannot purge your own account" — same invariant as Delete.
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	handler := NewUserHandler(userService, userOrgService, nil, nil)

	admin := createTestSuperAdmin(t, db)

	r := setupTestRouterWithUser(admin.ID)
	r.DELETE("/users/:userId/purge", handler.Purge)

	w := performRequest(r, "DELETE", fmt.Sprintf("/users/%d/purge", admin.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 self-purge, got %d body=%s", w.Code, w.Body.String())
	}
}

// NOTE: the "cannot purge the last superadmin" guard exists in
// UserService.HardDelete as defense-in-depth, but is unreachable via
// the HTTP handler in practice — the cannot-purge-self check fires
// first. The guard is exercised at the service layer via
// TestUserService_Delete_CannotDeleteLastSuperAdmin (delete shares
// the same guard logic).

func TestUserHandler_Purge_EmitsAuditEvent(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userOrgService := createUserOrganizationService(db)
	auditStore := store.NewAuditStore(db)
	auditService := service.NewAuditService(auditStore)
	handler := NewUserHandler(userService, userOrgService, auditService, nil)

	admin := createTestSuperAdmin(t, db)
	target := createTestUser(t, db, "Tracked", "tracked@example.com", "password")

	r := setupTestRouterWithUser(admin.ID)
	r.DELETE("/users/:userId/purge", handler.Purge)

	w := performRequest(r, "DELETE", fmt.Sprintf("/users/%d/purge", target.ID), nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("purge: %d", w.Code)
	}

	// Drain async audit channel before asserting on rows.
	auditService.Shutdown()

	logs, _, err := auditStore.FindByAction(context.Background(), models.AuditActionUserPurged, 100, 0)
	if err != nil {
		t.Fatalf("find audit: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected exactly 1 user_purged event, got %d", len(logs))
	}
	if logs[0].ResourceID == nil || *logs[0].ResourceID != target.ID {
		t.Errorf("audit row: ResourceID = %v, want %d", logs[0].ResourceID, target.ID)
	}
}
