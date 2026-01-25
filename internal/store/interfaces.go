package store

import (
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// UserStorer defines the interface for user storage operations
type UserStorer interface {
	FindAll(limit, offset int) ([]models.User, int64, error)
	FindByID(id uint) (*models.User, error)
	FindByEmail(email string) (*models.User, error)
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
	Update(org *models.Organization) error
	Delete(id uint) error
}

// GroupStorer defines the interface for group storage operations
type GroupStorer interface {
	FindAll(limit, offset int) ([]models.Group, int64, error)
	FindByID(id uint) (*models.Group, error)
	FindByOrganization(orgID uint) ([]models.Group, error)
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

// Compile-time interface compliance checks
var (
	_ UserStorer         = (*UserStore)(nil)
	_ OrganizationStorer = (*OrganizationStore)(nil)
	_ GroupStorer        = (*GroupStore)(nil)
	_ EmployeeStorer     = (*EmployeeStore)(nil)
	_ ChildStorer        = (*ChildStore)(nil)
)
