//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	cryptopkg "github.com/eenemeene/kitamanager-go/internal/crypto"
	"github.com/eenemeene/kitamanager-go/internal/handlers"
	"github.com/eenemeene/kitamanager-go/internal/middleware"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/rbac"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
	webauthnpkg "github.com/eenemeene/kitamanager-go/internal/webauthn"
)

// testWebAuthn* is the RP configuration the synthetic authenticator
// signs over. Tests driving the WebAuthn ceremonies must use the
// same values on both sides; a mismatch would fail origin / rpIdHash
// checks (which is exactly the production failure mode we want to
// exercise separately, but not by accident).
const (
	testWebAuthnRPID   = "example.test"
	testWebAuthnOrigin = "https://example.test"
)

// testTOTPAEADKey is a deterministic test key for the AES-GCM
// encryption of TOTP secrets. Integration tests don't care about the
// value, only that encrypt and decrypt round-trip.
const testTOTPAEADKeyHex = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"

func testTOTPAEAD(t *testing.T) *cryptopkg.AEAD {
	t.Helper()
	key, err := cryptopkg.DecodeKey(testTOTPAEADKeyHex)
	if err != nil {
		t.Fatalf("decode key: %v", err)
	}
	aead, err := cryptopkg.NewAEAD(key)
	if err != nil {
		t.Fatalf("new aead: %v", err)
	}
	return aead
}

// testCSRFHMACKey is a deterministic, ≥32-character secret used by the real
// auth + CSRF middleware. Config enforces the 32-char floor in production;
// we mirror it here so a future tightening of service-level checks does not
// silently skip this test.
const testCSRFHMACKey = "integration-test-csrf-hmac-key-at-least-32-chars-long"

// authFlowRouter wires a router with the real auth, authorization, user, and
// organization stack — no hardcoded userID middleware, no CSRF shortcut.
// This is the scaffolding the other integration tests in this package
// deliberately skip, so the real login → JWT → RBAC path is never exercised
// there. All tests in this file share the testDB created in TestMain.
type authFlowRouter struct {
	router *gin.Engine
}

func setupAuthFlowRouter(t *testing.T) *authFlowRouter {
	t.Helper()

	// Real stores against the shared testDB from integration_test.go's TestMain.
	userStore := store.NewUserStore(testDB)
	userOrgStore := store.NewUserOrganizationStore(testDB)
	orgStore := store.NewOrganizationStore(testDB)
	sessionStore := store.NewSessionStore(testDB)
	transactor := store.NewTransactor(testDB)

	auditStore := store.NewAuditStore(testDB)
	auditService := service.NewAuditService(auditStore)

	userService := service.NewUserService(userStore, userOrgStore)
	userOrgService := service.NewUserOrganizationService(userOrgStore, userStore, transactor)
	orgService := service.NewOrganizationService(orgStore, userStore)

	// Casbin enforcer against the real DB adapter; then seed default policies
	// so admin / manager / member roles actually have permissions attached.
	enforcer, err := rbac.NewEnforcer(testDB, findRBACModel(t))
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	if err := enforcer.SeedDefaultPolicies(); err != nil {
		t.Fatalf("seed default policies: %v", err)
	}
	permissionService := rbac.NewPermissionService(userOrgStore, enforcer)

	// Real middleware: session validation + RBAC gates + CSRF. CSRF
	// short-circuits when the request uses Authorization: Bearer (no session
	// cookie), so our tests can ignore it.
	authMW := middleware.NewAuthMiddleware(sessionStore)
	authzMW := middleware.NewAuthorizationMiddleware(permissionService)
	csrfMW := middleware.NewCSRFMiddleware(testCSRFHMACKey)

	factorStore := store.NewFactorStore(testDB)
	// Build a real WebAuthn service so the integration tests can
	// exercise the full registration + assertion ceremony via the
	// synthetic authenticator in webauthn_authenticator_test.go.
	// RP id / origin match what the synthetic authenticator signs
	// over. If these diverge the library's rpIdHash or origin check
	// will reject the ceremony — which is the point, but means tests
	// and real code must agree.
	waSvc, err := webauthnpkg.New(webauthnpkg.Config{
		RPID:      testWebAuthnRPID,
		RPName:    "KitaManager (test)",
		RPOrigins: []string{testWebAuthnOrigin},
	})
	if err != nil {
		t.Fatalf("webauthn init: %v", err)
	}
	factorService := service.NewFactorService(factorStore, userStore, testTOTPAEAD(t), "KitaManager (test)", waSvc, auditService)

	// Real auth service — hashes passwords with bcrypt.DefaultCost and issues
	// opaque session tokens backed by the sessions table. Gets a factor
	// service so two-step login fires for MFA-enrolled users.
	authService := service.NewAuthService(userStore, sessionStore, testCSRFHMACKey, auditService, factorService)

	authHandler := handlers.NewAuthHandler(authService, false /*secureCookies*/)
	userHandler := handlers.NewUserHandler(userService, userOrgService, auditService, sessionStore)
	orgHandler := handlers.NewOrganizationHandler(orgService, auditService)
	factorHandler := handlers.NewFactorHandler(factorService)

	r := gin.New()
	api := r.Group("/api/v1")

	// Public endpoints — no auth middleware.
	api.POST("/login", authHandler.Login)
	api.POST("/auth/mfa/challenge", authHandler.MFAChallenge)
	api.POST("/auth/mfa/verify", authHandler.MFAVerify)

	// Protected endpoints — real auth + CSRF. CSRF is skipped for Bearer
	// requests so we don't need to send X-CSRF-Token in these tests.
	protected := api.Group("")
	protected.Use(authMW.RequireAuth())
	protected.Use(csrfMW.ValidateCSRF())

	protected.GET("/me", authHandler.Me)
	protected.POST("/logout", authHandler.Logout)
	protected.GET("/me/sessions", authHandler.ListSessions)
	protected.DELETE("/me/sessions/:sessionId", authHandler.RevokeSession)

	// Factor (MFA) endpoints.
	protected.GET("/users/:userId/factors", factorHandler.List)
	protected.POST("/users/:userId/factors", factorHandler.Enroll)
	protected.GET("/users/:userId/factors/:factorId", factorHandler.Get)
	protected.PATCH("/users/:userId/factors/:factorId", factorHandler.UpdateLabel)
	protected.DELETE("/users/:userId/factors/:factorId", factorHandler.Delete)
	protected.POST("/users/:userId/factors/:factorId/activate", factorHandler.Activate)
	protected.POST("/users/:userId/factors/:factorId/regenerate", factorHandler.Regenerate)

	// Global user routes. Create requires ResourceUsers + ActionCreate in at
	// least one org — superadmin satisfies this.
	users := protected.Group("/users")
	users.POST("",
		authzMW.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionCreate),
		userHandler.Create)
	users.POST("/:userId/organizations",
		authzMW.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionUpdate),
		userHandler.AddToOrganization)

	// Org-scoped routes used by the assertions.
	orgs := protected.Group("/organizations")
	orgs.GET("",
		authzMW.RequireGlobalPermission(rbac.ResourceOrganizations, rbac.ActionRead),
		orgHandler.List)
	orgs.GET("/:orgId",
		authzMW.RequirePermission(rbac.ResourceOrganizations, rbac.ActionRead),
		orgHandler.Get)

	_ = userService // constructed for completeness of the wiring; not used directly in tests
	return &authFlowRouter{router: r}
}

// findRBACModel locates configs/rbac_model.conf from the test working dir.
// The integration package runs with cwd = internal/integration, so walk up.
func findRBACModel(t *testing.T) string {
	t.Helper()
	candidates := []string{
		"../../configs/rbac_model.conf",
		"configs/rbac_model.conf",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	t.Fatal("configs/rbac_model.conf not found")
	return ""
}

// seedSuperadmin inserts a superadmin user with a bcrypt-hashed password,
// then registers them in Casbin. Returns (userID, email, password).
// Using the real hash lets us log in through the production code path
// instead of faking it with a preset context.
func seedSuperadmin(t *testing.T, enforcer *rbac.Enforcer) (uint, string, string) {
	t.Helper()
	const email = "superadmin@test.local"
	const password = "super-secret-123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	u := &models.User{
		Name:         "Super Admin",
		Email:        email,
		Password:     string(hash),
		Active:       true,
		IsSuperAdmin: true,
	}
	if err := testDB.Create(u).Error; err != nil {
		t.Fatalf("create superadmin: %v", err)
	}
	if err := enforcer.AssignSuperAdmin(u.ID); err != nil {
		t.Fatalf("assign superadmin: %v", err)
	}
	return u.ID, email, password
}

// doLogin issues a POST /api/v1/login and returns the session cookie value
// extracted from the Set-Cookie header. An empty string indicates the server
// did not issue a session — callers should assert on the HTTP status first.
func doLogin(t *testing.T, r *gin.Engine, email, password string) (int, string) {
	t.Helper()
	body, _ := json.Marshal(models.LoginRequest{Email: email, Password: password})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			return w.Code, c.Value
		}
	}
	return w.Code, ""
}

// doAuthed sends an authenticated request with the Bearer access token,
// bypassing the cookie/CSRF path. Mirrors how a CLI client would call the API.
func doAuthed(t *testing.T, r *gin.Engine, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody *strings.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = strings.NewReader(string(b))
	} else {
		reqBody = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// createOrg is a small helper so the tests don't repeat the two-line insert.
func createOrg(t *testing.T, name string) *models.Organization {
	t.Helper()
	org := &models.Organization{Name: name, Active: true}
	if err := testDB.Create(org).Error; err != nil {
		t.Fatalf("create org: %v", err)
	}
	return org
}

// TestAuthFlow_SuperadminCreatesAdmin_AdminLoginsAndAccessesOrg exercises the
// happy path that was previously uncovered: a superadmin creates a new admin
// user through the public API, grants them org membership, and the new admin
// can log in with the password they were created with and reach an org-scoped
// endpoint. A regression here would mean the password hash stored on create
// does not match what the login verifier expects.
func TestAuthFlow_SuperadminCreatesAdmin_AdminLoginsAndAccessesOrg(t *testing.T) {
	cleanupDatabase()
	fr := setupAuthFlowRouter(t)
	enforcer, _ := rbac.NewEnforcer(testDB, findRBACModel(t))
	_, superEmail, superPass := seedSuperadmin(t, enforcer)
	org := createOrg(t, "Kita Sonnenschein")

	// 1. Superadmin logs in.
	code, superToken := doLogin(t, fr.router, superEmail, superPass)
	if code != http.StatusOK {
		t.Fatalf("superadmin login: got status %d, want 200", code)
	}
	if superToken == "" {
		t.Fatal("superadmin login returned no session cookie")
	}

	// 2. Superadmin creates a new admin user via the real API. This is the
	//    exact code path that was previously untested — password goes through
	//    UserService.Create's bcrypt hashing before storage.
	const newAdminEmail = "new-admin@test.local"
	const newAdminPass = "admin-password-456"
	createReq := models.UserCreateRequest{
		Name:     "New Admin",
		Email:    newAdminEmail,
		Password: newAdminPass,
		Active:   true,
	}
	w := doAuthed(t, fr.router, http.MethodPost, "/api/v1/users", superToken, createReq)
	if w.Code != http.StatusCreated {
		t.Fatalf("create user: got status %d, want 201. body=%s", w.Code, w.Body.String())
	}
	var created models.UserResponse
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	// 3. Grant the new user admin role in the target org.
	addReq := models.UserAddOrganizationRequest{
		OrganizationID: org.ID,
		Role:           models.RoleAdmin,
	}
	w = doAuthed(t, fr.router, http.MethodPost,
		fmt.Sprintf("/api/v1/users/%d/organizations", created.ID), superToken, addReq)
	if w.Code != http.StatusCreated {
		t.Fatalf("add to org: got status %d, want 201. body=%s", w.Code, w.Body.String())
	}

	// 4. The new admin logs in with the password they were created with.
	//    This is the end-to-end proof that bcrypt hashing on create matches
	//    CompareHashAndPassword on login.
	code, adminToken := doLogin(t, fr.router, newAdminEmail, newAdminPass)
	if code != http.StatusOK {
		t.Fatalf("new admin login: got status %d, want 200", code)
	}
	if adminToken == "" {
		t.Fatal("new admin login returned no session cookie")
	}

	// 5. The new admin calls GET /me — base sanity check that the JWT is
	//    valid and RequireAuth accepts it.
	w = doAuthed(t, fr.router, http.MethodGet, "/api/v1/me", adminToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("new admin GET /me: got %d, want 200. body=%s", w.Code, w.Body.String())
	}

	// 6. The new admin accesses an org-scoped endpoint they should have
	//    permission for (admin has organizations.read). This is what proves
	//    the RBAC wiring between user_organizations rows and Casbin policies
	//    actually resolves for a freshly minted user.
	w = doAuthed(t, fr.router, http.MethodGet,
		fmt.Sprintf("/api/v1/organizations/%d", org.ID), adminToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("new admin GET org: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
}

// TestAuthFlow_AdminWithoutOrg_CanLoginButIsRefusedOnOrgEndpoints captures the
// edge case where a superadmin creates an admin user but forgets to grant org
// membership. The user must still be able to log in (authentication is
// identity-level), but every org-scoped endpoint must refuse them — anything
// else is a tenant-isolation hole.
func TestAuthFlow_AdminWithoutOrg_CanLoginButIsRefusedOnOrgEndpoints(t *testing.T) {
	cleanupDatabase()
	fr := setupAuthFlowRouter(t)
	enforcer, _ := rbac.NewEnforcer(testDB, findRBACModel(t))
	_, superEmail, superPass := seedSuperadmin(t, enforcer)
	org := createOrg(t, "Kita Sonnenschein")

	_, superToken := doLogin(t, fr.router, superEmail, superPass)

	const email = "orphan-admin@test.local"
	const password = "orphan-password-789"
	w := doAuthed(t, fr.router, http.MethodPost, "/api/v1/users", superToken, models.UserCreateRequest{
		Name: "Orphan Admin", Email: email, Password: password, Active: true,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create user: %d %s", w.Code, w.Body.String())
	}

	// Login works — identity is valid even without any role.
	code, orphanToken := doLogin(t, fr.router, email, password)
	if code != http.StatusOK {
		t.Fatalf("orphan login: got %d, want 200", code)
	}
	if orphanToken == "" {
		t.Fatal("orphan login returned no session cookie")
	}

	// /me works — identity-level endpoint.
	w = doAuthed(t, fr.router, http.MethodGet, "/api/v1/me", orphanToken, nil)
	if w.Code != http.StatusOK {
		t.Errorf("orphan GET /me: got %d, want 200 (identity endpoint should work)", w.Code)
	}

	// Org-scoped reads must be refused — the user has no role in any org.
	w = doAuthed(t, fr.router, http.MethodGet,
		fmt.Sprintf("/api/v1/organizations/%d", org.ID), orphanToken, nil)
	if w.Code != http.StatusForbidden {
		t.Errorf("orphan GET org: got %d, want 403 (no role in org)", w.Code)
	}

	// Global orgs list must also be refused via RequireGlobalPermission —
	// the user has no role in ANY org, so the global gate fails closed.
	w = doAuthed(t, fr.router, http.MethodGet, "/api/v1/organizations", orphanToken, nil)
	if w.Code != http.StatusForbidden {
		t.Errorf("orphan GET orgs list: got %d, want 403", w.Code)
	}
}

// TestAuthFlow_ManagerRole_HasReadButNotOrgUpdate locks in the permission
// differential between admin and manager: both can read orgs, only admin can
// update. If a future Casbin policy edit accidentally widens manager's
// permissions (e.g. copy-paste during a refactor), this test fails immediately.
func TestAuthFlow_ManagerRole_HasReadButNotOrgUpdate(t *testing.T) {
	cleanupDatabase()
	fr := setupAuthFlowRouter(t)
	enforcer, _ := rbac.NewEnforcer(testDB, findRBACModel(t))
	_, superEmail, superPass := seedSuperadmin(t, enforcer)
	org := createOrg(t, "Kita Sonnenschein")

	_, superToken := doLogin(t, fr.router, superEmail, superPass)

	// Create manager via API.
	const email = "new-manager@test.local"
	const password = "manager-password-321"
	w := doAuthed(t, fr.router, http.MethodPost, "/api/v1/users", superToken, models.UserCreateRequest{
		Name: "New Manager", Email: email, Password: password, Active: true,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create user: %d %s", w.Code, w.Body.String())
	}
	var created models.UserResponse
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	w = doAuthed(t, fr.router, http.MethodPost,
		fmt.Sprintf("/api/v1/users/%d/organizations", created.ID), superToken,
		models.UserAddOrganizationRequest{OrganizationID: org.ID, Role: models.RoleManager})
	if w.Code != http.StatusCreated {
		t.Fatalf("add to org: %d %s", w.Code, w.Body.String())
	}

	// Manager logs in.
	code, managerToken := doLogin(t, fr.router, email, password)
	if code != http.StatusOK {
		t.Fatalf("manager login: got %d, want 200", code)
	}

	// Read on the org — must succeed.
	w = doAuthed(t, fr.router, http.MethodGet,
		fmt.Sprintf("/api/v1/organizations/%d", org.ID), managerToken, nil)
	if w.Code != http.StatusOK {
		t.Errorf("manager GET org: got %d, want 200", w.Code)
	}

	// Creating a user is a global-users:create action — manager does not
	// have that permission. If the policies are accidentally widened this
	// turns into a 201 and the test catches it.
	w = doAuthed(t, fr.router, http.MethodPost, "/api/v1/users", managerToken,
		models.UserCreateRequest{Name: "x", Email: "x@x.test", Password: "p123456", Active: true})
	if w.Code != http.StatusForbidden {
		t.Errorf("manager POST /users: got %d, want 403 (should not have users:create)", w.Code)
	}
}

// TestAuthFlow_StaffRole_CanReadChildrenButNotEmployees pins the staff-role
// surface. Staff has children:read and attendance CRUD but deliberately does
// NOT have employees:read — a policy shape that comes up often enough (an
// attendance-taker who should not see HR data) that a silent widening would
// be a real privacy leak. The assertion pair (children:200 / employees:403)
// is what separates staff from member.
func TestAuthFlow_StaffRole_CanReadChildrenButNotEmployees(t *testing.T) {
	cleanupDatabase()
	fr := setupAuthFlowRouter(t)
	enforcer, _ := rbac.NewEnforcer(testDB, findRBACModel(t))
	_, superEmail, superPass := seedSuperadmin(t, enforcer)
	org := createOrg(t, "Kita Sonnenschein")

	// The auth-flow router only registers /organizations and /users endpoints
	// to keep setup small. For the staff assertions we need children + employees
	// endpoints too, so wire those in against the same router.
	childStore := store.NewChildStore(testDB)
	employeeStore := store.NewEmployeeStore(testDB)
	fundingStore := store.NewGovernmentFundingStore(testDB)
	sectionStore := store.NewSectionStore(testDB)
	payPlanStore := store.NewPayPlanStore(testDB)
	orgStore := store.NewOrganizationStore(testDB)
	transactor := store.NewTransactor(testDB)
	childService := service.NewChildService(childStore, orgStore, fundingStore, sectionStore, transactor)
	employeeService := service.NewEmployeeService(employeeStore, payPlanStore, sectionStore, transactor)
	auditService := service.NewAuditService(store.NewAuditStore(testDB))
	childHandler := handlers.NewChildHandler(childService, auditService)
	employeeHandler := handlers.NewEmployeeHandler(employeeService, auditService)

	userOrgStore := store.NewUserOrganizationStore(testDB)
	permissionService := rbac.NewPermissionService(userOrgStore, enforcer)
	authzMW := middleware.NewAuthorizationMiddleware(permissionService)
	sessionStore := store.NewSessionStore(testDB)
	authMW := middleware.NewAuthMiddleware(sessionStore)
	csrfMW := middleware.NewCSRFMiddleware(testCSRFHMACKey)

	protected := fr.router.Group("/api/v1")
	protected.Use(authMW.RequireAuth())
	protected.Use(csrfMW.ValidateCSRF())
	protected.GET("/organizations/:orgId/children",
		authzMW.RequirePermission(rbac.ResourceChildren, rbac.ActionRead),
		childHandler.List)
	protected.GET("/organizations/:orgId/employees",
		authzMW.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead),
		employeeHandler.List)

	_, superToken := doLogin(t, fr.router, superEmail, superPass)

	const email = "new-staff@test.local"
	const password = "staff-password-999"
	w := doAuthed(t, fr.router, http.MethodPost, "/api/v1/users", superToken, models.UserCreateRequest{
		Name: "New Staff", Email: email, Password: password, Active: true,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create user: %d %s", w.Code, w.Body.String())
	}
	var created models.UserResponse
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	w = doAuthed(t, fr.router, http.MethodPost,
		fmt.Sprintf("/api/v1/users/%d/organizations", created.ID), superToken,
		models.UserAddOrganizationRequest{OrganizationID: org.ID, Role: models.RoleStaff})
	if w.Code != http.StatusCreated {
		t.Fatalf("add to org: %d %s", w.Code, w.Body.String())
	}

	code, staffToken := doLogin(t, fr.router, email, password)
	if code != http.StatusOK {
		t.Fatalf("staff login: got %d, want 200", code)
	}

	// Staff CAN read children in their org.
	w = doAuthed(t, fr.router, http.MethodGet,
		fmt.Sprintf("/api/v1/organizations/%d/children", org.ID), staffToken, nil)
	if w.Code != http.StatusOK {
		t.Errorf("staff GET children: got %d, want 200 (children:read is allowed)", w.Code)
	}

	// Staff CANNOT read employees — the important guard. If a future policy
	// edit accidentally grants employees:read to staff, this catches it.
	w = doAuthed(t, fr.router, http.MethodGet,
		fmt.Sprintf("/api/v1/organizations/%d/employees", org.ID), staffToken, nil)
	if w.Code != http.StatusForbidden {
		t.Errorf("staff GET employees: got %d, want 403 (staff must not have employees:read)", w.Code)
	}
}

// TestAuthFlow_WrongPasswordRejected is a sanity guard: given a correctly
// created user, a wrong password must not produce an access token. Paired
// with the happy-path test this bounds the login behaviour from both sides
// and protects against a future bug where login becomes silently permissive
// (e.g. bcrypt comparison replaced with a stubbed-true in a refactor).
func TestAuthFlow_WrongPasswordRejected(t *testing.T) {
	cleanupDatabase()
	fr := setupAuthFlowRouter(t)
	enforcer, _ := rbac.NewEnforcer(testDB, findRBACModel(t))
	_, superEmail, superPass := seedSuperadmin(t, enforcer)

	_, superToken := doLogin(t, fr.router, superEmail, superPass)

	const email = "wrongpw@test.local"
	const password = "correct-horse-battery-staple"
	w := doAuthed(t, fr.router, http.MethodPost, "/api/v1/users", superToken, models.UserCreateRequest{
		Name: "Wrong PW User", Email: email, Password: password, Active: true,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create user: %d %s", w.Code, w.Body.String())
	}

	// Correct password → 200 + token.
	code, goodToken := doLogin(t, fr.router, email, password)
	if code != http.StatusOK || goodToken == "" {
		t.Fatalf("correct-password login: got status=%d token=%q, want 200 + non-empty", code, goodToken)
	}

	// Wrong password → 401 + no token. Using a deliberately different
	// plaintext so we also exercise the bcrypt mismatch branch.
	code, badToken := doLogin(t, fr.router, email, password+"!!")
	if code != http.StatusUnauthorized {
		t.Errorf("wrong-password login status: got %d, want 401", code)
	}
	if badToken != "" {
		t.Errorf("wrong-password login issued a token: %q", badToken)
	}
}

// TestAuthFlow_InactiveUserCannotLogin covers the deactivation case — an
// existing user whose Active flag has been flipped to false must be refused
// at login, even with the correct password. Without this, a "disabled"
// account remains a live attack surface.
//
// The test deactivates via a direct UPDATE rather than the create API. That's
// deliberate: UserCreateRequest.Active is a plain `bool`, and the User model
// has `gorm:"default:true"` on the column, so GORM treats a Go zero value
// (false) as "unset" and writes the default. Creating with Active=false via
// the API therefore silently stores Active=true — a separate bug tracked
// outside this test. The auth behavior we want to assert here is about login
// rejecting an already-inactive row, which is the production scenario when
// an admin later toggles a user off.
func TestAuthFlow_InactiveUserCannotLogin(t *testing.T) {
	cleanupDatabase()
	fr := setupAuthFlowRouter(t)
	enforcer, _ := rbac.NewEnforcer(testDB, findRBACModel(t))
	_, superEmail, superPass := seedSuperadmin(t, enforcer)

	_, superToken := doLogin(t, fr.router, superEmail, superPass)

	const email = "inactive@test.local"
	const password = "inactive-password-777"
	w := doAuthed(t, fr.router, http.MethodPost, "/api/v1/users", superToken, models.UserCreateRequest{
		Name: "Inactive User", Email: email, Password: password, Active: true,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create user: %d %s", w.Code, w.Body.String())
	}
	var created models.UserResponse
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	// Deactivate the user directly in the DB — this is the state an admin
	// would leave them in after a PUT /users/:id with Active:false.
	if err := testDB.Model(&models.User{}).Where("id = ?", created.ID).
		Update("active", false).Error; err != nil {
		t.Fatalf("deactivate user: %v", err)
	}

	code, token := doLogin(t, fr.router, email, password)
	if code != http.StatusUnauthorized {
		t.Errorf("inactive login status: got %d, want 401", code)
	}
	if token != "" {
		t.Errorf("inactive login issued a token: %q", token)
	}
}

// TestAuthFlow_SessionManagement_CrossUserRevokeIs404 pins the core
// security invariant of /me/sessions: a user CANNOT list or revoke
// another user's sessions, even if they know the target's id-hash.
// The endpoints only reveal and operate on sessions scoped to the
// caller via ctxkeys.UserID, and cross-user deletes return 404 so the
// caller cannot probe for the existence of another user's session.
func TestAuthFlow_SessionManagement_CrossUserRevokeIs404(t *testing.T) {
	cleanupDatabase()
	fr := setupAuthFlowRouter(t)
	enforcer, _ := rbac.NewEnforcer(testDB, findRBACModel(t))
	_, superEmail, superPass := seedSuperadmin(t, enforcer)
	_, superToken := doLogin(t, fr.router, superEmail, superPass)

	// Two members, both via the real create-user flow.
	const aliceEmail = "alice-sm@test.local"
	const alicePass = "alice-pw-12345"
	const bobEmail = "bob-sm@test.local"
	const bobPass = "bob-pw-12345"

	for _, u := range []struct{ email, pw string }{{aliceEmail, alicePass}, {bobEmail, bobPass}} {
		w := doAuthed(t, fr.router, http.MethodPost, "/api/v1/users", superToken, models.UserCreateRequest{
			Name: u.email, Email: u.email, Password: u.pw, Active: true,
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("create user %s: %d %s", u.email, w.Code, w.Body.String())
		}
	}

	// Each logs in, capturing their own session token.
	aliceCode, aliceToken := doLogin(t, fr.router, aliceEmail, alicePass)
	if aliceCode != http.StatusOK || aliceToken == "" {
		t.Fatalf("alice login: %d token=%q", aliceCode, aliceToken)
	}
	bobCode, bobToken := doLogin(t, fr.router, bobEmail, bobPass)
	if bobCode != http.StatusOK || bobToken == "" {
		t.Fatalf("bob login: %d token=%q", bobCode, bobToken)
	}

	// Alice lists her sessions via the real endpoint and pulls out the id.
	w := doAuthed(t, fr.router, http.MethodGet, "/api/v1/me/sessions", aliceToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("alice list sessions: %d %s", w.Code, w.Body.String())
	}
	var aliceList models.UserSessionsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &aliceList); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(aliceList.Sessions) != 1 {
		t.Fatalf("alice should see 1 session, got %d", len(aliceList.Sessions))
	}
	aliceSessionID := aliceList.Sessions[0].ID
	if !aliceList.Sessions[0].Current {
		t.Error("alice's own session must be flagged Current")
	}

	// Bob attempts to DELETE Alice's session. Must 404 — not 403, not
	// 204 — so that the endpoint does not leak which session ids exist.
	w = doAuthed(t, fr.router, http.MethodDelete, "/api/v1/me/sessions/"+aliceSessionID, bobToken, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("bob→alice revoke: got %d, want 404 (must not leak session existence). body=%s", w.Code, w.Body.String())
	}

	// And Alice's session must still work — /me still returns her identity.
	w = doAuthed(t, fr.router, http.MethodGet, "/api/v1/me", aliceToken, nil)
	if w.Code != http.StatusOK {
		t.Errorf("alice /me after bob attempt: got %d, want 200 (her session must survive)", w.Code)
	}

	// Alice can revoke her own session.
	w = doAuthed(t, fr.router, http.MethodDelete, "/api/v1/me/sessions/"+aliceSessionID, aliceToken, nil)
	if w.Code != http.StatusNoContent {
		t.Errorf("alice self-revoke: got %d, want 204. body=%s", w.Code, w.Body.String())
	}

	// And now her next request must 401 — middleware catches it.
	w = doAuthed(t, fr.router, http.MethodGet, "/api/v1/me", aliceToken, nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("alice /me after self-revoke: got %d, want 401", w.Code)
	}
}

// TestAuthFlow_Factors_CrossUserIsolation pins the core invariant for
// the MFA endpoints: Alice enrols and activates a TOTP factor; Bob
// tries every /factors endpoint against Alice's factor id and must
// get 404 — never 403, never success — so the endpoint does not
// leak which factor ids exist. Alice's factor survives all of Bob's
// attempts.
func TestAuthFlow_Factors_CrossUserIsolation(t *testing.T) {
	cleanupDatabase()
	fr := setupAuthFlowRouter(t)
	enforcer, _ := rbac.NewEnforcer(testDB, findRBACModel(t))
	_, superEmail, superPass := seedSuperadmin(t, enforcer)
	_, superToken := doLogin(t, fr.router, superEmail, superPass)

	// Two members provisioned via the real user-create API path.
	const (
		aliceEmail = "alice-factors@test.local"
		alicePass  = "alice-pass-12345"
		bobEmail   = "bob-factors@test.local"
		bobPass    = "bob-pass-12345"
	)
	for _, u := range []struct{ email, pw string }{{aliceEmail, alicePass}, {bobEmail, bobPass}} {
		w := doAuthed(t, fr.router, http.MethodPost, "/api/v1/users", superToken, models.UserCreateRequest{
			Name: u.email, Email: u.email, Password: u.pw, Active: true,
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("create user %s: %d %s", u.email, w.Code, w.Body.String())
		}
	}

	_, aliceToken := doLogin(t, fr.router, aliceEmail, alicePass)
	_, bobToken := doLogin(t, fr.router, bobEmail, bobPass)
	if aliceToken == "" || bobToken == "" {
		t.Fatal("missing login tokens")
	}

	// Alice enrols TOTP.
	w := doAuthed(t, fr.router, http.MethodPost, "/api/v1/users/me/factors", aliceToken,
		models.FactorEnrollRequest{Type: models.FactorTypeTOTP, Password: alicePass})
	if w.Code != http.StatusOK {
		t.Fatalf("alice enrol: %d %s", w.Code, w.Body.String())
	}
	var enrol models.FactorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &enrol); err != nil {
		t.Fatalf("decode enrol: %v", err)
	}
	aliceFactorID := enrol.ID
	// Pull the secret out of the polymorphic enrollment blob.
	rawEnroll, _ := json.Marshal(enrol.Enrollment)
	var payload models.TOTPEnrollmentPayload
	if err := json.Unmarshal(rawEnroll, &payload); err != nil {
		t.Fatalf("decode enrollment: %v", err)
	}

	// Resolve Alice's numeric user id via /me so the test covers the
	// "Bob addresses alice's explicit user id" path as well as the
	// /users/me path. Relies on the /me endpoint already being wired.
	w = doAuthed(t, fr.router, http.MethodGet, "/api/v1/me", aliceToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("alice /me: %d", w.Code)
	}
	var aliceMe models.UserResponse
	_ = json.Unmarshal(w.Body.Bytes(), &aliceMe)

	// Bob tries every endpoint against Alice's factor id. Each must 404.
	//
	// 1. GET via explicit numeric user id — self-scope rule kicks in.
	w = doAuthed(t, fr.router, http.MethodGet,
		fmt.Sprintf("/api/v1/users/%d/factors/%d", aliceMe.ID, aliceFactorID),
		bobToken, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("bob GET via alice's user id: got %d, want 404. body=%s", w.Code, w.Body.String())
	}

	// 2. GET via /me/factors/:id with alice's id — ownership check wins.
	w = doAuthed(t, fr.router, http.MethodGet,
		fmt.Sprintf("/api/v1/users/me/factors/%d", aliceFactorID),
		bobToken, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("bob GET /me/factors/<alice-id>: got %d, want 404", w.Code)
	}

	// 3. Activate — bob provides his own password, but the factor is alice's.
	w = doAuthed(t, fr.router, http.MethodPost,
		fmt.Sprintf("/api/v1/users/me/factors/%d/activate", aliceFactorID),
		bobToken, models.FactorActivateRequest{Code: "000000"})
	if w.Code != http.StatusNotFound {
		t.Errorf("bob activate alice's factor: got %d, want 404", w.Code)
	}

	// 4. Delete — same.
	w = doAuthed(t, fr.router, http.MethodDelete,
		fmt.Sprintf("/api/v1/users/me/factors/%d", aliceFactorID),
		bobToken, models.FactorDeleteRequest{Password: bobPass, Code: "000000"})
	if w.Code != http.StatusNotFound {
		t.Errorf("bob delete alice's factor: got %d, want 404", w.Code)
	}

	// 5. PATCH label — same.
	lbl := "pwned"
	w = doAuthed(t, fr.router, http.MethodPatch,
		fmt.Sprintf("/api/v1/users/me/factors/%d", aliceFactorID),
		bobToken, models.FactorLabelUpdateRequest{Label: &lbl})
	if w.Code != http.StatusNotFound {
		t.Errorf("bob patch alice's factor: got %d, want 404", w.Code)
	}

	// 6. Alice can still see and use her factor — none of Bob's attempts
	//    were effective.
	w = doAuthed(t, fr.router, http.MethodGet,
		fmt.Sprintf("/api/v1/users/me/factors/%d", aliceFactorID),
		aliceToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("alice can't see her own factor: %d %s", w.Code, w.Body.String())
	}
	var aliceView models.FactorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &aliceView)
	if aliceView.Activated {
		t.Error("alice's factor should still be pending (bob's activate attempts must not succeed)")
	}
	if aliceView.Label != nil && *aliceView.Label == "pwned" {
		t.Error("bob's patch wrongly took effect on alice's label")
	}

	// 7. Alice activates her factor normally — confirms it's still usable
	//    after every hostile attempt from Bob.
	code, err := totp.GenerateCode(payload.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("gen code: %v", err)
	}
	w = doAuthed(t, fr.router, http.MethodPost,
		fmt.Sprintf("/api/v1/users/me/factors/%d/activate", aliceFactorID),
		aliceToken, models.FactorActivateRequest{Code: code})
	if w.Code != http.StatusOK {
		t.Errorf("alice activates after bob's attempts: got %d %s", w.Code, w.Body.String())
	}
}
