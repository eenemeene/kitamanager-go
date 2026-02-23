package models

import "time"

// GovernmentFunding represents a top-level government funding plan definition.
// Each funding is associated with a specific state (Bundesland).
// Organizations automatically use the funding for their state.
type GovernmentFunding struct {
	ID        uint                      `gorm:"primaryKey" json:"id" example:"1"`
	Name      string                    `gorm:"size:255;not null" json:"name" example:"Berlin Kita Funding"`
	State     string                    `gorm:"size:50;not null;uniqueIndex" json:"state" example:"berlin"`
	CreatedAt time.Time                 `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt time.Time                 `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	Periods   []GovernmentFundingPeriod `gorm:"foreignKey:GovernmentFundingID;constraint:OnDelete:CASCADE" json:"periods,omitempty"`
}

// TableName specifies the table name for GORM
func (GovernmentFunding) TableName() string {
	return "government_fundings"
}

// GovernmentFundingPeriod represents a time period within a government funding.
// Each period has its own set of properties with payment amounts.
// Periods within the same government funding must not overlap - this is enforced at the service layer.
// A period with nil To date is considered ongoing (extends indefinitely into the future).
type GovernmentFundingPeriod struct {
	ID                  uint                        `gorm:"primaryKey" json:"id" example:"1"`
	GovernmentFundingID uint                        `gorm:"not null;index" json:"government_funding_id" example:"1"`
	Period                                          // From, To (embedded)
	FullTimeWeeklyHours float64                     `gorm:"not null" json:"full_time_weekly_hours" example:"39.0"`
	Comment             string                      `gorm:"size:1000" json:"comment,omitempty" example:"Funding period 2023/2024"`
	CreatedAt           time.Time                   `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt           time.Time                   `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	Properties          []GovernmentFundingProperty `gorm:"foreignKey:PeriodID;constraint:OnDelete:CASCADE" json:"properties,omitempty"`
}

// TableName specifies the table name for GORM
func (GovernmentFundingPeriod) TableName() string {
	return "government_funding_periods"
}

// GovernmentFundingProperty represents a funding property with optional age range.
// Key/Value structure allows matching against child contract properties.
//
// Key: The property category (e.g., "care_type", "supplements")
// Value: The specific value within that category (e.g., "ganztag", "ndh")
//
// Matching is automatic: if the contract property is a scalar, it checks equality;
// if it's an array, it checks if the value is contained in the array.
//
// If MinAge and MaxAge are nil, the property applies to all ages.
// If set, both MinAge and MaxAge are inclusive (e.g., MinAge=0, MaxAge=2 covers ages 0, 1, and 2).
// Payment is stored in cents to avoid floating-point issues (e.g., 166847 = 1668.47 EUR).
type GovernmentFundingProperty struct {
	ID          uint      `gorm:"primaryKey" json:"id" example:"1"`
	PeriodID    uint      `gorm:"not null;index" json:"period_id" example:"1"`
	Key         string    `gorm:"size:100;not null" json:"key" example:"care_type"`
	Value       string    `gorm:"size:255;not null" json:"value" example:"ganztag"`
	Label       string    `gorm:"size:255;not null" json:"label" example:"Ganztag"`
	Payment     int       `gorm:"not null" json:"payment" example:"166847"`
	Requirement float64   `gorm:"not null" json:"requirement" example:"0.261"`
	MinAge      *int      `json:"min_age,omitempty" example:"0"`
	MaxAge      *int      `json:"max_age,omitempty" example:"3"`
	Comment     string    `gorm:"size:500" json:"comment,omitempty" example:"Full-day care funding for U3"`
	// ApplyToAllContracts marks this property as universal — it will be automatically
	// added to every child contract during creation/amendment, without the user needing
	// to select it. Example: parent meal contributions (Elternessen) in Berlin apply to
	// all children regardless of care type.
	ApplyToAllContracts bool      `gorm:"column:apply_to_all_contracts;not null;default:false" json:"apply_to_all_contracts" example:"false"`
	CreatedAt           time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// TableName specifies the table name for GORM
func (GovernmentFundingProperty) TableName() string {
	return "government_funding_properties"
}

// MatchesAge checks if the property applies to the given age.
// Returns true if no age filter is set, or if the age falls within the range.
// Both MinAge and MaxAge are inclusive: MinAge <= age <= MaxAge.
func (p *GovernmentFundingProperty) MatchesAge(age int) bool {
	// No age filter means it applies to all ages
	if p.MinAge == nil && p.MaxAge == nil {
		return true
	}
	// Check MinAge (inclusive)
	if p.MinAge != nil && age < *p.MinAge {
		return false
	}
	// Check MaxAge (inclusive)
	if p.MaxAge != nil && age > *p.MaxAge {
		return false
	}
	return true
}

// GovernmentFundingCreateRequest represents the request body for creating a government funding.
type GovernmentFundingCreateRequest struct {
	Name  string `json:"name" binding:"required,max=255" example:"Berlin Kita Funding"`
	State string `json:"state" binding:"required" example:"berlin"`
}

// GovernmentFundingUpdateRequest represents the request body for updating a government funding.
type GovernmentFundingUpdateRequest struct {
	Name *string `json:"name" binding:"omitempty,max=255" example:"Berlin Updated"`
}

// GovernmentFundingPeriodCreateRequest represents the request body for creating a government funding period.
type GovernmentFundingPeriodCreateRequest struct {
	From                time.Time  `json:"from" binding:"required" example:"2023-03-01"`
	To                  *time.Time `json:"to" example:"2024-02-29"`
	FullTimeWeeklyHours float64    `json:"full_time_weekly_hours" binding:"required,gt=0" example:"39.0"`
	Comment             string     `json:"comment" binding:"max=1000" example:"Funding period 2023/2024"`
}

// GovernmentFundingPeriodUpdateRequest represents the request body for updating a government funding period.
type GovernmentFundingPeriodUpdateRequest struct {
	From                *time.Time `json:"from" example:"2023-03-01"`
	To                  *time.Time `json:"to" example:"2024-02-29"`
	FullTimeWeeklyHours *float64   `json:"full_time_weekly_hours" binding:"omitempty,gt=0" example:"39.0"`
	Comment             *string    `json:"comment" binding:"omitempty,max=1000" example:"Updated comment"`
}

// GovernmentFundingPropertyCreateRequest represents the request body for creating a government funding property.
type GovernmentFundingPropertyCreateRequest struct {
	Key                 string  `json:"key" binding:"required,max=100" example:"care_type"`
	Value               string  `json:"value" binding:"required,max=255" example:"ganztag"`
	Label               string  `json:"label" binding:"required,max=255" example:"Ganztag"`
	Payment             int     `json:"payment" binding:"gte=0" example:"166847"`
	Requirement         float64 `json:"requirement" binding:"gte=0" example:"0.261"`
	MinAge              *int    `json:"min_age" binding:"omitempty,gte=0" example:"0"`
	MaxAge              *int    `json:"max_age" binding:"omitempty,gte=0" example:"3"`
	Comment             string  `json:"comment" binding:"max=500" example:"Full-day care funding for U3"`
	ApplyToAllContracts bool    `json:"apply_to_all_contracts" example:"false"`
}

// GovernmentFundingPropertyUpdateRequest represents the request body for updating a government funding property.
type GovernmentFundingPropertyUpdateRequest struct {
	Key                 *string  `json:"key" binding:"omitempty,max=100" example:"care_type"`
	Value               *string  `json:"value" binding:"omitempty,max=255" example:"ganztag"`
	Label               *string  `json:"label" binding:"omitempty,max=255" example:"Ganztag"`
	Payment             *int     `json:"payment" binding:"omitempty,gte=0" example:"166847"`
	Requirement         *float64 `json:"requirement" binding:"omitempty,gte=0" example:"0.261"`
	MinAge              *int     `json:"min_age" binding:"omitempty,gte=0" example:"0"`
	MaxAge              *int     `json:"max_age" binding:"omitempty,gte=0" example:"3"`
	Comment             *string  `json:"comment" binding:"omitempty,max=500" example:"Updated comment"`
	ApplyToAllContracts *bool    `json:"apply_to_all_contracts" example:"false"`
}

// GovernmentFundingResponse represents the government funding response
type GovernmentFundingResponse struct {
	ID        uint      `json:"id" example:"1"`
	Name      string    `json:"name" example:"Berlin Kita Funding"`
	State     string    `json:"state" example:"berlin"`
	CreatedAt time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

func (f *GovernmentFunding) ToResponse() GovernmentFundingResponse {
	return GovernmentFundingResponse{
		ID:        f.ID,
		Name:      f.Name,
		State:     f.State,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
}

// GovernmentFundingDetailResponse wraps the funding with metadata
type GovernmentFundingDetailResponse struct {
	*GovernmentFunding
	TotalPeriods int64 `json:"total_periods"`
}

// GovernmentFundingPeriodResponse represents the government funding period response
type GovernmentFundingPeriodResponse struct {
	ID                  uint       `json:"id" example:"1"`
	GovernmentFundingID uint       `json:"government_funding_id" example:"1"`
	From                time.Time  `json:"from" example:"2023-03-01"`
	To                  *time.Time `json:"to" example:"2024-02-29"`
	FullTimeWeeklyHours float64    `json:"full_time_weekly_hours" example:"39.0"`
	Comment             string     `json:"comment,omitempty" example:"Funding period 2023/2024"`
	CreatedAt           time.Time  `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt           time.Time  `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

func (p *GovernmentFundingPeriod) ToResponse() GovernmentFundingPeriodResponse {
	return GovernmentFundingPeriodResponse{
		ID:                  p.ID,
		GovernmentFundingID: p.GovernmentFundingID,
		From:                p.From,
		To:                  p.To,
		FullTimeWeeklyHours: p.FullTimeWeeklyHours,
		Comment:             p.Comment,
		CreatedAt:           p.CreatedAt,
		UpdatedAt:           p.UpdatedAt,
	}
}

// GovernmentFundingPropertyResponse represents the government funding property response.
// Key/Value structure defines how child contract properties are matched for funding calculation.
type GovernmentFundingPropertyResponse struct {
	ID                  uint      `json:"id" example:"1"`
	PeriodID            uint      `json:"period_id" example:"1"`
	Key                 string    `json:"key" example:"care_type"`
	Value               string    `json:"value" example:"ganztag"`
	Label               string    `json:"label" example:"Ganztag"`
	Payment             int       `json:"payment" example:"166847"`
	Requirement         float64   `json:"requirement" example:"0.261"`
	MinAge              *int      `json:"min_age,omitempty" example:"0"`
	MaxAge              *int      `json:"max_age,omitempty" example:"3"`
	Comment             string    `json:"comment,omitempty" example:"Full-day care funding for U3"`
	ApplyToAllContracts bool      `json:"apply_to_all_contracts" example:"false"`
	CreatedAt           time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

func (p *GovernmentFundingProperty) ToResponse() GovernmentFundingPropertyResponse {
	return GovernmentFundingPropertyResponse{
		ID:                  p.ID,
		PeriodID:            p.PeriodID,
		Key:                 p.Key,
		Value:               p.Value,
		Label:               p.Label,
		Payment:             p.Payment,
		Requirement:         p.Requirement,
		MinAge:              p.MinAge,
		MaxAge:              p.MaxAge,
		Comment:             p.Comment,
		ApplyToAllContracts: p.ApplyToAllContracts,
		CreatedAt:           p.CreatedAt,
	}
}

// ChildFundingResponse represents funding calculation for a single child
type ChildFundingResponse struct {
	ChildID             uint                      `json:"child_id" example:"1"`
	ChildName           string                    `json:"child_name" example:"Max Mustermann"`
	Age                 int                       `json:"age" example:"3"`
	Funding             int                       `json:"funding" example:"166847"`
	Requirement         float64                   `json:"requirement" example:"0.261"`
	MatchedProperties   []ChildFundingMatchedProp `json:"matched_properties"`
	UnmatchedProperties []ChildFundingMatchedProp `json:"unmatched_properties"`
}

// ChildFundingMatchedProp represents a matched or unmatched property in funding calculation
type ChildFundingMatchedProp struct {
	Key   string `json:"key" example:"care_type"`
	Value string `json:"value" example:"ganztag"`
}

// ChildrenFundingResponse represents the funding calculation response for all children
type ChildrenFundingResponse struct {
	Date             time.Time              `json:"date" example:"2025-01-27"`
	WeeklyHoursBasis float64                `json:"weekly_hours_basis" example:"39.0"`
	Children         []ChildFundingResponse `json:"children"`
}
