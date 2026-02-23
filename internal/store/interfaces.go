package store

import (
	"context"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// UserOrganizationStorer defines the interface for user-organization relationship operations
type UserOrganizationStorer interface {
	AddUserToOrg(ctx context.Context, userID, orgID uint, role models.Role, createdBy string) (*models.UserOrganization, error)
	UpdateRole(ctx context.Context, userID, orgID uint, role models.Role) error
	RemoveUserFromOrg(ctx context.Context, userID, orgID uint) error
	FindByUserAndOrg(ctx context.Context, userID, orgID uint) (*models.UserOrganization, error)
	FindByUser(ctx context.Context, userID uint) ([]models.UserOrganization, error)
	GetRoleInOrg(ctx context.Context, userID, orgID uint) (models.Role, error)
	GetUserOrganizationsWithRoles(ctx context.Context, userID uint) (map[uint]models.Role, error)
	SetSuperAdmin(ctx context.Context, userID uint, isSuperAdmin bool) error
	IsSuperAdmin(ctx context.Context, userID uint) (bool, error)
	Exists(ctx context.Context, userID, orgID uint) (bool, error)
}

// UserStorer defines the interface for user storage operations
type UserStorer interface {
	FindAll(ctx context.Context, search string, limit, offset int) ([]models.User, int64, error)
	FindByOrganization(ctx context.Context, orgID uint, search string, limit, offset int) ([]models.User, int64, error)
	FindByID(ctx context.Context, id uint) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	EmailExistsForOtherUser(ctx context.Context, email string, excludeUserID uint) (bool, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	UpdateLastLogin(ctx context.Context, userID uint) error
	Delete(ctx context.Context, id uint) error
	GetUserOrganizations(ctx context.Context, userID uint) ([]models.Organization, error)
	FindByOrganizations(ctx context.Context, orgIDs []uint, search string, limit, offset int) ([]models.User, int64, error)
	SharesOrganization(ctx context.Context, userID1, userID2 uint) (bool, error)
}

// OrganizationStorer defines the interface for organization storage operations
type OrganizationStorer interface {
	FindAll(ctx context.Context, search string, limit, offset int) ([]models.Organization, int64, error)
	FindByID(ctx context.Context, id uint) (*models.Organization, error)
	Create(ctx context.Context, org *models.Organization) error
	CreateWithDefaultSection(ctx context.Context, org *models.Organization, defaultSection *models.Section) error
	Update(ctx context.Context, org *models.Organization) error
	Delete(ctx context.Context, id uint) error
}

// EmployeeStorer defines the interface for employee storage operations
type EmployeeStorer interface {
	FindAll(ctx context.Context, limit, offset int) ([]models.Employee, int64, error)
	FindByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.Employee, int64, error)
	FindByOrganizationAndSection(ctx context.Context, orgID uint, sectionID *uint, activeOn *time.Time, search string, staffCategory *string, limit, offset int) ([]models.Employee, int64, error)
	FindByID(ctx context.Context, id uint) (*models.Employee, error)
	FindByIDAndOrg(ctx context.Context, id, orgID uint) (*models.Employee, error)
	FindByIDMinimal(ctx context.Context, id uint) (*models.Employee, error) // Without preloads, for org checks
	FindByIDMinimalAndOrg(ctx context.Context, id, orgID uint) (*models.Employee, error)
	Create(ctx context.Context, emp *models.Employee) error
	Update(ctx context.Context, emp *models.Employee) error
	Delete(ctx context.Context, id uint) error
	CreateContract(ctx context.Context, contract *models.EmployeeContract) error
	FindContractByID(ctx context.Context, id uint) (*models.EmployeeContract, error)
	UpdateContract(ctx context.Context, contract *models.EmployeeContract) error
	DeleteContract(ctx context.Context, id uint) error
	Contracts() PeriodStorer[models.EmployeeContract]
	FindByOrganizationWithContracts(ctx context.Context, orgID uint, date time.Time) ([]models.Employee, error)
	FindContractsByEmployeePaginated(ctx context.Context, employeeID uint, limit, offset int) ([]models.EmployeeContract, int64, error)
	FindContractsByOrganizationInDateRange(ctx context.Context, orgID uint, rangeStart, rangeEnd time.Time, staffCategories []string, sectionID *uint) ([]models.EmployeeContract, error)
	FindByOrganizationInDateRange(ctx context.Context, orgID uint, rangeStart, rangeEnd time.Time, sectionID *uint) ([]models.Employee, error)
	FindByNameBirthdateAndOrg(ctx context.Context, firstName, lastName string, birthdate time.Time, orgID uint) (*models.Employee, error)
	DeleteContractsByEmployee(ctx context.Context, employeeID uint) error
}

// ChildStorer defines the interface for child storage operations
type ChildStorer interface {
	FindAll(ctx context.Context, limit, offset int) ([]models.Child, int64, error)
	FindByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.Child, int64, error)
	FindByOrganizationAndSection(ctx context.Context, orgID uint, sectionID *uint, activeOn *time.Time, contractAfter *time.Time, search string, limit, offset int) ([]models.Child, int64, error)
	FindByOrganizationWithActiveOn(ctx context.Context, orgID uint, date time.Time) ([]models.Child, error)
	CountByOrganizationWithActiveOn(ctx context.Context, orgID uint, date time.Time) (int64, error)
	FindContractsByOrganizationInDateRange(ctx context.Context, orgID uint, rangeStart, rangeEnd time.Time) ([]models.ChildContract, error)
	FindByOrganizationInDateRange(ctx context.Context, orgID uint, rangeStart, rangeEnd time.Time, sectionID *uint) ([]models.Child, error)
	FindByID(ctx context.Context, id uint) (*models.Child, error)
	FindByIDAndOrg(ctx context.Context, id, orgID uint) (*models.Child, error)
	FindByIDMinimal(ctx context.Context, id uint) (*models.Child, error) // Without preloads, for org checks
	FindByIDMinimalAndOrg(ctx context.Context, id, orgID uint) (*models.Child, error)
	Create(ctx context.Context, child *models.Child) error
	Update(ctx context.Context, child *models.Child) error
	Delete(ctx context.Context, id uint) error
	CreateContract(ctx context.Context, contract *models.ChildContract) error
	FindContractByID(ctx context.Context, id uint) (*models.ChildContract, error)
	UpdateContract(ctx context.Context, contract *models.ChildContract) error
	DeleteContract(ctx context.Context, id uint) error
	Contracts() PeriodStorer[models.ChildContract]
	FindContractsByChildPaginated(ctx context.Context, childID uint, limit, offset int) ([]models.ChildContract, int64, error)
	FindContractsByVoucherNumbers(ctx context.Context, orgID uint, voucherNumbers []string, activeOn time.Time) ([]models.ChildContract, error)
	FindByNameBirthdateAndOrg(ctx context.Context, firstName, lastName string, birthdate time.Time, orgID uint) (*models.Child, error)
	DeleteContractsByChild(ctx context.Context, childID uint) error
}

// PeriodStorer defines the interface for time-bounded record operations
type PeriodStorer[T models.PeriodRecord] interface {
	GetCurrentRecord(ctx context.Context, ownerID uint) (*T, error)
	GetRecordOn(ctx context.Context, ownerID uint, date time.Time) (*T, error)
	ListRecordsPaginated(ctx context.Context, ownerID uint, limit, offset int) ([]T, int64, error)
	HasActiveRecord(ctx context.Context, ownerID uint, date time.Time) (bool, error)
	ValidateNoOverlap(ctx context.Context, ownerID uint, from time.Time, to *time.Time, excludeID *uint) error
	CloseCurrentRecord(ctx context.Context, ownerID uint, endDate time.Time) error
}

// SectionStorer defines the interface for section storage operations
type SectionStorer interface {
	FindByID(ctx context.Context, id uint) (*models.Section, error)
	FindByOrganizationPaginated(ctx context.Context, orgID uint, search string, limit, offset int) ([]models.Section, int64, error)
	FindDefaultSection(ctx context.Context, orgID uint) (*models.Section, error)
	FindByNameAndOrg(ctx context.Context, name string, orgID uint) (*models.Section, error)
	Create(ctx context.Context, section *models.Section) error
	Update(ctx context.Context, section *models.Section) error
	Delete(ctx context.Context, id uint) error
	HasChildren(ctx context.Context, id uint) (bool, error)
	HasEmployees(ctx context.Context, id uint) (bool, error)
}

// GovernmentFundingStorer defines the interface for government funding storage operations
type GovernmentFundingStorer interface {
	// GovernmentFunding CRUD
	FindAll(ctx context.Context, limit, offset int) ([]models.GovernmentFunding, int64, error)
	FindByID(ctx context.Context, id uint) (*models.GovernmentFunding, error)
	FindByIDWithDetails(ctx context.Context, id uint, periodsLimit int, activeOn *time.Time) (*models.GovernmentFunding, error)
	FindByState(ctx context.Context, state string) (*models.GovernmentFunding, error)
	FindByStateWithDetails(ctx context.Context, state string, periodsLimit int, activeOn *time.Time) (*models.GovernmentFunding, error)
	CountPeriods(ctx context.Context, fundingID uint) (int64, error)
	Create(ctx context.Context, funding *models.GovernmentFunding) error
	Update(ctx context.Context, funding *models.GovernmentFunding) error
	Delete(ctx context.Context, id uint) error

	// Period CRUD
	FindPeriodByID(ctx context.Context, id uint) (*models.GovernmentFundingPeriod, error)
	FindPeriodsByGovernmentFundingID(ctx context.Context, fundingID uint) ([]models.GovernmentFundingPeriod, error)
	CreatePeriod(ctx context.Context, period *models.GovernmentFundingPeriod) error
	UpdatePeriod(ctx context.Context, period *models.GovernmentFundingPeriod) error
	DeletePeriod(ctx context.Context, id uint) error

	// Property CRUD
	FindPropertyByID(ctx context.Context, id uint) (*models.GovernmentFundingProperty, error)
	CreateProperty(ctx context.Context, property *models.GovernmentFundingProperty) error
	UpdateProperty(ctx context.Context, property *models.GovernmentFundingProperty) error
	DeleteProperty(ctx context.Context, id uint) error
}

// ChildAttendanceStorer defines the interface for child attendance storage operations
type ChildAttendanceStorer interface {
	FindByID(ctx context.Context, id uint) (*models.ChildAttendance, error)
	FindByOrganizationAndDate(ctx context.Context, orgID uint, date time.Time, limit, offset int) ([]models.ChildAttendance, int64, error)
	FindByChildAndDate(ctx context.Context, childID uint, date time.Time) (*models.ChildAttendance, error)
	FindByChildAndDateRange(ctx context.Context, childID uint, from, to time.Time, limit, offset int) ([]models.ChildAttendance, int64, error)
	Create(ctx context.Context, attendance *models.ChildAttendance) error
	Update(ctx context.Context, attendance *models.ChildAttendance) error
	Delete(ctx context.Context, id uint) error
	GetDailySummary(ctx context.Context, orgID uint, date time.Time) (*models.ChildAttendanceDailySummaryResponse, error)
}

// PayPlanStorer defines the interface for pay plan storage operations
type PayPlanStorer interface {
	Create(ctx context.Context, payplan *models.PayPlan) error
	FindByID(ctx context.Context, id uint) (*models.PayPlan, error)
	FindByIDWithPeriods(ctx context.Context, id uint, activeOn *time.Time) (*models.PayPlan, error)
	FindByIDsWithPeriods(ctx context.Context, ids []uint) (map[uint]*models.PayPlan, error)
	FindByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.PayPlan, int64, error)
	FindByNameAndOrg(ctx context.Context, name string, orgID uint) (*models.PayPlan, error)
	Update(ctx context.Context, payplan *models.PayPlan) error
	Delete(ctx context.Context, id uint) error
	DeletePeriodsAndEntries(ctx context.Context, payplanID uint) error

	// Period operations
	CreatePeriod(ctx context.Context, period *models.PayPlanPeriod) error
	FindPeriodByID(ctx context.Context, id uint) (*models.PayPlanPeriod, error)
	FindPeriodByIDWithEntries(ctx context.Context, id uint) (*models.PayPlanPeriod, error)
	FindPeriodsByPayPlan(ctx context.Context, payplanID uint) ([]models.PayPlanPeriod, error)
	FindActivePeriod(ctx context.Context, payplanID uint, date time.Time) (*models.PayPlanPeriod, error)
	UpdatePeriod(ctx context.Context, period *models.PayPlanPeriod) error
	DeletePeriod(ctx context.Context, id uint) error

	// Entry operations
	CreateEntry(ctx context.Context, entry *models.PayPlanEntry) error
	FindEntryByID(ctx context.Context, id uint) (*models.PayPlanEntry, error)
	FindEntriesByPeriod(ctx context.Context, periodID uint) ([]models.PayPlanEntry, error)
	FindEntry(ctx context.Context, periodID uint, grade string, step int) (*models.PayPlanEntry, error)
	UpdateEntry(ctx context.Context, entry *models.PayPlanEntry) error
	DeleteEntry(ctx context.Context, id uint) error
}

// BudgetItemStorer defines the interface for budget item storage operations
type BudgetItemStorer interface {
	Create(ctx context.Context, item *models.BudgetItem) error
	FindByID(ctx context.Context, id uint) (*models.BudgetItem, error)
	FindByIDWithEntries(ctx context.Context, id uint) (*models.BudgetItem, error)
	FindByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.BudgetItem, int64, error)
	FindByOrganizationWithEntries(ctx context.Context, orgID uint) ([]models.BudgetItem, error)
	Update(ctx context.Context, item *models.BudgetItem) error
	Delete(ctx context.Context, id uint) error

	// Entry operations
	CreateEntry(ctx context.Context, entry *models.BudgetItemEntry) error
	FindEntryByID(ctx context.Context, id uint) (*models.BudgetItemEntry, error)
	FindEntriesByBudgetItemPaginated(ctx context.Context, budgetItemID uint, limit, offset int) ([]models.BudgetItemEntry, int64, error)
	UpdateEntry(ctx context.Context, entry *models.BudgetItemEntry) error
	DeleteEntry(ctx context.Context, id uint) error
	Entries() PeriodStorer[models.BudgetItemEntry]
}

// AuditStorer defines the interface for audit log storage operations
type AuditStorer interface {
	Create(ctx context.Context, log *models.AuditLog) error
	FindByUser(ctx context.Context, userID uint, limit, offset int) ([]models.AuditLog, int64, error)
	FindByAction(ctx context.Context, action models.AuditAction, limit, offset int) ([]models.AuditLog, int64, error)
	FindByDateRange(ctx context.Context, from, to time.Time, limit, offset int) ([]models.AuditLog, int64, error)
	FindAll(ctx context.Context, limit, offset int) ([]models.AuditLog, int64, error)
	FindFailedLogins(ctx context.Context, email string, since time.Time, limit int) ([]models.AuditLog, error)
	CountFailedLoginsSince(ctx context.Context, email string, since time.Time) (int64, error)
	Cleanup(ctx context.Context, olderThan time.Time) (int64, error)
}

// GovernmentFundingBillPeriodStorer defines the interface for government funding bill period storage operations
type GovernmentFundingBillPeriodStorer interface {
	Create(ctx context.Context, period *models.GovernmentFundingBillPeriod) error
	FindByID(ctx context.Context, id uint) (*models.GovernmentFundingBillPeriod, error)
	FindByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.GovernmentFundingBillPeriod, int64, error)
	FindByOrganizationAndVoucherNumber(ctx context.Context, orgID uint, voucherNumber string) ([]models.BillAppearance, error)
	Delete(ctx context.Context, id uint) error
}

// Compile-time interface compliance checks
var (
	_ UserStorer                        = (*UserStore)(nil)
	_ OrganizationStorer                = (*OrganizationStore)(nil)
	_ EmployeeStorer                    = (*EmployeeStore)(nil)
	_ ChildStorer                       = (*ChildStore)(nil)
	_ UserOrganizationStorer            = (*UserOrganizationStore)(nil)
	_ GovernmentFundingStorer           = (*GovernmentFundingStore)(nil)
	_ GovernmentFundingBillPeriodStorer = (*GovernmentFundingBillPeriodStore)(nil)
	_ SectionStorer                     = (*SectionStore)(nil)
	_ ChildAttendanceStorer             = (*ChildAttendanceStore)(nil)
	_ PayPlanStorer                     = (*PayPlanStore)(nil)
	_ AuditStorer                       = (*AuditStore)(nil)
	_ BudgetItemStorer                  = (*BudgetItemStore)(nil)
	_ Transactor                        = (*GormTransactor)(nil)
)
