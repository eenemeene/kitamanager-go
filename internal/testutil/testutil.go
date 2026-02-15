package testutil

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// SetupTestDB creates an in-memory SQLite database for testing with all models migrated.
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// SQLite :memory: databases are per-connection. Limit to 1 connection
	// so all operations use the same database with the migrated schema.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	err = db.AutoMigrate(
		&models.Organization{},
		&models.User{},
		&models.Group{},
		&models.Section{},
		&models.UserGroup{},
		&models.Employee{},
		&models.EmployeeContract{},
		&models.Child{},
		&models.ChildContract{},
		&models.GovernmentFunding{},
		&models.GovernmentFundingPeriod{},
		&models.GovernmentFundingProperty{},
		&models.PayPlan{},
		&models.PayPlanPeriod{},
		&models.PayPlanEntry{},
		&models.AuditLog{},
		&models.ChildAttendance{},
		&models.Cost{},
		&models.CostEntry{},
		&models.RevokedToken{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
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
