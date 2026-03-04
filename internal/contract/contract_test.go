//go:build contract

package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
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

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("kitamanager_contract"),
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
	userOrgStore := store.NewUserOrganizationStore(testDB)

	// Setup services
	orgService := service.NewOrganizationService(orgStore, userStore)
	userService := service.NewUserService(userStore, userOrgStore)
	transactor := store.NewTransactor(testDB)
	userOrgService := service.NewUserOrganizationService(userOrgStore, userStore, transactor)

	// Setup audit service
	auditStore := store.NewAuditStore(testDB)
	auditService := service.NewAuditService(auditStore)

	// Setup handlers
	orgHandler := handlers.NewOrganizationHandler(orgService, auditService)
	userHandler := handlers.NewUserHandler(userService, userOrgService, auditService, nil)

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
		"user_organizations",
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
		"name":                 "New Organization",
		"active":               true,
		"state":                "berlin",
		"default_section_name": "Default",
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
