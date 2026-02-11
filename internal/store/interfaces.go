package store

import (
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// UserGroupStorer defines the interface for user-group relationship operations
type UserGroupStorer interface {
	AddUserToGroup(userID, groupID uint, role models.Role, createdBy string) (*models.UserGroup, error)
	UpdateRole(userID, groupID uint, role models.Role) error
	RemoveUserFromGroup(userID, groupID uint) error
	FindByUserAndGroup(userID, groupID uint) (*models.UserGroup, error)
	FindByUser(userID uint) ([]models.UserGroup, error)
	FindByGroup(groupID uint) ([]models.UserGroup, error)
	FindByUserAndOrg(userID, orgID uint) ([]models.UserGroup, error)
	GetEffectiveRoleInOrg(userID, orgID uint) (models.Role, error)
	GetUserOrganizationsWithRoles(userID uint) (map[uint]models.Role, error)
	RemoveUserFromOrg(userID, orgID uint) error
	SetSuperAdmin(userID uint, isSuperAdmin bool) error
	IsSuperAdmin(userID uint) (bool, error)
	Exists(userID, groupID uint) (bool, error)
}

// UserStorer defines the interface for user storage operations
type UserStorer interface {
	FindAll(limit, offset int) ([]models.User, int64, error)
	FindByOrganization(orgID uint, limit, offset int) ([]models.User, int64, error)
	FindByID(id uint) (*models.User, error)
	FindByEmail(email string) (*models.User, error)
	EmailExistsForOtherUser(email string, excludeUserID uint) (bool, error)
	Create(user *models.User) error
	Update(user *models.User) error
	UpdateLastLogin(userID uint) error
	Delete(id uint) error
	AddToGroup(userID, groupID uint) error
	RemoveFromGroup(userID, groupID uint) error
	RemoveFromAllGroupsInOrg(userID, orgID uint) error
	GetUserOrganizations(userID uint) ([]models.Organization, error)
}

// OrganizationStorer defines the interface for organization storage operations
type OrganizationStorer interface {
	FindAll(limit, offset int) ([]models.Organization, int64, error)
	FindByID(id uint) (*models.Organization, error)
	Create(org *models.Organization) error
	CreateWithDefaultGroup(org *models.Organization, defaultGroup *models.Group) error
	Update(org *models.Organization) error
	Delete(id uint) error
}

// GroupStorer defines the interface for group storage operations
type GroupStorer interface {
	FindAll(limit, offset int) ([]models.Group, int64, error)
	FindByID(id uint) (*models.Group, error)
	FindByOrganization(orgID uint) ([]models.Group, error)
	FindByOrganizationPaginated(orgID uint, limit, offset int) ([]models.Group, int64, error)
	FindDefaultGroup(orgID uint) (*models.Group, error)
	Create(group *models.Group) error
	Update(group *models.Group) error
	Delete(id uint) error
}

// EmployeeStorer defines the interface for employee storage operations
type EmployeeStorer interface {
	FindAll(limit, offset int) ([]models.Employee, int64, error)
	FindByOrganization(orgID uint, limit, offset int) ([]models.Employee, int64, error)
	FindByOrganizationAndSection(orgID uint, sectionID *uint, activeOn *time.Time, limit, offset int) ([]models.Employee, int64, error)
	FindByID(id uint) (*models.Employee, error)
	FindByIDMinimal(id uint) (*models.Employee, error) // Without preloads, for org checks
	Create(emp *models.Employee) error
	Update(emp *models.Employee) error
	Delete(id uint) error
	CreateContract(contract *models.EmployeeContract) error
	FindContractByID(id uint) (*models.EmployeeContract, error)
	UpdateContract(contract *models.EmployeeContract) error
	DeleteContract(id uint) error
	Contracts() ContractStorer[models.EmployeeContract]
}

// ChildStorer defines the interface for child storage operations
type ChildStorer interface {
	FindAll(limit, offset int) ([]models.Child, int64, error)
	FindByOrganization(orgID uint, limit, offset int) ([]models.Child, int64, error)
	FindByOrganizationAndSection(orgID uint, sectionID *uint, activeOn *time.Time, limit, offset int) ([]models.Child, int64, error)
	FindByOrganizationWithActiveOn(orgID uint, date time.Time) ([]models.Child, error)
	CountByOrganizationWithActiveOn(orgID uint, date time.Time) (int64, error)
	FindByID(id uint) (*models.Child, error)
	FindByIDMinimal(id uint) (*models.Child, error) // Without preloads, for org checks
	Create(child *models.Child) error
	Update(child *models.Child) error
	Delete(id uint) error
	CreateContract(contract *models.ChildContract) error
	FindContractByID(id uint) (*models.ChildContract, error)
	UpdateContract(contract *models.ChildContract) error
	DeleteContract(id uint) error
	Contracts() ContractStorer[models.ChildContract]
}

// ContractStorer defines the interface for contract storage operations
type ContractStorer[T models.HasPeriod] interface {
	GetCurrentContract(personID uint) (*T, error)
	GetContractOn(personID uint, date time.Time) (*T, error)
	GetHistory(personID uint) ([]T, error)
	GetHistoryPaginated(personID uint, limit, offset int) ([]T, int64, error)
	HasActiveContract(personID uint, date time.Time) (bool, error)
	ValidateNoOverlap(personID uint, from time.Time, to *time.Time, excludeID *uint) error
	CloseCurrentContract(personID uint, endDate time.Time) error
}

// SectionStorer defines the interface for section storage operations
type SectionStorer interface {
	FindByID(id uint) (*models.Section, error)
	FindByOrganization(orgID uint) ([]models.Section, error)
	FindByOrganizationPaginated(orgID uint, limit, offset int) ([]models.Section, int64, error)
	FindDefaultSection(orgID uint) (*models.Section, error)
	FindByNameAndOrg(name string, orgID uint) (*models.Section, error)
	Create(section *models.Section) error
	Update(section *models.Section) error
	Delete(id uint) error
	HasChildren(id uint) (bool, error)
	HasEmployees(id uint) (bool, error)
}

// GovernmentFundingStorer defines the interface for government funding storage operations
type GovernmentFundingStorer interface {
	// GovernmentFunding CRUD
	FindAll(limit, offset int) ([]models.GovernmentFunding, int64, error)
	FindByID(id uint) (*models.GovernmentFunding, error)
	FindByIDWithDetails(id uint, periodsLimit int, activeOn *time.Time) (*models.GovernmentFunding, error)
	FindByState(state string) (*models.GovernmentFunding, error)
	FindByStateWithDetails(state string, periodsLimit int, activeOn *time.Time) (*models.GovernmentFunding, error)
	CountPeriods(fundingID uint) (int64, error)
	FindByName(name string) (*models.GovernmentFunding, error)
	Create(funding *models.GovernmentFunding) error
	Update(funding *models.GovernmentFunding) error
	Delete(id uint) error

	// Period CRUD
	FindPeriodByID(id uint) (*models.GovernmentFundingPeriod, error)
	FindPeriodsByGovernmentFundingID(fundingID uint) ([]models.GovernmentFundingPeriod, error)
	CreatePeriod(period *models.GovernmentFundingPeriod) error
	UpdatePeriod(period *models.GovernmentFundingPeriod) error
	DeletePeriod(id uint) error

	// Property CRUD
	FindPropertyByID(id uint) (*models.GovernmentFundingProperty, error)
	CreateProperty(property *models.GovernmentFundingProperty) error
	UpdateProperty(property *models.GovernmentFundingProperty) error
	DeleteProperty(id uint) error
}

// ChildAttendanceStorer defines the interface for child attendance storage operations
type ChildAttendanceStorer interface {
	FindByID(id uint) (*models.ChildAttendance, error)
	FindByOrganizationAndDate(orgID uint, date time.Time, limit, offset int) ([]models.ChildAttendance, int64, error)
	FindByChildAndDate(childID uint, date time.Time) (*models.ChildAttendance, error)
	FindByChildAndDateRange(childID uint, from, to time.Time, limit, offset int) ([]models.ChildAttendance, int64, error)
	Create(attendance *models.ChildAttendance) error
	Update(attendance *models.ChildAttendance) error
	Delete(id uint) error
	GetDailySummary(orgID uint, date time.Time) (*models.ChildAttendanceDailySummaryResponse, error)
}

// Compile-time interface compliance checks
var (
	_ UserStorer              = (*UserStore)(nil)
	_ OrganizationStorer      = (*OrganizationStore)(nil)
	_ GroupStorer             = (*GroupStore)(nil)
	_ EmployeeStorer          = (*EmployeeStore)(nil)
	_ ChildStorer             = (*ChildStore)(nil)
	_ UserGroupStorer         = (*UserGroupStore)(nil)
	_ GovernmentFundingStorer = (*GovernmentFundingStore)(nil)
	_ SectionStorer           = (*SectionStore)(nil)
	_ ChildAttendanceStorer   = (*ChildAttendanceStore)(nil)
)
