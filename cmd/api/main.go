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
	"github.com/eenemeene/kitamanager-go/internal/database"
	"github.com/eenemeene/kitamanager-go/internal/handlers"
	"github.com/eenemeene/kitamanager-go/internal/importer"
	"github.com/eenemeene/kitamanager-go/internal/middleware"
	"github.com/eenemeene/kitamanager-go/internal/rbac"
	"github.com/eenemeene/kitamanager-go/internal/routes"
	"github.com/eenemeene/kitamanager-go/internal/seed"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/version"
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
	token                       *store.TokenStore
	governmentFundingBillPeriod *store.GovernmentFundingBillPeriodStore
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
	seedData(cfg, db, stores, enforcer)

	transactor := store.NewTransactor(db)
	permissionService := rbac.NewPermissionService(stores.userOrganization, enforcer)

	svc := initServices(stores, cfg, transactor)
	mw := initMiddleware(stores, cfg, permissionService)
	r := setupRouter(cfg, db, stores, svc, mw)

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	tokenCleanupDone := startTokenCleanup(stores.token)

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

	shutdown(srv, tokenCleanupDone, mw, svc, db)
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
		token:                       store.NewTokenStore(db),
		governmentFundingBillPeriod: store.NewGovernmentFundingBillPeriodStore(db),
	}
}

func seedData(cfg *config.Config, db *gorm.DB, s *appStores, enforcer *rbac.Enforcer) {
	if err := seed.SeedAdmin(cfg, s.user, s.userOrganization, enforcer); err != nil {
		slog.Error("Failed to seed admin user", "error", err)
		os.Exit(1)
	}

	if err := seed.SeedGovernmentFunding(cfg, db, s.governmentFunding); err != nil {
		slog.Error("Failed to seed government funding", "error", err)
		os.Exit(1)
	}

	if err := seed.SeedTestData(cfg, db, s.governmentFunding); err != nil {
		slog.Error("Failed to seed test data", "error", err)
		os.Exit(1)
	}
}

func initServices(s *appStores, cfg *config.Config, transactor store.Transactor) *appServices {
	auditService := service.NewAuditService(s.audit)
	return &appServices{
		audit:                 auditService,
		auth:                  service.NewAuthService(s.user, s.token, cfg.JWTSecret, auditService),
		user:                  service.NewUserService(s.user, s.userOrganization),
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
		statistics:            service.NewStatisticsService(s.child, s.employee, s.organization, s.governmentFunding, s.payPlan, s.budgetItem),
		governmentFundingBill: service.NewGovernmentFundingBillService(s.child, s.governmentFundingBillPeriod, s.organization, s.governmentFunding),
	}
}

func initMiddleware(s *appStores, cfg *config.Config, permissionService *rbac.PermissionService) *appMiddleware {
	return &appMiddleware{
		auth:             middleware.NewAuthMiddleware(cfg.JWTSecret, s.token),
		authz:            middleware.NewAuthorizationMiddleware(permissionService),
		csrf:             middleware.NewCSRFMiddleware(cfg.JWTSecret),
		loginRateLimiter: middleware.LoginRateLimiter(cfg.LoginRateLimitPerMinute),
		apiRateLimiter:   middleware.APIRateLimiter(cfg.APIRateLimitPerMinute),
	}
}

func setupRouter(cfg *config.Config, db *gorm.DB, s *appStores, svc *appServices, mw *appMiddleware) *gin.Engine {
	r := gin.New()

	// Configure trusted proxies for accurate client IP detection
	if len(cfg.TrustedProxies) > 0 {
		if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
			slog.Error("Failed to set trusted proxies", "error", err)
			os.Exit(1)
		}
	} else if cfg.IsProduction() {
		if err := r.SetTrustedProxies(nil); err != nil {
			slog.Error("Failed to set trusted proxies", "error", err)
			os.Exit(1)
		}
	}

	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.StructuredLogger())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.Metrics())
	r.Use(middleware.BodySizeLimit(middleware.MaxRequestBodySize))
	r.Use(middleware.RequestTimeout(middleware.DefaultRequestTimeout))

	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: cfg.CORSAllowCredentials,
		MaxAge:           12 * time.Hour,
	}
	if len(cfg.CORSAllowOrigins) == 1 && cfg.CORSAllowOrigins[0] == "*" {
		corsConfig.AllowOriginFunc = func(origin string) bool { return true }
	} else {
		corsConfig.AllowOrigins = cfg.CORSAllowOrigins
	}
	r.Use(cors.New(corsConfig))

	// Health check endpoints (no auth required)
	healthHandler := handlers.NewHealthHandler(db)
	r.GET("/api/v1/health", healthHandler.Check)
	r.GET("/api/v1/ready", healthHandler.Ready)
	r.GET("/api/v1/live", healthHandler.Live)

	// Metrics endpoint (requires authentication)
	r.GET("/metrics", mw.auth.RequireAuth(), gin.WrapH(promhttp.Handler()))

	// Swagger UI
	if cfg.IsDevelopment() {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	} else {
		swagger := r.Group("/swagger")
		swagger.Use(mw.auth.RequireAuth())
		swagger.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// API routes
	routes.Setup(r, routes.Deps{
		Auth:                  handlers.NewAuthHandler(svc.auth, cfg.SecureCookies),
		User:                  handlers.NewUserHandler(svc.user, svc.userOrganization, svc.audit, s.token),
		Section:               handlers.NewSectionHandler(svc.section, svc.audit),
		Organization:          handlers.NewOrganizationHandler(svc.organization, svc.audit),
		Employee:              handlers.NewEmployeeHandler(svc.employee, svc.audit),
		Child:                 handlers.NewChildHandler(svc.child, svc.audit),
		ChildStatistics:       handlers.NewChildStatisticsHandler(svc.child),
		GovernmentFunding:     handlers.NewGovernmentFundingHandler(svc.governmentFunding, svc.audit, importer.NewGovernmentFundingImporter(db, s.governmentFunding)),
		PayPlan:               handlers.NewPayPlanHandler(svc.payPlan, svc.audit),
		ChildAttendance:       handlers.NewChildAttendanceHandler(svc.childAttendance, svc.audit),
		BudgetItem:            handlers.NewBudgetItemHandler(svc.budgetItem, svc.audit),
		StepPromotion:         handlers.NewStepPromotionHandler(svc.stepPromotion),
		Statistics:            handlers.NewStatisticsHandler(svc.statistics),
		Export:                handlers.NewExportHandler(svc.employee, svc.child),
		GovernmentFundingBill: handlers.NewGovernmentFundingBillHandler(svc.governmentFundingBill, svc.audit),
		AuthMiddleware:        mw.auth,
		AuthzMiddleware:       mw.authz,
		CSRFMiddleware:        mw.csrf,
		LoginRateLimiter:      mw.loginRateLimiter,
		APIRateLimiter:        mw.apiRateLimiter,
	})

	return r
}

func startTokenCleanup(tokenStore *store.TokenStore) chan struct{} {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := tokenStore.CleanupExpired(context.Background()); err != nil {
					slog.Error("Failed to cleanup expired tokens", "error", err)
				}
			case <-done:
				return
			}
		}
	}()
	return done
}

func shutdown(srv *http.Server, tokenCleanupDone chan struct{}, mw *appMiddleware, svc *appServices, db *gorm.DB) {
	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	close(tokenCleanupDone)

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
