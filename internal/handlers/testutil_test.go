package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupTestDB creates a PostgreSQL test database using testcontainers.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.SetupTestDB(t)
}

// mustMarshal marshals body to JSON, panicking on failure (test bug).
func mustMarshal(body interface{}) []byte {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		panic(fmt.Sprintf("test bug: failed to marshal request body: %v", err))
	}
	return jsonBody
}

// mustNewRequest creates an HTTP request, panicking on failure (test bug).
func mustNewRequest(method, path string, body *bytes.Buffer) *http.Request {
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, path, body)
	} else {
		req, err = http.NewRequest(method, path, nil)
	}
	if err != nil {
		panic(fmt.Sprintf("test bug: failed to create request: %v", err))
	}
	return req
}

// performRequest executes an HTTP request against the router.
func performRequest(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		jsonBody := mustMarshal(body)
		req = mustNewRequest(method, path, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = mustNewRequest(method, path, nil)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// performRequestWithCookies executes an HTTP request with cookies from a previous response.
func performRequestWithCookies(r *gin.Engine, method, path string, body interface{}, cookies []*http.Cookie) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		jsonBody := mustMarshal(body)
		req = mustNewRequest(method, path, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = mustNewRequest(method, path, nil)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// performRequestRaw executes an HTTP request with a raw string body.
func performRequestRaw(r *gin.Engine, method, path string, body string) *httptest.ResponseRecorder {
	req := mustNewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
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

// setupTestRouterWithUser creates a gin router with the given user ID in auth context.
func setupTestRouterWithUser(userID uint) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ctxkeys.UserID, userID)
		c.Set(ctxkeys.UserEmail, "test@example.com")
		c.Next()
	})
	return r
}

// setupTestRouter creates a gin router with userID=1 in auth context.
// Use setupTestRouterWithUser when the test creates a real user whose ID matters.
func setupTestRouter() *gin.Engine {
	return setupTestRouterWithUser(1)
}

// createTestOrganization creates an organization for testing.
func createTestOrganization(t *testing.T, db *gorm.DB, name string) *models.Organization {
	t.Helper()
	return testutil.CreateTestOrganization(t, db, name)
}

// createTestUser creates a user for testing.
func createTestUser(t *testing.T, db *gorm.DB, name, email, password string) *models.User {
	t.Helper()
	return testutil.CreateTestUser(t, db, name, email, password)
}

// createTestSuperAdmin creates a superadmin user for testing.
// Callers that need the router to match should use setupTestRouterWithUser(admin.ID).
func createTestSuperAdmin(t *testing.T, db *gorm.DB) *models.User {
	t.Helper()

	user := &models.User{
		Name:         "Test Admin",
		Email:        "admin@test.com",
		Password:     "password",
		Active:       true,
		IsSuperAdmin: true,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create test superadmin: %v", err)
	}
	return user
}

// createTestUserOrganization creates a user-organization membership for testing.
func createTestUserOrganization(t *testing.T, db *gorm.DB, userID, orgID uint, role models.Role) *models.UserOrganization {
	t.Helper()
	return testutil.CreateTestUserOrganization(t, db, userID, orgID, role)
}

// ensureTestPayPlan finds or creates a pay plan for the given organization.
func ensureTestPayPlan(t *testing.T, db *gorm.DB, orgID uint) uint {
	t.Helper()
	var payPlan models.PayPlan
	result := db.Where("organization_id = ?", orgID).First(&payPlan)
	if result.Error == nil {
		return payPlan.ID
	}
	payPlan = models.PayPlan{Name: "Test Pay Plan", OrganizationID: orgID}
	if err := db.Create(&payPlan).Error; err != nil {
		t.Fatalf("failed to create test pay plan: %v", err)
	}
	return payPlan.ID
}

// ensureTestSection finds or creates a default section for the given organization.
func ensureTestSection(t *testing.T, db *gorm.DB, orgID uint) uint {
	t.Helper()
	var section models.Section
	result := db.Where("organization_id = ? AND is_default = ?", orgID, true).First(&section)
	if result.Error == nil {
		return section.ID
	}
	section = models.Section{OrganizationID: orgID, Name: "Default", IsDefault: true}
	if err := db.Create(&section).Error; err != nil {
		t.Fatalf("failed to create test section: %v", err)
	}
	return section.ID
}

// createActiveChildContract creates an open-ended contract for a child (active today).
func createActiveChildContract(t *testing.T, db *gorm.DB, childID uint) {
	t.Helper()
	var child models.Child
	if err := db.First(&child, childID).Error; err != nil {
		t.Fatalf("failed to find child %d: %v", childID, err)
	}
	sectionID := ensureTestSection(t, db, child.OrganizationID)
	if err := db.Create(&models.ChildContract{
		ChildID:      childID,
		BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}, SectionID: sectionID},
	}).Error; err != nil {
		t.Fatalf("failed to create child contract: %v", err)
	}
}

// createActiveEmployeeContract creates an open-ended contract for an employee (active today).
func createActiveEmployeeContract(t *testing.T, db *gorm.DB, employeeID uint) {
	t.Helper()
	var emp models.Employee
	if err := db.First(&emp, employeeID).Error; err != nil {
		t.Fatalf("failed to find employee %d: %v", employeeID, err)
	}
	sectionID := ensureTestSection(t, db, emp.OrganizationID)
	payPlanID := ensureTestPayPlan(t, db, emp.OrganizationID)
	if err := db.Create(&models.EmployeeContract{
		EmployeeID:    employeeID,
		BaseContract:  models.BaseContract{Period: models.Period{From: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}, SectionID: sectionID},
		StaffCategory: "qualified",
		WeeklyHours:   40,
		PayPlanID:     payPlanID,
	}).Error; err != nil {
		t.Fatalf("failed to create employee contract: %v", err)
	}
}

// createUserService creates a user service for testing.
func createUserService(db *gorm.DB) *service.UserService {
	userStore := store.NewUserStore(db)
	userOrgStore := store.NewUserOrganizationStore(db)
	return service.NewUserService(userStore, userOrgStore)
}

// createUserOrganizationService creates a user organization service for testing.
func createUserOrganizationService(db *gorm.DB) *service.UserOrganizationService {
	userOrgStore := store.NewUserOrganizationStore(db)
	userStore := store.NewUserStore(db)
	transactor := store.NewTransactor(db)
	return service.NewUserOrganizationService(userOrgStore, userStore, transactor)
}

// createOrganizationService creates an organization service for testing.
func createOrganizationService(db *gorm.DB) *service.OrganizationService {
	orgStore := store.NewOrganizationStore(db)
	userStore := store.NewUserStore(db)
	return service.NewOrganizationService(orgStore, userStore)
}

// createEmployeeService creates an employee service for testing.
func createEmployeeService(db *gorm.DB) *service.EmployeeService {
	employeeStore := store.NewEmployeeStore(db)
	payPlanStore := store.NewPayPlanStore(db)
	sectionStore := store.NewSectionStore(db)
	transactor := store.NewTransactor(db)
	return service.NewEmployeeService(employeeStore, payPlanStore, sectionStore, transactor)
}

// createChildService creates a child service for testing.
func createChildService(db *gorm.DB) *service.ChildService {
	childStore := store.NewChildStore(db)
	orgStore := store.NewOrganizationStore(db)
	fundingStore := store.NewGovernmentFundingStore(db)
	sectionStore := store.NewSectionStore(db)
	transactor := store.NewTransactor(db)
	return service.NewChildService(childStore, orgStore, fundingStore, sectionStore, transactor)
}

// createAuditService creates an audit service for testing.
func createAuditService(db *gorm.DB) *service.AuditService {
	auditStore := store.NewAuditStore(db)
	return service.NewAuditService(auditStore)
}

// createAuthService creates an auth service for testing.
func createAuthService(db *gorm.DB) *service.AuthService {
	userStore := store.NewUserStore(db)
	tokenStore := store.NewTokenStore(db)
	auditService := createAuditService(db)
	return service.NewAuthService(userStore, tokenStore, "test-jwt-secret", auditService)
}

// createAuthHandler creates an auth handler for testing.
func createAuthHandler(db *gorm.DB) *AuthHandler {
	return NewAuthHandler(createAuthService(db), false)
}

// createAttendanceService creates a child attendance service for testing.
func createAttendanceService(db *gorm.DB) *service.ChildAttendanceService {
	attendanceStore := store.NewChildAttendanceStore(db)
	childStore := store.NewChildStore(db)
	return service.NewChildAttendanceService(attendanceStore, childStore)
}

// createTestPayPlan creates a pay plan for testing.
func createTestPayPlan(t *testing.T, db *gorm.DB, name string, orgID uint) *models.PayPlan {
	t.Helper()
	return testutil.CreateTestPayPlan(t, db, name, orgID)
}

// createStatisticsService creates a statistics service for testing.
func createStatisticsService(db *gorm.DB) *service.StatisticsService {
	childStore := store.NewChildStore(db)
	employeeStore := store.NewEmployeeStore(db)
	orgStore := store.NewOrganizationStore(db)
	fundingStore := store.NewGovernmentFundingStore(db)
	payPlanStore := store.NewPayPlanStore(db)
	budgetItemStore := store.NewBudgetItemStore(db)
	sectionStore := store.NewSectionStore(db)
	billStore := store.NewGovernmentFundingBillPeriodStore(db)
	return service.NewStatisticsService(childStore, employeeStore, orgStore, fundingStore, payPlanStore, budgetItemStore, sectionStore, billStore)
}

func parseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("failed to parse time %q: %v", value, err)
	}
	return parsed
}
