package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/handlers"
	"github.com/eenemeene/kitamanager-go/internal/middleware"
	"github.com/eenemeene/kitamanager-go/internal/rbac"
)

// Deps groups all dependencies needed for route setup.
type Deps struct {
	Auth                  *handlers.AuthHandler
	User                  *handlers.UserHandler
	Section               *handlers.SectionHandler
	Organization          *handlers.OrganizationHandler
	Employee              *handlers.EmployeeHandler
	Child                 *handlers.ChildHandler
	GovernmentFunding     *handlers.GovernmentFundingHandler
	PayPlan               *handlers.PayPlanHandler
	ChildAttendance       *handlers.ChildAttendanceHandler
	BudgetItem            *handlers.BudgetItemHandler
	StepPromotion         *handlers.StepPromotionHandler
	Statistics            *handlers.StatisticsHandler
	Export                *handlers.ExportHandler
	GovernmentFundingBill *handlers.GovernmentFundingBillHandler
	AuditLog              *handlers.AuditLogHandler
	AuthMiddleware        *middleware.AuthMiddleware
	AuthzMiddleware       *middleware.AuthorizationMiddleware
	CSRFMiddleware        *middleware.CSRFMiddleware
	LoginRateLimiter      *middleware.RateLimiter
	APIRateLimiter        *middleware.RateLimiter
}

func Setup(r *gin.Engine, d Deps) {
	authHandler := d.Auth
	userHandler := d.User
	sectionHandler := d.Section
	orgHandler := d.Organization
	employeeHandler := d.Employee
	childHandler := d.Child
	governmentFundingHandler := d.GovernmentFunding
	payPlanHandler := d.PayPlan
	childAttendanceHandler := d.ChildAttendance
	budgetItemHandler := d.BudgetItem
	stepPromotionHandler := d.StepPromotion
	statisticsHandler := d.Statistics
	exportHandler := d.Export
	governmentFundingBillHandler := d.GovernmentFundingBill
	auditLogHandler := d.AuditLog
	authMiddleware := d.AuthMiddleware
	authzMiddleware := d.AuthzMiddleware
	csrfMiddleware := d.CSRFMiddleware
	loginRateLimiter := d.LoginRateLimiter
	apiRateLimiter := d.APIRateLimiter
	api := r.Group("/api/v1")
	{
		// Public endpoints with rate limiting
		if loginRateLimiter != nil {
			api.POST("/login", loginRateLimiter.RateLimit(), authHandler.Login)
			api.POST("/refresh", loginRateLimiter.RateLimit(), authHandler.Refresh)
		} else {
			api.POST("/login", authHandler.Login)
			api.POST("/refresh", authHandler.Refresh)
		}

		// Protected endpoints (require authentication and CSRF for cookie-based auth)
		protected := api.Group("")
		protected.Use(authMiddleware.RequireAuth())
		protected.Use(csrfMiddleware.ValidateCSRF())
		if apiRateLimiter != nil {
			protected.Use(apiRateLimiter.RateLimitMutations())
		}

		// Current user endpoints
		protected.POST("/logout", authHandler.Logout)
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
			governmentFundings := protected.Group("/government-funding-rates")
			{
				// Import (must be before :fundingId to avoid route conflict)
				governmentFundings.POST("/import", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.Import)

				governmentFundings.GET("", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.List)
				governmentFundings.GET("/:fundingId", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.Get)
				governmentFundings.POST("", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.Create)
				governmentFundings.PUT("/:fundingId", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.Update)
				governmentFundings.DELETE("/:fundingId", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.Delete)

				// Period management
				governmentFundings.GET("/:fundingId/periods/:periodId", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.GetPeriod)
				governmentFundings.POST("/:fundingId/periods", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.CreatePeriod)
				governmentFundings.PUT("/:fundingId/periods/:periodId", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.UpdatePeriod)
				governmentFundings.DELETE("/:fundingId/periods/:periodId", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.DeletePeriod)

				// Property management (directly under periods)
				governmentFundings.GET("/:fundingId/periods/:periodId/properties/:propertyId", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.GetProperty)
				governmentFundings.POST("/:fundingId/periods/:periodId/properties", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.CreateProperty)
				governmentFundings.PUT("/:fundingId/periods/:periodId/properties/:propertyId", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.UpdateProperty)
				governmentFundings.DELETE("/:fundingId/periods/:periodId/properties/:propertyId", authzMiddleware.RequireSuperAdmin(), governmentFundingHandler.DeleteProperty)
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
				users.POST("/:userId/organizations",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
					userHandler.AddToOrganization)
				users.PUT("/:userId/organizations/:orgId",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
					userHandler.UpdateOrganizationRole)
				users.DELETE("/:userId/organizations/:orgId",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
					userHandler.RemoveFromOrganization)
				users.GET("/:userId/memberships",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionRead),
					userHandler.GetMemberships)
				users.PUT("/:userId/password",
					authzMiddleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
					userHandler.ResetPassword)
				users.PUT("/:userId/superadmin",
					authzMiddleware.RequireSuperAdmin(),
					userHandler.SetSuperAdmin)
			}

			// ============================================================
			// Audit logs (superadmin only, read-only)
			// ============================================================
			auditLogs := protected.Group("/audit-logs")
			{
				auditLogs.GET("", authzMiddleware.RequireSuperAdmin(), auditLogHandler.List)
				auditLogs.GET("/:auditLogId", authzMiddleware.RequireSuperAdmin(), auditLogHandler.Get)
			}

			// ============================================================
			// Organization-scoped resources
			// All routes under /organizations/:orgId/... require org access
			// ============================================================
			orgScoped := protected.Group("/organizations/:orgId")
			{
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
					authzMiddleware.RequirePermission(rbac.ResourceStatistics, rbac.ActionRead),
					statisticsHandler.GetStaffingHours)
				orgScoped.GET("/statistics/financials",
					authzMiddleware.RequirePermission(rbac.ResourceStatistics, rbac.ActionRead),
					statisticsHandler.GetFinancials)
				orgScoped.GET("/statistics/occupancy",
					authzMiddleware.RequirePermission(rbac.ResourceStatistics, rbac.ActionRead),
					statisticsHandler.GetOccupancy)
				orgScoped.GET("/statistics/staffing-hours/employees",
					authzMiddleware.RequirePermission(rbac.ResourceStatistics, rbac.ActionRead),
					statisticsHandler.GetEmployeeStaffingHours)
				orgScoped.GET("/statistics/age-distribution",
					authzMiddleware.RequirePermission(rbac.ResourceStatistics, rbac.ActionRead),
					statisticsHandler.GetAgeDistribution)
				orgScoped.GET("/statistics/contract-properties",
					authzMiddleware.RequirePermission(rbac.ResourceStatistics, rbac.ActionRead),
					statisticsHandler.GetContractPropertiesDistribution)
				orgScoped.GET("/statistics/funding",
					authzMiddleware.RequirePermission(rbac.ResourceStatistics, rbac.ActionRead),
					statisticsHandler.GetFunding)
				orgScoped.POST("/statistics/forecast",
					authzMiddleware.RequirePermission(rbac.ResourceStatistics, rbac.ActionRead),
					statisticsHandler.GetForecast)
				orgScoped.POST("/statistics/estimates/child-funding",
					authzMiddleware.RequirePermission(rbac.ResourceStatistics, rbac.ActionRead),
					statisticsHandler.EstimateChildFunding)
				orgScoped.POST("/statistics/estimates/employee-cost",
					authzMiddleware.RequirePermission(rbac.ResourceStatistics, rbac.ActionRead),
					statisticsHandler.EstimateEmployeeCost)

				// Employees
				employees := orgScoped.Group("/employees")
				{
					// Export (must be before /:employeeId to avoid route conflict)
					employees.GET("/export/excel",
						authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead),
						exportHandler.ExportEmployees)
					employees.GET("/export/yaml",
						authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead),
						employeeHandler.ExportYAML)

					// Import (must be before /:employeeId to avoid route conflict)
					employees.POST("/import",
						authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionCreate),
						employeeHandler.Import)

					// Step promotions (must be before /:employeeId to avoid route conflict)
					employees.GET("/step-promotions",
						authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead),
						stepPromotionHandler.GetStepPromotions)

					employees.GET("",
						authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead),
						employeeHandler.List)
					employees.GET("/:employeeId",
						authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead),
						employeeHandler.Get)
					employees.POST("",
						authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionCreate),
						employeeHandler.Create)
					employees.PUT("/:employeeId",
						authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionUpdate),
						employeeHandler.Update)
					employees.DELETE("/:employeeId",
						authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionDelete),
						employeeHandler.Delete)

					// Employee contracts
					employees.GET("/:employeeId/contracts",
						authzMiddleware.RequirePermission(rbac.ResourceEmployeeContracts, rbac.ActionRead),
						employeeHandler.ListContracts)
					employees.GET("/:employeeId/contracts/current",
						authzMiddleware.RequirePermission(rbac.ResourceEmployeeContracts, rbac.ActionRead),
						employeeHandler.GetCurrentRecord)
					employees.POST("/:employeeId/contracts",
						authzMiddleware.RequirePermission(rbac.ResourceEmployeeContracts, rbac.ActionCreate),
						employeeHandler.CreateContract)
					employees.PUT("/:employeeId/contracts/batch",
						authzMiddleware.RequirePermission(rbac.ResourceEmployeeContracts, rbac.ActionUpdate),
						employeeHandler.BatchUpdateContracts)
					employees.GET("/:employeeId/contracts/:contractId",
						authzMiddleware.RequirePermission(rbac.ResourceEmployeeContracts, rbac.ActionRead),
						employeeHandler.GetContract)
					employees.PUT("/:employeeId/contracts/:contractId",
						authzMiddleware.RequirePermission(rbac.ResourceEmployeeContracts, rbac.ActionUpdate),
						employeeHandler.UpdateContract)
					employees.DELETE("/:employeeId/contracts/:contractId",
						authzMiddleware.RequirePermission(rbac.ResourceEmployeeContracts, rbac.ActionDelete),
						employeeHandler.DeleteContract)
				}

				// ============================================================
				// Government funding bill management (org-scoped)
				// ============================================================
				fundingBills := orgScoped.Group("/government-funding-bills")
				{
					fundingBills.GET("",
						authzMiddleware.RequirePermission(rbac.ResourceGovernmentFundingBills, rbac.ActionRead),
						governmentFundingBillHandler.List)
					fundingBills.GET("/:billId",
						authzMiddleware.RequirePermission(rbac.ResourceGovernmentFundingBills, rbac.ActionRead),
						governmentFundingBillHandler.Get)
					fundingBills.GET("/:billId/compare",
						authzMiddleware.RequirePermission(rbac.ResourceGovernmentFundingBills, rbac.ActionRead),
						governmentFundingBillHandler.Compare)
					fundingBills.POST("",
						authzMiddleware.RequirePermission(rbac.ResourceGovernmentFundingBills, rbac.ActionCreate),
						governmentFundingBillHandler.UploadISBJ)
					fundingBills.DELETE("/:billId",
						authzMiddleware.RequirePermission(rbac.ResourceGovernmentFundingBills, rbac.ActionDelete),
						governmentFundingBillHandler.Delete)
				}

				// Children
				children := orgScoped.Group("/children")
				{
					// Export (must be before /:childId to avoid route conflict)
					children.GET("/export/excel",
						authzMiddleware.RequirePermission(rbac.ResourceChildren, rbac.ActionRead),
						exportHandler.ExportChildren)
					children.GET("/export/yaml",
						authzMiddleware.RequirePermission(rbac.ResourceChildren, rbac.ActionRead),
						childHandler.ExportYAML)

					// Import (must be before /:childId to avoid route conflict)
					children.POST("/import",
						authzMiddleware.RequirePermission(rbac.ResourceChildren, rbac.ActionCreate),
						childHandler.Import)

					// ============================================================
					// Org-wide child attendance endpoints (must come before /:childId)
					// ============================================================
					children.GET("/attendance",
						authzMiddleware.RequirePermission(rbac.ResourceChildAttendance, rbac.ActionRead),
						childAttendanceHandler.ListByDate)
					children.GET("/attendance/summary",
						authzMiddleware.RequirePermission(rbac.ResourceChildAttendance, rbac.ActionRead),
						childAttendanceHandler.GetDailySummary)

					children.GET("",
						authzMiddleware.RequirePermission(rbac.ResourceChildren, rbac.ActionRead),
						childHandler.List)
					children.GET("/:childId",
						authzMiddleware.RequirePermission(rbac.ResourceChildren, rbac.ActionRead),
						childHandler.Get)
					children.POST("",
						authzMiddleware.RequirePermission(rbac.ResourceChildren, rbac.ActionCreate),
						childHandler.Create)
					children.PUT("/:childId",
						authzMiddleware.RequirePermission(rbac.ResourceChildren, rbac.ActionUpdate),
						childHandler.Update)
					children.DELETE("/:childId",
						authzMiddleware.RequirePermission(rbac.ResourceChildren, rbac.ActionDelete),
						childHandler.Delete)

					// Child contracts
					children.GET("/:childId/contracts",
						authzMiddleware.RequirePermission(rbac.ResourceChildContracts, rbac.ActionRead),
						childHandler.ListContracts)
					children.GET("/:childId/contracts/current",
						authzMiddleware.RequirePermission(rbac.ResourceChildContracts, rbac.ActionRead),
						childHandler.GetCurrentRecord)
					children.POST("/:childId/contracts",
						authzMiddleware.RequirePermission(rbac.ResourceChildContracts, rbac.ActionCreate),
						childHandler.CreateContract)
					children.PUT("/:childId/contracts/batch",
						authzMiddleware.RequirePermission(rbac.ResourceChildContracts, rbac.ActionUpdate),
						childHandler.BatchUpdateContracts)
					children.GET("/:childId/contracts/:contractId",
						authzMiddleware.RequirePermission(rbac.ResourceChildContracts, rbac.ActionRead),
						childHandler.GetContract)
					children.PUT("/:childId/contracts/:contractId",
						authzMiddleware.RequirePermission(rbac.ResourceChildContracts, rbac.ActionUpdate),
						childHandler.UpdateContract)
					children.DELETE("/:childId/contracts/:contractId",
						authzMiddleware.RequirePermission(rbac.ResourceChildContracts, rbac.ActionDelete),
						childHandler.DeleteContract)

					// ============================================================
					// Per-child attendance tracking
					// Routes: /children/:childId/attendance/...
					// Uses same :childId param as children resource (Gin resolves
					// based on route structure). Sub-resource uses :attendanceId.
					// ============================================================
					childAttendance := children.Group("/:childId/attendance")
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
				payplans := orgScoped.Group("/pay-plans")
				{
					// Import (must come before /:payPlanId to avoid route conflict)
					payplans.POST("/import",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionCreate),
						payPlanHandler.Import)

					payplans.GET("",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionRead),
						payPlanHandler.List)
					payplans.GET("/:payPlanId",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionRead),
						payPlanHandler.Get)
					payplans.GET("/:payPlanId/export",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionRead),
						payPlanHandler.Export)
					payplans.POST("",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionCreate),
						payPlanHandler.Create)
					payplans.PUT("/:payPlanId",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionUpdate),
						payPlanHandler.Update)
					payplans.DELETE("/:payPlanId",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionDelete),
						payPlanHandler.Delete)

					// Period management
					payplans.POST("/:payPlanId/periods",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionCreate),
						payPlanHandler.CreatePeriod)
					payplans.GET("/:payPlanId/periods/:periodId",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionRead),
						payPlanHandler.GetPeriod)
					payplans.PUT("/:payPlanId/periods/:periodId",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionUpdate),
						payPlanHandler.UpdatePeriod)
					payplans.DELETE("/:payPlanId/periods/:periodId",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionDelete),
						payPlanHandler.DeletePeriod)

					// Entry management
					payplans.POST("/:payPlanId/periods/:periodId/entries",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionCreate),
						payPlanHandler.CreateEntry)
					payplans.GET("/:payPlanId/periods/:periodId/entries/:entryId",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionRead),
						payPlanHandler.GetEntry)
					payplans.PUT("/:payPlanId/periods/:periodId/entries/:entryId",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionUpdate),
						payPlanHandler.UpdateEntry)
					payplans.DELETE("/:payPlanId/periods/:periodId/entries/:entryId",
						authzMiddleware.RequirePermission(rbac.ResourcePayPlans, rbac.ActionDelete),
						payPlanHandler.DeleteEntry)
				}

				// ============================================================
				// Budget Item management (org-scoped)
				// ============================================================
				budgetItems := orgScoped.Group("/budget-items")
				{
					budgetItems.GET("",
						authzMiddleware.RequirePermission(rbac.ResourceBudgetItems, rbac.ActionRead),
						budgetItemHandler.List)
					budgetItems.GET("/:budgetItemId",
						authzMiddleware.RequirePermission(rbac.ResourceBudgetItems, rbac.ActionRead),
						budgetItemHandler.Get)
					budgetItems.POST("",
						authzMiddleware.RequirePermission(rbac.ResourceBudgetItems, rbac.ActionCreate),
						budgetItemHandler.Create)
					budgetItems.PUT("/:budgetItemId",
						authzMiddleware.RequirePermission(rbac.ResourceBudgetItems, rbac.ActionUpdate),
						budgetItemHandler.Update)
					budgetItems.DELETE("/:budgetItemId",
						authzMiddleware.RequirePermission(rbac.ResourceBudgetItems, rbac.ActionDelete),
						budgetItemHandler.Delete)

					// Budget item entry management
					budgetItems.GET("/:budgetItemId/entries",
						authzMiddleware.RequirePermission(rbac.ResourceBudgetItemEntries, rbac.ActionRead),
						budgetItemHandler.ListEntries)
					budgetItems.POST("/:budgetItemId/entries",
						authzMiddleware.RequirePermission(rbac.ResourceBudgetItemEntries, rbac.ActionCreate),
						budgetItemHandler.CreateEntry)
					budgetItems.GET("/:budgetItemId/entries/:entryId",
						authzMiddleware.RequirePermission(rbac.ResourceBudgetItemEntries, rbac.ActionRead),
						budgetItemHandler.GetEntry)
					budgetItems.PUT("/:budgetItemId/entries/:entryId",
						authzMiddleware.RequirePermission(rbac.ResourceBudgetItemEntries, rbac.ActionUpdate),
						budgetItemHandler.UpdateEntry)
					budgetItems.DELETE("/:budgetItemId/entries/:entryId",
						authzMiddleware.RequirePermission(rbac.ResourceBudgetItemEntries, rbac.ActionDelete),
						budgetItemHandler.DeleteEntry)
				}
			}

		}
	}
}
