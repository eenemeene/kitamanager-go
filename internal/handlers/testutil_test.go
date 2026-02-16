package handlers

import (
	"bytes"
	"encoding/json"
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

// performRequestWithCookies executes an HTTP request with cookies from a previous response.
func performRequestWithCookies(r *gin.Engine, method, path string, body interface{}, cookies []*http.Cookie) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req, _ = http.NewRequest(method, path, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, path, nil)
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
	req, _ := http.NewRequest(method, path, bytes.NewBufferString(body))
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

// setupTestRouter creates a gin router with auth context middleware for testing.
func setupTestRouter() *gin.Engine {
	r := gin.New()
	// Add middleware to set user context (simulating authenticated user)
	r.Use(func(c *gin.Context) {
		c.Set(ctxkeys.UserID, uint(1))
		c.Set(ctxkeys.UserEmail, "test@example.com")
		c.Next()
	})
	return r
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
// Must be the first user created after SetupTestDB truncation so it gets ID 1,
// matching the userID set in setupTestRouter.
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
	if user.ID != 1 {
		t.Fatalf("expected superadmin to get ID 1 (must be first user created), got %d", user.ID)
	}
	return user
}

// createTestGroup creates a group for testing.
func createTestGroup(t *testing.T, db *gorm.DB, name string) *models.Group {
	t.Helper()
	return testutil.CreateTestGroup(t, db, name)
}

// createTestGroupWithOrg creates a group for testing with a specific organization.
func createTestGroupWithOrg(t *testing.T, db *gorm.DB, name string, orgID uint) *models.Group {
	t.Helper()
	return testutil.CreateTestGroupWithOrg(t, db, name, orgID)
}

// createTestGroupWithOrgAndDefault creates a group for testing with a specific organization and default flag.
func createTestGroupWithOrgAndDefault(t *testing.T, db *gorm.DB, name string, orgID uint, isDefault bool) *models.Group {
	t.Helper()
	return testutil.CreateTestGroupWithOrgAndDefault(t, db, name, orgID, isDefault)
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
	db.First(&child, childID)
	sectionID := ensureTestSection(t, db, child.OrganizationID)
	db.Create(&models.ChildContract{
		ChildID:      childID,
		BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}, SectionID: sectionID},
	})
}

// createActiveEmployeeContract creates an open-ended contract for an employee (active today).
func createActiveEmployeeContract(t *testing.T, db *gorm.DB, employeeID uint) {
	t.Helper()
	var emp models.Employee
	db.First(&emp, employeeID)
	sectionID := ensureTestSection(t, db, emp.OrganizationID)
	payPlanID := ensureTestPayPlan(t, db, emp.OrganizationID)
	db.Create(&models.EmployeeContract{
		EmployeeID:    employeeID,
		BaseContract:  models.BaseContract{Period: models.Period{From: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}, SectionID: sectionID},
		StaffCategory: "qualified",
		WeeklyHours:   40,
		PayPlanID:     payPlanID,
	})
}

// createUserService creates a user service for testing.
func createUserService(db *gorm.DB) *service.UserService {
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	userGroupStore := store.NewUserGroupStore(db)
	return service.NewUserService(userStore, groupStore, userGroupStore)
}

// createUserGroupService creates a user group service for testing.
func createUserGroupService(db *gorm.DB) *service.UserGroupService {
	userGroupStore := store.NewUserGroupStore(db)
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	return service.NewUserGroupService(userGroupStore, userStore, groupStore)
}

// createGroupService creates a group service for testing.
func createGroupService(db *gorm.DB) *service.GroupService {
	groupStore := store.NewGroupStore(db)
	return service.NewGroupService(groupStore)
}

// createOrganizationService creates an organization service for testing.
func createOrganizationService(db *gorm.DB) *service.OrganizationService {
	orgStore := store.NewOrganizationStore(db)
	groupStore := store.NewGroupStore(db)
	userStore := store.NewUserStore(db)
	return service.NewOrganizationService(orgStore, groupStore, userStore)
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
	return service.NewStatisticsService(childStore, employeeStore, orgStore, fundingStore, payPlanStore, budgetItemStore)
}

func parseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("failed to parse time %q: %v", value, err)
	}
	return parsed
}
