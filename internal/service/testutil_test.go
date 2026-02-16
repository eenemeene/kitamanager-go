package service

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
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
func createTestUser(t *testing.T, db *gorm.DB, name, email, password string) *models.User {
	t.Helper()
	return testutil.CreateTestUser(t, db, name, email, password)
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

// createTestGroupWithOrgAndDefault creates a group with specific organization and default flag.
func createTestGroupWithOrgAndDefault(t *testing.T, db *gorm.DB, name string, orgID uint, isDefault bool) *models.Group {
	t.Helper()
	return testutil.CreateTestGroupWithOrgAndDefault(t, db, name, orgID, isDefault)
}

// createTestUserGroup creates a user-group relationship for testing.
func createTestUserGroup(t *testing.T, db *gorm.DB, userID, groupID uint, role models.Role) *models.UserGroup {
	t.Helper()
	return testutil.CreateTestUserGroup(t, db, userID, groupID, role)
}

// createTestChild creates a child for testing.
func createTestChild(t *testing.T, db *gorm.DB, firstName, lastName string, orgID uint) *models.Child {
	t.Helper()
	return testutil.CreateTestChild(t, db, firstName, lastName, orgID)
}

// createTestEmployee creates an employee for testing.
func createTestEmployee(t *testing.T, db *gorm.DB, firstName, lastName string, orgID uint) *models.Employee {
	t.Helper()
	return testutil.CreateTestEmployee(t, db, firstName, lastName, orgID)
}

// createTestSection creates a section for testing.
func createTestSection(t *testing.T, db *gorm.DB, name string, orgID uint, isDefault bool) *models.Section {
	t.Helper()
	return testutil.CreateTestSection(t, db, name, orgID, isDefault)
}

// getDefaultSection returns the default section already created by createTestOrganization.
func getDefaultSection(t *testing.T, db *gorm.DB, orgID uint) *models.Section {
	t.Helper()
	var section models.Section
	if err := db.Where("organization_id = ? AND is_default = ?", orgID, true).First(&section).Error; err != nil {
		t.Fatalf("failed to find default section: %v", err)
	}
	return &section
}

// createTestChildWithContract creates a child with an active contract assigned to a specific section.
func createTestChildWithContract(t *testing.T, db *gorm.DB, firstName, lastName string, orgID, sectionID uint) *models.Child {
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
	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			SectionID: sectionID,
		},
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("failed to create test child contract: %v", err)
	}
	return child
}

// createTestSuperAdmin creates a superadmin user for authorization tests.
func createTestSuperAdmin(t *testing.T, db *gorm.DB) *models.User {
	t.Helper()
	user := testutil.CreateTestUser(t, db, "Super Admin", "superadmin@example.com", "password")
	if err := db.Model(user).Update("is_superadmin", true).Error; err != nil {
		t.Fatalf("failed to set superadmin: %v", err)
	}
	user.IsSuperAdmin = true
	return user
}

// Service creation helpers

func createOrganizationService(db *gorm.DB) *OrganizationService {
	orgStore := store.NewOrganizationStore(db)
	groupStore := store.NewGroupStore(db)
	userStore := store.NewUserStore(db)
	return NewOrganizationService(orgStore, groupStore, userStore)
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
	orgStore := store.NewOrganizationStore(db)
	fundingStore := store.NewGovernmentFundingStore(db)
	sectionStore := store.NewSectionStore(db)
	transactor := store.NewTransactor(db)
	return NewChildService(childStore, orgStore, fundingStore, sectionStore, transactor)
}

func createEmployeeService(db *gorm.DB) *EmployeeService {
	employeeStore := store.NewEmployeeStore(db)
	payPlanStore := store.NewPayPlanStore(db)
	sectionStore := store.NewSectionStore(db)
	transactor := store.NewTransactor(db)
	return NewEmployeeService(employeeStore, payPlanStore, sectionStore, transactor)
}

// createTestPayPlan creates a pay plan for testing.
func createTestPayPlan(t *testing.T, db *gorm.DB, name string, orgID uint) *models.PayPlan {
	t.Helper()
	return testutil.CreateTestPayPlan(t, db, name, orgID)
}

// createTestPayPlanPeriod creates a pay plan period for testing.
func createTestPayPlanPeriod(t *testing.T, db *gorm.DB, payplanID uint, from time.Time, to *time.Time, weeklyHours float64) *models.PayPlanPeriod {
	t.Helper()

	period := &models.PayPlanPeriod{
		PayPlanID:   payplanID,
		Period:      models.Period{From: from, To: to},
		WeeklyHours: weeklyHours,
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("failed to create test pay plan period: %v", err)
	}
	return period
}

// createTestPayPlanEntry creates a pay plan entry for testing.
func createTestPayPlanEntry(t *testing.T, db *gorm.DB, periodID uint, grade string, step int, monthlyAmount int, stepMinYears *int) *models.PayPlanEntry {
	t.Helper()

	entry := &models.PayPlanEntry{
		PeriodID:      periodID,
		Grade:         grade,
		Step:          step,
		MonthlyAmount: monthlyAmount,
		StepMinYears:  stepMinYears,
	}
	if err := db.Create(entry).Error; err != nil {
		t.Fatalf("failed to create test pay plan entry: %v", err)
	}
	return entry
}

// createTestEmployeeContract creates an employee contract for testing.
func createTestEmployeeContract(t *testing.T, db *gorm.DB, employeeID uint, payplanID uint, from time.Time, to *time.Time, grade string, step int, weeklyHours float64) *models.EmployeeContract {
	t.Helper()

	contract := &models.EmployeeContract{
		EmployeeID: employeeID,
		BaseContract: models.BaseContract{
			Period: models.Period{From: from, To: to},
		},
		StaffCategory: "qualified",
		Grade:         grade,
		Step:          step,
		WeeklyHours:   weeklyHours,
		PayPlanID:     payplanID,
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("failed to create test employee contract: %v", err)
	}
	return contract
}

func createSectionService(db *gorm.DB) *SectionService {
	sectionStore := store.NewSectionStore(db)
	return NewSectionService(sectionStore)
}

// createPayPlanService creates a PayPlanService for testing.
func createPayPlanService(db *gorm.DB) *PayPlanService {
	payPlanStore := store.NewPayPlanStore(db)
	return NewPayPlanService(payPlanStore)
}

// createStepPromotionService creates a StepPromotionService for testing.
func createStepPromotionService(db *gorm.DB) *StepPromotionService {
	payPlanStore := store.NewPayPlanStore(db)
	employeeStore := store.NewEmployeeStore(db)
	return NewStepPromotionService(payPlanStore, employeeStore)
}

// createTestBudgetItem creates a budget item for testing.
func createTestBudgetItem(t *testing.T, db *gorm.DB, name string, orgID uint, category string, perChild bool) *models.BudgetItem {
	t.Helper()
	item := &models.BudgetItem{
		OrganizationID: orgID,
		Name:           name,
		Category:       category,
		PerChild:       perChild,
	}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("failed to create test budget item: %v", err)
	}
	return item
}

// createTestBudgetItemEntry creates a budget item entry for testing.
func createTestBudgetItemEntry(t *testing.T, db *gorm.DB, budgetItemID uint, from time.Time, to *time.Time, amountCents int, notes string) *models.BudgetItemEntry {
	t.Helper()
	entry := &models.BudgetItemEntry{
		BudgetItemID: budgetItemID,
		Period:       models.Period{From: from, To: to},
		AmountCents:  amountCents,
		Notes:        notes,
	}
	if err := db.Create(entry).Error; err != nil {
		t.Fatalf("failed to create test budget item entry: %v", err)
	}
	return entry
}

// createTestFundingPropertyFull creates a funding property with both payment and requirement.
func createTestFundingPropertyFull(t *testing.T, db *gorm.DB, periodID uint, key, value string, payment int, requirement float64, minAge, maxAge int) *models.GovernmentFundingProperty {
	t.Helper()
	var minAgePtr, maxAgePtr *int
	if minAge >= 0 {
		minAgePtr = &minAge
	}
	if maxAge >= 0 {
		maxAgePtr = &maxAge
	}
	prop := &models.GovernmentFundingProperty{
		PeriodID:    periodID,
		Key:         key,
		Value:       value,
		Payment:     payment,
		Requirement: requirement,
		MinAge:      minAgePtr,
		MaxAge:      maxAgePtr,
	}
	if err := db.Create(prop).Error; err != nil {
		t.Fatalf("failed to create test funding property: %v", err)
	}
	return prop
}

// createTestPayPlanPeriodWithContrib creates a pay plan period with employer contribution rate.
func createTestPayPlanPeriodWithContrib(t *testing.T, db *gorm.DB, payplanID uint, from time.Time, to *time.Time, weeklyHours float64, contribRate int) *models.PayPlanPeriod {
	t.Helper()
	period := &models.PayPlanPeriod{
		PayPlanID:                payplanID,
		Period:                   models.Period{From: from, To: to},
		WeeklyHours:              weeklyHours,
		EmployerContributionRate: contribRate,
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("failed to create test pay plan period: %v", err)
	}
	return period
}

// createTestGovernmentFunding creates a government funding plan for testing.
func createTestGovernmentFunding(t *testing.T, db *gorm.DB, name string) *models.GovernmentFunding {
	t.Helper()

	funding := &models.GovernmentFunding{
		Name:  name,
		State: string(models.StateBerlin),
	}
	if err := db.Create(funding).Error; err != nil {
		t.Fatalf("failed to create test government funding: %v", err)
	}
	return funding
}

// createTestFundingPeriod creates a funding period for testing.
func createTestFundingPeriod(t *testing.T, db *gorm.DB, fundingID uint, from time.Time, to *time.Time, fullTimeWeeklyHours float64) *models.GovernmentFundingPeriod {
	t.Helper()

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: fundingID,
		Period:              models.Period{From: from, To: to},
		FullTimeWeeklyHours: fullTimeWeeklyHours,
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("failed to create test funding period: %v", err)
	}
	return period
}

// createTestFundingProperty creates a funding property for testing.
func createTestFundingProperty(t *testing.T, db *gorm.DB, periodID uint, key, value string, payment int, minAge, maxAge int) *models.GovernmentFundingProperty {
	t.Helper()

	var minAgePtr, maxAgePtr *int
	if minAge >= 0 {
		minAgePtr = &minAge
	}
	if maxAge >= 0 {
		maxAgePtr = &maxAge
	}

	prop := &models.GovernmentFundingProperty{
		PeriodID:    periodID,
		Key:         key,
		Value:       value,
		Payment:     payment,
		Requirement: 0.1,
		MinAge:      minAgePtr,
		MaxAge:      maxAgePtr,
	}
	if err := db.Create(prop).Error; err != nil {
		t.Fatalf("failed to create test funding property: %v", err)
	}
	return prop
}
