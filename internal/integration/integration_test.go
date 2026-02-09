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
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	testDB      *gorm.DB
	testRouter  *gin.Engine
	usingSQLite bool
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	var err error
	dbType := getEnv("TEST_DB_TYPE", "sqlite")
	usingSQLite = dbType != "postgres"

	if dbType == "postgres" {
		// PostgreSQL connection for CI/production-like testing
		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			getEnv("TEST_DB_HOST", "localhost"),
			getEnv("TEST_DB_PORT", "5432"),
			getEnv("TEST_DB_USER", "kitamanager"),
			getEnv("TEST_DB_PASSWORD", "kitamanager"),
			getEnv("TEST_DB_NAME", "kitamanager_test"),
		)
		testDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
	} else {
		// SQLite in-memory database for fast local testing (pre-commit hooks)
		testDB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
	}

	if err != nil {
		fmt.Printf("Failed to connect to test database: %v\n", err)
		os.Exit(1)
	}

	// Run migrations
	if err := testDB.AutoMigrate(
		&models.GovernmentFunding{},
		&models.GovernmentFundingPeriod{},
		&models.GovernmentFundingProperty{},
		&models.Organization{},
		&models.User{},
		&models.Group{},
		&models.Section{},
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

	// Add middleware to set user context (simulating authenticated user)
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Set("userEmail", "admin@test.com")
		c.Next()
	})

	// Setup stores
	orgStore := store.NewOrganizationStore(testDB)
	userStore := store.NewUserStore(testDB)
	groupStore := store.NewGroupStore(testDB)
	employeeStore := store.NewEmployeeStore(testDB)
	childStore := store.NewChildStore(testDB)
	userGroupStore := store.NewUserGroupStore(testDB)
	fundingStore := store.NewGovernmentFundingStore(testDB)

	// Setup services
	orgService := service.NewOrganizationService(orgStore, groupStore, userStore)
	userService := service.NewUserService(userStore, groupStore)
	userGroupService := service.NewUserGroupService(userGroupStore, userStore, groupStore)
	groupService := service.NewGroupService(groupStore)
	employeeService := service.NewEmployeeService(employeeStore)
	childService := service.NewChildService(childStore, orgStore, fundingStore)

	// Setup handlers (passing nil for auditService in tests)
	orgHandler := handlers.NewOrganizationHandler(orgService, nil)
	userHandler := handlers.NewUserHandler(userService, userGroupService, nil)
	groupHandler := handlers.NewGroupHandler(groupService)
	employeeHandler := handlers.NewEmployeeHandler(employeeService, nil)
	childHandler := handlers.NewChildHandler(childService, nil)

	// Routes - matching the actual API structure
	api := r.Group("/api/v1")
	{
		// Organizations
		api.GET("/organizations", orgHandler.List)
		api.POST("/organizations", orgHandler.Create)
		api.GET("/organizations/:orgId", orgHandler.Get)
		api.PUT("/organizations/:orgId", orgHandler.Update)
		api.DELETE("/organizations/:orgId", orgHandler.Delete)

		// Global user routes
		api.GET("/users", userHandler.List)
		api.POST("/users", userHandler.Create)
		api.GET("/users/:userId", userHandler.Get)

		// Org-scoped routes
		orgScoped := api.Group("/organizations/:orgId")
		{
			// Groups
			orgScoped.GET("/groups", groupHandler.List)
			orgScoped.POST("/groups", groupHandler.Create)
			orgScoped.GET("/groups/:groupId", groupHandler.Get)

			// Employees
			orgScoped.GET("/employees", employeeHandler.List)
			orgScoped.POST("/employees", employeeHandler.Create)
			orgScoped.GET("/employees/:id", employeeHandler.Get)

			// Children
			orgScoped.GET("/children", childHandler.List)
			orgScoped.POST("/children", childHandler.Create)
			orgScoped.GET("/children/:id", childHandler.Get)
		}
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
	testDB.Exec("DELETE FROM groups")
	testDB.Exec("DELETE FROM users")
	testDB.Exec("DELETE FROM organizations")
	testDB.Exec("DELETE FROM government_funding_properties")
	testDB.Exec("DELETE FROM government_funding_periods")
	testDB.Exec("DELETE FROM government_fundings")
}

func cleanupBetweenTests() {
	cleanupDatabase()
	// Create superadmin user with ID 1 to match the userID set in middleware
	user := &models.User{
		Name:         "Test Admin",
		Email:        "admin@test.com",
		Password:     "password",
		Active:       true,
		IsSuperAdmin: true,
	}
	user.ID = 1
	testDB.Create(user)
	// Reset the sequence so the next auto-generated ID is 2, not 1
	testDB.Exec("SELECT setval(pg_get_serial_sequence('users', 'id'), (SELECT MAX(id) FROM users))")
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

// PaginatedResponse wraps paginated API responses
type PaginatedResponse[T any] struct {
	Data []T `json:"data"`
	Meta struct {
		Total   int `json:"total"`
		Page    int `json:"page"`
		PerPage int `json:"per_page"`
	} `json:"meta"`
}

// Integration Tests

func TestOrganizationCRUD(t *testing.T) {
	cleanupBetweenTests()

	// Create
	createResp := performRequest("POST", "/api/v1/organizations", map[string]interface{}{
		"name":   "Test Organization",
		"active": true,
		"state":  "berlin",
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

	var orgsResp PaginatedResponse[models.Organization]
	parseResponse(t, listResp, &orgsResp)
	if len(orgsResp.Data) != 1 {
		t.Errorf("expected 1 organization, got %d", len(orgsResp.Data))
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
		"state":  "berlin",
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
		"state":  "berlin",
	})
	var org models.Organization
	parseResponse(t, orgResp, &org)

	// Create employee (using org-scoped route)
	empResp := performRequest("POST", fmt.Sprintf("/api/v1/organizations/%d/employees", org.ID), map[string]interface{}{
		"first_name": "John",
		"last_name":  "Doe",
		"gender":     "male",
		"birthdate":  "1990-01-15T00:00:00Z",
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
		"state":  "berlin",
	})
	var org models.Organization
	parseResponse(t, orgResp, &org)

	// Create child (using org-scoped route)
	childResp := performRequest("POST", fmt.Sprintf("/api/v1/organizations/%d/children", org.ID), map[string]interface{}{
		"first_name": "Emma",
		"last_name":  "Smith",
		"gender":     "female",
		"birthdate":  "2020-06-15T00:00:00Z",
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
		"state":  "berlin",
	})
	if orgResp.Code != http.StatusCreated {
		t.Fatalf("failed to create organization: %d: %s", orgResp.Code, orgResp.Body.String())
	}
	var org models.Organization
	parseResponse(t, orgResp, &org)

	// Create group (using org-scoped route)
	groupResp := performRequest("POST", fmt.Sprintf("/api/v1/organizations/%d/groups", org.ID), map[string]interface{}{
		"name":   "Test Group",
		"active": true,
	})
	if groupResp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", groupResp.Code, groupResp.Body.String())
	}

	var group models.Group
	parseResponse(t, groupResp, &group)
	if group.Name != "Test Group" {
		t.Errorf("expected name 'Test Group', got '%s'", group.Name)
	}

	// List groups (org-scoped)
	// Should have 2 groups: default "Members" group + the one we created
	listResp := performRequest("GET", fmt.Sprintf("/api/v1/organizations/%d/groups", org.ID), nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", listResp.Code)
	}

	var groupsResp PaginatedResponse[models.Group]
	parseResponse(t, listResp, &groupsResp)
	if len(groupsResp.Data) != 2 {
		t.Errorf("expected 2 groups (default + created), got %d", len(groupsResp.Data))
	}
}

func TestConcurrentOrganizationCreation(t *testing.T) {
	if usingSQLite {
		t.Skip("skipping concurrent test with SQLite (doesn't support concurrent writes)")
	}

	cleanupBetweenTests()

	// Test concurrent creation to ensure no race conditions
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(idx int) {
			resp := performRequest("POST", "/api/v1/organizations", map[string]interface{}{
				"name":   fmt.Sprintf("Concurrent Org %d", idx),
				"active": true,
				"state":  "berlin",
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
	var orgsResp PaginatedResponse[models.Organization]
	parseResponse(t, listResp, &orgsResp)
	if len(orgsResp.Data) != 5 {
		t.Errorf("expected 5 organizations, got %d", len(orgsResp.Data))
	}
}
