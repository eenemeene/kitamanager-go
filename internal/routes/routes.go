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
			// Organization management (superadmin only for create/delete)
			// ============================================================
			orgs := protected.Group("/organizations")
			{
				// Superadmin only
				orgs.POST("", authzMiddleware.RequireSuperAdmin(), orgHandler.Create)
				orgs.DELETE("/:id", authzMiddleware.RequireSuperAdmin(), orgHandler.Delete)

				// Any authenticated user can list (they'll see based on their access)
				orgs.GET("", orgHandler.List)
				orgs.GET("/:id", orgHandler.Get)
				orgs.PUT("/:id", orgHandler.Update) // TODO: Add org-level permission check
			}

			// ============================================================
			// User management (global, not org-scoped)
			// ============================================================
			users := protected.Group("/users")
			{
				users.GET("", userHandler.List)
				users.GET("/:id", userHandler.Get)
				users.POST("", userHandler.Create)
				users.PUT("/:id", userHandler.Update)
				users.DELETE("/:id", userHandler.Delete)
				users.POST("/:id/groups", userHandler.AddToGroup)
				users.DELETE("/:id/groups/:gid", userHandler.RemoveFromGroup)
				users.POST("/:id/organizations", userHandler.AddToOrganization)
				users.DELETE("/:id/organizations/:oid", userHandler.RemoveFromOrganization)
			}

			// ============================================================
			// Group management (org-scoped - each group belongs to one org)
			// ============================================================
			groups := protected.Group("/groups")
			{
				groups.GET("", groupHandler.List)
				groups.GET("/:id", groupHandler.Get)
				groups.POST("", groupHandler.Create)
				groups.PUT("/:id", groupHandler.Update)
				groups.DELETE("/:id", groupHandler.Delete)
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
