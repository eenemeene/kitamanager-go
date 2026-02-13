package models

import "time"

// PayPlan represents a salary pay plan (e.g., TVöD-SuE) for an organization.
// Each organization can have multiple pay plans.
type PayPlan struct {
	ID             uint            `gorm:"primaryKey" json:"id" example:"1"`
	OrganizationID uint            `gorm:"not null;index" json:"organization_id" example:"1"`
	Organization   *Organization   `gorm:"foreignKey:OrganizationID" json:"-"`
	Name           string          `gorm:"not null" json:"name" example:"TVöD-SuE"`
	Periods        []PayPlanPeriod `gorm:"foreignKey:PayPlanID" json:"periods,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// PayPlanPeriod represents a time period with specific pay rates.
// WeeklyHours defines the reference hours for full-time employment.
type PayPlanPeriod struct {
	ID          uint           `gorm:"primaryKey" json:"id" example:"1"`
	PayPlanID   uint           `gorm:"not null;index" json:"payplan_id" example:"1"`
	PayPlan     *PayPlan       `gorm:"foreignKey:PayPlanID" json:"-"`
	From        time.Time      `gorm:"not null" json:"from" example:"2024-01-01"`
	To          *time.Time     `json:"to,omitempty" example:"2024-12-31"`
	WeeklyHours float64        `gorm:"not null" json:"weekly_hours" example:"39.0"`
	Entries     []PayPlanEntry `gorm:"foreignKey:PeriodID" json:"entries,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// PayPlanEntry represents a specific pay grade and step with its monthly amount.
// Grade is the pay grade (e.g., "S8a", "S11b") and Step is the experience level (1-6).
type PayPlanEntry struct {
	ID            uint           `gorm:"primaryKey" json:"id" example:"1"`
	PeriodID      uint           `gorm:"not null;index" json:"period_id" example:"1"`
	Period        *PayPlanPeriod `gorm:"foreignKey:PeriodID" json:"-"`
	Grade         string         `gorm:"not null" json:"grade" example:"S8a"`
	Step          int            `gorm:"not null" json:"step" example:"3"`
	MonthlyAmount int            `gorm:"not null" json:"monthly_amount" example:"350000"` // cents
	StepMinYears  *int           `json:"step_min_years,omitempty" example:"3"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// PayPlanCreateRequest is the request body for creating a pay plan.
type PayPlanCreateRequest struct {
	Name string `json:"name" binding:"required" example:"TVöD-SuE"`
}

// PayPlanUpdateRequest is the request body for updating a pay plan.
type PayPlanUpdateRequest struct {
	Name string `json:"name" binding:"required" example:"TVöD-SuE"`
}

// PayPlanResponse is the response for a pay plan.
type PayPlanResponse struct {
	ID             uint      `json:"id" example:"1"`
	OrganizationID uint      `json:"organization_id" example:"1"`
	Name           string    `json:"name" example:"TVöD-SuE"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// PayPlanDetailResponse includes periods for detail view.
type PayPlanDetailResponse struct {
	ID             uint                    `json:"id" example:"1"`
	OrganizationID uint                    `json:"organization_id" example:"1"`
	Name           string                  `json:"name" example:"TVöD-SuE"`
	Periods        []PayPlanPeriodResponse `json:"periods"`
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
}

// PayPlanPeriodCreateRequest is the request body for creating a period.
type PayPlanPeriodCreateRequest struct {
	From        string  `json:"from" binding:"required" example:"2024-01-01"`
	To          *string `json:"to,omitempty" example:"2024-12-31"`
	WeeklyHours float64 `json:"weekly_hours" binding:"required,gt=0" example:"39.0"`
}

// PayPlanPeriodUpdateRequest is the request body for updating a period.
type PayPlanPeriodUpdateRequest struct {
	From        string  `json:"from" binding:"required" example:"2024-01-01"`
	To          *string `json:"to,omitempty" example:"2024-12-31"`
	WeeklyHours float64 `json:"weekly_hours" binding:"required,gt=0" example:"39.0"`
}

// PayPlanPeriodResponse is the response for a period.
type PayPlanPeriodResponse struct {
	ID          uint                   `json:"id" example:"1"`
	PayPlanID   uint                   `json:"payplan_id" example:"1"`
	From        string                 `json:"from" example:"2024-01-01"`
	To          *string                `json:"to,omitempty" example:"2024-12-31"`
	WeeklyHours float64                `json:"weekly_hours" example:"39.0"`
	Entries     []PayPlanEntryResponse `json:"entries,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// PayPlanEntryCreateRequest is the request body for creating an entry.
type PayPlanEntryCreateRequest struct {
	Grade         string `json:"grade" binding:"required" example:"S8a"`
	Step          int    `json:"step" binding:"required,min=1" example:"3"`
	MonthlyAmount int    `json:"monthly_amount" binding:"required,min=0" example:"350000"`
	StepMinYears  *int   `json:"step_min_years,omitempty" binding:"omitempty,min=0" example:"3"`
}

// PayPlanEntryUpdateRequest is the request body for updating an entry.
type PayPlanEntryUpdateRequest struct {
	Grade         string `json:"grade" binding:"required" example:"S8a"`
	Step          int    `json:"step" binding:"required,min=1" example:"3"`
	MonthlyAmount int    `json:"monthly_amount" binding:"required,min=0" example:"350000"`
	StepMinYears  *int   `json:"step_min_years,omitempty" binding:"omitempty,min=0" example:"3"`
}

// PayPlanEntryResponse is the response for an entry.
type PayPlanEntryResponse struct {
	ID            uint      `json:"id" example:"1"`
	PeriodID      uint      `json:"period_id" example:"1"`
	Grade         string    `json:"grade" example:"S8a"`
	Step          int       `json:"step" example:"3"`
	MonthlyAmount int       `json:"monthly_amount" example:"350000"`
	StepMinYears  *int      `json:"step_min_years,omitempty" example:"3"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ToResponse converts a PayPlan to PayPlanResponse.
func (p *PayPlan) ToResponse() PayPlanResponse {
	return PayPlanResponse{
		ID:             p.ID,
		OrganizationID: p.OrganizationID,
		Name:           p.Name,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

// ToDetailResponse converts a PayPlan to PayPlanDetailResponse.
func (p *PayPlan) ToDetailResponse() PayPlanDetailResponse {
	periods := make([]PayPlanPeriodResponse, len(p.Periods))
	for i, period := range p.Periods {
		periods[i] = period.ToResponse()
	}
	return PayPlanDetailResponse{
		ID:             p.ID,
		OrganizationID: p.OrganizationID,
		Name:           p.Name,
		Periods:        periods,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

// ToResponse converts a PayPlanPeriod to PayPlanPeriodResponse.
func (p *PayPlanPeriod) ToResponse() PayPlanPeriodResponse {
	var to *string
	if p.To != nil {
		toStr := p.To.Format(DateFormat)
		to = &toStr
	}
	entries := make([]PayPlanEntryResponse, len(p.Entries))
	for i, entry := range p.Entries {
		entries[i] = entry.ToResponse()
	}
	return PayPlanPeriodResponse{
		ID:          p.ID,
		PayPlanID:   p.PayPlanID,
		From:        p.From.Format(DateFormat),
		To:          to,
		WeeklyHours: p.WeeklyHours,
		Entries:     entries,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// ToResponse converts a PayPlanEntry to PayPlanEntryResponse.
func (e *PayPlanEntry) ToResponse() PayPlanEntryResponse {
	return PayPlanEntryResponse{
		ID:            e.ID,
		PeriodID:      e.PeriodID,
		Grade:         e.Grade,
		Step:          e.Step,
		MonthlyAmount: e.MonthlyAmount,
		StepMinYears:  e.StepMinYears,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}
}
