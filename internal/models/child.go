package models

import (
	"fmt"
	"time"
)

// Child represents a child enrolled in the Kita.
type Child struct {
	Person
	// SchoolEntryDate is when the child leaves for school, when that is not the
	// date their birthdate implies -- a Zurückstellung being the reason it
	// usually differs. Nil means "compute it", which is what every child gets
	// until somebody records otherwise.
	//
	// A date rather than a flag or a year offset: Bayern's Einschulungskorridor
	// and Bremen's Karenzzeit leave the regular date genuinely undecided for a
	// band of birthdates, so there is no base to offset from, and a Bayern
	// deferral granted up to 31 January does not land a whole number of years
	// from anything.
	//
	// On Child and not Person: employees do not go to school.
	SchoolEntryDate *time.Time `gorm:"type:date" json:"school_entry_date,omitempty" format:"date-time" example:"2028-08-01"`

	Contracts []ChildContract `gorm:"foreignKey:ChildID" json:"contracts,omitempty"`
	Vouchers  []ChildVoucher  `gorm:"foreignKey:ChildID" json:"vouchers,omitempty"`
}

// ChildContract represents an enrollment contract for a specific period.
// Contracts for the same child cannot overlap.
type ChildContract struct {
	ID      uint `gorm:"primaryKey" json:"id" example:"1"`
	ChildID uint `gorm:"not null;index" json:"child_id" example:"1"`
	BaseContract
}

// GetOwnerID returns the child ID for the PeriodRecord interface.
func (c ChildContract) GetOwnerID() uint {
	return c.ChildID
}

// ChildContractCreateRequest represents the request body for creating a child contract.
type ChildContractCreateRequest struct {
	From       time.Time          `json:"from" binding:"required" format:"date-time" example:"2025-01-01"`
	To         *time.Time         `json:"to" format:"date-time" example:"2025-12-31"`
	SectionID  uint               `json:"section_id" binding:"required" example:"2"`
	Properties ContractProperties `json:"properties,omitempty"`
}

// ChildCreateRequest represents the request body for creating a child.
// OrganizationID is derived from the URL path parameter.
type ChildCreateRequest struct {
	FirstName string `json:"first_name" binding:"required,max=255" example:"Emma"`
	LastName  string `json:"last_name" binding:"required,max=255" example:"Schmidt"`
	Gender    string `json:"gender" enums:"male,female,diverse" binding:"required" example:"female"`
	Birthdate string `json:"birthdate" binding:"required" example:"2020-03-10"`
	// SchoolEntryDate is optional; omitted means the date is computed from the
	// birthdate. A plain pointer suffices here because on a create there is no
	// stored value that omitting could destroy.
	SchoolEntryDate *time.Time `json:"school_entry_date,omitempty" format:"date-time" example:"2028-08-01"`
}

// ChildUpdateRequest represents the request body for updating a child.
type ChildUpdateRequest struct {
	FirstName *string `json:"first_name" binding:"omitempty,max=255" example:"Emma"`
	LastName  *string `json:"last_name" binding:"omitempty,max=255" example:"Schmidt"`
	Gender    *string `json:"gender" enums:"male,female,diverse" binding:"omitempty" example:"female"`
	Birthdate *string `json:"birthdate" binding:"omitempty" example:"2020-03-10"`
	// Opt rather than *time.Time, unlike its neighbours above: a reversed
	// Zurückstellung has to be expressible, and `*T` collapses "left the field
	// alone" into "set it to null" -- so a plain pointer could record a deferral
	// and never undo one.
	SchoolEntryDate Opt[time.Time] `json:"school_entry_date,omitzero" swaggertype:"string" format:"date-time" extensions:"x-nullable" example:"2028-08-01"`
}

// ChildResponse represents the child response
type ChildResponse struct {
	ID             uint      `json:"id" yaml:"id" example:"1"`
	OrganizationID uint      `json:"organization_id" yaml:"organization_id" example:"1"`
	FirstName      string    `json:"first_name" yaml:"first_name" example:"Emma"`
	LastName       string    `json:"last_name" yaml:"last_name" example:"Schmidt"`
	Gender         string    `json:"gender" enums:"male,female,diverse" yaml:"gender" example:"female"`
	Birthdate      time.Time `json:"birthdate" format:"date-time" yaml:"birthdate" example:"2020-03-10"`
	// Carries a yaml tag like its neighbours so a YAML round-trip preserves it.
	SchoolEntryDate *time.Time              `json:"school_entry_date,omitempty" format:"date-time" yaml:"school_entry_date,omitempty" example:"2028-08-01"`
	Vouchers        []string                `json:"vouchers,omitempty" yaml:"vouchers"`
	Contracts       []ChildContractResponse `json:"contracts,omitempty" yaml:"contracts"`
	CreatedAt       time.Time               `json:"created_at" format:"date-time" yaml:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at" format:"date-time" yaml:"updated_at"`
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
		ID:              c.ID,
		OrganizationID:  c.OrganizationID,
		FirstName:       c.FirstName,
		LastName:        c.LastName,
		Gender:          c.Gender,
		Birthdate:       c.Birthdate,
		SchoolEntryDate: c.SchoolEntryDate,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
	if len(c.Vouchers) > 0 {
		resp.Vouchers = make([]string, len(c.Vouchers))
		for i, v := range c.Vouchers {
			resp.Vouchers[i] = v.VoucherNumber
		}
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
	ID          uint               `json:"id" yaml:"id" example:"1"`
	ChildID     uint               `json:"child_id" yaml:"child_id" example:"1"`
	From        time.Time          `json:"from" format:"date-time" yaml:"from" example:"2025-01-01"`
	To          *time.Time         `json:"to" format:"date-time" yaml:"to" example:"2025-12-31"`
	SectionID   uint               `json:"section_id" yaml:"section_id" example:"2"`
	SectionName *string            `json:"section_name,omitempty" yaml:"section_name" example:"Krippe"`
	Properties  ContractProperties `json:"properties,omitempty" yaml:"properties"`
	// Version is the optimistic-concurrency token a client echoes back as an
	// If-Match precondition on the next write. yaml:"-" keeps it out of the
	// person YAML dumps: it describes a row's revision, not the contract.
	Version   int64     `json:"version" yaml:"-" example:"3"`
	CreatedAt time.Time `json:"created_at" format:"date-time" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" format:"date-time" yaml:"updated_at"`
}

func (c *ChildContract) ToResponse() ChildContractResponse {
	resp := ChildContractResponse{
		ID:         c.ID,
		ChildID:    c.ChildID,
		From:       c.From,
		To:         c.To,
		SectionID:  c.SectionID,
		Properties: c.Properties,
		Version:    c.Version,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
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
