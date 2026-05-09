//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/handlers"
	"github.com/eenemeene/kitamanager-go/internal/importer"
	"github.com/eenemeene/kitamanager-go/internal/middleware"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/rbac"
	"github.com/eenemeene/kitamanager-go/internal/routes"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// Why this file exists: the Casbin policy is well-pinned at the unit
// level in internal/rbac/rbac_test.go. That layer answers "is this
// (role, resource, action) triple allowed?" — but it cannot answer the
// question that actually breaks production: "did the route handler in
// routes.go remember to call RequirePermission?". A new endpoint that
// forgets the middleware passes every Casbin test (the policy is
// unchanged) and every handler unit test (those mostly run as admin).
// The only layer that catches it is the real HTTP middleware chain
// against the real router.
//
// This file therefore wires the SAME routes.Setup the production
// binary uses and drives requests against it with role-scoped session
// tokens. The matrix is hand-curated: any new endpoint that needs a
// permission gate should add a row here at the same time the gate
// gets added to routes.go. If the developer forgets either side, the
// test fails.

// setupFullProductionRouter mirrors cmd/api/main.go's wiring closely
// enough that routes.Setup sees the same handler/middleware shapes it
// does in production. The integration test infrastructure already
// gives us a real Postgres (testDB), so every store hits real SQL and
// every middleware runs the real authorization path.
func setupFullProductionRouter(t *testing.T) (*gin.Engine, *rbac.Enforcer) {
	t.Helper()

	enforcer, err := rbac.NewEnforcer(testDB, findRBACModel(t))
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	if err := enforcer.SeedDefaultPolicies(); err != nil {
		t.Fatalf("seed policies: %v", err)
	}

	transactor := store.NewTransactor(testDB)

	// Stores — full set, in the same order as cmd/api/main.go's
	// initStores. Keep this aligned: a missing store here means a
	// missing handler dependency three blocks down.
	userStore := store.NewUserStore(testDB)
	sectionStore := store.NewSectionStore(testDB)
	orgStore := store.NewOrganizationStore(testDB)
	employeeStore := store.NewEmployeeStore(testDB)
	childStore := store.NewChildStore(testDB)
	userOrgStore := store.NewUserOrganizationStore(testDB)
	fundingStore := store.NewGovernmentFundingStore(testDB)
	payPlanStore := store.NewPayPlanStore(testDB)
	attendanceStore := store.NewChildAttendanceStore(testDB)
	budgetItemStore := store.NewBudgetItemStore(testDB)
	auditStore := store.NewAuditStore(testDB)
	sessionStore := store.NewSessionStore(testDB)
	factorStore := store.NewFactorStore(testDB)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(testDB)
	childVoucherStore := store.NewChildVoucherStore(testDB)

	// Services. AuthService needs the JWT/CSRF secret; FactorService
	// needs the AEAD; both come from the existing test fixtures.
	auditService := service.NewAuditService(auditStore)
	factorService := service.NewFactorService(factorStore, userStore, testTOTPAEAD(t), "KitaManager (test)", nil, auditService)
	authService := service.NewAuthService(userStore, sessionStore, testJWTSecret, auditService, factorService)
	userService := service.NewUserService(userStore, userOrgStore, sessionStore).WithAuditService(auditService)
	userOrgService := service.NewUserOrganizationService(userOrgStore, userStore, transactor)
	orgService := service.NewOrganizationService(orgStore, userStore)
	sectionService := service.NewSectionService(sectionStore, transactor)
	employeeService := service.NewEmployeeService(employeeStore, payPlanStore, sectionStore, transactor)
	childService := service.NewChildService(childStore, orgStore, fundingStore, sectionStore, transactor)
	fundingService := service.NewGovernmentFundingService(fundingStore, transactor)
	payPlanService := service.NewPayPlanService(payPlanStore, transactor)
	attendanceService := service.NewChildAttendanceService(attendanceStore, childStore)
	budgetItemService := service.NewBudgetItemService(budgetItemStore, transactor)
	stepPromotionService := service.NewStepPromotionService(payPlanStore, employeeStore)
	statisticsService := service.NewStatisticsService(childStore, employeeStore, orgStore, fundingStore, payPlanStore, budgetItemStore, sectionStore, billPeriodStore)
	billService := service.NewGovernmentFundingBillService(childStore, childVoucherStore, billPeriodStore, orgStore, fundingStore, transactor)

	// Middlewares.
	permissionService := rbac.NewPermissionService(userOrgStore, enforcer)
	authMW := middleware.NewAuthMiddleware(sessionStore)
	authzMW := middleware.NewAuthorizationMiddleware(permissionService)
	csrfMW := middleware.NewCSRFMiddleware(testJWTSecret)

	r := gin.New()
	routes.Setup(r, routes.Deps{
		Auth:                  handlers.NewAuthHandler(authService, false),
		User:                  handlers.NewUserHandler(userService, userOrgService, auditService, sessionStore),
		Section:               handlers.NewSectionHandler(sectionService, auditService),
		Organization:          handlers.NewOrganizationHandler(orgService, auditService),
		Employee:              handlers.NewEmployeeHandler(employeeService, auditService),
		Child:                 handlers.NewChildHandler(childService, auditService),
		GovernmentFunding:     handlers.NewGovernmentFundingHandler(fundingService, auditService, importer.NewGovernmentFundingImporter(fundingService, transactor)),
		PayPlan:               handlers.NewPayPlanHandler(payPlanService, auditService),
		ChildAttendance:       handlers.NewChildAttendanceHandler(attendanceService, auditService),
		BudgetItem:            handlers.NewBudgetItemHandler(budgetItemService, auditService),
		StepPromotion:         handlers.NewStepPromotionHandler(stepPromotionService),
		Statistics:            handlers.NewStatisticsHandler(statisticsService),
		Export:                handlers.NewExportHandler(employeeService, childService, auditService),
		GovernmentFundingBill: handlers.NewGovernmentFundingBillHandler(billService, auditService),
		AuditLog:              handlers.NewAuditLogHandler(auditService),
		Factor:                handlers.NewFactorHandler(factorService),
		AuthMiddleware:        authMW,
		AuthzMiddleware:       authzMW,
		CSRFMiddleware:        csrfMW,
		// Rate limiters: nil is honoured by routes.Setup as "skip the
		// limiter middleware" — exactly what we want in tests since
		// the matrix fires many requests in quick succession.
	})
	return r, enforcer
}

// provisionRoleUser creates a user, assigns them the requested role
// in `orgID`, and returns a session token. Unlike the helpers in
// auth_login_flow_test.go this one talks directly to the DB so a test
// can quickly fan out across roles without a per-role admin-creates-user
// flow.
func provisionRoleUser(t *testing.T, r *gin.Engine, role models.Role, orgID uint) string {
	t.Helper()
	const password = "role-auth-test-pw-1234"
	email := fmt.Sprintf("%s-%d@role-auth.test", role, orgID)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	u := &models.User{
		Name:     fmt.Sprintf("Role %s in %d", role, orgID),
		Email:    email,
		Password: string(hash),
		Active:   true,
	}
	if err := testDB.Create(u).Error; err != nil {
		t.Fatalf("create %s user: %v", role, err)
	}
	uo := &models.UserOrganization{UserID: u.ID, OrganizationID: orgID, Role: role}
	if err := testDB.Create(uo).Error; err != nil {
		t.Fatalf("assign %s role: %v", role, err)
	}
	code, token := doLogin(t, r, email, password)
	if code != http.StatusOK || token == "" {
		t.Fatalf("login %s: code=%d token=%q", role, code, token)
	}
	return token
}

// roleCase is one row in the negative-coverage matrix. wantStatus is
// almost always 403 — the exception is "admin trying to create an org"
// (the route is gated by RequireSuperAdmin which also returns 403, so
// the assertion shape matches). 200/201 cases live in the separate
// positive baseline test below to avoid mixing concerns.
type roleCase struct {
	name       string
	role       models.Role
	method     string
	path       string // %d will be Sprintf'd with orgID
	wantStatus int
}

// TestRoleAuthorization_NegativeMatrix is the answer to "would we
// catch a developer forgetting RequirePermission on a new endpoint?"
// It exercises the production router (routes.Setup) with role-scoped
// session tokens and asserts every entry in the matrix below returns
// 403. Each row pins a specific Casbin invariant against the real
// HTTP middleware chain — the only layer above Casbin that can drift
// silently.
//
// Adding a new row when adding a permission-gated endpoint is the
// social contract this file is asking for.
func TestRoleAuthorization_NegativeMatrix(t *testing.T) {
	cleanupDatabase()
	r, _ := setupFullProductionRouter(t)
	org := createOrg(t, "Kita Sonnenschein")

	// Provision one user per role, each in the same org. The
	// cross-org isolation property is checked separately below.
	tokens := map[models.Role]string{
		models.RoleStaff:   provisionRoleUser(t, r, models.RoleStaff, org.ID),
		models.RoleMember:  provisionRoleUser(t, r, models.RoleMember, org.ID),
		models.RoleManager: provisionRoleUser(t, r, models.RoleManager, org.ID),
		models.RoleAdmin:   provisionRoleUser(t, r, models.RoleAdmin, org.ID),
	}

	// The matrix. POST/PUT/DELETE bodies are deliberately empty —
	// 403 from the authz middleware fires before body validation, so
	// the test does not need a syntactically-valid payload.
	cases := []roleCase{
		// --- staff ---------------------------------------------------------
		// Staff has attendance CRUD + read on children/sections, nothing
		// else. The headline negatives are "must not see HR data" and
		// "must not touch settings".
		{"staff cannot list employees", models.RoleStaff, "GET", "/api/v1/organizations/%d/employees", http.StatusForbidden},
		{"staff cannot create employees", models.RoleStaff, "POST", "/api/v1/organizations/%d/employees", http.StatusForbidden},
		{"staff cannot create children", models.RoleStaff, "POST", "/api/v1/organizations/%d/children", http.StatusForbidden},
		{"staff cannot create sections", models.RoleStaff, "POST", "/api/v1/organizations/%d/sections", http.StatusForbidden},
		{"staff cannot create pay plans", models.RoleStaff, "POST", "/api/v1/organizations/%d/pay-plans", http.StatusForbidden},
		{"staff cannot list budget items", models.RoleStaff, "GET", "/api/v1/organizations/%d/budget-items", http.StatusForbidden},
		{"staff cannot read audit log", models.RoleStaff, "GET", "/api/v1/organizations/%d/audit-logs", http.StatusForbidden},

		// --- member --------------------------------------------------------
		// Member is a read-only observer. The "member cannot edit pay
		// plans" line is the one the user explicitly asked us to pin;
		// the others extend the same invariant to every other resource
		// they have read access to.
		{"member cannot create pay plans", models.RoleMember, "POST", "/api/v1/organizations/%d/pay-plans", http.StatusForbidden},
		{"member cannot create employees", models.RoleMember, "POST", "/api/v1/organizations/%d/employees", http.StatusForbidden},
		{"member cannot create children", models.RoleMember, "POST", "/api/v1/organizations/%d/children", http.StatusForbidden},
		{"member cannot create sections", models.RoleMember, "POST", "/api/v1/organizations/%d/sections", http.StatusForbidden},
		{"member cannot create budget items", models.RoleMember, "POST", "/api/v1/organizations/%d/budget-items", http.StatusForbidden},
		{"member cannot create attendance", models.RoleMember, "POST", "/api/v1/organizations/%d/children/1/attendance", http.StatusForbidden},
		{"member cannot create funding bill", models.RoleMember, "POST", "/api/v1/organizations/%d/government-funding-bills", http.StatusForbidden},
		{"member cannot list users (global)", models.RoleMember, "GET", "/api/v1/users", http.StatusForbidden},
		{"member cannot read audit log", models.RoleMember, "GET", "/api/v1/organizations/%d/audit-logs", http.StatusForbidden},

		// --- manager -------------------------------------------------------
		// Manager runs operations + finance. Settings, audit log, and
		// org-update are admin-only. These rows pin every admin-vs-
		// manager difference rbac_test.go calls out, but at the HTTP
		// layer where a missing middleware actually shows up.
		{"manager cannot create sections", models.RoleManager, "POST", "/api/v1/organizations/%d/sections", http.StatusForbidden},
		{"manager cannot create pay plans", models.RoleManager, "POST", "/api/v1/organizations/%d/pay-plans", http.StatusForbidden},
		{"manager cannot create users (global)", models.RoleManager, "POST", "/api/v1/users", http.StatusForbidden},
		{"manager cannot update org", models.RoleManager, "PUT", "/api/v1/organizations/%d", http.StatusForbidden},
		{"manager cannot read audit log", models.RoleManager, "GET", "/api/v1/organizations/%d/audit-logs", http.StatusForbidden},

		// --- admin ---------------------------------------------------------
		// Admin runs the org but cannot modify the platform: only
		// superadmin creates/deletes orgs and edits global funding
		// rates. These three rows are the regression guard against
		// "someone widened RequirePermission to admin where it should
		// have stayed RequireSuperAdmin".
		{"admin cannot create org (global)", models.RoleAdmin, "POST", "/api/v1/organizations", http.StatusForbidden},
		{"admin cannot delete org", models.RoleAdmin, "DELETE", "/api/v1/organizations/%d", http.StatusForbidden},
		{"admin cannot create funding rate", models.RoleAdmin, "POST", "/api/v1/government-funding-rates", http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.path
			if hasPathPlaceholder(path) {
				path = fmt.Sprintf(path, org.ID)
			}
			w := doAuthed(t, r, tc.method, path, tokens[tc.role], nil)
			if w.Code != tc.wantStatus {
				t.Errorf("[%s] %s %s: got %d, want %d. body=%s",
					tc.role, tc.method, path, w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

// TestRoleAuthorization_PositiveBaseline confirms each role token
// actually works on at least one allowed path. Without this, all the
// 403 assertions above could pass for the wrong reason — for example,
// if provisionRoleUser silently returned an empty token, every
// `doAuthed` call would 401 (read by the test as "not 403, fails")…
// or worse, depending on a future change, "401 ≠ 403 but maybe still
// counted as denied". The positive baseline catches that drift.
func TestRoleAuthorization_PositiveBaseline(t *testing.T) {
	cleanupDatabase()
	r, _ := setupFullProductionRouter(t)
	org := createOrg(t, "Kita Sonnenschein")

	tokens := map[models.Role]string{
		models.RoleStaff:   provisionRoleUser(t, r, models.RoleStaff, org.ID),
		models.RoleMember:  provisionRoleUser(t, r, models.RoleMember, org.ID),
		models.RoleManager: provisionRoleUser(t, r, models.RoleManager, org.ID),
		models.RoleAdmin:   provisionRoleUser(t, r, models.RoleAdmin, org.ID),
	}

	// One allowed read per role. GETs avoid CSRF + body-validation
	// concerns so they isolate the authz path cleanly.
	cases := []roleCase{
		{"staff can list children", models.RoleStaff, "GET", "/api/v1/organizations/%d/children", http.StatusOK},
		{"member can list employees", models.RoleMember, "GET", "/api/v1/organizations/%d/employees", http.StatusOK},
		{"manager can list budget items", models.RoleManager, "GET", "/api/v1/organizations/%d/budget-items", http.StatusOK},
		{"admin can read audit log", models.RoleAdmin, "GET", "/api/v1/organizations/%d/audit-logs", http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := fmt.Sprintf(tc.path, org.ID)
			w := doAuthed(t, r, tc.method, path, tokens[tc.role], nil)
			if w.Code != tc.wantStatus {
				t.Errorf("[%s] %s %s: got %d, want %d. body=%s",
					tc.role, tc.method, path, w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

// TestRoleAuthorization_CrossOrgIsolation is the multi-tenant guard.
// A user with full permissions in org A must not be able to use those
// permissions in org B. The Casbin policy enforces this via the
// domain check, but a route that pulls the orgID from the wrong place
// (e.g. a body field instead of the URL param the middleware reads)
// would slip past every other test in this file. This pins the
// invariant at the HTTP layer.
func TestRoleAuthorization_CrossOrgIsolation(t *testing.T) {
	cleanupDatabase()
	r, _ := setupFullProductionRouter(t)
	orgA := createOrg(t, "Kita Alpha")
	orgB := createOrg(t, "Kita Beta")

	// User is admin in orgA only.
	adminInA := provisionRoleUser(t, r, models.RoleAdmin, orgA.ID)

	cases := []roleCase{
		{"admin-in-A cannot read orgB", models.RoleAdmin, "GET", fmt.Sprintf("/api/v1/organizations/%d", orgB.ID), http.StatusForbidden},
		{"admin-in-A cannot list orgB employees", models.RoleAdmin, "GET", fmt.Sprintf("/api/v1/organizations/%d/employees", orgB.ID), http.StatusForbidden},
		{"admin-in-A cannot create orgB employee", models.RoleAdmin, "POST", fmt.Sprintf("/api/v1/organizations/%d/employees", orgB.ID), http.StatusForbidden},
		{"admin-in-A cannot read orgB audit log", models.RoleAdmin, "GET", fmt.Sprintf("/api/v1/organizations/%d/audit-logs", orgB.ID), http.StatusForbidden},
		{"admin-in-A cannot update orgB", models.RoleAdmin, "PUT", fmt.Sprintf("/api/v1/organizations/%d", orgB.ID), http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doAuthed(t, r, tc.method, tc.path, adminInA, nil)
			if w.Code != tc.wantStatus {
				t.Errorf("%s %s: got %d, want %d. body=%s",
					tc.method, tc.path, w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

// hasPathPlaceholder reports whether the path template contains a
// `%d` slot that needs the org id substituted in. Plain paths
// (already-formatted, or global routes like /api/v1/users) skip the
// Sprintf to avoid "%d" appearing literally in a URL.
func hasPathPlaceholder(s string) bool {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '%' && s[i+1] == 'd' {
			return true
		}
	}
	return false
}
