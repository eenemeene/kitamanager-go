package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	err = db.AutoMigrate(
		&models.Organization{},
		&models.User{},
		&models.Group{},
		&models.Employee{},
		&models.EmployeeContract{},
		&models.Child{},
		&models.ChildContract{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

// performRequest executes an HTTP request against the router.
func performRequest(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req, _ = http.NewRequest(method, path, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// parseResponse parses the JSON response body into the provided interface.
func parseResponse(t *testing.T, w *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
}

// setupTestRouter creates a gin router with auth context middleware for testing.
func setupTestRouter() *gin.Engine {
	r := gin.New()
	// Add middleware to set user context (simulating authenticated user)
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Set("userEmail", "test@example.com")
		c.Next()
	})
	return r
}

// createTestOrganization creates an organization for testing.
func createTestOrganization(t *testing.T, db *gorm.DB, name string) *models.Organization {
	t.Helper()

	org := &models.Organization{
		Name:   name,
		Active: true,
	}
	if err := db.Create(org).Error; err != nil {
		t.Fatalf("failed to create test organization: %v", err)
	}
	return org
}

// createTestUser creates a user for testing.
func createTestUser(t *testing.T, db *gorm.DB, name, email, password string) *models.User {
	t.Helper()

	user := &models.User{
		Name:     name,
		Email:    email,
		Password: password,
		Active:   true,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

// createTestGroup creates a group for testing.
// If orgID is 0, it will create a test organization first.
func createTestGroup(t *testing.T, db *gorm.DB, name string) *models.Group {
	t.Helper()

	// Create a default organization for the group
	org := createTestOrganization(t, db, name+" Org")

	group := &models.Group{
		Name:           name,
		OrganizationID: org.ID,
		Active:         true,
	}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("failed to create test group: %v", err)
	}
	return group
}

// createTestGroupWithOrg creates a group for testing with a specific organization.
func createTestGroupWithOrg(t *testing.T, db *gorm.DB, name string, orgID uint) *models.Group {
	t.Helper()

	group := &models.Group{
		Name:           name,
		OrganizationID: orgID,
		Active:         true,
	}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("failed to create test group: %v", err)
	}
	return group
}

// createUserService creates a user service for testing.
func createUserService(db *gorm.DB) *service.UserService {
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	return service.NewUserService(userStore, groupStore)
}

// createGroupService creates a group service for testing.
func createGroupService(db *gorm.DB) *service.GroupService {
	groupStore := store.NewGroupStore(db)
	return service.NewGroupService(groupStore)
}

// createOrganizationService creates an organization service for testing.
func createOrganizationService(db *gorm.DB) *service.OrganizationService {
	orgStore := store.NewOrganizationStore(db)
	return service.NewOrganizationService(orgStore)
}

// createEmployeeService creates an employee service for testing.
func createEmployeeService(db *gorm.DB) *service.EmployeeService {
	employeeStore := store.NewEmployeeStore(db)
	return service.NewEmployeeService(employeeStore)
}

// createChildService creates a child service for testing.
func createChildService(db *gorm.DB) *service.ChildService {
	childStore := store.NewChildStore(db)
	groupStore := store.NewGroupStore(db)
	return service.NewChildService(childStore, groupStore)
}
