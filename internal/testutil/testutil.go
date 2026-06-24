package testutil

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/eenemeene/kitamanager-go/internal/database"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

// testDatabaseURLEnv, when set, points at an already-running PostgreSQL
// *server* (e.g. a CI `services:` container). In that mode SetupTestDB does
// NOT start a per-package testcontainer — instead each test binary carves out
// its own freshly-migrated database on that shared server (see startContainer).
//
// This exists because `go test ./...` runs one test binary per package in
// parallel, and the testcontainers path starts one Postgres container per
// binary. Under that fan-out the Docker daemon / Ryuk reaper flakes on CI
// runners (postmaster exits mid-run, "connection refused" cascades). Pointing
// every binary at one shared server removes the per-package container churn
// while a unique database per process preserves test isolation.
const testDatabaseURLEnv = "TEST_DATABASE_URL"

var (
	once         sync.Once
	pgContainer  *postgres.PostgresContainer
	containerDSN string
	sharedDB     *gorm.DB
	initErr      error
)

// truncation order: leaf tables first, then parents
var truncateTables = []string{
	"factor_backup_codes",
	"factor_totp_secrets",
	"factors",
	"sessions",
	"audit_logs",
	"budget_item_entries",
	"budget_items",
	"child_attendances",
	"pay_plan_entries",
	"pay_plan_periods",
	"pay_plans",
	"government_funding_properties",
	"government_funding_periods",
	"government_fundings",
	"child_contracts",
	"children",
	"employee_contracts",
	"employees",
	"sections",
	"user_organizations",
	"users",
	"organizations",
}

func startContainer() {
	// Shared-server mode: reuse an external Postgres provided by CI instead of
	// spinning up a testcontainer for this package. Each test binary gets its
	// own database on that server for isolation.
	if serverURL := os.Getenv(testDatabaseURLEnv); serverURL != "" {
		initErr = setupExternalDB(serverURL)
		return
	}

	ctx := context.Background()

	pgContainer, initErr = postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("kitamanager_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if initErr != nil {
		return
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		initErr = fmt.Errorf("failed to get connection string: %w", err)
		return
	}
	containerDSN = connStr

	// Run production migrations
	initErr = database.RunMigrationsWithURL(connStr)
	if initErr != nil {
		return
	}

	// Open a shared GORM connection pool (reused across all tests)
	sharedDB, initErr = gorm.Open(gormpostgres.Open(containerDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

// setupExternalDB carves out a fresh, uniquely-named database on the shared
// PostgreSQL server given by serverURL, migrates it, and opens the package's
// shared GORM pool against it. A database per test binary keeps packages that
// `go test ./...` runs in parallel from truncating each other's data — the
// same isolation the per-package testcontainer used to provide.
//
// The database is intentionally not dropped on exit: the shared server only
// exists for the lifetime of a CI job and is discarded with it, so a teardown
// (which Go's per-package test lifecycle makes awkward without a TestMain in
// every package) would buy nothing.
func setupExternalDB(serverURL string) error {
	admin, err := gorm.Open(gormpostgres.Open(serverURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("connect to shared test server: %w", err)
	}

	// Unique per process: go test runs each package in its own process, so
	// pid is unique among concurrently-running binaries; the nanosecond clock
	// guards against pid reuse across a long run. Both are digits/underscore,
	// so the identifier needs no escaping.
	dbName := fmt.Sprintf("km_test_%d_%d", os.Getpid(), time.Now().UnixNano())
	if err := admin.Exec(`CREATE DATABASE "` + dbName + `"`).Error; err != nil {
		return fmt.Errorf("create test database %q: %w", dbName, err)
	}
	if sqlDB, derr := admin.DB(); derr == nil {
		_ = sqlDB.Close() // CREATE DATABASE done; the admin handle is no longer needed.
	}

	procURL, err := withDatabaseName(serverURL, dbName)
	if err != nil {
		return err
	}
	containerDSN = procURL

	if err := database.RunMigrationsWithURL(procURL); err != nil {
		return fmt.Errorf("migrate test database %q: %w", dbName, err)
	}

	sharedDB, err = gorm.Open(gormpostgres.Open(procURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("open test database %q: %w", dbName, err)
	}
	return nil
}

// withDatabaseName returns serverURL with its database path replaced by name,
// preserving credentials, host, and query parameters (e.g. sslmode).
func withDatabaseName(serverURL, name string) (string, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", testDatabaseURLEnv, err)
	}
	u.Path = "/" + name
	return u.String(), nil
}

// SetupTestDB starts a shared PostgreSQL testcontainer (once per process),
// returns a GORM *gorm.DB, and truncates all tables before the test runs.
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	once.Do(startContainer)
	if initErr != nil {
		t.Fatalf("failed to start test database container: %v", initErr)
	}

	// Truncate all tables for test isolation
	for _, table := range truncateTables {
		if err := sharedDB.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table)).Error; err != nil {
			t.Fatalf("failed to truncate table %s: %v", table, err)
		}
	}

	return sharedDB
}

// CreateTestOrganization creates an organization for testing.
// It also creates a default section ("Default") for the organization,
// mirroring what the real org creation endpoint does.
func CreateTestOrganization(t *testing.T, db *gorm.DB, name string) *models.Organization {
	t.Helper()

	org := &models.Organization{
		Name:   name,
		Active: true,
		State:  string(models.StateBerlin),
	}
	if err := db.Create(org).Error; err != nil {
		t.Fatalf("failed to create test organization: %v", err)
	}

	// Auto-create a default section (mirrors production behavior)
	section := &models.Section{
		OrganizationID: org.ID,
		Name:           "Default",
		IsDefault:      true,
		CreatedBy:      "test",
	}
	if err := db.Create(section).Error; err != nil {
		t.Fatalf("failed to create default section for test organization: %v", err)
	}

	return org
}

// CreateTestUser creates a user for testing.
func CreateTestUser(t *testing.T, db *gorm.DB, name, email, password string) *models.User {
	t.Helper()

	user := &models.User{
		Name:     name,
		Email:    email,
		Password: password,
		Active:   true,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

// CreateTestUserOrganization creates a user-organization membership for testing.
func CreateTestUserOrganization(t *testing.T, db *gorm.DB, userID, orgID uint, role models.Role) *models.UserOrganization {
	t.Helper()

	uo := &models.UserOrganization{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           role,
		CreatedBy:      "test@example.com",
	}
	if err := db.Create(uo).Error; err != nil {
		t.Fatalf("failed to create test user organization: %v", err)
	}
	return uo
}

// CreateTestSection creates a section for testing.
func CreateTestSection(t *testing.T, db *gorm.DB, name string, orgID uint, isDefault bool) *models.Section {
	t.Helper()

	section := &models.Section{
		OrganizationID: orgID,
		Name:           name,
		IsDefault:      isDefault,
		CreatedBy:      "test",
	}
	if err := db.Create(section).Error; err != nil {
		t.Fatalf("failed to create test section: %v", err)
	}
	return section
}

// CreateTestChild creates a child for testing.
func CreateTestChild(t *testing.T, db *gorm.DB, firstName, lastName string, orgID uint) *models.Child {
	t.Helper()

	child := &models.Child{
		Person: models.Person{
			OrganizationID: orgID,
			FirstName:      firstName,
			LastName:       lastName,
			Birthdate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	if err := db.Create(child).Error; err != nil {
		t.Fatalf("failed to create test child: %v", err)
	}
	return child
}

// CreateTestEmployee creates an employee for testing.
func CreateTestEmployee(t *testing.T, db *gorm.DB, firstName, lastName string, orgID uint) *models.Employee {
	t.Helper()

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: orgID,
			FirstName:      firstName,
			LastName:       lastName,
			Birthdate:      time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	if err := db.Create(employee).Error; err != nil {
		t.Fatalf("failed to create test employee: %v", err)
	}
	return employee
}

// CreateTestPayPlan creates a pay plan for testing.
func CreateTestPayPlan(t *testing.T, db *gorm.DB, name string, orgID uint) *models.PayPlan {
	t.Helper()

	payPlan := &models.PayPlan{
		OrganizationID: orgID,
		Name:           name,
	}
	if err := db.Create(payPlan).Error; err != nil {
		t.Fatalf("failed to create test pay plan: %v", err)
	}
	return payPlan
}
