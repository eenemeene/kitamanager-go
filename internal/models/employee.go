package models

import (
	"strconv"
	"time"
)

// Employee represents a staff member of the Kita.
type Employee struct {
	Person
	Contracts []EmployeeContract `gorm:"foreignKey:EmployeeID" json:"contracts,omitempty"`
}

// EmployeeContract represents an employment contract for a specific period.
// Contracts for the same employee cannot overlap.
type EmployeeContract struct {
	ID         uint `gorm:"primaryKey" json:"id" example:"1"`
	EmployeeID uint `gorm:"not null;index" json:"employee_id" example:"1"`
	Period

	// Contract properties
	Position    string  `gorm:"size:255" json:"position" example:"Erzieher"`
	Grade       string  `gorm:"size:20" json:"grade" example:"S8a"`
	Step        int     `json:"step" example:"3"`
	WeeklyHours float64 `json:"weekly_hours" example:"40"`

	// Key-value properties for additional contract attributes
	Properties []EmployeeContractProperty `gorm:"foreignKey:ContractID" json:"properties,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Employee contract property name constants
const (
	PropActualHours      = "actual_hours"       // float64: 0-168
	PropChristmasBonus   = "christmas_bonus"    // int: >= 0 (cents)
	PropEmployerType     = "employer_type"      // string: "normal" or "minijob"
	PropIsQualifiedStaff = "is_qualified_staff" // bool: "true" or "false"
)

// PropertyType represents the expected type for a property value
type PropertyType string

const (
	PropertyTypeFloat  PropertyType = "float"
	PropertyTypeInt    PropertyType = "int"
	PropertyTypeBool   PropertyType = "bool"
	PropertyTypeString PropertyType = "string"
)

// PropertySchema defines validation rules for a property
type PropertySchema struct {
	Type          PropertyType
	AllowedValues []string // For string types with restricted values
	MinFloat      *float64 // For float types
	MaxFloat      *float64 // For float types
	MinInt        *int     // For int types
}

// PropertySchemas defines validation schemas for known properties
var PropertySchemas = map[string]PropertySchema{
	PropActualHours: {
		Type:     PropertyTypeFloat,
		MinFloat: ptrFloat(0),
		MaxFloat: ptrFloat(168),
	},
	PropChristmasBonus: {
		Type:   PropertyTypeInt,
		MinInt: ptrInt(0),
	},
	PropEmployerType: {
		Type:          PropertyTypeString,
		AllowedValues: []string{"normal", "minijob"},
	},
	PropIsQualifiedStaff: {
		Type: PropertyTypeBool,
	},
}

func ptrFloat(f float64) *float64 { return &f }
func ptrInt(i int) *int           { return &i }

// EmployeeContractProperty represents a key-value property for an employee contract
type EmployeeContractProperty struct {
	ID         uint   `gorm:"primaryKey" json:"id" example:"1"`
	ContractID uint   `gorm:"not null;index" json:"contract_id" example:"1"`
	Name       string `gorm:"size:255;not null" json:"name" example:"actual_hours"`
	Value      string `gorm:"size:1024;not null" json:"value" example:"38.5"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetFloat parses the property value as a float64
func (p *EmployeeContractProperty) GetFloat() (float64, error) {
	return strconv.ParseFloat(p.Value, 64)
}

// GetInt parses the property value as an int
func (p *EmployeeContractProperty) GetInt() (int, error) {
	return strconv.Atoi(p.Value)
}

// GetBool parses the property value as a bool
func (p *EmployeeContractProperty) GetBool() (bool, error) {
	return strconv.ParseBool(p.Value)
}

// GetString returns the property value as a string
func (p *EmployeeContractProperty) GetString() string {
	return p.Value
}

// GetPersonID returns the employee ID for the HasPeriod interface.
func (c EmployeeContract) GetPersonID() uint {
	return c.EmployeeID
}

// EmployeeContractCreateRequest represents the request body for creating an employee contract.
type EmployeeContractCreateRequest struct {
	From        time.Time  `json:"from" binding:"required" example:"2025-01-01"`
	To          *time.Time `json:"to" example:"2025-12-31"`
	Position    string     `json:"position" binding:"required,max=255" example:"Erzieher"`
	Grade       string     `json:"grade" binding:"max=20" example:"S8a"`
	Step        int        `json:"step" binding:"gte=0,lte=10" example:"3"`
	WeeklyHours float64    `json:"weekly_hours" binding:"required,gte=0,lte=168" example:"40"`
}

// EmployeeContractUpdateRequest represents the request body for updating an employee contract.
type EmployeeContractUpdateRequest struct {
	From        *time.Time `json:"from" example:"2025-01-01"`
	To          *time.Time `json:"to" example:"2025-12-31"`
	Position    *string    `json:"position" binding:"omitempty,max=255" example:"Erzieher"`
	Grade       *string    `json:"grade" binding:"omitempty,max=20" example:"S8a"`
	Step        *int       `json:"step" binding:"omitempty,gte=0,lte=10" example:"3"`
	WeeklyHours *float64   `json:"weekly_hours" binding:"omitempty,gte=0,lte=168" example:"40"`
}

// EmployeeContractPropertyCreateRequest represents the request body for creating a contract property.
type EmployeeContractPropertyCreateRequest struct {
	Name  string `json:"name" binding:"required,max=255" example:"actual_hours"`
	Value string `json:"value" binding:"required,max=1024" example:"38.5"`
}

// EmployeeContractPropertyUpdateRequest represents the request body for updating a contract property.
type EmployeeContractPropertyUpdateRequest struct {
	Value string `json:"value" binding:"required,max=1024" example:"38.5"`
}

// EmployeeContractPropertyResponse represents the contract property response
type EmployeeContractPropertyResponse struct {
	ID         uint      `json:"id" example:"1"`
	ContractID uint      `json:"contract_id" example:"1"`
	Name       string    `json:"name" example:"actual_hours"`
	Value      string    `json:"value" example:"38.5"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (p *EmployeeContractProperty) ToResponse() EmployeeContractPropertyResponse {
	return EmployeeContractPropertyResponse{
		ID:         p.ID,
		ContractID: p.ContractID,
		Name:       p.Name,
		Value:      p.Value,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

// EmployeeCreateRequest represents the request body for creating an employee.
// OrganizationID is derived from the URL path parameter.
type EmployeeCreateRequest struct {
	FirstName string    `json:"first_name" binding:"required,max=255" example:"Max"`
	LastName  string    `json:"last_name" binding:"required,max=255" example:"Mustermann"`
	Gender    string    `json:"gender" binding:"required" example:"male"`
	Birthdate time.Time `json:"birthdate" binding:"required" example:"1990-05-15"`
}

// EmployeeUpdateRequest represents the request body for updating an employee.
type EmployeeUpdateRequest struct {
	FirstName *string    `json:"first_name" binding:"omitempty,max=255" example:"Max"`
	LastName  *string    `json:"last_name" binding:"omitempty,max=255" example:"Mustermann"`
	Gender    *string    `json:"gender" binding:"omitempty" example:"male"`
	Birthdate *time.Time `json:"birthdate" example:"1990-05-15"`
}

// EmployeeResponse represents the employee response
type EmployeeResponse struct {
	ID             uint                       `json:"id" example:"1"`
	OrganizationID uint                       `json:"organization_id" example:"1"`
	FirstName      string                     `json:"first_name" example:"Max"`
	LastName       string                     `json:"last_name" example:"Mustermann"`
	Gender         string                     `json:"gender" example:"male"`
	Birthdate      time.Time                  `json:"birthdate" example:"1990-05-15"`
	Contracts      []EmployeeContractResponse `json:"contracts,omitempty"`
	CreatedAt      time.Time                  `json:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at"`
}

func (e *Employee) ToResponse() EmployeeResponse {
	resp := EmployeeResponse{
		ID:             e.ID,
		OrganizationID: e.OrganizationID,
		FirstName:      e.FirstName,
		LastName:       e.LastName,
		Gender:         e.Gender,
		Birthdate:      e.Birthdate,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}

	if len(e.Contracts) > 0 {
		resp.Contracts = make([]EmployeeContractResponse, len(e.Contracts))
		for i, c := range e.Contracts {
			resp.Contracts[i] = c.ToResponse()
		}
	}

	return resp
}

// EmployeeContractResponse represents the employee contract response
type EmployeeContractResponse struct {
	ID          uint                               `json:"id" example:"1"`
	EmployeeID  uint                               `json:"employee_id" example:"1"`
	From        time.Time                          `json:"from" example:"2025-01-01"`
	To          *time.Time                         `json:"to" example:"2025-12-31"`
	Position    string                             `json:"position" example:"Erzieher"`
	Grade       string                             `json:"grade" example:"S8a"`
	Step        int                                `json:"step" example:"3"`
	WeeklyHours float64                            `json:"weekly_hours" example:"40"`
	Properties  []EmployeeContractPropertyResponse `json:"properties,omitempty"`
	CreatedAt   time.Time                          `json:"created_at"`
	UpdatedAt   time.Time                          `json:"updated_at"`
}

func (c *EmployeeContract) ToResponse() EmployeeContractResponse {
	resp := EmployeeContractResponse{
		ID:          c.ID,
		EmployeeID:  c.EmployeeID,
		From:        c.From,
		To:          c.To,
		Position:    c.Position,
		Grade:       c.Grade,
		Step:        c.Step,
		WeeklyHours: c.WeeklyHours,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}

	if len(c.Properties) > 0 {
		resp.Properties = make([]EmployeeContractPropertyResponse, len(c.Properties))
		for i, p := range c.Properties {
			resp.Properties[i] = p.ToResponse()
		}
	}

	return resp
}
