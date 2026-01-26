//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/handlers"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	testDB     *gorm.DB
	testRouter *gin.Engine
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	// Get database connection from environment
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("TEST_DB_HOST", "localhost"),
		getEnv("TEST_DB_PORT", "5432"),
		getEnv("TEST_DB_USER", "kitamanager"),
		getEnv("TEST_DB_PASSWORD", "kitamanager"),
		getEnv("TEST_DB_NAME", "kitamanager_test"),
	)

	var err error
	testDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fmt.Printf("Failed to connect to test database: %v\n", err)
		os.Exit(1)
	}

	// Run migrations
	if err := testDB.AutoMigrate(
		&models.Organization{},
		&models.User{},
		&models.Group{},
		&models.UserGroup{},
		&models.Employee{},
		&models.EmployeeContract{},
		&models.Child{},
		&models.ChildContract{},
	); err != nil {
		fmt.Printf("Failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	// Setup router
	testRouter = setupRouter()

	// Run tests
	code := m.Run()

	// Cleanup
	cleanupDatabase()

	os.Exit(code)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupRouter() *gin.Engine {
	r := gin.New()

	// Setup stores
	orgStore := store.NewOrganizationStore(testDB)
	userStore := store.NewUserStore(testDB)
	groupStore := store.NewGroupStore(testDB)
	employeeStore := store.NewEmployeeStore(testDB)
	childStore := store.NewChildStore(testDB)
	userGroupStore := store.NewUserGroupStore(testDB)

	// Setup services
	orgService := service.NewOrganizationService(orgStore, groupStore)
	userService := service.NewUserService(userStore, groupStore)
	userGroupService := service.NewUserGroupService(userGroupStore, userStore, groupStore)
	groupService := service.NewGroupService(groupStore)
	employeeService := service.NewEmployeeService(employeeStore)
	childService := service.NewChildService(childStore, groupStore)

	// Setup handlers
	orgHandler := handlers.NewOrganizationHandler(orgService)
	userHandler := handlers.NewUserHandler(userService, userGroupService)
	groupHandler := handlers.NewGroupHandler(groupService)
	employeeHandler := handlers.NewEmployeeHandler(employeeService)
	childHandler := handlers.NewChildHandler(childService)

	// Routes
	api := r.Group("/api/v1")
	{
		api.GET("/organizations", orgHandler.List)
		api.POST("/organizations", orgHandler.Create)
		api.GET("/organizations/:id", orgHandler.Get)
		api.PUT("/organizations/:id", orgHandler.Update)
		api.DELETE("/organizations/:id", orgHandler.Delete)

		api.GET("/users", userHandler.List)
		api.POST("/users", userHandler.Create)
		api.GET("/users/:id", userHandler.Get)

		api.GET("/groups", groupHandler.List)
		api.POST("/groups", groupHandler.Create)
		api.GET("/groups/:id", groupHandler.Get)

		api.GET("/employees", employeeHandler.List)
		api.POST("/employees", employeeHandler.Create)
		api.GET("/employees/:id", employeeHandler.Get)

		api.GET("/children", childHandler.List)
		api.POST("/children", childHandler.Create)
		api.GET("/children/:id", childHandler.Get)
	}

	return r
}

func cleanupDatabase() {
	// Clean up in reverse order of dependencies
	testDB.Exec("DELETE FROM child_contracts")
	testDB.Exec("DELETE FROM employee_contracts")
	testDB.Exec("DELETE FROM children")
	testDB.Exec("DELETE FROM employees")
	testDB.Exec("DELETE FROM user_groups")
	testDB.Exec("DELETE FROM user_organizations")
	testDB.Exec("DELETE FROM group_organizations")
	testDB.Exec("DELETE FROM groups")
	testDB.Exec("DELETE FROM users")
	testDB.Exec("DELETE FROM organizations")
}

func cleanupBetweenTests() {
	cleanupDatabase()
}

// Helper functions
func performRequest(method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	return w
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
}

// Integration Tests

func TestOrganizationCRUD(t *testing.T) {
	cleanupBetweenTests()

	// Create
	createResp := performRequest("POST", "/api/v1/organizations", map[string]interface{}{
		"name":   "Test Organization",
		"active": true,
	})
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", createResp.Code, createResp.Body.String())
	}

	var created models.Organization
	parseResponse(t, createResp, &created)
	if created.Name != "Test Organization" {
		t.Errorf("expected name 'Test Organization', got '%s'", created.Name)
	}

	// Read
	getResp := performRequest("GET", fmt.Sprintf("/api/v1/organizations/%d", created.ID), nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", getResp.Code)
	}

	var fetched models.Organization
	parseResponse(t, getResp, &fetched)
	if fetched.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, fetched.ID)
	}

	// Update
	updateResp := performRequest("PUT", fmt.Sprintf("/api/v1/organizations/%d", created.ID), map[string]interface{}{
		"name": "Updated Organization",
	})
	if updateResp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", updateResp.Code, updateResp.Body.String())
	}

	var updated models.Organization
	parseResponse(t, updateResp, &updated)
	if updated.Name != "Updated Organization" {
		t.Errorf("expected name 'Updated Organization', got '%s'", updated.Name)
	}

	// List
	listResp := performRequest("GET", "/api/v1/organizations", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", listResp.Code)
	}

	var orgs []models.Organization
	parseResponse(t, listResp, &orgs)
	if len(orgs) != 1 {
		t.Errorf("expected 1 organization, got %d", len(orgs))
	}

	// Delete
	deleteResp := performRequest("DELETE", fmt.Sprintf("/api/v1/organizations/%d", created.ID), nil)
	if deleteResp.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", deleteResp.Code)
	}

	// Verify deleted
	getDeletedResp := performRequest("GET", fmt.Sprintf("/api/v1/organizations/%d", created.ID), nil)
	if getDeletedResp.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", getDeletedResp.Code)
	}
}

func TestUserCreationWithOrganization(t *testing.T) {
	cleanupBetweenTests()

	// Create organization first
	orgResp := performRequest("POST", "/api/v1/organizations", map[string]interface{}{
		"name":   "User Test Org",
		"active": true,
	})
	if orgResp.Code != http.StatusCreated {
		t.Fatalf("failed to create org: %s", orgResp.Body.String())
	}

	// Create user
	userResp := performRequest("POST", "/api/v1/users", map[string]interface{}{
		"name":     "Test User",
		"email":    "test@example.com",
		"password": "password123",
		"active":   true,
	})
	if userResp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", userResp.Code, userResp.Body.String())
	}

	var user models.UserResponse
	parseResponse(t, userResp, &user)
	if user.Name != "Test User" {
		t.Errorf("expected name 'Test User', got '%s'", user.Name)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", user.Email)
	}
}

func TestEmployeeWithContracts(t *testing.T) {
	cleanupBetweenTests()

	// Create organization
	orgResp := performRequest("POST", "/api/v1/organizations", map[string]interface{}{
		"name":   "Employee Test Org",
		"active": true,
	})
	var org models.Organization
	parseResponse(t, orgResp, &org)

	// Create employee
	empResp := performRequest("POST", "/api/v1/employees", map[string]interface{}{
		"organization_id": org.ID,
		"first_name":      "John",
		"last_name":       "Doe",
		"birthdate":       "1990-01-15T00:00:00Z",
	})
	if empResp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", empResp.Code, empResp.Body.String())
	}

	var employee models.Employee
	parseResponse(t, empResp, &employee)
	if employee.FirstName != "John" {
		t.Errorf("expected first name 'John', got '%s'", employee.FirstName)
	}

	// Verify organization association
	if employee.OrganizationID != org.ID {
		t.Errorf("expected organization ID %d, got %d", org.ID, employee.OrganizationID)
	}
}

func TestChildWithContracts(t *testing.T) {
	cleanupBetweenTests()

	// Create organization
	orgResp := performRequest("POST", "/api/v1/organizations", map[string]interface{}{
		"name":   "Child Test Org",
		"active": true,
	})
	var org models.Organization
	parseResponse(t, orgResp, &org)

	// Create child
	childResp := performRequest("POST", "/api/v1/children", map[string]interface{}{
		"organization_id": org.ID,
		"first_name":      "Emma",
		"last_name":       "Smith",
		"birthdate":       "2020-06-15T00:00:00Z",
	})
	if childResp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", childResp.Code, childResp.Body.String())
	}

	var child models.Child
	parseResponse(t, childResp, &child)
	if child.FirstName != "Emma" {
		t.Errorf("expected first name 'Emma', got '%s'", child.FirstName)
	}
}

func TestGroupOperations(t *testing.T) {
	cleanupBetweenTests()

	// Create organization first
	orgResp := performRequest("POST", "/api/v1/organizations", map[string]interface{}{
		"name":   "Group Test Org",
		"active": true,
	})
	if orgResp.Code != http.StatusCreated {
		t.Fatalf("failed to create organization: %d: %s", orgResp.Code, orgResp.Body.String())
	}
	var org models.Organization
	parseResponse(t, orgResp, &org)

	// Create group
	groupResp := performRequest("POST", "/api/v1/groups", map[string]interface{}{
		"name":            "Test Group",
		"organization_id": org.ID,
		"active":          true,
	})
	if groupResp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", groupResp.Code, groupResp.Body.String())
	}

	var group models.Group
	parseResponse(t, groupResp, &group)
	if group.Name != "Test Group" {
		t.Errorf("expected name 'Test Group', got '%s'", group.Name)
	}

	// List groups
	listResp := performRequest("GET", "/api/v1/groups", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", listResp.Code)
	}

	var groups []models.Group
	parseResponse(t, listResp, &groups)
	if len(groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(groups))
	}
}

func TestConcurrentOrganizationCreation(t *testing.T) {
	cleanupBetweenTests()

	// Test concurrent creation to ensure no race conditions
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(idx int) {
			resp := performRequest("POST", "/api/v1/organizations", map[string]interface{}{
				"name":   fmt.Sprintf("Concurrent Org %d", idx),
				"active": true,
			})
			if resp.Code != http.StatusCreated {
				t.Errorf("concurrent creation %d failed: %d", idx, resp.Code)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify all were created
	listResp := performRequest("GET", "/api/v1/organizations", nil)
	var orgs []models.Organization
	parseResponse(t, listResp, &orgs)
	if len(orgs) != 5 {
		t.Errorf("expected 5 organizations, got %d", len(orgs))
	}
}
