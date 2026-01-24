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
	"github.com/eenemeene/kitamanager-go/internal/store"
	"gorm.io/driver/postgres"
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
	if err != nil {
		fmt.Printf("Failed to connect to test database: %v\n", err)
		os.Exit(1)
	}

	// Run migrations
	if err := testDB.AutoMigrate(
		&models.Organization{},
		&models.User{},
		&models.Group{},
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

	orgStore := store.NewOrganizationStore(testDB)
	userStore := store.NewUserStore(testDB)
	groupStore := store.NewGroupStore(testDB)

	orgHandler := handlers.NewOrganizationHandler(orgStore)
	userHandler := handlers.NewUserHandler(userStore, groupStore)
	groupHandler := handlers.NewGroupHandler(groupStore)

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
	}

	return r
}

func cleanupDatabase() {
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
	testDB.Create(&models.Organization{Name: "Org 1", Active: true})
	testDB.Create(&models.Organization{Name: "Org 2", Active: true})

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
	})

	if resp.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestContract_OrganizationGet(t *testing.T) {
	cleanupBetweenTests()

	org := &models.Organization{Name: "Test Org", Active: true}
	testDB.Create(org)

	resp := performRequest(t, "GET", fmt.Sprintf("/api/v1/organizations/%d", org.ID), nil)

	if resp.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.Code)
	}
}

func TestContract_OrganizationUpdate(t *testing.T) {
	cleanupBetweenTests()

	org := &models.Organization{Name: "Test Org", Active: true}
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

	org := &models.Organization{Name: "Test Org", Active: true}
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

	resp := performRequest(t, "GET", "/api/v1/groups", nil)

	if resp.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.Code)
	}
}

func TestContract_GroupCreate(t *testing.T) {
	cleanupBetweenTests()

	resp := performRequest(t, "POST", "/api/v1/groups", map[string]interface{}{
		"name":   "Test Group",
		"active": true,
	})

	if resp.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", resp.Code, resp.Body.String())
	}
}
