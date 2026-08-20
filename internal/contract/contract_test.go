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
	"strings"
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
	sectionStore := store.NewSectionStore(testDB)
	childStore := store.NewChildStore(testDB)
	employeeStore := store.NewEmployeeStore(testDB)
	payPlanStore := store.NewPayPlanStore(testDB)
	fundingStore := store.NewGovernmentFundingStore(testDB)
	attendanceStore := store.NewChildAttendanceStore(testDB)

	// Setup services
	transactor := store.NewTransactor(testDB)
	orgService := service.NewOrganizationService(orgStore, userStore)
	userService := service.NewUserService(userStore, userOrgStore)
	userOrgService := service.NewUserOrganizationService(userOrgStore, userStore, transactor)
	sectionService := service.NewSectionService(sectionStore, transactor)
	childService := service.NewChildService(childStore, orgStore, fundingStore, sectionStore, transactor)
	employeeService := service.NewEmployeeService(employeeStore, payPlanStore, sectionStore, transactor)
	attendanceService := service.NewChildAttendanceService(attendanceStore, childStore)

	// Setup audit service
	auditStore := store.NewAuditStore(testDB)
	auditService := service.NewAuditService(auditStore)

	// Setup handlers
	orgHandler := handlers.NewOrganizationHandler(orgService, auditService)
	userHandler := handlers.NewUserHandler(userService, userOrgService, auditService, nil)
	sectionHandler := handlers.NewSectionHandler(sectionService, auditService)
	childHandler := handlers.NewChildHandler(childService, auditService)
	employeeHandler := handlers.NewEmployeeHandler(employeeService, auditService)
	attendanceHandler := handlers.NewChildAttendanceHandler(attendanceService, auditService)

	// Routes. Paths and parameter names must match internal/routes/routes.go
	// exactly — TestContract_RegisteredRoutesAreDocumented pins that, so a
	// divergence here fails rather than quietly validating a route the real
	// API does not serve.
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

		// Sections
		api.GET("/organizations/:orgId/sections", sectionHandler.List)
		api.POST("/organizations/:orgId/sections", sectionHandler.Create)
		api.GET("/organizations/:orgId/sections/:sectionId", sectionHandler.Get)

		// Children and their contracts
		api.GET("/organizations/:orgId/children", childHandler.List)
		api.POST("/organizations/:orgId/children", childHandler.Create)
		api.GET("/organizations/:orgId/children/:childId", childHandler.Get)
		api.PUT("/organizations/:orgId/children/:childId", childHandler.Update)
		api.GET("/organizations/:orgId/children/:childId/contracts", childHandler.ListContracts)
		api.POST("/organizations/:orgId/children/:childId/contracts", childHandler.CreateContract)

		// Employees and their contracts
		api.GET("/organizations/:orgId/employees", employeeHandler.List)
		api.POST("/organizations/:orgId/employees", employeeHandler.Create)
		api.GET("/organizations/:orgId/employees/:employeeId", employeeHandler.Get)
		api.PUT("/organizations/:orgId/employees/:employeeId", employeeHandler.Update)

		// Attendance
		api.POST("/organizations/:orgId/children/:childId/attendance", attendanceHandler.Create)
		api.GET("/organizations/:orgId/children/:childId/attendance", attendanceHandler.ListByChild)
		api.PUT("/organizations/:orgId/children/:childId/attendance/:attendanceId", attendanceHandler.Update)
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
func performRequest(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
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

func parseResponse(t *testing.T, w *httptest.ResponseRecorder, v any) {
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

	resp := performRequest(t, "POST", "/api/v1/organizations", map[string]any{
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

	resp := performRequest(t, "PUT", fmt.Sprintf("/api/v1/organizations/%d", org.ID), map[string]any{
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

	resp := performRequest(t, "POST", "/api/v1/users", map[string]any{
		"name":     "Test User",
		"email":    "test@example.com",
		"password": "password123",
		"active":   true,
	})

	if resp.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", resp.Code, resp.Body.String())
	}
}

// TestContract_RegisteredRoutesAreDocumented pins this file's router to the
// published contract.
//
// setupRouter hand-registers its routes rather than calling routes.Setup,
// because the real wiring needs auth, RBAC and CSRF that these tests
// deliberately skip. That hand-copy can drift, and a drifted route is worse
// than an untested one: it validates a shape the real API never serves. The
// `/users/:uid` route here was exactly that — routes.go says `:userId`, and
// nothing noticed because no test exercised it.
//
// The spec is verified against routes.go by `make swagger-check`, so matching
// the spec transitively matches the real routes.
func TestContract_RegisteredRoutesAreDocumented(t *testing.T) {
	documented := make(map[string]bool)
	for path, item := range openAPIDoc.Paths.Map() {
		ginPath := strings.NewReplacer("{", ":", "}", "").Replace(path)
		for method := range item.Operations() {
			documented[method+" "+ginPath] = true
		}
	}

	for _, route := range testRouter.Routes() {
		key := route.Method + " " + route.Path
		if !documented[key] {
			t.Errorf("route %s is registered here but not in the OpenAPI spec — "+
				"check it against internal/routes/routes.go", key)
		}
	}
}

// setupOrgFixture creates an organization and returns its id along with the id
// of the default section every organization is created with. Contract tests
// need both to address the nested resources.
func setupOrgFixture(t *testing.T) (orgID, sectionID uint) {
	t.Helper()

	org := &models.Organization{Name: "Kita Sonnenschein", Active: true, State: "berlin"}
	if err := testDB.Create(org).Error; err != nil {
		t.Fatalf("create organization: %v", err)
	}
	section := &models.Section{OrganizationID: org.ID, Name: "Sonnengruppe"}
	if err := testDB.Create(section).Error; err != nil {
		t.Fatalf("create section: %v", err)
	}
	return org.ID, section.ID
}

func TestContract_SectionsList(t *testing.T) {
	cleanupBetweenTests()
	orgID, _ := setupOrgFixture(t)

	resp := performRequest(t, "GET", fmt.Sprintf("/api/v1/organizations/%d/sections", orgID), nil)

	if resp.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestContract_SectionCreate(t *testing.T) {
	cleanupBetweenTests()
	orgID, _ := setupOrgFixture(t)

	resp := performRequest(t, "POST", fmt.Sprintf("/api/v1/organizations/%d/sections", orgID), map[string]any{
		"name": "Mondgruppe",
	})

	if resp.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestContract_ChildCreateAndGet(t *testing.T) {
	cleanupBetweenTests()
	orgID, _ := setupOrgFixture(t)

	created := performRequest(t, "POST", fmt.Sprintf("/api/v1/organizations/%d/children", orgID), map[string]any{
		"first_name": "Emma",
		"last_name":  "Schmidt",
		"gender":     "female",
		// A plain date string, not RFC3339: birthdate is a string on the wire
		// while its neighbour school_entry_date is a time.Time. The contract
		// says so, and this pins it.
		"birthdate": "2022-03-10",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create child: expected 201, got %d: %s", created.Code, created.Body.String())
	}

	var child models.ChildResponse
	parseResponse(t, created, &child)

	got := performRequest(t, "GET", fmt.Sprintf("/api/v1/organizations/%d/children/%d", orgID, child.ID), nil)
	if got.Code != http.StatusOK {
		t.Errorf("get child: expected 200, got %d: %s", got.Code, got.Body.String())
	}
}

func TestContract_ChildrenList(t *testing.T) {
	cleanupBetweenTests()
	orgID, _ := setupOrgFixture(t)

	resp := performRequest(t, "GET", fmt.Sprintf("/api/v1/organizations/%d/children", orgID), nil)

	if resp.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestContract_ChildContractCreateAndList(t *testing.T) {
	cleanupBetweenTests()
	orgID, sectionID := setupOrgFixture(t)

	created := performRequest(t, "POST", fmt.Sprintf("/api/v1/organizations/%d/children", orgID), map[string]any{
		"first_name": "Emma",
		"last_name":  "Schmidt",
		"gender":     "female",
		"birthdate":  "2022-03-10",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create child: expected 201, got %d: %s", created.Code, created.Body.String())
	}
	var child models.ChildResponse
	parseResponse(t, created, &child)

	base := fmt.Sprintf("/api/v1/organizations/%d/children/%d/contracts", orgID, child.ID)
	// RFC3339 here, unlike birthdate above: contract dates are time.Time.
	contract := performRequest(t, "POST", base, map[string]any{
		"from":       "2026-08-01T00:00:00Z",
		"section_id": sectionID,
		"properties": map[string]any{"care_type": "ganztag"},
	})
	if contract.Code != http.StatusCreated {
		t.Fatalf("create contract: expected 201, got %d: %s", contract.Code, contract.Body.String())
	}

	// The paginated envelope is the shape the frontend reads whole pages from.
	list := performRequest(t, "GET", base, nil)
	if list.Code != http.StatusOK {
		t.Errorf("list contracts: expected 200, got %d: %s", list.Code, list.Body.String())
	}
}

func TestContract_EmployeeCreateAndGet(t *testing.T) {
	cleanupBetweenTests()
	orgID, _ := setupOrgFixture(t)

	created := performRequest(t, "POST", fmt.Sprintf("/api/v1/organizations/%d/employees", orgID), map[string]any{
		"first_name": "Max",
		"last_name":  "Mustermann",
		"gender":     "male",
		"birthdate":  "1990-05-15",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create employee: expected 201, got %d: %s", created.Code, created.Body.String())
	}

	var employee models.EmployeeResponse
	parseResponse(t, created, &employee)

	got := performRequest(t, "GET", fmt.Sprintf("/api/v1/organizations/%d/employees/%d", orgID, employee.ID), nil)
	if got.Code != http.StatusOK {
		t.Errorf("get employee: expected 200, got %d: %s", got.Code, got.Body.String())
	}
}

func TestContract_EmployeesList(t *testing.T) {
	cleanupBetweenTests()
	orgID, _ := setupOrgFixture(t)

	resp := performRequest(t, "GET", fmt.Sprintf("/api/v1/organizations/%d/employees", orgID), nil)

	if resp.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

// TestContract_AttendanceCheckOutAndUndo walks the check-in / check-out / undo
// sequence over HTTP, which is the path that was broken: the undo sent `""` for
// check_out_time and time.Time rejected it. Both the null that clears and the
// timestamp that sets are validated against the spec here.
func TestContract_AttendanceCheckOutAndUndo(t *testing.T) {
	cleanupBetweenTests()
	orgID, _ := setupOrgFixture(t)

	created := performRequest(t, "POST", fmt.Sprintf("/api/v1/organizations/%d/children", orgID), map[string]any{
		"first_name": "Emma",
		"last_name":  "Schmidt",
		"gender":     "female",
		"birthdate":  "2022-03-10",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create child: expected 201, got %d: %s", created.Code, created.Body.String())
	}
	var child models.ChildResponse
	parseResponse(t, created, &child)

	base := fmt.Sprintf("/api/v1/organizations/%d/children/%d/attendance", orgID, child.ID)
	checkIn := performRequest(t, "POST", base, map[string]any{
		"date":          "2026-06-15",
		"status":        "present",
		"check_in_time": "2026-06-15T08:00:00Z",
	})
	if checkIn.Code != http.StatusCreated {
		t.Fatalf("check in: expected 201, got %d: %s", checkIn.Code, checkIn.Body.String())
	}
	var attendance models.ChildAttendanceResponse
	parseResponse(t, checkIn, &attendance)

	one := fmt.Sprintf("%s/%d", base, attendance.ID)
	checkOut := performRequest(t, "PUT", one, map[string]any{
		"check_out_time": "2026-06-15T16:00:00Z",
	})
	if checkOut.Code != http.StatusOK {
		t.Fatalf("check out: expected 200, got %d: %s", checkOut.Code, checkOut.Body.String())
	}
	parseResponse(t, checkOut, &attendance)
	if attendance.CheckOutTime == nil {
		t.Fatal("check out: expected check_out_time to be set")
	}

	undo := performRequest(t, "PUT", one, map[string]any{"check_out_time": nil})
	if undo.Code != http.StatusOK {
		t.Fatalf("undo check out: expected 200, got %d: %s", undo.Code, undo.Body.String())
	}
	parseResponse(t, undo, &attendance)
	if attendance.CheckOutTime != nil {
		t.Errorf("undo check out: expected check_out_time to be cleared, got %v", attendance.CheckOutTime)
	}
	if attendance.CheckInTime == nil {
		t.Error("undo check out: expected check_in_time to survive")
	}
}
