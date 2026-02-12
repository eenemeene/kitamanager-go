package store

import (
	"testing"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/testutil"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.SetupTestDB(t)
}

// createTestOrganization creates an organization for testing.
func createTestOrganization(t *testing.T, db *gorm.DB, name string) *models.Organization {
	t.Helper()
	return testutil.CreateTestOrganization(t, db, name)
}

// createTestUser creates a user for testing.
func createTestUser(t *testing.T, db *gorm.DB, name, email string) *models.User {
	t.Helper()
	return testutil.CreateTestUser(t, db, name, email, "hashedpassword")
}

// createTestGroup creates a group for testing.
func createTestGroup(t *testing.T, db *gorm.DB, name string) *models.Group {
	t.Helper()
	return testutil.CreateTestGroup(t, db, name)
}

// createTestGroupWithOrg creates a group for testing with a specific organization.
func createTestGroupWithOrg(t *testing.T, db *gorm.DB, name string, orgID uint) *models.Group {
	t.Helper()
	return testutil.CreateTestGroupWithOrg(t, db, name, orgID)
}

// createTestUserGroup creates a user-group relationship for testing.
func createTestUserGroup(t *testing.T, db *gorm.DB, userID, groupID uint, role models.Role) *models.UserGroup {
	t.Helper()
	return testutil.CreateTestUserGroup(t, db, userID, groupID, role)
}
