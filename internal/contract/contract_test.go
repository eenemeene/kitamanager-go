//go:build contract

package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
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
	testDB     *gorm.DB
	testRouter *gin.Engine
	openAPIDoc *openapi3.T
	apiRouter  routers.Router
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	// Load OpenAPI spec
	var err error
	openAPIDoc, err = openapi3.NewLoader().LoadFromFile("../../docs/swagger.json")
	if err != nil {
		fmt.Printf("Failed to load OpenAPI spec: %v\n", err)
		os.Exit(1)
	}

	// Create router for spec validation
	apiRouter, err = gorillamux.NewRouter(openAPIDoc)
	if err != nil {
		fmt.Printf("Failed to create OpenAPI router: %v\n", err)
		os.Exit(1)
	}

	// Get database connection
	dbType := getEnv("TEST_DB_TYPE", "sqlite")

	if dbType == "postgres" {
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
		// SQLite in-memory database for fast local testing
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

	code := m.Run()

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
	userGroupStore := store.NewUserGroupStore(testDB)

	// Setup services
	orgService := service.NewOrganizationService(orgStore, groupStore, userStore)
	userService := service.NewUserService(userStore, groupStore)
	userGroupService := service.NewUserGroupService(userGroupStore, userStore, groupStore)
	groupService := service.NewGroupService(groupStore)

	// Setup handlers (nil audit service for contract tests - audit has nil-safety)
	orgHandler := handlers.NewOrganizationHandler(orgService, nil)
	userHandler := handlers.NewUserHandler(userService, userGroupService, nil)
	groupHandler := handlers.NewGroupHandler(groupService)

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
		api.GET("/users/:uid", userHandler.Get)

		// Org-scoped groups
		orgScoped := api.Group("/organizations/:orgId")
		{
			orgScoped.GET("/groups", groupHandler.List)
			orgScoped.POST("/groups", groupHandler.Create)
			orgScoped.GET("/groups/:groupId", groupHandler.Get)
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
}

// validateResponse validates an HTTP response against the OpenAPI spec
func validateResponse(t *testing.T, req *http.Request, resp *httptest.ResponseRecorder) {
	t.Helper()

	// Find route in OpenAPI spec
	route, pathParams, err := apiRouter.FindRoute(req)
	if err != nil {
		t.Logf("Warning: Route not found in OpenAPI spec: %s %s", req.Method, req.URL.Path)
		return
	}

	// Validate request
	requestValidationInput := &openapi3filter.RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
	}

	// Validate response
	responseValidationInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: requestValidationInput,
		Status:                 resp.Code,
		Header:                 resp.Header(),
		Body:                   io.NopCloser(resp.Body),
	}

	if err := openapi3filter.ValidateResponse(req.Context(), responseValidationInput); err != nil {
		t.Errorf("Response does not match OpenAPI spec: %v", err)
	}
}

// performRequest makes a request and validates it against the OpenAPI spec
func performRequest(t *testing.T, method, path string, body interface{}) *httptest.ResponseRecorder {
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

	// Validate response against OpenAPI spec
	validateResponse(t, req, w)

	return w
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
}

// Contract Tests

func TestContract_OrganizationsList(t *testing.T) {
	cleanupBetweenTests()

	// Create test data
	testDB.Create(&models.Organization{Name: "Org 1", Active: true, State: "berlin"})
	testDB.Create(&models.Organization{Name: "Org 2", Active: true, State: "berlin"})

	resp := performRequest(t, "GET", "/api/v1/organizations", nil)

	if resp.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.Code)
	}
}

func TestContract_OrganizationCreate(t *testing.T) {
	cleanupBetweenTests()

	resp := performRequest(t, "POST", "/api/v1/organizations", map[string]interface{}{
		"name":   "New Organization",
		"active": true,
		"state":  "berlin",
	})

	if resp.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestContract_OrganizationGet(t *testing.T) {
	cleanupBetweenTests()

	org := &models.Organization{Name: "Test Org", Active: true, State: "berlin"}
	testDB.Create(org)

	resp := performRequest(t, "GET", fmt.Sprintf("/api/v1/organizations/%d", org.ID), nil)

	if resp.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.Code)
	}
}

func TestContract_OrganizationUpdate(t *testing.T) {
	cleanupBetweenTests()

	org := &models.Organization{Name: "Test Org", Active: true, State: "berlin"}
	testDB.Create(org)

	resp := performRequest(t, "PUT", fmt.Sprintf("/api/v1/organizations/%d", org.ID), map[string]interface{}{
		"name": "Updated Org",
	})

	if resp.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestContract_OrganizationDelete(t *testing.T) {
	cleanupBetweenTests()

	org := &models.Organization{Name: "Test Org", Active: true, State: "berlin"}
	testDB.Create(org)

	resp := performRequest(t, "DELETE", fmt.Sprintf("/api/v1/organizations/%d", org.ID), nil)

	if resp.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", resp.Code)
	}
}

func TestContract_OrganizationNotFound(t *testing.T) {
	cleanupBetweenTests()

	resp := performRequest(t, "GET", "/api/v1/organizations/99999", nil)

	if resp.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.Code)
	}
}

func TestContract_UsersList(t *testing.T) {
	cleanupBetweenTests()

	resp := performRequest(t, "GET", "/api/v1/users", nil)

	if resp.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.Code)
	}
}

func TestContract_UserCreate(t *testing.T) {
	cleanupBetweenTests()

	resp := performRequest(t, "POST", "/api/v1/users", map[string]interface{}{
		"name":     "Test User",
		"email":    "test@example.com",
		"password": "password123",
		"active":   true,
	})

	if resp.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestContract_GroupsList(t *testing.T) {
	cleanupBetweenTests()

	// Create organization first (groups are org-scoped)
	orgResp := performRequest(t, "POST", "/api/v1/organizations", map[string]interface{}{
		"name":   "Groups List Test Org",
		"active": true,
		"state":  "berlin",
	})
	if orgResp.Code != http.StatusCreated {
		t.Fatalf("failed to create organization: %d: %s", orgResp.Code, orgResp.Body.String())
	}
	var org models.Organization
	if err := json.Unmarshal(orgResp.Body.Bytes(), &org); err != nil {
		t.Fatalf("failed to parse organization response: %v", err)
	}

	resp := performRequest(t, "GET", fmt.Sprintf("/api/v1/organizations/%d/groups", org.ID), nil)

	if resp.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.Code)
	}
}

func TestContract_GroupCreate(t *testing.T) {
	cleanupBetweenTests()

	// Create organization first
	orgResp := performRequest(t, "POST", "/api/v1/organizations", map[string]interface{}{
		"name":   "Group Test Org",
		"active": true,
		"state":  "berlin",
	})
	if orgResp.Code != http.StatusCreated {
		t.Fatalf("failed to create organization: %d: %s", orgResp.Code, orgResp.Body.String())
	}
	var org models.Organization
	if err := json.Unmarshal(orgResp.Body.Bytes(), &org); err != nil {
		t.Fatalf("failed to parse organization response: %v", err)
	}

	resp := performRequest(t, "POST", fmt.Sprintf("/api/v1/organizations/%d/groups", org.ID), map[string]interface{}{
		"name":   "Test Group",
		"active": true,
	})

	if resp.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", resp.Code, resp.Body.String())
	}
}
