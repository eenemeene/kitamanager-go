package models

import "time"

// GovernmentFunding represents a top-level government funding plan definition.
// Organizations are assigned to a government funding to determine funding calculations.
type GovernmentFunding struct {
	ID        uint                      `gorm:"primaryKey" json:"id" example:"1"`
	Name      string                    `gorm:"size:255;not null;uniqueIndex" json:"name" example:"Berlin"`
	CreatedAt time.Time                 `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt time.Time                 `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	Periods   []GovernmentFundingPeriod `gorm:"foreignKey:GovernmentFundingID;constraint:OnDelete:CASCADE" json:"periods,omitempty"`
}

// TableName specifies the table name for GORM
func (GovernmentFunding) TableName() string {
	return "government_fundings"
}

// GovernmentFundingPeriod represents a time period within a government funding.
// Each period has its own set of age-based entries with payment amounts.
// Periods within the same government funding must not overlap - this is enforced at the service layer.
// A period with nil To date is considered ongoing (extends indefinitely into the future).
type GovernmentFundingPeriod struct {
	ID                  uint                     `gorm:"primaryKey" json:"id" example:"1"`
	GovernmentFundingID uint                     `gorm:"not null;index" json:"government_funding_id" example:"1"`
	From                time.Time                `gorm:"column:from_date;type:date;not null" json:"from" example:"2023-03-01"`
	To                  *time.Time               `gorm:"column:to_date;type:date" json:"to" example:"2024-02-29"`
	Comment             string                   `gorm:"size:1000" json:"comment,omitempty" example:"Funding period 2023/2024"`
	CreatedAt           time.Time                `json:"created_at" example:"2024-01-15T10:30:00Z"`
	Entries             []GovernmentFundingEntry `gorm:"foreignKey:PeriodID;constraint:OnDelete:CASCADE" json:"entries,omitempty"`
}

// TableName specifies the table name for GORM
func (GovernmentFundingPeriod) TableName() string {
	return "government_funding_periods"
}

// GovernmentFundingEntry represents an age range entry within a period.
// MinAge is inclusive, MaxAge is exclusive (e.g., MinAge=0, MaxAge=2 covers ages 0 and 1,
// meaning children from birth up to but not including their 2nd birthday).
type GovernmentFundingEntry struct {
	ID       uint `gorm:"primaryKey" json:"id" example:"1"`
	PeriodID uint `gorm:"not null;index" json:"period_id" example:"1"`
	// MinAge is the minimum age in years (inclusive). A child whose age >= MinAge qualifies.
	MinAge int `gorm:"not null" json:"min_age" example:"0"`
	// MaxAge is the maximum age in years (exclusive). A child whose age < MaxAge qualifies.
	MaxAge     int                         `gorm:"not null" json:"max_age" example:"2"`
	CreatedAt  time.Time                   `json:"created_at" example:"2024-01-15T10:30:00Z"`
	Properties []GovernmentFundingProperty `gorm:"foreignKey:EntryID;constraint:OnDelete:CASCADE" json:"properties,omitempty"`
}

// TableName specifies the table name for GORM
func (GovernmentFundingEntry) TableName() string {
	return "government_funding_entries"
}

// GovernmentFundingProperty represents a property value with payment and staffing requirement.
// Payment is stored in cents to avoid floating-point issues (e.g., 166847 = 1668.47 EUR).
type GovernmentFundingProperty struct {
	ID          uint      `gorm:"primaryKey" json:"id" example:"1"`
	EntryID     uint      `gorm:"not null;index" json:"entry_id" example:"1"`
	Name        string    `gorm:"size:255;not null" json:"name" example:"ganztag"`
	Payment     int       `gorm:"not null" json:"payment" example:"166847"`
	Requirement float64   `gorm:"not null" json:"requirement" example:"0.261"`
	Comment     string    `gorm:"size:500" json:"comment,omitempty" example:"Full-day care funding"`
	CreatedAt   time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// TableName specifies the table name for GORM
func (GovernmentFundingProperty) TableName() string {
	return "government_funding_properties"
}

// GovernmentFundingCreateRequest represents the request body for creating a government funding.
type GovernmentFundingCreateRequest struct {
	Name string `json:"name" binding:"required,max=255" example:"Berlin"`
}

// GovernmentFundingUpdateRequest represents the request body for updating a government funding.
type GovernmentFundingUpdateRequest struct {
	Name *string `json:"name" binding:"omitempty,max=255" example:"Berlin Updated"`
}

// GovernmentFundingPeriodCreateRequest represents the request body for creating a government funding period.
type GovernmentFundingPeriodCreateRequest struct {
	From    time.Time  `json:"from" binding:"required" example:"2023-03-01"`
	To      *time.Time `json:"to" example:"2024-02-29"`
	Comment string     `json:"comment" binding:"max=1000" example:"Funding period 2023/2024"`
}

// GovernmentFundingPeriodUpdateRequest represents the request body for updating a government funding period.
type GovernmentFundingPeriodUpdateRequest struct {
	From    *time.Time `json:"from" example:"2023-03-01"`
	To      *time.Time `json:"to" example:"2024-02-29"`
	Comment *string    `json:"comment" binding:"omitempty,max=1000" example:"Updated comment"`
}

// GovernmentFundingEntryCreateRequest represents the request body for creating a government funding entry.
// MinAge is inclusive, MaxAge is exclusive (e.g., MinAge=0, MaxAge=2 covers ages 0 and 1).
type GovernmentFundingEntryCreateRequest struct {
	MinAge int `json:"min_age" binding:"gte=0" example:"0"`
	MaxAge int `json:"max_age" binding:"required,gtfield=MinAge" example:"2"`
}

// GovernmentFundingEntryUpdateRequest represents the request body for updating a government funding entry.
// MinAge is inclusive, MaxAge is exclusive (e.g., MinAge=0, MaxAge=2 covers ages 0 and 1).
type GovernmentFundingEntryUpdateRequest struct {
	MinAge *int `json:"min_age" binding:"omitempty,gte=0" example:"0"`
	MaxAge *int `json:"max_age" example:"3"`
}

// GovernmentFundingPropertyCreateRequest represents the request body for creating a government funding property.
type GovernmentFundingPropertyCreateRequest struct {
	Name        string  `json:"name" binding:"required,max=255" example:"ganztag"`
	Payment     int     `json:"payment" binding:"gte=0" example:"166847"`
	Requirement float64 `json:"requirement" binding:"gte=0" example:"0.261"`
	Comment     string  `json:"comment" binding:"max=500" example:"Full-day care funding"`
}

// GovernmentFundingPropertyUpdateRequest represents the request body for updating a government funding property.
type GovernmentFundingPropertyUpdateRequest struct {
	Name        *string  `json:"name" binding:"omitempty,max=255" example:"ganztag"`
	Payment     *int     `json:"payment" binding:"omitempty,gte=0" example:"166847"`
	Requirement *float64 `json:"requirement" binding:"omitempty,gte=0" example:"0.261"`
	Comment     *string  `json:"comment" binding:"omitempty,max=500" example:"Updated comment"`
}

// AssignGovernmentFundingRequest represents the request body for assigning a government funding to an organization.
type AssignGovernmentFundingRequest struct {
	GovernmentFundingID uint `json:"government_funding_id" binding:"required" example:"1"`
}

// GovernmentFundingResponse represents the government funding response
type GovernmentFundingResponse struct {
	ID        uint      `json:"id" example:"1"`
	Name      string    `json:"name" example:"Berlin"`
	CreatedAt time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

func (f *GovernmentFunding) ToResponse() GovernmentFundingResponse {
	return GovernmentFundingResponse{
		ID:        f.ID,
		Name:      f.Name,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
}

// GovernmentFundingPeriodResponse represents the government funding period response
type GovernmentFundingPeriodResponse struct {
	ID                  uint       `json:"id" example:"1"`
	GovernmentFundingID uint       `json:"government_funding_id" example:"1"`
	From                time.Time  `json:"from" example:"2023-03-01"`
	To                  *time.Time `json:"to" example:"2024-02-29"`
	Comment             string     `json:"comment,omitempty" example:"Funding period 2023/2024"`
	CreatedAt           time.Time  `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

func (p *GovernmentFundingPeriod) ToResponse() GovernmentFundingPeriodResponse {
	return GovernmentFundingPeriodResponse{
		ID:                  p.ID,
		GovernmentFundingID: p.GovernmentFundingID,
		From:                p.From,
		To:                  p.To,
		Comment:             p.Comment,
		CreatedAt:           p.CreatedAt,
	}
}

// GovernmentFundingEntryResponse represents the government funding entry response
type GovernmentFundingEntryResponse struct {
	ID        uint      `json:"id" example:"1"`
	PeriodID  uint      `json:"period_id" example:"1"`
	MinAge    int       `json:"min_age" example:"0"`
	MaxAge    int       `json:"max_age" example:"2"`
	CreatedAt time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

func (e *GovernmentFundingEntry) ToResponse() GovernmentFundingEntryResponse {
	return GovernmentFundingEntryResponse{
		ID:        e.ID,
		PeriodID:  e.PeriodID,
		MinAge:    e.MinAge,
		MaxAge:    e.MaxAge,
		CreatedAt: e.CreatedAt,
	}
}

// GovernmentFundingPropertyResponse represents the government funding property response
type GovernmentFundingPropertyResponse struct {
	ID          uint      `json:"id" example:"1"`
	EntryID     uint      `json:"entry_id" example:"1"`
	Name        string    `json:"name" example:"ganztag"`
	Payment     int       `json:"payment" example:"166847"`
	Requirement float64   `json:"requirement" example:"0.261"`
	Comment     string    `json:"comment,omitempty" example:"Full-day care funding"`
	CreatedAt   time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

func (p *GovernmentFundingProperty) ToResponse() GovernmentFundingPropertyResponse {
	return GovernmentFundingPropertyResponse{
		ID:          p.ID,
		EntryID:     p.EntryID,
		Name:        p.Name,
		Payment:     p.Payment,
		Requirement: p.Requirement,
		Comment:     p.Comment,
		CreatedAt:   p.CreatedAt,
	}
}
