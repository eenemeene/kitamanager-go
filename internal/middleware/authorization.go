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

		// First check if user is superadmin (can access everything). The
		// resolved flag is stashed in the context so downstream handlers
		// (e.g. audit-log IP redaction) do not need to repeat the DB
		// lookup — see M2.
		isSuperAdmin, err := m.permissionService.IsSuperAdmin(c.Request.Context(), userIDUint)
		if err != nil {
			problem.Write(c, http.StatusInternalServerError, apperror.CodeInternal, "authorization check failed")
			return
		}
		c.Set(ctxkeys.IsSuperAdmin, isSuperAdmin)
		if isSuperAdmin {
			c.Next()
			return
		}

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
			problem.Write(c, http.StatusForbidden, apperror.CodeForbidden, "organization context required")
			return
		}

		orgID, err := strconv.ParseUint(orgIDStr, 10, 32)
		if err != nil {
			problem.Write(c, http.StatusBadRequest, apperror.CodeBadRequest, "invalid organization id")
			return
		}

		// Check permission
		allowed, err := m.permissionService.CheckPermission(c.Request.Context(), userIDUint, uint(orgID), resource, action)
		if err != nil {
			problem.Write(c, http.StatusInternalServerError, apperror.CodeInternal, "authorization check failed")
			return
		}

		if !allowed {
			problem.Write(c, http.StatusForbidden, apperror.CodeForbidden, "forbidden")
			return
		}

		// Store orgID in context for handlers to use
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
