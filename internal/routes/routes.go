package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/handlers"
	"github.com/eenemeene/kitamanager-go/internal/middleware"
	"github.com/eenemeene/kitamanager-go/internal/rbac"
)

func Setup(
	r *gin.Engine,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	groupHandler *handlers.GroupHandler,
	orgHandler *handlers.OrganizationHandler,
	employeeHandler *handlers.EmployeeHandler,
	childHandler *handlers.ChildHandler,
	authMiddleware *middleware.AuthMiddleware,
	authzMiddleware *middleware.AuthorizationMiddleware,
) {
	api := r.Group("/api/v1")
	{
		// Public endpoints
		api.POST("/login", authHandler.Login)

		// Protected endpoints (require authentication)
		protected := api.Group("")
		protected.Use(authMiddleware.RequireAuth())
		{
			// ============================================================
			// Organization management
			// Create/Delete: superadmin only
			// Read: any role
			// Update: admin+ (superadmin, admin)
			// ============================================================
			orgs := protected.Group("/organizations")
			{
				// Superadmin only
				orgs.POST("", authzMiddleware.RequireSuperAdmin(), orgHandler.Create)
				orgs.DELETE("/:orgId", authzMiddleware.RequireSuperAdmin(), orgHandler.Delete)

				// List: requires any role (filtered by access in handler/service)
				orgs.GET("", orgHandler.List)

				// Get specific org: requires read permission for that org
				orgs.GET("/:orgId",
					authzMiddleware.RequirePermission(rbac.ResourceOrganizations, rbac.ActionRead),
					orgHandler.Get)

				// Update specific org: requires update permission for that org
				orgs.PUT("/:orgId",
					authzMiddleware.RequirePermission(rbac.ResourceOrganizations, rbac.ActionUpdate),
					orgHandler.Update)
			}

			// ============================================================
			// User management (global, not org-scoped)
			// Permissions checked against any org the user has a role in
			// ============================================================
			users := protected.Group("/users")
			{
				users.GET("",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionRead),
					userHandler.List)
				users.GET("/:uid",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionRead),
					userHandler.Get)
				users.POST("",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionCreate),
					userHandler.Create)
				users.PUT("/:uid",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
					userHandler.Update)
				users.DELETE("/:uid",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionDelete),
					userHandler.Delete)
				users.POST("/:uid/groups",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
					userHandler.AddToGroup)
				users.PUT("/:uid/groups/:gid",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
					userHandler.UpdateGroupRole)
				users.DELETE("/:uid/groups/:gid",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
					userHandler.RemoveFromGroup)
				users.GET("/:uid/memberships",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionRead),
					userHandler.GetMemberships)
				users.PUT("/:uid/superadmin",
					authzMiddleware.RequireSuperAdmin(),
					userHandler.SetSuperAdmin)
				users.POST("/:uid/organizations",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
					userHandler.AddToOrganization)
				users.DELETE("/:uid/organizations/:oid",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
					userHandler.RemoveFromOrganization)
			}

			// ============================================================
			// Group management (org-scoped - each group belongs to one org)
			// Permissions checked against any org the user has a role in
			// ============================================================
			groups := protected.Group("/groups")
			{
				groups.GET("",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceGroups, rbac.ActionRead),
					groupHandler.List)
				groups.GET("/:groupId",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceGroups, rbac.ActionRead),
					groupHandler.Get)
				groups.POST("",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceGroups, rbac.ActionCreate),
					groupHandler.Create)
				groups.PUT("/:groupId",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceGroups, rbac.ActionUpdate),
					groupHandler.Update)
				groups.DELETE("/:groupId",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceGroups, rbac.ActionDelete),
					groupHandler.Delete)
			}

			// ============================================================
			// Organization-scoped resources
			// All routes under /organizations/:id/... require org access
			// ============================================================
			orgScoped := protected.Group("/organizations/:orgId")
			{
				// Employees
				employees := orgScoped.Group("/employees")
				{
					employees.GET("",
						authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead),
						employeeHandler.List)
					employees.GET("/:id",
						authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead),
						employeeHandler.Get)
					employees.POST("",
						authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionCreate),
						employeeHandler.Create)
					employees.PUT("/:id",
						authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionUpdate),
						employeeHandler.Update)
					employees.DELETE("/:id",
						authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionDelete),
						employeeHandler.Delete)

					// Employee contracts
					employees.GET("/:id/contracts",
						authzMiddleware.RequirePermission(rbac.ResourceEmployeeContracts, rbac.ActionRead),
						employeeHandler.ListContracts)
					employees.GET("/:id/contracts/current",
						authzMiddleware.RequirePermission(rbac.ResourceEmployeeContracts, rbac.ActionRead),
						employeeHandler.GetCurrentContract)
					employees.POST("/:id/contracts",
						authzMiddleware.RequirePermission(rbac.ResourceEmployeeContracts, rbac.ActionCreate),
						employeeHandler.CreateContract)
					employees.DELETE("/:id/contracts/:contractId",
						authzMiddleware.RequirePermission(rbac.ResourceEmployeeContracts, rbac.ActionDelete),
						employeeHandler.DeleteContract)
				}

				// Children
				children := orgScoped.Group("/children")
				{
					children.GET("",
						authzMiddleware.RequirePermission(rbac.ResourceChildren, rbac.ActionRead),
						childHandler.List)
					children.GET("/:id",
						authzMiddleware.RequirePermission(rbac.ResourceChildren, rbac.ActionRead),
						childHandler.Get)
					children.POST("",
						authzMiddleware.RequirePermission(rbac.ResourceChildren, rbac.ActionCreate),
						childHandler.Create)
					children.PUT("/:id",
						authzMiddleware.RequirePermission(rbac.ResourceChildren, rbac.ActionUpdate),
						childHandler.Update)
					children.DELETE("/:id",
						authzMiddleware.RequirePermission(rbac.ResourceChildren, rbac.ActionDelete),
						childHandler.Delete)

					// Child contracts
					children.GET("/:id/contracts",
						authzMiddleware.RequirePermission(rbac.ResourceChildContracts, rbac.ActionRead),
						childHandler.ListContracts)
					children.GET("/:id/contracts/current",
						authzMiddleware.RequirePermission(rbac.ResourceChildContracts, rbac.ActionRead),
						childHandler.GetCurrentContract)
					children.POST("/:id/contracts",
						authzMiddleware.RequirePermission(rbac.ResourceChildContracts, rbac.ActionCreate),
						childHandler.CreateContract)
					children.DELETE("/:id/contracts/:contractId",
						authzMiddleware.RequirePermission(rbac.ResourceChildContracts, rbac.ActionDelete),
						childHandler.DeleteContract)
				}
			}

		}
	}
}
