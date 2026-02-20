package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/models"
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    apperror.CodeUnauthorized,
				Message: "unauthorized",
			})
			return
		}

		userIDUint, ok := userID.(uint)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, models.ErrorResponse{
				Code:    apperror.CodeInternal,
				Message: "invalid user id",
			})
			return
		}

		// First check if user is superadmin (can access everything)
		isSuperAdmin, err := m.permissionService.IsSuperAdmin(c.Request.Context(), userIDUint)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, models.ErrorResponse{
				Code:    apperror.CodeInternal,
				Message: "authorization check failed",
			})
			return
		}
		if isSuperAdmin {
			c.Next()
			return
		}

		// Get organization ID from path parameter
		orgIDStr := c.Param("orgId")
		if orgIDStr == "" {
			// For endpoints without orgId, try to get it from the resource itself
			// This requires looking up the resource - for now, deny access
			c.AbortWithStatusJSON(http.StatusForbidden, models.ErrorResponse{
				Code:    apperror.CodeForbidden,
				Message: "organization context required",
			})
			return
		}

		orgID, err := strconv.ParseUint(orgIDStr, 10, 32)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, models.ErrorResponse{
				Code:    apperror.CodeBadRequest,
				Message: "invalid organization id",
			})
			return
		}

		// Check permission
		allowed, err := m.permissionService.CheckPermission(c.Request.Context(), userIDUint, uint(orgID), resource, action)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, models.ErrorResponse{
				Code:    apperror.CodeInternal,
				Message: "authorization check failed",
			})
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, models.ErrorResponse{
				Code:    apperror.CodeForbidden,
				Message: "forbidden",
			})
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    apperror.CodeUnauthorized,
				Message: "unauthorized",
			})
			return
		}

		userIDUint, ok := userID.(uint)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, models.ErrorResponse{
				Code:    apperror.CodeInternal,
				Message: "invalid user id",
			})
			return
		}

		isSuperAdmin, err := m.permissionService.IsSuperAdmin(c.Request.Context(), userIDUint)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, models.ErrorResponse{
				Code:    apperror.CodeInternal,
				Message: "authorization check failed",
			})
			return
		}

		if !isSuperAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, models.ErrorResponse{
				Code:    apperror.CodeForbidden,
				Message: "superadmin access required",
			})
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    apperror.CodeUnauthorized,
				Message: "unauthorized",
			})
			return
		}

		userIDUint, ok := userID.(uint)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, models.ErrorResponse{
				Code:    apperror.CodeInternal,
				Message: "invalid user id",
			})
			return
		}

		allowed, err := m.permissionService.HasPermissionInAnyOrg(c.Request.Context(), userIDUint, resource, action)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, models.ErrorResponse{
				Code:    apperror.CodeInternal,
				Message: "authorization check failed",
			})
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, models.ErrorResponse{
				Code:    apperror.CodeForbidden,
				Message: "forbidden",
			})
			return
		}

		c.Next()
	}
}
