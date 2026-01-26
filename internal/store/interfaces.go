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
	FindByID(id uint) (*models.Employee, error)
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
	FindByID(id uint) (*models.Child, error)
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
	HasActiveContract(personID uint, date time.Time) (bool, error)
	ValidateNoOverlap(personID uint, from time.Time, to *time.Time, excludeID *uint) error
	CloseCurrentContract(personID uint, endDate time.Time) error
}

// PayplanStorer defines the interface for payplan storage operations
type PayplanStorer interface {
	// Payplan CRUD
	FindAll(limit, offset int) ([]models.Payplan, int64, error)
	FindByID(id uint) (*models.Payplan, error)
	FindByIDWithDetails(id uint) (*models.Payplan, error)
	Create(payplan *models.Payplan) error
	Update(payplan *models.Payplan) error
	Delete(id uint) error

	// Period CRUD
	FindPeriodByID(id uint) (*models.PayplanPeriod, error)
	FindPeriodsByPayplanID(payplanID uint) ([]models.PayplanPeriod, error)
	CreatePeriod(period *models.PayplanPeriod) error
	UpdatePeriod(period *models.PayplanPeriod) error
	DeletePeriod(id uint) error

	// Entry CRUD
	FindEntryByID(id uint) (*models.PayplanEntry, error)
	CreateEntry(entry *models.PayplanEntry) error
	UpdateEntry(entry *models.PayplanEntry) error
	DeleteEntry(id uint) error

	// Property CRUD
	FindPropertyByID(id uint) (*models.PayplanProperty, error)
	CreateProperty(property *models.PayplanProperty) error
	UpdateProperty(property *models.PayplanProperty) error
	DeleteProperty(id uint) error

	// Organization payplan assignment
	AssignPayplanToOrg(orgID, payplanID uint) error
	RemovePayplanFromOrg(orgID uint) error
}

// Compile-time interface compliance checks
var (
	_ UserStorer         = (*UserStore)(nil)
	_ OrganizationStorer = (*OrganizationStore)(nil)
	_ GroupStorer        = (*GroupStore)(nil)
	_ EmployeeStorer     = (*EmployeeStore)(nil)
	_ ChildStorer        = (*ChildStore)(nil)
	_ UserGroupStorer    = (*UserGroupStore)(nil)
	_ PayplanStorer      = (*PayplanStore)(nil)
)
