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
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/eenemeene/kitamanager-go/docs"
	"github.com/eenemeene/kitamanager-go/internal/config"
	"github.com/eenemeene/kitamanager-go/internal/database"
	"github.com/eenemeene/kitamanager-go/internal/handlers"
	"github.com/eenemeene/kitamanager-go/internal/middleware"
	"github.com/eenemeene/kitamanager-go/internal/rbac"
	"github.com/eenemeene/kitamanager-go/internal/routes"
	"github.com/eenemeene/kitamanager-go/internal/seed"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/version"
	"github.com/eenemeene/kitamanager-go/internal/web"
)

// @title KitaManager API
// @version 1.0
// @description REST API for managing Users, Groups, and Organizations with RBAC support
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

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Setup structured logging
	setupLogging(cfg)

	slog.Info("Starting KitaManager API",
		"version", version.Version(),
		"commit", version.GitCommit,
		"built", version.BuildTime,
		"port", cfg.ServerPort,
	)

	// Connect to database
	db, err := database.Connect(cfg)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Initialize RBAC enforcer
	enforcer, err := rbac.NewEnforcer(db, cfg.RBACModelPath)
	if err != nil {
		slog.Error("Failed to initialize RBAC enforcer", "error", err)
		os.Exit(1)
	}

	// Seed default policies if requested
	if os.Getenv("SEED_RBAC_POLICIES") == "true" {
		slog.Info("Seeding RBAC policies...")
		if err := enforcer.SeedDefaultPolicies(); err != nil {
			slog.Error("Failed to seed RBAC policies", "error", err)
			os.Exit(1)
		}
		slog.Info("RBAC policies seeded successfully")
	}

	// Initialize stores
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	sectionStore := store.NewSectionStore(db)
	orgStore := store.NewOrganizationStore(db)
	employeeStore := store.NewEmployeeStore(db)
	childStore := store.NewChildStore(db)
	userGroupStore := store.NewUserGroupStore(db)
	governmentFundingStore := store.NewGovernmentFundingStore(db)
	payPlanStore := store.NewPayPlanStore(db)
	childAttendanceStore := store.NewChildAttendanceStore(db)
	auditStore := store.NewAuditStore(db)

	// Seed admin user if configured
	if err := seed.SeedAdmin(cfg, userStore, userGroupStore, enforcer); err != nil {
		slog.Error("Failed to seed admin user", "error", err)
		os.Exit(1)
	}

	// Seed government funding if configured
	if err := seed.SeedGovernmentFunding(cfg, db, governmentFundingStore); err != nil {
		slog.Error("Failed to seed government funding", "error", err)
		os.Exit(1)
	}

	// Seed test data if configured
	if err := seed.SeedTestData(cfg, db, governmentFundingStore); err != nil {
		slog.Error("Failed to seed test data", "error", err)
		os.Exit(1)
	}

	// Initialize RBAC permission service
	permissionService := rbac.NewPermissionService(userGroupStore, enforcer)

	// Initialize services
	auditService := service.NewAuditService(auditStore)
	userService := service.NewUserService(userStore, groupStore)
	userGroupService := service.NewUserGroupService(userGroupStore, userStore, groupStore)
	orgService := service.NewOrganizationService(orgStore, groupStore, userStore)
	groupService := service.NewGroupService(groupStore)
	sectionService := service.NewSectionService(sectionStore)
	employeeService := service.NewEmployeeService(employeeStore)
	childService := service.NewChildService(childStore, orgStore, governmentFundingStore)
	governmentFundingService := service.NewGovernmentFundingService(governmentFundingStore)
	payPlanService := service.NewPayPlanService(payPlanStore)
	childAttendanceService := service.NewChildAttendanceService(childAttendanceStore, childStore)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userStore, cfg.JWTSecret, auditService)
	userHandler := handlers.NewUserHandler(userService, userGroupService, auditService)
	groupHandler := handlers.NewGroupHandler(groupService, auditService)
	sectionHandler := handlers.NewSectionHandler(sectionService, auditService)
	orgHandler := handlers.NewOrganizationHandler(orgService, auditService)
	employeeHandler := handlers.NewEmployeeHandler(employeeService, auditService)
	childHandler := handlers.NewChildHandler(childService, auditService)
	governmentFundingHandler := handlers.NewGovernmentFundingHandler(governmentFundingService, auditService)
	payPlanHandler := handlers.NewPayPlanHandler(payPlanService, auditService)
	childAttendanceHandler := handlers.NewChildAttendanceHandler(childAttendanceService, auditService)
	healthHandler := handlers.NewHealthHandler(db)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWTSecret)
	authzMiddleware := middleware.NewAuthorizationMiddleware(permissionService)
	csrfMiddleware := middleware.NewCSRFMiddleware()
	loginRateLimiter := middleware.LoginRateLimiter(cfg.LoginRateLimitPerMinute)

	// Create Gin router
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.StructuredLogger())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.BodySizeLimit(middleware.MaxRequestBodySize))
	r.Use(middleware.RequestTimeout(middleware.DefaultRequestTimeout))

	// Configure CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSAllowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: cfg.CORSAllowCredentials,
		MaxAge:           12 * time.Hour,
	}))

	// Health check endpoints (no auth required)
	r.GET("/api/v1/health", healthHandler.Check)
	r.GET("/api/v1/ready", healthHandler.Ready)
	r.GET("/api/v1/live", healthHandler.Live)

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Setup API routes
	routes.Setup(r, authHandler, userHandler, groupHandler, sectionHandler, orgHandler, employeeHandler, childHandler, governmentFundingHandler, payPlanHandler, childAttendanceHandler, authMiddleware, authzMiddleware, csrfMiddleware, loginRateLimiter)

	// Register embedded web UI
	if err := web.RegisterHandlers(r); err != nil {
		slog.Error("Failed to register web handlers", "error", err)
		os.Exit(1)
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("Server started",
			"port", cfg.ServerPort,
			"swagger", "http://localhost:"+cfg.ServerPort+"/swagger/index.html",
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	// Close database connection
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
