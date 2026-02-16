//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/database"
	"github.com/eenemeene/kitamanager-go/internal/handlers"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

var (
	testDB     *gorm.DB
	testRouter *gin.Engine
	testUserID uint
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("kitamanager_integration"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		fmt.Printf("Failed to start PostgreSQL container: %v\n", err)
		os.Exit(1)
	}
	defer pgContainer.Terminate(ctx) //nolint:errcheck

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Printf("Failed to get connection string: %v\n", err)
		os.Exit(1)
	}

	// Run production migrations
	if err := database.RunMigrationsWithURL(connStr); err != nil {
		fmt.Printf("Failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	testDB, err = gorm.Open(gormpostgres.Open(connStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fmt.Printf("Failed to connect to test database: %v\n", err)
		os.Exit(1)
	}

	// Setup router
	testRouter = setupRouter()

	// Run tests
	code := m.Run()

	os.Exit(code)
}

func setupRouter() *gin.Engine {
	r := gin.New()

	// Add middleware to set user context (simulating authenticated user)
	r.Use(func(c *gin.Context) {
		c.Set(ctxkeys.UserID, testUserID)
		c.Set(ctxkeys.UserEmail, "admin@test.com")
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
	userService := service.NewUserService(userStore, groupStore, userGroupStore)
	userGroupService := service.NewUserGroupService(userGroupStore, userStore, groupStore)
	groupService := service.NewGroupService(groupStore)
	payPlanStore := store.NewPayPlanStore(testDB)
	sectionStore := store.NewSectionStore(testDB)
	transactor := store.NewTransactor(testDB)
	employeeService := service.NewEmployeeService(employeeStore, payPlanStore, sectionStore, transactor)
	childService := service.NewChildService(childStore, orgStore, fundingStore, sectionStore, transactor)

	// Setup audit service
	auditStore := store.NewAuditStore(testDB)
	auditService := service.NewAuditService(auditStore)

	// Setup handlers
	orgHandler := handlers.NewOrganizationHandler(orgService, auditService)
	userHandler := handlers.NewUserHandler(userService, userGroupService, auditService, nil)
	groupHandler := handlers.NewGroupHandler(groupService, auditService)
	employeeHandler := handlers.NewEmployeeHandler(employeeService, auditService)
	childHandler := handlers.NewChildHandler(childService, auditService)

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
	// truncation order: leaf tables first, then parents (must match testutil.truncateTables)
	tables := []string{
		"revoked_tokens",
		"audit_logs",
		"budget_item_entries",
		"budget_items",
		"child_attendances",
		"pay_plan_entries",
		"pay_plan_periods",
		"pay_plans",
		"government_funding_properties",
		"government_funding_periods",
		"government_fundings",
		"child_contracts",
		"children",
		"employee_contracts",
		"employees",
		"sections",
		"user_groups",
		"groups",
		"users",
		"organizations",
	}
	for _, table := range tables {
		testDB.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table))
	}
}

func cleanupBetweenTests() {
	cleanupDatabase()
	// Create superadmin user; let the DB assign the ID via auto-increment.
	user := &models.User{
		Name:         "Test Admin",
		Email:        "admin@test.com",
		Password:     "password",
		Active:       true,
		IsSuperAdmin: true,
	}
	testDB.Create(user)
	testUserID = user.ID
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
		"name":                 "Test Organization",
		"active":               true,
		"state":                "berlin",
		"default_section_name": "Default",
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
		"name":                 "User Test Org",
		"active":               true,
		"state":                "berlin",
		"default_section_name": "Default",
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
		"name":                 "Employee Test Org",
		"active":               true,
		"state":                "berlin",
		"default_section_name": "Default",
	})
	var org models.Organization
	parseResponse(t, orgResp, &org)

	// Create employee (using org-scoped route)
	empResp := performRequest("POST", fmt.Sprintf("/api/v1/organizations/%d/employees", org.ID), map[string]interface{}{
		"first_name": "John",
		"last_name":  "Doe",
		"gender":     "male",
		"birthdate":  "1990-01-15",
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
		"name":                 "Child Test Org",
		"active":               true,
		"state":                "berlin",
		"default_section_name": "Default",
	})
	var org models.Organization
	parseResponse(t, orgResp, &org)

	// Create child (using org-scoped route)
	childResp := performRequest("POST", fmt.Sprintf("/api/v1/organizations/%d/children", org.ID), map[string]interface{}{
		"first_name": "Emma",
		"last_name":  "Smith",
		"gender":     "female",
		"birthdate":  "2020-06-15",
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
		"name":                 "Group Test Org",
		"active":               true,
		"state":                "berlin",
		"default_section_name": "Default",
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
	cleanupBetweenTests()

	// Test concurrent creation to ensure no race conditions
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(idx int) {
			resp := performRequest("POST", "/api/v1/organizations", map[string]interface{}{
				"name":                 fmt.Sprintf("Concurrent Org %d", idx),
				"active":               true,
				"state":                "berlin",
				"default_section_name": "Default",
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
