package models

import (
	"fmt"
	"time"
)

// Child represents a child enrolled in the Kita.
type Child struct {
	Person
	Contracts []ChildContract `gorm:"foreignKey:ChildID" json:"contracts,omitempty"`
}

// ChildContract represents an enrollment contract for a specific period.
// Contracts for the same child cannot overlap.
type ChildContract struct {
	ID            uint    `gorm:"primaryKey" json:"id" example:"1"`
	ChildID       uint    `gorm:"not null;index" json:"child_id" example:"1"`
	VoucherNumber *string `gorm:"size:50;index" json:"voucher_number,omitempty" example:"GB-12345678901-02"`
	BaseContract
}

// GetOwnerID returns the child ID for the PeriodRecord interface.
func (c ChildContract) GetOwnerID() uint {
	return c.ChildID
}

// ChildContractCreateRequest represents the request body for creating a child contract.
type ChildContractCreateRequest struct {
	From          time.Time          `json:"from" binding:"required" example:"2025-01-01"`
	To            *time.Time         `json:"to" example:"2025-12-31"`
	SectionID     uint               `json:"section_id" binding:"required" example:"2"`
	VoucherNumber *string            `json:"voucher_number,omitempty" example:"GB-12345678901-02"`
	Properties    ContractProperties `json:"properties,omitempty"`
}

// ChildContractUpdateRequest represents the request body for updating a child contract.
type ChildContractUpdateRequest struct {
	From          *time.Time         `json:"from" example:"2025-01-01"`
	To            *time.Time         `json:"to" example:"2025-12-31"`
	SectionID     *uint              `json:"section_id,omitempty" example:"2"`
	VoucherNumber *string            `json:"voucher_number,omitempty" example:"GB-12345678901-02"`
	Properties    ContractProperties `json:"properties,omitempty"`
}

// ChildContractBatchUpdateEntry represents a single contract update within a batch.
type ChildContractBatchUpdateEntry struct {
	ID uint `json:"id" binding:"required" example:"5"`
	ChildContractUpdateRequest
}

// ChildContractBatchUpdateRequest represents a batch of contract updates applied atomically.
type ChildContractBatchUpdateRequest struct {
	Updates []ChildContractBatchUpdateEntry `json:"updates" binding:"required,min=1,max=20"`
}

// ChildCreateRequest represents the request body for creating a child.
// OrganizationID is derived from the URL path parameter.
type ChildCreateRequest struct {
	FirstName string `json:"first_name" binding:"required,max=255" example:"Emma"`
	LastName  string `json:"last_name" binding:"required,max=255" example:"Schmidt"`
	Gender    string `json:"gender" binding:"required" example:"female"`
	Birthdate string `json:"birthdate" binding:"required" example:"2020-03-10"`
}

// ChildUpdateRequest represents the request body for updating a child.
type ChildUpdateRequest struct {
	FirstName *string `json:"first_name" binding:"omitempty,max=255" example:"Emma"`
	LastName  *string `json:"last_name" binding:"omitempty,max=255" example:"Schmidt"`
	Gender    *string `json:"gender" binding:"omitempty" example:"female"`
	Birthdate *string `json:"birthdate" binding:"omitempty" example:"2020-03-10"`
}

// ChildResponse represents the child response
type ChildResponse struct {
	ID             uint                    `json:"id" yaml:"id" example:"1"`
	OrganizationID uint                    `json:"organization_id" yaml:"organization_id" example:"1"`
	FirstName      string                  `json:"first_name" yaml:"first_name" example:"Emma"`
	LastName       string                  `json:"last_name" yaml:"last_name" example:"Schmidt"`
	Gender         string                  `json:"gender" yaml:"gender" example:"female"`
	Birthdate      time.Time               `json:"birthdate" yaml:"birthdate" example:"2020-03-10"`
	Contracts      []ChildContractResponse `json:"contracts,omitempty" yaml:"contracts"`
	CreatedAt      time.Time               `json:"created_at" yaml:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at" yaml:"updated_at"`
}

// ChildImportExportData wraps a list of children for YAML import/export.
type ChildImportExportData struct {
	Children []ChildResponse `json:"children" yaml:"children"`
}

// FullName returns the full name.
func (r ChildResponse) FullName() string {
	return r.FirstName + " " + r.LastName
}

func (c *Child) ToResponse() ChildResponse {
	resp := ChildResponse{
		ID:             c.ID,
		OrganizationID: c.OrganizationID,
		FirstName:      c.FirstName,
		LastName:       c.LastName,
		Gender:         c.Gender,
		Birthdate:      c.Birthdate,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
	if len(c.Contracts) > 0 {
		resp.Contracts = make([]ChildContractResponse, len(c.Contracts))
		for i, contract := range c.Contracts {
			resp.Contracts[i] = contract.ToResponse()
		}
	}
	return resp
}

// ChildContractResponse represents the child contract response
type ChildContractResponse struct {
	ID            uint               `json:"id" yaml:"id" example:"1"`
	ChildID       uint               `json:"child_id" yaml:"child_id" example:"1"`
	From          time.Time          `json:"from" yaml:"from" example:"2025-01-01"`
	To            *time.Time         `json:"to" yaml:"to" example:"2025-12-31"`
	SectionID     uint               `json:"section_id" yaml:"section_id" example:"2"`
	SectionName   *string            `json:"section_name,omitempty" yaml:"section_name" example:"Krippe"`
	VoucherNumber *string            `json:"voucher_number,omitempty" yaml:"voucher_number" example:"GB-12345678901-02"`
	Properties    ContractProperties `json:"properties,omitempty" yaml:"properties"`
	CreatedAt     time.Time          `json:"created_at" yaml:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at" yaml:"updated_at"`
}

func (c *ChildContract) ToResponse() ChildContractResponse {
	resp := ChildContractResponse{
		ID:            c.ID,
		ChildID:       c.ChildID,
		From:          c.From,
		To:            c.To,
		SectionID:     c.SectionID,
		VoucherNumber: c.VoucherNumber,
		Properties:    c.Properties,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
	if c.Section != nil {
		resp.SectionName = &c.Section.Name
	}
	return resp
}

// ChildListFilter represents filter options for listing children.
type ChildListFilter struct {
	SectionID     *uint
	ActiveOn      *time.Time
	ContractAfter *time.Time
	Search        string
}

// Validate checks the filter for conflicting options.
func (f *ChildListFilter) Validate() error {
	if f.ActiveOn != nil && f.ContractAfter != nil {
		return fmt.Errorf("active_on and contract_after are mutually exclusive")
	}
	return nil
}

// AgeDistributionResponse represents the age distribution of children with active contracts
type AgeDistributionResponse struct {
	Date         string                  `json:"date" example:"2025-01-28"`
	TotalCount   int                     `json:"total_count" example:"50"`
	Distribution []AgeDistributionBucket `json:"distribution"`
}

// AgeDistributionBucket represents count of children in an age range
type AgeDistributionBucket struct {
	AgeLabel     string `json:"age_label" example:"3"` // e.g., "0", "1", "2", "3", "4", "5", "6+"
	MinAge       int    `json:"min_age" example:"3"`
	MaxAge       *int   `json:"max_age,omitempty" example:"3"` // nil for open-ended (6+)
	Count        int    `json:"count" example:"12"`
	MaleCount    int    `json:"male_count" example:"6"`
	FemaleCount  int    `json:"female_count" example:"5"`
	DiverseCount int    `json:"diverse_count" example:"1"`
}
