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
	sectionHandler *handlers.SectionHandler,
	orgHandler *handlers.OrganizationHandler,
	employeeHandler *handlers.EmployeeHandler,
	childHandler *handlers.ChildHandler,
	governmentFundingHandler *handlers.GovernmentFundingHandler,
	payPlanHandler *handlers.PayPlanHandler,
	childAttendanceHandler *handlers.ChildAttendanceHandler,
	costHandler *handlers.CostHandler,
	stepPromotionHandler *handlers.StepPromotionHandler,
	statisticsHandler *handlers.StatisticsHandler,
	authMiddleware *middleware.AuthMiddleware,
	authzMiddleware *middleware.AuthorizationMiddleware,
	csrfMiddleware *middleware.CSRFMiddleware,
	loginRateLimiter *middleware.RateLimiter,
	apiRateLimiter *middleware.RateLimiter,
) {
	api := r.Group("/api/v1")
	{
		// Public endpoints with optional rate limiting
		if loginRateLimiter != nil {
			api.POST("/login", loginRateLimiter.RateLimit(), authHandler.Login)
			api.POST("/refresh", loginRateLimiter.RateLimit(), authHandler.Refresh)
		} else {
			api.POST("/login", authHandler.Login)
			api.POST("/refresh", authHandler.Refresh)
		}

		// Logout endpoint (no auth required - just clears cookies)
		api.POST("/logout", authHandler.Logout)

		// Protected endpoints (require authentication and CSRF for cookie-based auth)
		protected := api.Group("")
		protected.Use(authMiddleware.RequireAuth())
		protected.Use(csrfMiddleware.ValidateCSRF())
		if apiRateLimiter != nil {
			protected.Use(apiRateLimiter.RateLimitMutations())
		}

		// Current user endpoints
		protected.GET("/me", authHandler.Me)
		protected.PUT("/me/password", authHandler.ChangePassword)
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

				// List: requires read permission in any org (results filtered in service)
				orgs.GET("",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceOrganizations, rbac.ActionRead),
					orgHandler.List)

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
			// Government Funding management (superadmin only)
			// ============================================================
			governmentFundings := protected.Group("/government-fundings")
			{
				governmentFundings.GET("", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.List)
				governmentFundings.GET("/:id", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.Get)
				governmentFundings.POST("", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.Create)
				governmentFundings.PUT("/:id", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.Update)
				governmentFundings.DELETE("/:id", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.Delete)

				// Period management
				governmentFundings.POST("/:id/periods", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.CreatePeriod)
				governmentFundings.PUT("/:id/periods/:periodId", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.UpdatePeriod)
				governmentFundings.DELETE("/:id/periods/:periodId", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.DeletePeriod)

				// Property management (directly under periods)
				governmentFundings.POST("/:id/periods/:periodId/properties", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.CreateProperty)
				governmentFundings.PUT("/:id/periods/:periodId/properties/:propId", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.UpdateProperty)
				governmentFundings.DELETE("/:id/periods/:periodId/properties/:propId", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.DeleteProperty)
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
				users.GET("/:userId",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionRead),
					userHandler.Get)
				users.POST("",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionCreate),
					userHandler.Create)
				users.PUT("/:userId",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
					userHandler.Update)
				users.DELETE("/:userId",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionDelete),
					userHandler.Delete)
				users.POST("/:userId/groups",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
					userHandler.AddToGroup)
				users.PUT("/:userId/groups/:groupId",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
					userHandler.UpdateGroupRole)
				users.DELETE("/:userId/groups/:groupId",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
					userHandler.RemoveFromGroup)
				users.GET("/:userId/memberships",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionRead),
					userHandler.GetMemberships)
				users.PUT("/:userId/password",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
					userHandler.ResetPassword)
				users.PUT("/:userId/superadmin",
					authzMiddleware.RequireSuperAdmin(),
					userHandler.SetSuperAdmin)
				users.POST("/:userId/organizations",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
					userHandler.AddToOrganization)
				users.DELETE("/:userId/organizations/:orgId",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
					userHandler.RemoveFromOrganization)
			}

			// ============================================================
			// Organization-scoped resources
			// All routes under /organizations/:id/... require org access
			// ============================================================
			orgScoped := protected.Group("/organizations/:orgId")
			{
				// ============================================================
				// Group management (org-scoped - each group belongs to one org)
				// ============================================================
				groups := orgScoped.Group("/groups")
				{
					groups.GET("",
						authzMiddleware.RequirePermission(rbac.ResourceGroups, rbac.ActionRead),
						groupHandler.List)
					groups.GET("/:groupId",
						authzMiddleware.RequirePermission(rbac.ResourceGroups, rbac.ActionRead),
						groupHandler.Get)
					groups.POST("",
						authzMiddleware.RequirePermission(rbac.ResourceGroups, rbac.ActionCreate),
						groupHandler.Create)
					groups.PUT("/:groupId",
						authzMiddleware.RequirePermission(rbac.ResourceGroups, rbac.ActionUpdate),
						groupHandler.Update)
					groups.DELETE("/:groupId",
						authzMiddleware.RequirePermission(rbac.ResourceGroups, rbac.ActionDelete),
						groupHandler.Delete)
				}

				// ============================================================
				// Section management (org-scoped - each section belongs to one org)
				// ============================================================
				sections := orgScoped.Group("/sections")
				{
					sections.GET("",
						authzMiddleware.RequirePermission(rbac.ResourceSections, rbac.ActionRead),
						sectionHandler.List)
					sections.GET("/:sectionId",
						authzMiddleware.RequirePermission(rbac.ResourceSections, rbac.ActionRead),
						sectionHandler.Get)
					sections.POST("",
						authzMiddleware.RequirePermission(rbac.ResourceSections, rbac.ActionCreate),
						sectionHandler.Create)
					sections.PUT("/:sectionId",
						authzMiddleware.RequirePermission(rbac.ResourceSections, rbac.ActionUpdate),
						sectionHandler.Update)
					sections.DELETE("/:sectionId",
						authzMiddleware.RequirePermission(rbac.ResourceSections, rbac.ActionDelete),
						sectionHandler.Delete)
				}

				// ============================================================
				// Users in organization (read-only list)
				// ============================================================
				orgScoped.GET("/users",
					authzMiddleware.RequirePermission(rbac.ResourceUsers, rbac.ActionRead),
					userHandler.ListByOrganization)

				// ============================================================
				// Organization-wide statistics
				// ============================================================
				orgScoped.GET("/statistics/staffing-hours",
					authzMiddleware.RequirePermission(rbac.ResourceChildren, rbac.ActionRead),
					statisticsHandler.GetStaffingHours)

				// Employees
				employees := orgScoped.Group("/employees")
				{
					// Step promotions (must be before /:id to avoid route conflict)
					employees.GET("/step-promotions",
						authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead),
						stepPromotionHandler.GetStepPromotions)

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
						employeeHandler.GetCurrentRecord)
					employees.POST("/:id/contracts",
						authzMiddleware.RequirePermission(rbac.ResourceEmployeeContracts, rbac.ActionCreate),
						employeeHandler.CreateContract)
					employees.GET("/:id/contracts/:contractId",
						authzMiddleware.RequirePermission(rbac.ResourceEmployeeContracts, rbac.ActionRead),
						employeeHandler.GetContract)
					employees.PUT("/:id/contracts/:contractId",
						authzMiddleware.RequirePermission(rbac.ResourceEmployeeContracts, rbac.ActionUpdate),
						employeeHandler.UpdateContract)
					employees.DELETE("/:id/contracts/:contractId",
						authzMiddleware.RequirePermission(rbac.ResourceEmployeeContracts, rbac.ActionDelete),
						employeeHandler.DeleteContract)
				}

				// Children
				children := orgScoped.Group("/children")
				{
					// ============================================================
					// Org-wide child attendance endpoints (must come before /:id)
					// ============================================================
					children.GET("/attendance",
						authzMiddleware.RequirePermission(rbac.ResourceChildAttendance, rbac.ActionRead),
						childAttendanceHandler.ListByDate)
					children.GET("/attendance/summary",
						authzMiddleware.RequirePermission(rbac.ResourceChildAttendance, rbac.ActionRead),
						childAttendanceHandler.GetDailySummary)

					// Statistics endpoint (must be before /:id to avoid conflict)
					children.GET("/statistics/contract-count-by-month",
						authzMiddleware.RequirePermission(rbac.ResourceChildren, rbac.ActionRead),
						childHandler.GetContractCountByMonth)

					children.GET("/statistics/age-distribution",
						authzMiddleware.RequirePermission(rbac.ResourceChildren, rbac.ActionRead),
						childHandler.GetAgeDistribution)

					// Funding calculation endpoint (must be before /:id to avoid conflict)
					children.GET("/funding",
						authzMiddleware.RequirePermission(rbac.ResourceChildren, rbac.ActionRead),
						childHandler.GetFunding)

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
						childHandler.GetCurrentRecord)
					children.POST("/:id/contracts",
						authzMiddleware.RequirePermission(rbac.ResourceChildContracts, rbac.ActionCreate),
						childHandler.CreateContract)
					children.GET("/:id/contracts/:contractId",
						authzMiddleware.RequirePermission(rbac.ResourceChildContracts, rbac.ActionRead),
						childHandler.GetContract)
					children.PUT("/:id/contracts/:contractId",
						authzMiddleware.RequirePermission(rbac.ResourceChildContracts, rbac.ActionUpdate),
						childHandler.UpdateContract)
					children.DELETE("/:id/contracts/:contractId",
						authzMiddleware.RequirePermission(rbac.ResourceChildContracts, rbac.ActionDelete),
						childHandler.DeleteContract)

					// ============================================================
					// Per-child attendance tracking
					// Routes: /children/:id/attendance/...
					// Uses same :id param as children resource (Gin resolves
					// based on route structure). Sub-resource uses :attendanceId.
					// ============================================================
					childAttendance := children.Group("/:id/attendance")
					{
						childAttendance.POST("",
							authzMiddleware.RequirePermission(rbac.ResourceChildAttendance, rbac.ActionCreate),
							childAttendanceHandler.Create)
						childAttendance.GET("",
							authzMiddleware.RequirePermission(rbac.ResourceChildAttendance, rbac.ActionRead),
							childAttendanceHandler.ListByChild)
						childAttendance.GET("/:attendanceId",
							authzMiddleware.RequirePermission(rbac.ResourceChildAttendance, rbac.ActionRead),
							childAttendanceHandler.Get)
						childAttendance.PUT("/:attendanceId",
							authzMiddleware.RequirePermission(rbac.ResourceChildAttendance, rbac.ActionUpdate),
							childAttendanceHandler.Update)
						childAttendance.DELETE("/:attendanceId",
							authzMiddleware.RequirePermission(rbac.ResourceChildAttendance, rbac.ActionDelete),
							childAttendanceHandler.Delete)
					}
				}

				// ============================================================
				// Pay Plan management (org-scoped)
				// ============================================================
				payplans := orgScoped.Group("/payplans")
				{
					payplans.GET("",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionRead),
						payPlanHandler.List)
					payplans.GET("/:id",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionRead),
						payPlanHandler.Get)
					payplans.POST("",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionCreate),
						payPlanHandler.Create)
					payplans.PUT("/:id",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionUpdate),
						payPlanHandler.Update)
					payplans.DELETE("/:id",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionDelete),
						payPlanHandler.Delete)

					// Period management
					payplans.POST("/:id/periods",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionCreate),
						payPlanHandler.CreatePeriod)
					payplans.GET("/:id/periods/:periodId",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionRead),
						payPlanHandler.GetPeriod)
					payplans.PUT("/:id/periods/:periodId",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionUpdate),
						payPlanHandler.UpdatePeriod)
					payplans.DELETE("/:id/periods/:periodId",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionDelete),
						payPlanHandler.DeletePeriod)

					// Entry management
					payplans.POST("/:id/periods/:periodId/entries",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionCreate),
						payPlanHandler.CreateEntry)
					payplans.GET("/:id/periods/:periodId/entries/:entryId",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionRead),
						payPlanHandler.GetEntry)
					payplans.PUT("/:id/periods/:periodId/entries/:entryId",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionUpdate),
						payPlanHandler.UpdateEntry)
					payplans.DELETE("/:id/periods/:periodId/entries/:entryId",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionDelete),
						payPlanHandler.DeleteEntry)
				}

				// ============================================================
				// Cost management (org-scoped)
				// ============================================================
				costs := orgScoped.Group("/costs")
				{
					costs.GET("",
						authzMiddleware.RequirePermission(rbac.ResourceCosts, rbac.ActionRead),
						costHandler.List)
					costs.GET("/:id",
						authzMiddleware.RequirePermission(rbac.ResourceCosts, rbac.ActionRead),
						costHandler.Get)
					costs.POST("",
						authzMiddleware.RequirePermission(rbac.ResourceCosts, rbac.ActionCreate),
						costHandler.Create)
					costs.PUT("/:id",
						authzMiddleware.RequirePermission(rbac.ResourceCosts, rbac.ActionUpdate),
						costHandler.Update)
					costs.DELETE("/:id",
						authzMiddleware.RequirePermission(rbac.ResourceCosts, rbac.ActionDelete),
						costHandler.Delete)

					// Cost entry management
					costs.GET("/:id/entries",
						authzMiddleware.RequirePermission(rbac.ResourceCostEntries, rbac.ActionRead),
						costHandler.ListEntries)
					costs.POST("/:id/entries",
						authzMiddleware.RequirePermission(rbac.ResourceCostEntries, rbac.ActionCreate),
						costHandler.CreateEntry)
					costs.GET("/:id/entries/:entryId",
						authzMiddleware.RequirePermission(rbac.ResourceCostEntries, rbac.ActionRead),
						costHandler.GetEntry)
					costs.PUT("/:id/entries/:entryId",
						authzMiddleware.RequirePermission(rbac.ResourceCostEntries, rbac.ActionUpdate),
						costHandler.UpdateEntry)
					costs.DELETE("/:id/entries/:entryId",
						authzMiddleware.RequirePermission(rbac.ResourceCostEntries, rbac.ActionDelete),
						costHandler.DeleteEntry)
				}
			}

		}
	}
}
