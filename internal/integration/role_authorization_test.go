//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"strings"
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
		// Voucher writes (POST + DELETE) are admin/manager only — they're
		// gated on ResourceGovernmentFundingBills, the same resource the
		// rest of the funding-bill matrix uses. Read is intentionally
		// broader (Children.Read), tested separately in PositiveBaseline.
		{"member cannot assign voucher", models.RoleMember, "POST", "/api/v1/organizations/%d/children/1/vouchers", http.StatusForbidden},
		{"member cannot remove voucher", models.RoleMember, "DELETE", "/api/v1/organizations/%d/children/1/vouchers/1", http.StatusForbidden},
		{"staff cannot assign voucher", models.RoleStaff, "POST", "/api/v1/organizations/%d/children/1/vouchers", http.StatusForbidden},
		{"staff cannot remove voucher", models.RoleStaff, "DELETE", "/api/v1/organizations/%d/children/1/vouchers/1", http.StatusForbidden},
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

		// Voucher reads must be reachable by every role with access to
		// the child (member + staff included) — this is the whole reason
		// the GET endpoint uses Children.Read, not GovernmentFundingBills.
		// Without this, the new VouchersDialog would 403 for member/staff
		// on click. The path includes a non-existent child id; we accept
		// any non-403 response — what matters is that authz didn't deny.
		{"staff can read child vouchers", models.RoleStaff, "GET", "/api/v1/organizations/%d/children/1/vouchers", http.StatusNotFound},
		{"member can read child vouchers", models.RoleMember, "GET", "/api/v1/organizations/%d/children/1/vouchers", http.StatusNotFound},
		{"manager can read child vouchers", models.RoleManager, "GET", "/api/v1/organizations/%d/children/1/vouchers", http.StatusNotFound},
		{"admin can read child vouchers", models.RoleAdmin, "GET", "/api/v1/organizations/%d/children/1/vouchers", http.StatusNotFound},

		// /me/memberships must be reachable by EVERY authenticated user
		// regardless of role. The frontend's auth-store loadUser() calls
		// it on every page load to populate orgRoleMap, which the
		// sidebar's role-based nav filtering depends on. A 403 here
		// silently breaks the dashboard for staff and member users
		// (currentRole resolves to null, no nav items render). We had
		// exactly that bug shipped — the admin-facing
		// /users/{userId}/memberships requires users:read which staff
		// and member don't have. This row pins the "/me/memberships
		// must NOT be permission-gated" invariant.
		{"staff can read own memberships", models.RoleStaff, "GET", "/api/v1/me/memberships", http.StatusOK},
		{"member can read own memberships", models.RoleMember, "GET", "/api/v1/me/memberships", http.StatusOK},
		{"manager can read own memberships", models.RoleManager, "GET", "/api/v1/me/memberships", http.StatusOK},
		{"admin can read own memberships", models.RoleAdmin, "GET", "/api/v1/me/memberships", http.StatusOK},
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

// TestMyMemberships_Edges covers the failure surface around the
// self-memberships route specifically. The positive matrix above
// confirms each role gets a 200 with their data; this file pins the
// behaviour at the rough edges:
//
//  1. No auth at all → 401 (the endpoint must NOT silently leak the
//     calling user's identity to anonymous callers).
//  2. Authenticated but with zero org memberships → 200 + empty list.
//     Not 404, not 500. The frontend renders the empty case as "no
//     orgs assigned" — a 4xx/5xx here would brick the dashboard.
//  3. Authenticated with multiple org memberships → 200 + every
//     membership returned. The auth-store rebuilds orgRoleMap from
//     this list; missing entries would silently strip nav items.
//  4. The response contains ONLY the caller's data, never another
//     user's. There is no userId path param to abuse, but a future
//     refactor that mistakenly reads the param from a query string
//     or body would need to be caught somewhere — here.
func TestMyMemberships_Edges(t *testing.T) {
	cleanupDatabase()
	r, _ := setupFullProductionRouter(t)
	orgA := createOrg(t, "Kita Alpha")
	orgB := createOrg(t, "Kita Beta")

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		// No Authorization header, no session cookie. The auth
		// middleware must reject this before the handler runs.
		w := doAuthed(t, r, "GET", "/api/v1/me/memberships", "", nil)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("got %d, want 401. body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid bearer token returns 401", func(t *testing.T) {
		// A syntactically-shaped but unknown token. The auth
		// middleware should treat this the same as no token —
		// 401, not 200, not 500.
		w := doAuthed(t, r, "GET", "/api/v1/me/memberships", "not-a-real-session-token", nil)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("got %d, want 401. body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("user with zero memberships returns empty list", func(t *testing.T) {
		// Provision a user but do NOT assign any role in any org.
		// The frontend's auth-store handles this case by rendering
		// only the global nav, but it expects 200 + {memberships: []}.
		// A 404 or 500 would leave orgRoleMap permanently empty in a
		// way that looks indistinguishable from "fetch failed".
		const email = "no-memberships@role-auth.test"
		const password = "no-memberships-pw-1234"
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		u := &models.User{Name: "Lonely User", Email: email, Password: string(hash), Active: true}
		if err := testDB.Create(u).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
		_, token := doLogin(t, r, email, password)

		w := doAuthed(t, r, "GET", "/api/v1/me/memberships", token, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("got %d, want 200. body=%s", w.Code, w.Body.String())
		}
		// Body shape: {"memberships": []}. The frontend buildOrgRoleMap
		// helper expects the array to be present even when empty.
		body := w.Body.String()
		if !strings.Contains(body, `"memberships"`) {
			t.Errorf("response missing 'memberships' field: %s", body)
		}
	})

	t.Run("user with multiple memberships returns all", func(t *testing.T) {
		// Roles deliberately chosen to differ across orgs so a sloppy
		// implementation that returns the first match (or that loses
		// the role enum) would visibly fail the assertion.
		const email = "multi-org@role-auth.test"
		const password = "multi-org-pw-1234"
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		u := &models.User{Name: "Multi User", Email: email, Password: string(hash), Active: true}
		if err := testDB.Create(u).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
		// admin in A, manager in B.
		for _, m := range []models.UserOrganization{
			{UserID: u.ID, OrganizationID: orgA.ID, Role: models.RoleAdmin},
			{UserID: u.ID, OrganizationID: orgB.ID, Role: models.RoleManager},
		} {
			if err := testDB.Create(&m).Error; err != nil {
				t.Fatalf("create membership: %v", err)
			}
		}
		_, token := doLogin(t, r, email, password)

		w := doAuthed(t, r, "GET", "/api/v1/me/memberships", token, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("got %d, want 200. body=%s", w.Code, w.Body.String())
		}
		body := w.Body.String()
		if !strings.Contains(body, fmt.Sprintf(`"organization_id":%d`, orgA.ID)) {
			t.Errorf("orgA membership missing from response: %s", body)
		}
		if !strings.Contains(body, fmt.Sprintf(`"organization_id":%d`, orgB.ID)) {
			t.Errorf("orgB membership missing from response: %s", body)
		}
		if !strings.Contains(body, `"role":"admin"`) {
			t.Errorf("admin role missing from response: %s", body)
		}
		if !strings.Contains(body, `"role":"manager"`) {
			t.Errorf("manager role missing from response: %s", body)
		}
	})

	t.Run("response is scoped to the calling user only", func(t *testing.T) {
		// Two users, distinct memberships. User one calls the endpoint
		// and must see ONLY their own row, not user two's. The route
		// has no userId path param so cross-user is structurally
		// impossible today — this test pins it so it stays that way.
		const userOneEmail = "scope-one@role-auth.test"
		const userTwoEmail = "scope-two@role-auth.test"
		const password = "scope-pw-1234"
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		userOne := &models.User{Name: "Scope One", Email: userOneEmail, Password: string(hash), Active: true}
		userTwo := &models.User{Name: "Scope Two", Email: userTwoEmail, Password: string(hash), Active: true}
		if err := testDB.Create(userOne).Error; err != nil {
			t.Fatalf("create userOne: %v", err)
		}
		if err := testDB.Create(userTwo).Error; err != nil {
			t.Fatalf("create userTwo: %v", err)
		}
		// userOne is admin in A; userTwo is staff in B. They share no
		// org membership.
		if err := testDB.Create(&models.UserOrganization{UserID: userOne.ID, OrganizationID: orgA.ID, Role: models.RoleAdmin}).Error; err != nil {
			t.Fatalf("create userOne membership: %v", err)
		}
		if err := testDB.Create(&models.UserOrganization{UserID: userTwo.ID, OrganizationID: orgB.ID, Role: models.RoleStaff}).Error; err != nil {
			t.Fatalf("create userTwo membership: %v", err)
		}
		_, oneToken := doLogin(t, r, userOneEmail, password)

		w := doAuthed(t, r, "GET", "/api/v1/me/memberships", oneToken, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("got %d, want 200. body=%s", w.Code, w.Body.String())
		}
		body := w.Body.String()
		// userOne sees their orgA admin row.
		if !strings.Contains(body, fmt.Sprintf(`"organization_id":%d`, orgA.ID)) {
			t.Errorf("userOne should see orgA in their own response: %s", body)
		}
		// userOne must NOT see orgB at all — that's userTwo's org.
		// If a future bug returns "all memberships in any org the
		// caller can see" the orgB id would leak in. Pin it.
		if strings.Contains(body, fmt.Sprintf(`"organization_id":%d`, orgB.ID)) {
			t.Errorf("userOne response leaked orgB (userTwo's org): %s", body)
		}
		if strings.Contains(body, `"role":"staff"`) {
			t.Errorf("userOne response leaked userTwo's staff role: %s", body)
		}
	})
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
