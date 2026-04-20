package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/rbac"
)

// TestCrossOrgIsolation_RequirePermission_DeniesForeignOrgAccess is the
// load-bearing security test for the multi-tenancy boundary: a user who has
// any role in Org A must NOT be able to invoke an org-scoped endpoint with
// orgId=B. The previous coverage exercised role denials within the same org
// but never asserted the org boundary itself.
//
// Every org-scoped resource that is gated by RequirePermission appears in the
// table below. If a new org-scoped resource is added, append it here so the
// boundary is enforced by default rather than by reviewer attention.
func TestCrossOrgIsolation_RequirePermission_DeniesForeignOrgAccess(t *testing.T) {
	type endpoint struct {
		name     string
		method   string
		pattern  string // gin route template
		path     string // concrete request path with orgId=2
		resource string
		action   string
	}

	endpoints := []endpoint{
		// Children
		{"children_list", "GET", "/organizations/:orgId/children", "/organizations/2/children", rbac.ResourceChildren, rbac.ActionRead},
		{"children_get", "GET", "/organizations/:orgId/children/:childId", "/organizations/2/children/1", rbac.ResourceChildren, rbac.ActionRead},
		{"children_create", "POST", "/organizations/:orgId/children", "/organizations/2/children", rbac.ResourceChildren, rbac.ActionCreate},
		{"children_update", "PUT", "/organizations/:orgId/children/:childId", "/organizations/2/children/1", rbac.ResourceChildren, rbac.ActionUpdate},
		{"children_delete", "DELETE", "/organizations/:orgId/children/:childId", "/organizations/2/children/1", rbac.ResourceChildren, rbac.ActionDelete},

		// Child contracts
		{"child_contracts_list", "GET", "/organizations/:orgId/children/:childId/contracts", "/organizations/2/children/1/contracts", rbac.ResourceChildContracts, rbac.ActionRead},
		{"child_contracts_create", "POST", "/organizations/:orgId/children/:childId/contracts", "/organizations/2/children/1/contracts", rbac.ResourceChildContracts, rbac.ActionCreate},
		{"child_contracts_delete", "DELETE", "/organizations/:orgId/children/:childId/contracts/:contractId", "/organizations/2/children/1/contracts/1", rbac.ResourceChildContracts, rbac.ActionDelete},

		// Child attendance
		{"child_attendance_list_org", "GET", "/organizations/:orgId/children/attendance", "/organizations/2/children/attendance", rbac.ResourceChildAttendance, rbac.ActionRead},
		{"child_attendance_create", "POST", "/organizations/:orgId/children/:childId/attendance", "/organizations/2/children/1/attendance", rbac.ResourceChildAttendance, rbac.ActionCreate},
		{"child_attendance_update", "PUT", "/organizations/:orgId/children/:childId/attendance/:attendanceId", "/organizations/2/children/1/attendance/1", rbac.ResourceChildAttendance, rbac.ActionUpdate},
		{"child_attendance_delete", "DELETE", "/organizations/:orgId/children/:childId/attendance/:attendanceId", "/organizations/2/children/1/attendance/1", rbac.ResourceChildAttendance, rbac.ActionDelete},

		// Employees
		{"employees_list", "GET", "/organizations/:orgId/employees", "/organizations/2/employees", rbac.ResourceEmployees, rbac.ActionRead},
		{"employees_get", "GET", "/organizations/:orgId/employees/:employeeId", "/organizations/2/employees/1", rbac.ResourceEmployees, rbac.ActionRead},
		{"employees_create", "POST", "/organizations/:orgId/employees", "/organizations/2/employees", rbac.ResourceEmployees, rbac.ActionCreate},
		{"employees_update", "PUT", "/organizations/:orgId/employees/:employeeId", "/organizations/2/employees/1", rbac.ResourceEmployees, rbac.ActionUpdate},
		{"employees_delete", "DELETE", "/organizations/:orgId/employees/:employeeId", "/organizations/2/employees/1", rbac.ResourceEmployees, rbac.ActionDelete},

		// Employee contracts
		{"employee_contracts_list", "GET", "/organizations/:orgId/employees/:employeeId/contracts", "/organizations/2/employees/1/contracts", rbac.ResourceEmployeeContracts, rbac.ActionRead},
		{"employee_contracts_create", "POST", "/organizations/:orgId/employees/:employeeId/contracts", "/organizations/2/employees/1/contracts", rbac.ResourceEmployeeContracts, rbac.ActionCreate},
		{"employee_contracts_delete", "DELETE", "/organizations/:orgId/employees/:employeeId/contracts/:contractId", "/organizations/2/employees/1/contracts/1", rbac.ResourceEmployeeContracts, rbac.ActionDelete},

		// Sections
		{"sections_list", "GET", "/organizations/:orgId/sections", "/organizations/2/sections", rbac.ResourceSections, rbac.ActionRead},
		{"sections_create", "POST", "/organizations/:orgId/sections", "/organizations/2/sections", rbac.ResourceSections, rbac.ActionCreate},
		{"sections_delete", "DELETE", "/organizations/:orgId/sections/:sectionId", "/organizations/2/sections/1", rbac.ResourceSections, rbac.ActionDelete},

		// Pay plans
		{"payplans_list", "GET", "/organizations/:orgId/pay-plans", "/organizations/2/pay-plans", rbac.ResourcePayPlans, rbac.ActionRead},
		{"payplans_create", "POST", "/organizations/:orgId/pay-plans", "/organizations/2/pay-plans", rbac.ResourcePayPlans, rbac.ActionCreate},
		{"payplans_delete", "DELETE", "/organizations/:orgId/pay-plans/:payPlanId", "/organizations/2/pay-plans/1", rbac.ResourcePayPlans, rbac.ActionDelete},

		// Budget items
		{"budget_items_list", "GET", "/organizations/:orgId/budget-items", "/organizations/2/budget-items", rbac.ResourceBudgetItems, rbac.ActionRead},
		{"budget_items_create", "POST", "/organizations/:orgId/budget-items", "/organizations/2/budget-items", rbac.ResourceBudgetItems, rbac.ActionCreate},
		{"budget_item_entries_list", "GET", "/organizations/:orgId/budget-items/:budgetItemId/entries", "/organizations/2/budget-items/1/entries", rbac.ResourceBudgetItemEntries, rbac.ActionRead},

		// Government funding bills
		{"funding_bills_list", "GET", "/organizations/:orgId/government-funding-bills", "/organizations/2/government-funding-bills", rbac.ResourceGovernmentFundingBills, rbac.ActionRead},
		{"funding_bills_upload", "POST", "/organizations/:orgId/government-funding-bills", "/organizations/2/government-funding-bills", rbac.ResourceGovernmentFundingBills, rbac.ActionCreate},

		// Statistics
		{"statistics_staffing", "GET", "/organizations/:orgId/statistics/staffing-hours", "/organizations/2/statistics/staffing-hours", rbac.ResourceStatistics, rbac.ActionRead},
		{"statistics_financials", "GET", "/organizations/:orgId/statistics/financials", "/organizations/2/statistics/financials", rbac.ResourceStatistics, rbac.ActionRead},
		{"statistics_occupancy", "GET", "/organizations/:orgId/statistics/occupancy", "/organizations/2/statistics/occupancy", rbac.ResourceStatistics, rbac.ActionRead},
	}

	roles := []models.Role{
		models.RoleAdmin,   // Highest non-superadmin org role
		models.RoleManager, // Operational role
		models.RoleStaff,   // Limited operational role
		models.RoleMember,  // Read-only role
	}

	for _, role := range roles {
		for _, ep := range endpoints {
			t.Run(string(role)+"/"+ep.name, func(t *testing.T) {
				db := setupTestDB(t)
				enforcer := setupTestEnforcer(t)
				// User has the role in org 1, but tries to reach org 2.
				assignRole(t, db, 1, role, 1)
				permissionService := setupTestPermissionService(t, db, enforcer)
				mw := NewAuthorizationMiddleware(permissionService)

				r := gin.New()
				r.Use(func(c *gin.Context) {
					c.Set(ctxkeys.UserID, uint(1))
					c.Next()
				})
				r.Handle(ep.method, ep.pattern,
					mw.RequirePermission(ep.resource, ep.action),
					func(c *gin.Context) {
						// If we reach this handler, the boundary leaked.
						c.JSON(http.StatusOK, gin.H{"leaked": true})
					})

				req, _ := http.NewRequest(ep.method, ep.path, nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				if w.Code != http.StatusForbidden {
					t.Errorf("cross-org access leaked: %s %s as %s in foreign org returned %d, want %d. body=%s",
						ep.method, ep.path, role, w.Code, http.StatusForbidden, w.Body.String())
				}
			})
		}
	}
}

// TestCrossOrgIsolation_RequirePermission_AllowsHomeOrgAccess is the positive
// counterpart: with the same setup, a user with the appropriate role on their
// HOME org must succeed. This guards against a regression where the matrix
// above starts passing because authorization is broken-by-default rather than
// because the boundary holds.
func TestCrossOrgIsolation_RequirePermission_AllowsHomeOrgAccess(t *testing.T) {
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignRole(t, db, 1, models.RoleAdmin, 1)
	permissionService := setupTestPermissionService(t, db, enforcer)
	mw := NewAuthorizationMiddleware(permissionService)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ctxkeys.UserID, uint(1))
		c.Next()
	})
	r.GET("/organizations/:orgId/employees",
		mw.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	req, _ := http.NewRequest("GET", "/organizations/1/employees", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("home-org access denied: got %d, want %d. body=%s",
			w.Code, http.StatusOK, w.Body.String())
	}
}

// TestCrossOrgIsolation_RequirePermission_MultiOrgUserCannotCrossBoundary
// covers the trickier case: a user is a legitimate member of two orgs (1 and
// 3), with admin role in both. They must still receive 403 when they aim a
// request at org 2, where they have no role at all. This rules out a regression
// where "user has any membership" gets confused with "user has membership in
// THIS org".
func TestCrossOrgIsolation_RequirePermission_MultiOrgUserCannotCrossBoundary(t *testing.T) {
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	// Pre-create orgs with unique names so assignRole's "create if missing"
	// branch does not collide on the unique-name constraint.
	for _, name := range []struct {
		id   uint
		name string
	}{{1, "Org A"}, {3, "Org C"}} {
		org := models.Organization{Name: name.name, Active: true}
		org.ID = name.id
		if err := db.Create(&org).Error; err != nil {
			t.Fatalf("failed to seed organization %s: %v", name.name, err)
		}
	}
	assignRole(t, db, 1, models.RoleAdmin, 1)
	assignRole(t, db, 1, models.RoleAdmin, 3)
	permissionService := setupTestPermissionService(t, db, enforcer)
	mw := NewAuthorizationMiddleware(permissionService)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ctxkeys.UserID, uint(1))
		c.Next()
	})
	r.GET("/organizations/:orgId/employees",
		mw.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"leaked": true}) })

	req, _ := http.NewRequest("GET", "/organizations/2/employees", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("multi-org user reached non-member org: got %d, want %d. body=%s",
			w.Code, http.StatusForbidden, w.Body.String())
	}
}
