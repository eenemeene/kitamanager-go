package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/problem"
	"github.com/eenemeene/kitamanager-go/internal/rbac"
)

// AuthorizationMiddleware handles RBAC authorization.
type AuthorizationMiddleware struct {
	permissionService *rbac.PermissionService
}

// NewAuthorizationMiddleware creates a new authorization middleware.
func NewAuthorizationMiddleware(permissionService *rbac.PermissionService) *AuthorizationMiddleware {
	return &AuthorizationMiddleware{permissionService: permissionService}
}

// RequirePermission returns a middleware that checks if the user has permission
// to perform the specified action on the resource.
// The organization ID is extracted from the "orgId" path parameter.
func (m *AuthorizationMiddleware) RequirePermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get(ctxkeys.UserID)
		if !exists {
			problem.Write(c, http.StatusUnauthorized, apperror.CodeUnauthorized, "unauthorized")
			return
		}

		userIDUint, ok := userID.(uint)
		if !ok {
			problem.Write(c, http.StatusInternalServerError, apperror.CodeInternal, "invalid user id")
			return
		}

		// Resolve superadmin up front. The flag is stashed in the context
		// so downstream handlers do not need to repeat the DB lookup, and
		// it is resolved before the org checks below so that the value is
		// present on every exit path, not only the ones that get through.
		isSuperAdmin, err := m.permissionService.IsSuperAdmin(c.Request.Context(), userIDUint)
		if err != nil {
			problem.Write(c, http.StatusInternalServerError, apperror.CodeInternal, "authorization check failed")
			return
		}
		c.Set(ctxkeys.IsSuperAdmin, isSuperAdmin)

		// Get organization ID from path parameter
		orgIDStr := c.Param("orgId")
		if orgIDStr == "" {
			// RequirePermission is intended for routes that include
			// :orgId in their pattern; everything else uses
			// RequireGlobalPermission. Reaching this branch means a
			// developer wired a route through the wrong middleware —
			// fail closed with a 403 rather than try to infer the
			// tenant scope from a resource lookup (which would need
			// the resource's type and a DB round-trip per request).
			//
			// Applies to superadmins too. The superadmin short-circuit
			// used to sit above this check, so a mis-wired route failed
			// closed for everyone except the one caller who could do the
			// most damage through it.
			problem.Write(c, http.StatusForbidden, apperror.CodeForbidden, "organization context required")
			return
		}

		orgID, err := strconv.ParseUint(orgIDStr, 10, 32)
		if err != nil {
			problem.Write(c, http.StatusBadRequest, apperror.CodeBadRequest, "invalid organization id")
			return
		}

		// Authorize first, then check that the organization still exists.
		//
		// The order matters and is not the obvious one. Checking liveness
		// up front would answer 404 for an org that does not exist and 403
		// for one that does but is closed to the caller — an existence
		// oracle letting any authenticated user enumerate organization ids.
		// Running the permission check first means an outsider gets 403
		// either way, exactly as before, and only a caller who already has
		// a role in the organization — someone who necessarily knows it
		// exists — can observe the difference.
		//
		// It also costs nothing to detect the case we care about. A missing
		// org has no user_organizations rows, so GetRoleInOrg returns "" and
		// the caller is refused here. A TOMBSTONED org is the opposite: its
		// membership rows survive the soft-delete, so roles keep resolving
		// and the caller sails through — which is precisely why the data
		// stayed readable and why the liveness check has to come next.
		if !isSuperAdmin {
			allowed, err := m.permissionService.CheckPermission(c.Request.Context(), userIDUint, uint(orgID), resource, action)
			if err != nil {
				problem.Write(c, http.StatusInternalServerError, apperror.CodeInternal, "authorization check failed")
				return
			}
			if !allowed {
				problem.Write(c, http.StatusForbidden, apperror.CodeForbidden, "forbidden")
				return
			}
		}

		// The organization must still exist. Soft-deleting one leaves every
		// user_organizations row in place, and no org-scoped store query
		// filters on the parent, so children, employees, contracts and bills
		// all stayed readable through /organizations/{id}/... indefinitely
		// after the org was deleted — for children's data, the highest
		// protection class in the system.
		//
		// Enforced here rather than in ~30 store queries because this is the
		// single gate every org-scoped route already passes through, which
		// makes it the only place the invariant can be stated once.
		//
		// Superadmins are subject to it too: the point is that the data is
		// gone, not that the caller lacks a role. The erasure path is
		// unaffected — DELETE /organizations/:orgId/purge is gated by
		// RequireSuperAdmin, not by this middleware, so a tombstoned org can
		// still be purged.
		orgLive, err := m.permissionService.OrganizationIsLive(c.Request.Context(), uint(orgID))
		if err != nil {
			problem.Write(c, http.StatusInternalServerError, apperror.CodeInternal, "authorization check failed")
			return
		}
		if !orgLive {
			problem.Write(c, http.StatusNotFound, apperror.CodeNotFound, "organization not found")
			return
		}

		// Store orgID in context for handlers to use. Set on both paths —
		// the superadmin short-circuit used to skip it.
		c.Set(ctxkeys.OrgID, uint(orgID))
		c.Next()
	}
}

// RequireSuperAdmin returns a middleware that only allows superadmins.
func (m *AuthorizationMiddleware) RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get(ctxkeys.UserID)
		if !exists {
			problem.Write(c, http.StatusUnauthorized, apperror.CodeUnauthorized, "unauthorized")
			return
		}

		userIDUint, ok := userID.(uint)
		if !ok {
			problem.Write(c, http.StatusInternalServerError, apperror.CodeInternal, "invalid user id")
			return
		}

		isSuperAdmin, err := m.permissionService.IsSuperAdmin(c.Request.Context(), userIDUint)
		if err != nil {
			problem.Write(c, http.StatusInternalServerError, apperror.CodeInternal, "authorization check failed")
			return
		}
		c.Set(ctxkeys.IsSuperAdmin, isSuperAdmin)

		if !isSuperAdmin {
			problem.Write(c, http.StatusForbidden, apperror.CodeForbidden, "superadmin access required")
			return
		}

		c.Next()
	}
}

// RequireGlobalPermission returns a middleware that checks if the user has permission
// to perform the specified action on a global resource (like users or groups) in ANY
// of their assigned organizations. This is for resources that aren't org-scoped in URLs.
func (m *AuthorizationMiddleware) RequireGlobalPermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get(ctxkeys.UserID)
		if !exists {
			problem.Write(c, http.StatusUnauthorized, apperror.CodeUnauthorized, "unauthorized")
			return
		}

		userIDUint, ok := userID.(uint)
		if !ok {
			problem.Write(c, http.StatusInternalServerError, apperror.CodeInternal, "invalid user id")
			return
		}

		// Resolve superadmin up-front so downstream handlers can read the
		// ctx key without repeating the lookup (M2).
		isSuperAdmin, err := m.permissionService.IsSuperAdmin(c.Request.Context(), userIDUint)
		if err != nil {
			problem.Write(c, http.StatusInternalServerError, apperror.CodeInternal, "authorization check failed")
			return
		}
		c.Set(ctxkeys.IsSuperAdmin, isSuperAdmin)

		allowed, err := m.permissionService.HasPermissionInAnyOrg(c.Request.Context(), userIDUint, resource, action)
		if err != nil {
			problem.Write(c, http.StatusInternalServerError, apperror.CodeInternal, "authorization check failed")
			return
		}

		if !allowed {
			problem.Write(c, http.StatusForbidden, apperror.CodeForbidden, "forbidden")
			return
		}

		c.Next()
	}
}
