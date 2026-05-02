package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	_ "github.com/eenemeene/kitamanager-go/docs"
	"github.com/eenemeene/kitamanager-go/internal/config"
	cryptopkg "github.com/eenemeene/kitamanager-go/internal/crypto"
	"github.com/eenemeene/kitamanager-go/internal/database"
	"github.com/eenemeene/kitamanager-go/internal/handlers"
	"github.com/eenemeene/kitamanager-go/internal/importer"
	"github.com/eenemeene/kitamanager-go/internal/middleware"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/rbac"
	"github.com/eenemeene/kitamanager-go/internal/routes"
	"github.com/eenemeene/kitamanager-go/internal/seed"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/version"
	webauthnpkg "github.com/eenemeene/kitamanager-go/internal/webauthn"
)

// @title KitaManager API
// @version 1.0
// @description REST API for managing Users and Organizations with RBAC support
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@kitamanager.example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// appStores holds all data access layer instances.
type appStores struct {
	user                        *store.UserStore
	section                     *store.SectionStore
	organization                *store.OrganizationStore
	employee                    *store.EmployeeStore
	child                       *store.ChildStore
	userOrganization            *store.UserOrganizationStore
	governmentFunding           *store.GovernmentFundingStore
	payPlan                     *store.PayPlanStore
	childAttendance             *store.ChildAttendanceStore
	budgetItem                  *store.BudgetItemStore
	audit                       *store.AuditStore
	session                     *store.SessionStore
	factor                      *store.FactorStore
	governmentFundingBillPeriod *store.GovernmentFundingBillPeriodStore
	childVoucher                *store.ChildVoucherStore
}

// appServices holds all business logic layer instances.
type appServices struct {
	audit                 *service.AuditService
	auth                  *service.AuthService
	user                  *service.UserService
	userOrganization      *service.UserOrganizationService
	organization          *service.OrganizationService
	section               *service.SectionService
	employee              *service.EmployeeService
	child                 *service.ChildService
	governmentFunding     *service.GovernmentFundingService
	payPlan               *service.PayPlanService
	childAttendance       *service.ChildAttendanceService
	budgetItem            *service.BudgetItemService
	stepPromotion         *service.StepPromotionService
	statistics            *service.StatisticsService
	governmentFundingBill *service.GovernmentFundingBillService
	email                 *service.EmailService
	factor                *service.FactorService
}

// appMiddleware holds all middleware instances.
type appMiddleware struct {
	auth             *middleware.AuthMiddleware
	authz            *middleware.AuthorizationMiddleware
	csrf             *middleware.CSRFMiddleware
	loginRateLimiter *middleware.RateLimiter
	apiRateLimiter   *middleware.RateLimiter
}

func main() {
	// Force release mode before any gin package function runs so startup never
	// emits debug route tables or verbose diagnostics. Tests override via
	// gin.SetMode(gin.TestMode).
	gin.SetMode(gin.ReleaseMode)

	// Register custom binding validators (e.g. "voucher" for Berlin
	// Gutschein format) before any handler binds JSON. Tests that exercise
	// the binding path call this from their own setup; here it must run
	// before routes/handlers are wired below.
	if err := models.RegisterCustomValidators(); err != nil {
		slog.Error("Failed to register custom validators", "error", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	setupLogging(cfg)

	slog.Info("Starting KitaManager API",
		"version", version.Version(),
		"commit", version.GitCommit,
		"built", version.BuildTime,
		"port", cfg.ServerPort,
	)

	db, err := database.Connect(cfg)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	enforcer, err := rbac.NewEnforcer(db, cfg.RBACModelPath)
	if err != nil {
		slog.Error("Failed to initialize RBAC enforcer", "error", err)
		os.Exit(1)
	}

	if os.Getenv("SEED_RBAC_POLICIES") == "true" {
		slog.Info("Seeding RBAC policies...")
		if err := enforcer.SeedDefaultPolicies(); err != nil {
			slog.Error("Failed to seed RBAC policies", "error", err)
			os.Exit(1)
		}
		slog.Info("RBAC policies seeded successfully")
	}

	stores := initStores(db)
	transactor := store.NewTransactor(db)
	permissionService := rbac.NewPermissionService(stores.userOrganization, enforcer)

	svc := initServices(stores, cfg, transactor)
	seedData(cfg, db, stores, enforcer, svc.governmentFunding, transactor)
	mw := initMiddleware(stores, cfg, permissionService)
	r := setupRouter(cfg, db, stores, svc, mw, transactor)

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	sessionCleanupDone := startSessionCleanup(stores.session, stores.factor, stores.audit, svc.audit, cfg.AuditLogRetentionDays)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("Server started",
			"port", cfg.ServerPort,
			"swagger", "http://localhost:"+cfg.ServerPort+"/swagger/index.html",
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server error", "error", err)
			quit <- syscall.SIGTERM
		}
	}()

	<-quit

	shutdown(srv, sessionCleanupDone, mw, svc, db)
}

func initStores(db *gorm.DB) *appStores {
	return &appStores{
		user:                        store.NewUserStore(db),
		section:                     store.NewSectionStore(db),
		organization:                store.NewOrganizationStore(db),
		employee:                    store.NewEmployeeStore(db),
		child:                       store.NewChildStore(db),
		userOrganization:            store.NewUserOrganizationStore(db),
		governmentFunding:           store.NewGovernmentFundingStore(db),
		payPlan:                     store.NewPayPlanStore(db),
		childAttendance:             store.NewChildAttendanceStore(db),
		budgetItem:                  store.NewBudgetItemStore(db),
		audit:                       store.NewAuditStore(db),
		session:                     store.NewSessionStore(db),
		factor:                      store.NewFactorStore(db),
		governmentFundingBillPeriod: store.NewGovernmentFundingBillPeriodStore(db),
		childVoucher:                store.NewChildVoucherStore(db),
	}
}

func seedData(cfg *config.Config, db *gorm.DB, s *appStores, enforcer *rbac.Enforcer, fundingSvc *service.GovernmentFundingService, transactor store.Transactor) {
	if err := seed.SeedAdmin(cfg, s.user, s.userOrganization, enforcer); err != nil {
		slog.Error("Failed to seed admin user", "error", err)
		os.Exit(1)
	}

	fundingImporter := importer.NewGovernmentFundingImporter(fundingSvc, transactor)

	if err := seed.SeedGovernmentFunding(cfg, fundingImporter); err != nil {
		slog.Error("Failed to seed government funding", "error", err)
		os.Exit(1)
	}

	if err := seed.SeedTestData(cfg, db, fundingImporter); err != nil {
		slog.Error("Failed to seed test data", "error", err)
		os.Exit(1)
	}
}

func initServices(s *appStores, cfg *config.Config, transactor store.Transactor) *appServices {
	auditService := service.NewAuditService(s.audit)

	// TOTP_ENCRYPTION_KEY shape is validated at config.Load() time;
	// the hex decode + AEAD construction here is guaranteed not to
	// fail on valid config. We still panic on failure rather than
	// silently skipping — a nil AEAD would surface as a runtime NPE
	// on the first enrollment attempt, which is much worse.
	totpKey, err := cryptopkg.DecodeKey(cfg.TOTPEncryptionKey)
	if err != nil {
		slog.Error("TOTP_ENCRYPTION_KEY invalid despite config validation", "error", err)
		os.Exit(1)
	}
	aead, err := cryptopkg.NewAEAD(totpKey)
	if err != nil {
		slog.Error("failed to construct TOTP AEAD", "error", err)
		os.Exit(1)
	}

	// Optional WebAuthn wiring. All three env vars together enable
	// the factor type; otherwise WebAuthn endpoints return a clear
	// "not enabled" error rather than crashing.
	var webAuthnSvc *webauthnpkg.Service
	if cfg.WebAuthnRPID != "" {
		webAuthnSvc, err = webauthnpkg.New(webauthnpkg.Config{
			RPID:      cfg.WebAuthnRPID,
			RPName:    cfg.WebAuthnRPName,
			RPOrigins: cfg.WebAuthnOrigins,
		})
		if err != nil {
			slog.Error("WebAuthn config invalid despite startup validation", "error", err)
			os.Exit(1)
		}
	}

	factorSvc := service.NewFactorService(s.factor, s.user, aead, cfg.TOTPIssuer, webAuthnSvc, auditService)
	return &appServices{
		audit:                 auditService,
		auth:                  service.NewAuthService(s.user, s.session, cfg.CSRFHMACKey, auditService, factorSvc),
		user:                  service.NewUserService(s.user, s.userOrganization, s.session).WithAuditService(auditService),
		factor:                factorSvc,
		userOrganization:      service.NewUserOrganizationService(s.userOrganization, s.user, transactor),
		organization:          service.NewOrganizationService(s.organization, s.user),
		section:               service.NewSectionService(s.section, transactor),
		employee:              service.NewEmployeeService(s.employee, s.payPlan, s.section, transactor),
		child:                 service.NewChildService(s.child, s.organization, s.governmentFunding, s.section, transactor),
		governmentFunding:     service.NewGovernmentFundingService(s.governmentFunding, transactor),
		payPlan:               service.NewPayPlanService(s.payPlan, transactor),
		childAttendance:       service.NewChildAttendanceService(s.childAttendance, s.child),
		budgetItem:            service.NewBudgetItemService(s.budgetItem, transactor),
		stepPromotion:         service.NewStepPromotionService(s.payPlan, s.employee),
		statistics:            service.NewStatisticsService(s.child, s.employee, s.organization, s.governmentFunding, s.payPlan, s.budgetItem, s.section, s.governmentFundingBillPeriod),
		governmentFundingBill: service.NewGovernmentFundingBillService(s.child, s.childVoucher, s.governmentFundingBillPeriod, s.organization, s.governmentFunding),
		email:                 service.NewEmailService(cfg),
	}
}

func initMiddleware(s *appStores, cfg *config.Config, permissionService *rbac.PermissionService) *appMiddleware {
	slog.Info("Rate limiter is in-memory — not suitable for multi-instance deployments. Use a Redis-backed limiter when scaling horizontally.")

	authMW := middleware.NewAuthMiddleware(s.session)
	authMW.SetSecureCookies(cfg.SecureCookies)

	return &appMiddleware{
		auth:             authMW,
		authz:            middleware.NewAuthorizationMiddleware(permissionService),
		csrf:             middleware.NewCSRFMiddleware(cfg.CSRFHMACKey),
		loginRateLimiter: middleware.LoginRateLimiter(cfg.LoginRateLimitPerMinute),
		apiRateLimiter:   middleware.APIRateLimiter(cfg.APIRateLimitPerMinute),
	}
}

func setupRouter(cfg *config.Config, db *gorm.DB, s *appStores, svc *appServices, mw *appMiddleware, transactor store.Transactor) *gin.Engine {
	r := gin.New()

	// Trusted proxies: explicit allowlist only. An empty list means the app does
	// not honor any X-Forwarded-* headers, and c.ClientIP() returns the direct
	// peer's address. Operators put their reverse-proxy CIDRs in TRUSTED_PROXIES.
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		slog.Error("Failed to set trusted proxies", "error", err)
		os.Exit(1)
	}

	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.StructuredLogger())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.Metrics())
	r.Use(middleware.BodySizeLimit(middleware.MaxRequestBodySize))
	r.Use(middleware.RequestTimeout(middleware.DefaultRequestTimeout))

	corsConfig := cors.Config{
		AllowOrigins:     cfg.CORSAllowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: cfg.CORSAllowCredentials,
		MaxAge:           12 * time.Hour,
	}
	r.Use(cors.New(corsConfig))

	// Health check endpoints (no auth required)
	healthHandler := handlers.NewHealthHandler(db)
	r.GET("/api/v1/health", healthHandler.Check)
	r.GET("/api/v1/ready", healthHandler.Ready)
	r.GET("/api/v1/live", healthHandler.Live)

	// Metrics endpoint — superadmin only to avoid leaking request-path/label
	// cardinality to authenticated tenant users.
	r.GET("/metrics",
		mw.auth.RequireAuth(),
		mw.authz.RequireSuperAdmin(),
		gin.WrapH(promhttp.Handler()))

	// Swagger UI — always requires superadmin. Developers log in as the seeded
	// admin to use it; there is no anonymous path to the API surface.
	swagger := r.Group("/swagger")
	swagger.Use(mw.auth.RequireAuth())
	swagger.Use(mw.authz.RequireSuperAdmin())
	swagger.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API routes
	routes.Setup(r, routes.Deps{
		Auth:                  handlers.NewAuthHandler(svc.auth, cfg.SecureCookies),
		User:                  handlers.NewUserHandler(svc.user, svc.userOrganization, svc.audit, s.session),
		Section:               handlers.NewSectionHandler(svc.section, svc.audit),
		Organization:          handlers.NewOrganizationHandler(svc.organization, svc.audit),
		Employee:              handlers.NewEmployeeHandler(svc.employee, svc.audit),
		Child:                 handlers.NewChildHandler(svc.child, svc.audit),
		GovernmentFunding:     handlers.NewGovernmentFundingHandler(svc.governmentFunding, svc.audit, importer.NewGovernmentFundingImporter(svc.governmentFunding, transactor)),
		PayPlan:               handlers.NewPayPlanHandler(svc.payPlan, svc.audit),
		ChildAttendance:       handlers.NewChildAttendanceHandler(svc.childAttendance, svc.audit),
		BudgetItem:            handlers.NewBudgetItemHandler(svc.budgetItem, svc.audit),
		StepPromotion:         handlers.NewStepPromotionHandler(svc.stepPromotion),
		Statistics:            handlers.NewStatisticsHandler(svc.statistics),
		Export:                handlers.NewExportHandler(svc.employee, svc.child, svc.audit),
		GovernmentFundingBill: handlers.NewGovernmentFundingBillHandler(svc.governmentFundingBill, svc.audit),
		AuditLog:              handlers.NewAuditLogHandler(svc.audit),
		Factor:                handlers.NewFactorHandler(svc.factor),
		AuthMiddleware:        mw.auth,
		AuthzMiddleware:       mw.authz,
		CSRFMiddleware:        mw.csrf,
		LoginRateLimiter:      mw.loginRateLimiter,
		APIRateLimiter:        mw.apiRateLimiter,
	})

	return r
}

// startSessionCleanup runs the hourly background sweeper. Beyond
// expired sessions and abandoned MFA enrolments, it also enforces the
// configured audit-log retention window (closes audit finding O-M-8:
// audit_logs.Cleanup exists but was never invoked). When
// auditRetentionDays <= 0 the audit cleanup branch is skipped and the
// log table grows unbounded — operators with an external retention
// pipeline can opt into that.
//
// Each successful audit-log purge ALSO writes a self-marker
// (audit_log_purged) so an investigator looking at the table later
// can ratify the deletion pattern as scheduled retention rather than
// tampering.
func startSessionCleanup(sessionStore *store.SessionStore, factorStore *store.FactorStore, auditStore *store.AuditStore, auditService *service.AuditService, auditRetentionDays int) chan struct{} {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx := context.Background()
				if err := sessionStore.CleanupExpired(ctx); err != nil {
					slog.Error("Failed to cleanup expired sessions", "error", err)
				}
				// Abandoned enrollment rows — users who started MFA
				// enrollment but never completed it. One hour
				// matches the UX promise "finish within the hour
				// or start over."
				if _, err := factorStore.CleanupAbandonedPending(ctx, time.Hour); err != nil {
					slog.Error("Failed to cleanup abandoned pending factors", "error", err)
				}
				if auditRetentionDays > 0 {
					cutoff := time.Now().UTC().AddDate(0, 0, -auditRetentionDays)
					n, err := auditStore.Cleanup(ctx, cutoff)
					if err != nil {
						slog.Error("Failed to cleanup old audit logs", "error", err, "older_than", cutoff)
					} else if n > 0 {
						slog.Info("Audit log retention purge complete", "deleted_rows", n, "older_than", cutoff)
						if auditService != nil {
							auditService.LogAuditLogPurged(ctx, n, cutoff)
						}
					}
				}
			case <-done:
				return
			}
		}
	}()
	return done
}

func shutdown(srv *http.Server, sessionCleanupDone chan struct{}, mw *appMiddleware, svc *appServices, db *gorm.DB) {
	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	close(sessionCleanupDone)

	if mw.loginRateLimiter != nil {
		mw.loginRateLimiter.Stop()
	}
	if mw.apiRateLimiter != nil {
		mw.apiRateLimiter.Stop()
	}

	slog.Info("Draining audit logs...")
	svc.audit.Shutdown()

	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}

	slog.Info("Server stopped gracefully")
}

func setupLogging(cfg *config.Config) {
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var handler slog.Handler
	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}

	slog.SetDefault(slog.New(handler))
}
