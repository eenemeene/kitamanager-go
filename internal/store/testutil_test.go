package store

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// AutoMigrate all models
	err = db.AutoMigrate(
		&models.Organization{},
		&models.User{},
		&models.Group{},
		&models.UserGroup{},
		&models.Employee{},
		&models.EmployeeContract{},
		&models.Child{},
		&models.ChildContract{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

// createTestOrganization creates an organization for testing.
func createTestOrganization(t *testing.T, db *gorm.DB, name string) *models.Organization {
	t.Helper()

	org := &models.Organization{
		Name:   name,
		Active: true,
	}
	if err := db.Create(org).Error; err != nil {
		t.Fatalf("failed to create test organization: %v", err)
	}
	return org
}

// createTestUser creates a user for testing.
func createTestUser(t *testing.T, db *gorm.DB, name, email string) *models.User {
	t.Helper()

	user := &models.User{
		Name:     name,
		Email:    email,
		Password: "hashedpassword",
		Active:   true,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

// createTestGroup creates a group for testing.
// It creates a default organization for the group.
func createTestGroup(t *testing.T, db *gorm.DB, name string) *models.Group {
	t.Helper()

	// Create a default organization for the group
	org := createTestOrganization(t, db, name+" Org")

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

// createTestGroupWithOrg creates a group for testing with a specific organization.
func createTestGroupWithOrg(t *testing.T, db *gorm.DB, name string, orgID uint) *models.Group {
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

// createTestUserGroup creates a user-group relationship for testing.
func createTestUserGroup(t *testing.T, db *gorm.DB, userID, groupID uint, role models.Role) *models.UserGroup {
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
