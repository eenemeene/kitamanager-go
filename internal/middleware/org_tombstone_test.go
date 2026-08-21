package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/rbac"
)

// Soft-deleting an organization must close every org-scoped route.
//
// It did not. user_organizations rows survive the tombstone, so GetRoleInOrg
// kept resolving roles in a deleted org; and no org-scoped store query filters
// on the parent, so children, employees, contracts and bills stayed readable
// through /organizations/{id}/... indefinitely. OrganizationService.Delete
// documented the opposite — "their own lookup paths go through the org
// resolver, which returns NotFound once the parent is tombstoned" — and no such
// resolver existed.
//
// The gate is in RequirePermission because that is the one place every
// org-scoped route already passes through. These tests pin it there.

// orgScopedRouter wires one org-scoped route through RequirePermission and
// reports what the middleware did with the request.
func orgScopedRouter(t *testing.T, db *gorm.DB, userID uint) *gin.Engine {
	t.Helper()
	enforcer := setupTestEnforcer(t)
	permissionService := setupTestPermissionService(t, db, enforcer)
	mw := NewAuthorizationMiddleware(permissionService)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ctxkeys.UserID, userID)
		c.Next()
	})
	r.GET("/organizations/:orgId/children",
		mw.RequirePermission(rbac.ResourceChildren, rbac.ActionRead),
		func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestRequirePermission_TombstonedOrgIsNotFound_ForMember(t *testing.T) {
	db := setupTestDB(t)
	assignRole(t, db, 1, models.RoleAdmin, 1)
	r := orgScopedRouter(t, db, 1)

	// Live org: the admin gets through, so the 404 below is attributable to
	// the tombstone and not to a missing role.
	req, _ := http.NewRequest("GET", "/organizations/1/children", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("setup: expected 200 while the org is live, got %d: %s", w.Code, w.Body.String())
	}

	if err := db.Delete(&models.Organization{}, 1).Error; err != nil {
		t.Fatalf("failed to tombstone organization: %v", err)
	}

	req, _ = http.NewRequest("GET", "/organizations/1/children", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 once the org is tombstoned, got %d: %s", w.Code, w.Body.String())
	}
}

// Superadmins are subject to the same rule. The data is gone; that is not a
// question about the caller's role. The erasure path is unaffected because
// DELETE /organizations/:orgId/purge is gated by RequireSuperAdmin rather than
// by this middleware.
func TestRequirePermission_TombstonedOrgIsNotFound_ForSuperAdmin(t *testing.T) {
	db := setupTestDB(t)
	assignSuperAdmin(t, db, 1)
	createOrg(t, db, 1)
	r := orgScopedRouter(t, db, 1)

	req, _ := http.NewRequest("GET", "/organizations/1/children", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("setup: expected 200 while the org is live, got %d: %s", w.Code, w.Body.String())
	}

	if err := db.Delete(&models.Organization{}, 1).Error; err != nil {
		t.Fatalf("failed to tombstone organization: %v", err)
	}

	req, _ = http.NewRequest("GET", "/organizations/1/children", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for a superadmin once the org is tombstoned, got %d: %s", w.Code, w.Body.String())
	}
}

// An organization id that never existed is the same answer as one that was
// deleted — both mean "no such organization".
func TestRequirePermission_UnknownOrgIsNotFound(t *testing.T) {
	db := setupTestDB(t)
	assignSuperAdmin(t, db, 1)
	r := orgScopedRouter(t, db, 1)

	req, _ := http.NewRequest("GET", "/organizations/424242/children", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for an organization that does not exist, got %d: %s", w.Code, w.Body.String())
	}
}

// The liveness check must not become an organization-existence oracle. A
// caller with no role gets 403 whether or not the organization exists, so
// nothing can be learned by comparing the two responses. Only a caller who
// already holds a role — and therefore already knows the org existed — ever
// sees the 404.
func TestRequirePermission_DoesNotLeakOrgExistenceToOutsiders(t *testing.T) {
	db := setupTestDB(t)
	// User 1 is an admin in org 1 and has no relationship with any other org.
	assignRole(t, db, 1, models.RoleAdmin, 1)
	createOrg(t, db, 2)
	r := orgScopedRouter(t, db, 1)

	// Org 2 exists but is closed to this caller.
	req, _ := http.NewRequest("GET", "/organizations/2/children", nil)
	existing := httptest.NewRecorder()
	r.ServeHTTP(existing, req)

	// Org 424242 does not exist at all.
	req, _ = http.NewRequest("GET", "/organizations/424242/children", nil)
	missing := httptest.NewRecorder()
	r.ServeHTTP(missing, req)

	if existing.Code != http.StatusForbidden {
		t.Errorf("foreign org that exists: expected 403, got %d: %s", existing.Code, existing.Body.String())
	}
	if missing.Code != existing.Code {
		t.Errorf("org existence is observable: existing org gave %d, missing org gave %d",
			existing.Code, missing.Code)
	}
}
