package testutil

import (
	"context"
	"fmt"
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

var (
	once         sync.Once
	pgContainer  *postgres.PostgresContainer
	containerDSN string
	sharedDB     *gorm.DB
	initErr      error
)

// truncation order: leaf tables first, then parents
var truncateTables = []string{
	"revoked_tokens",
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
	"user_groups",
	"groups",
	"users",
	"organizations",
}

func startContainer() {
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

// CreateTestGroup creates a group for testing. It creates a default organization for the group.
func CreateTestGroup(t *testing.T, db *gorm.DB, name string) *models.Group {
	t.Helper()

	org := CreateTestOrganization(t, db, name+" Org")

	group := &models.Group{
		Name:           name,
		OrganizationID: org.ID,
		Active:         true,
	}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("failed to create test group: %v", err)
	}
	return group
}

// CreateTestGroupWithOrg creates a group for testing with a specific organization.
func CreateTestGroupWithOrg(t *testing.T, db *gorm.DB, name string, orgID uint) *models.Group {
	t.Helper()

	group := &models.Group{
		Name:           name,
		OrganizationID: orgID,
		Active:         true,
	}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("failed to create test group: %v", err)
	}
	return group
}

// CreateTestGroupWithOrgAndDefault creates a group with specific organization and default flag.
func CreateTestGroupWithOrgAndDefault(t *testing.T, db *gorm.DB, name string, orgID uint, isDefault bool) *models.Group {
	t.Helper()

	group := &models.Group{
		Name:           name,
		OrganizationID: orgID,
		IsDefault:      isDefault,
		Active:         true,
	}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("failed to create test group: %v", err)
	}
	return group
}

// CreateTestUserGroup creates a user-group relationship for testing.
func CreateTestUserGroup(t *testing.T, db *gorm.DB, userID, groupID uint, role models.Role) *models.UserGroup {
	t.Helper()

	ug := &models.UserGroup{
		UserID:    userID,
		GroupID:   groupID,
		Role:      role,
		CreatedBy: "test@example.com",
	}
	if err := db.Create(ug).Error; err != nil {
		t.Fatalf("failed to create test user group: %v", err)
	}
	return ug
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
