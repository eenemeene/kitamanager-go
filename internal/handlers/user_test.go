package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestUserHandler_List(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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

func TestUserHandler_List_IncludesGroups(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	// Create org, group, and user
	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "Test Group", org.ID)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	// Add user to group (this also makes them a member of the org)
	if err := db.Model(user).Association("Groups").Append(group); err != nil {
		t.Fatalf("failed to add user to group: %v", err)
	}

	r := setupTestRouter()
	r.GET("/users", handler.List)

	w := performRequest(r, "GET", "/users", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.UserResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Fatalf("expected 1 user, got %d", len(response.Data))
	}

	if len(response.Data[0].Groups) != 1 {
		t.Errorf("expected 1 group in response, got %d", len(response.Data[0].Groups))
	}

	if response.Data[0].Groups[0].OrganizationID != org.ID {
		t.Errorf("expected group organization_id %d, got %d", org.ID, response.Data[0].Groups[0].OrganizationID)
	}

	if response.Data[0].Groups[0].Name != "Test Group" {
		t.Errorf("expected group name 'Test Group', got '%s'", response.Data[0].Groups[0].Name)
	}
}

func TestUserHandler_ListByOrganization(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	// Create two orgs with groups
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	group1 := createTestGroupWithOrg(t, db, "Group 1", org1.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org2.ID)

	// Create users and add them to groups
	user1 := createTestUser(t, db, "User 1", "user1@example.com", "password")
	user2 := createTestUser(t, db, "User 2", "user2@example.com", "password")
	user3 := createTestUser(t, db, "User 3", "user3@example.com", "password")

	// user1 and user2 in org1, user3 in org2
	if err := db.Model(user1).Association("Groups").Append(group1); err != nil {
		t.Fatalf("failed to add user1 to group1: %v", err)
	}
	if err := db.Model(user2).Association("Groups").Append(group1); err != nil {
		t.Fatalf("failed to add user2 to group1: %v", err)
	}
	if err := db.Model(user3).Association("Groups").Append(group2); err != nil {
		t.Fatalf("failed to add user3 to group2: %v", err)
	}

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	r := setupTestRouter()
	r.GET("/users/:userId", handler.Get)

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
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	createTestUser(t, db, "Original Name", "test@example.com", "password")

	r := setupTestRouter()
	r.PUT("/users/:userId", handler.Update)

	body := models.UserUpdateRequest{
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
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	createTestUser(t, db, "To Delete", "delete@example.com", "password")

	r := setupTestRouter()
	r.DELETE("/users/:userId", handler.Delete)

	w := performRequest(r, "DELETE", "/users/1", nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Verify user was deleted
	var users []models.User
	db.Find(&users)
	if len(users) != 0 {
		t.Error("expected user to be deleted")
	}
}

func TestUserHandler_AddToGroup(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	// Create org, group, and user
	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "Test Group", org.ID)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	_ = org // org is used to create the group

	r := setupTestRouterWithUser(admin.ID)
	r.POST("/users/:userId/groups", handler.AddToGroup)

	body := models.UserGroupAddRequest{
		GroupID: group.ID,
		Role:    models.RoleMember,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/users/%d/groups", user.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	// Verify user was added to group
	var foundUser models.User
	db.Preload("Groups").First(&foundUser, user.ID)
	if len(foundUser.Groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(foundUser.Groups))
	}
}

func TestUserHandler_RemoveFromGroup(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	// Create org, group, and user
	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "Test Group", org.ID)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	_ = org // org is used to create the group

	// Add user to group
	if err := db.Model(user).Association("Groups").Append(group); err != nil {
		t.Fatalf("failed to add user to group: %v", err)
	}

	r := setupTestRouterWithUser(admin.ID)
	r.DELETE("/users/:userId/groups/:groupId", handler.RemoveFromGroup)

	w := performRequest(r, "DELETE", fmt.Sprintf("/users/%d/groups/%d", user.ID, group.ID), nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
	}

	// Verify user was removed from group
	var foundUser models.User
	db.Preload("Groups").First(&foundUser, user.ID)
	if len(foundUser.Groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(foundUser.Groups))
	}
}

func TestUserHandler_AddToOrganization(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")
	// Create a default group for the organization
	defaultGroup := createTestGroupWithOrgAndDefault(t, db, "Members", org.ID, true)

	r := setupTestRouterWithUser(admin.ID)
	r.POST("/users/:userId/organizations", handler.AddToOrganization)

	body := models.UserOrganizationAddRequest{
		OrganizationID: org.ID,
	}

	w := performRequest(r, "POST", "/users/1/organizations", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	// Verify user was added to the default group
	var foundUser models.User
	db.Preload("Groups").First(&foundUser, 1)
	if len(foundUser.Groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(foundUser.Groups))
	}
	if foundUser.Groups[0].ID != defaultGroup.ID {
		t.Errorf("expected user to be in default group %d, got %d", defaultGroup.ID, foundUser.Groups[0].ID)
	}
}

func TestUserHandler_RemoveFromOrganization(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "Test Group", org.ID)
	// Add user to the group
	if err := db.Model(user).Association("Groups").Append(group); err != nil {
		t.Fatalf("failed to add user to group: %v", err)
	}

	r := setupTestRouterWithUser(admin.ID)
	r.DELETE("/users/:userId/organizations/:orgId", handler.RemoveFromOrganization)

	w := performRequest(r, "DELETE", "/users/1/organizations/1", nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
	}

	// Verify user was removed from all groups in the organization
	var removedUser models.User
	db.Preload("Groups").First(&removedUser, 1)
	if len(removedUser.Groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(removedUser.Groups))
	}
}

// Edge case tests

func TestUserHandler_Get_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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

func TestUserHandler_AddToGroup_InvalidUserID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	r := setupTestRouter()
	r.POST("/users/:userId/groups", handler.AddToGroup)

	body := models.UserGroupAddRequest{
		GroupID: 1,
		Role:    models.RoleMember,
	}

	w := performRequest(r, "POST", "/users/invalid/groups", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_AddToGroup_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	group := createTestGroup(t, db, "Test Group")

	r := setupTestRouter()
	r.POST("/users/:userId/groups", handler.AddToGroup)

	body := models.UserGroupAddRequest{
		GroupID: group.ID,
		Role:    models.RoleMember,
	}

	w := performRequest(r, "POST", "/users/999/groups", body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUserHandler_AddToGroup_GroupNotFound(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	createTestUser(t, db, "Test User", "test@example.com", "password")

	r := setupTestRouter()
	r.POST("/users/:userId/groups", handler.AddToGroup)

	body := models.UserGroupAddRequest{
		GroupID: 999, // Non-existent group
		Role:    models.RoleMember,
	}

	w := performRequest(r, "POST", "/users/1/groups", body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestUserHandler_RemoveFromGroup_InvalidUserID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	r := setupTestRouter()
	r.DELETE("/users/:userId/groups/:groupId", handler.RemoveFromGroup)

	w := performRequest(r, "DELETE", "/users/invalid/groups/1", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_RemoveFromGroup_InvalidGroupID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	createTestUser(t, db, "Test User", "test@example.com", "password")

	r := setupTestRouter()
	r.DELETE("/users/:userId/groups/:groupId", handler.RemoveFromGroup)

	w := performRequest(r, "DELETE", "/users/1/groups/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_AddToOrganization_InvalidUserID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	r := setupTestRouter()
	r.POST("/users/:userId/organizations", handler.AddToOrganization)

	body := models.UserOrganizationAddRequest{
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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouterWithUser(admin.ID)
	r.POST("/users/:userId/organizations", handler.AddToOrganization)

	body := models.UserOrganizationAddRequest{
		OrganizationID: org.ID,
	}

	w := performRequest(r, "POST", "/users/999/organizations", body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUserHandler_RemoveFromOrganization_InvalidUserID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	createTestUser(t, db, "Test User", "test@example.com", "password")

	r := setupTestRouter()
	r.DELETE("/users/:userId/organizations/:orgId", handler.RemoveFromOrganization)

	w := performRequest(r, "DELETE", "/users/1/organizations/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Validation edge case tests

func TestUserHandler_Create_WhitespaceOnlyName(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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

// UpdateGroupRole tests

func TestUserHandler_UpdateGroupRole(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "Test Group", org.ID)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	// Add user to group first
	ug := &models.UserGroup{
		UserID:    user.ID,
		GroupID:   group.ID,
		Role:      models.RoleMember,
		CreatedBy: "test@example.com",
	}
	db.Create(ug)

	r := setupTestRouterWithUser(admin.ID)
	r.PUT("/users/:userId/groups/:groupId", handler.UpdateGroupRole)

	body := models.UserGroupRoleUpdateRequest{
		Role: models.RoleAdmin,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/groups/%d", user.ID, group.ID), body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.UserGroupResponse
	parseResponse(t, w, &result)

	if result.Role != models.RoleAdmin {
		t.Errorf("expected role %v, got %v", models.RoleAdmin, result.Role)
	}
}

func TestUserHandler_UpdateGroupRole_InvalidRole(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "Test Group", org.ID)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	// Add user to group
	ug := &models.UserGroup{
		UserID:    user.ID,
		GroupID:   group.ID,
		Role:      models.RoleMember,
		CreatedBy: "test@example.com",
	}
	db.Create(ug)

	r := setupTestRouter()
	r.PUT("/users/:userId/groups/:groupId", handler.UpdateGroupRole)

	body := map[string]interface{}{
		"role": "invalid_role",
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/groups/%d", user.ID, group.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid role, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_UpdateGroupRole_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	group := createTestGroup(t, db, "Test Group")

	r := setupTestRouter()
	r.PUT("/users/:userId/groups/:groupId", handler.UpdateGroupRole)

	body := models.UserGroupRoleUpdateRequest{
		Role: models.RoleAdmin,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/users/999/groups/%d", group.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUserHandler_UpdateGroupRole_UserNotInGroup(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "Test Group", org.ID)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	// User is NOT added to group

	r := setupTestRouter()
	r.PUT("/users/:userId/groups/:groupId", handler.UpdateGroupRole)

	body := models.UserGroupRoleUpdateRequest{
		Role: models.RoleAdmin,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/groups/%d", user.ID, group.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for user not in group, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUserHandler_UpdateGroupRole_InvalidUserID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	r := setupTestRouter()
	r.PUT("/users/:userId/groups/:groupId", handler.UpdateGroupRole)

	body := models.UserGroupRoleUpdateRequest{
		Role: models.RoleAdmin,
	}

	w := performRequest(r, "PUT", "/users/invalid/groups/1", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_UpdateGroupRole_InvalidGroupID(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	createTestUser(t, db, "Test User", "test@example.com", "password")

	r := setupTestRouter()
	r.PUT("/users/:userId/groups/:groupId", handler.UpdateGroupRole)

	body := models.UserGroupRoleUpdateRequest{
		Role: models.RoleAdmin,
	}

	w := performRequest(r, "PUT", "/users/1/groups/invalid", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// GetMemberships tests

func TestUserHandler_GetMemberships(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	org := createTestOrganization(t, db, "Test Org")
	group1 := createTestGroupWithOrg(t, db, "Group 1", org.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org.ID)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	// Add user to groups
	db.Create(&models.UserGroup{UserID: user.ID, GroupID: group1.ID, Role: models.RoleAdmin, CreatedBy: "test@example.com"})
	db.Create(&models.UserGroup{UserID: user.ID, GroupID: group2.ID, Role: models.RoleMember, CreatedBy: "test@example.com"})

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

func TestUserHandler_GetMemberships_WithEffectiveRole(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	org := createTestOrganization(t, db, "Test Org")
	group1 := createTestGroupWithOrg(t, db, "Group 1", org.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org.ID)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	// Admin in group1, member in group2 (same org)
	db.Create(&models.UserGroup{UserID: user.ID, GroupID: group1.ID, Role: models.RoleAdmin, CreatedBy: "test@example.com"})
	db.Create(&models.UserGroup{UserID: user.ID, GroupID: group2.ID, Role: models.RoleMember, CreatedBy: "test@example.com"})

	r := setupTestRouter()
	r.GET("/users/:userId/memberships", handler.GetMemberships)

	w := performRequest(r, "GET", fmt.Sprintf("/users/%d/memberships", user.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result models.UserMembershipsResponse
	parseResponse(t, w, &result)

	// Both memberships should have effective org role = admin (highest)
	for _, m := range result.Memberships {
		if m.EffectiveOrgRole != models.RoleAdmin {
			t.Errorf("expected EffectiveOrgRole = admin, got %v", m.EffectiveOrgRole)
		}
	}
}

func TestUserHandler_GetMemberships_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	// Set as superadmin first
	db.Model(&models.User{}).Where("id = ?", user.ID).Update("is_superadmin", true)

	r := setupTestRouter()
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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	r := setupTestRouter()
	r.PUT("/users/:userId/superadmin", handler.SetSuperAdmin)

	// Send empty body
	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d/superadmin", user.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for missing body, got %d", http.StatusBadRequest, w.Code)
	}
}

// Additional AddToOrganization tests

func TestUserHandler_AddToOrganization_OrgNotFound(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	r := setupTestRouterWithUser(admin.ID)
	r.POST("/users/:userId/organizations", handler.AddToOrganization)

	body := models.UserOrganizationAddRequest{
		OrganizationID: 999, // Non-existent
	}

	w := performRequest(r, "POST", fmt.Sprintf("/users/%d/organizations", user.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for non-existent org, got %d", http.StatusNotFound, w.Code)
	}
}

// Additional RemoveFromOrganization tests

func TestUserHandler_RemoveFromOrganization_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouterWithUser(admin.ID)
	r.DELETE("/users/:userId/organizations/:orgId", handler.RemoveFromOrganization)

	w := performRequest(r, "DELETE", fmt.Sprintf("/users/999/organizations/%d", org.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUserHandler_List_Search(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	userService := createUserService(db)
	userGroupService := createUserGroupService(db)
	handler := NewUserHandler(userService, userGroupService, nil, nil)

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
