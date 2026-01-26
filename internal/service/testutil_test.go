package service

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

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
func createTestUser(t *testing.T, db *gorm.DB, name, email, password string) *models.User {
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

// createTestGroup creates a group for testing.
func createTestGroup(t *testing.T, db *gorm.DB, name string) *models.Group {
	t.Helper()

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

// createTestGroupWithOrgAndDefault creates a group with specific organization and default flag.
func createTestGroupWithOrgAndDefault(t *testing.T, db *gorm.DB, name string, orgID uint, isDefault bool) *models.Group {
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

// createTestChild creates a child for testing.
func createTestChild(t *testing.T, db *gorm.DB, firstName, lastName string, orgID uint) *models.Child {
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

// createTestEmployee creates an employee for testing.
func createTestEmployee(t *testing.T, db *gorm.DB, firstName, lastName string, orgID uint) *models.Employee {
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

// Service creation helpers

func createOrganizationService(db *gorm.DB) *OrganizationService {
	orgStore := store.NewOrganizationStore(db)
	groupStore := store.NewGroupStore(db)
	return NewOrganizationService(orgStore, groupStore)
}

func createUserService(db *gorm.DB) *UserService {
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	return NewUserService(userStore, groupStore)
}

func createUserGroupService(db *gorm.DB) *UserGroupService {
	userGroupStore := store.NewUserGroupStore(db)
	userStore := store.NewUserStore(db)
	groupStore := store.NewGroupStore(db)
	return NewUserGroupService(userGroupStore, userStore, groupStore)
}

func createGroupService(db *gorm.DB) *GroupService {
	groupStore := store.NewGroupStore(db)
	return NewGroupService(groupStore)
}

func createChildService(db *gorm.DB) *ChildService {
	childStore := store.NewChildStore(db)
	groupStore := store.NewGroupStore(db)
	return NewChildService(childStore, groupStore)
}

func createEmployeeService(db *gorm.DB) *EmployeeService {
	employeeStore := store.NewEmployeeStore(db)
	return NewEmployeeService(employeeStore)
}
