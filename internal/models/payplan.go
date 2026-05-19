package models

import (
	"cmp"
	"regexp"
	"strconv"
	"time"
)

// gradeNumberRe extracts the digit run from a grade string like "S8a" → "8".
// Anything before is treated as the prefix, anything after as the suffix.
var gradeNumberRe = regexp.MustCompile(`(\d+)`)

// CompareGrade orders grades naturally: prefix (alpha), then numeric part, then
// suffix. Plain string comparison would put "S10" before "S2" which is what
// every consumer wants to avoid. Used by the store layer so callers don't
// have to re-sort, and so non-UI consumers (YAML export, etc.) get sane
// ordering too.
func CompareGrade(a, b string) int {
	pa, na, sa := splitGrade(a)
	pb, nb, sb := splitGrade(b)
	return cmp.Or(
		cmp.Compare(pa, pb),
		cmp.Compare(na, nb),
		cmp.Compare(sa, sb),
	)
}

// splitGrade splits a grade like "S8a" into ("S", 8, "a"). If no digit run is
// found the prefix is the whole string and number is 0; that keeps the
// comparator total without panicking on unexpected input.
func splitGrade(g string) (prefix string, number int, suffix string) {
	loc := gradeNumberRe.FindStringIndex(g)
	if loc == nil {
		return g, 0, ""
	}
	prefix = g[:loc[0]]
	number, _ = strconv.Atoi(g[loc[0]:loc[1]])
	suffix = g[loc[1]:]
	return
}

// PayPlan represents a salary pay plan (e.g., TVöD-SuE) for an organization.
// Each organization can have multiple pay plans.
type PayPlan struct {
	ID             uint            `gorm:"primaryKey" json:"id" example:"1"`
	OrganizationID uint            `gorm:"not null;index;uniqueIndex:idx_pay_plans_org_name" json:"organization_id" example:"1"`
	Organization   *Organization   `gorm:"foreignKey:OrganizationID" json:"-"`
	Name           string          `gorm:"not null;uniqueIndex:idx_pay_plans_org_name" json:"name" example:"TVöD-SuE"`
	Periods        []PayPlanPeriod `gorm:"foreignKey:PayPlanID" json:"periods,omitempty"`
	CreatedAt      time.Time       `json:"created_at" format:"date-time"`
	UpdatedAt      time.Time       `json:"updated_at" format:"date-time"`

	// PeriodsCount is populated by the list/service layer via a
	// dedicated GROUP BY query so the slim list response can carry
	// the count without preloading the full Periods slice. `gorm:"-"`
	// keeps GORM out of it — the column doesn't exist on the table.
	PeriodsCount int `gorm:"-" json:"-"`
}

// GetOrganizationID returns the organization ID for the OrgOwned interface.
func (p *PayPlan) GetOrganizationID() uint {
	return p.OrganizationID
}

// PayPlanPeriod represents a time period with specific pay rates.
// WeeklyHours defines the reference hours for full-time employment.
// EmployerContributionRate is stored in hundredths of a percent (e.g. 2200 = 22.00%).
type PayPlanPeriod struct {
	ID                       uint           `gorm:"primaryKey" json:"id" example:"1"`
	PayPlanID                uint           `gorm:"not null;index" json:"payplan_id" example:"1"`
	PayPlan                  *PayPlan       `gorm:"foreignKey:PayPlanID" json:"-"`
	Period                                  // From, To (embedded)
	WeeklyHours              float64        `gorm:"not null" json:"weekly_hours" example:"39.0"`
	EmployerContributionRate int            `json:"employer_contribution_rate" example:"2200"` // hundredths of percent: 2200 = 22.00%
	Entries                  []PayPlanEntry `gorm:"foreignKey:PeriodID" json:"entries,omitempty"`
	CreatedAt                time.Time      `json:"created_at" format:"date-time"`
	UpdatedAt                time.Time      `json:"updated_at" format:"date-time"`
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
	CreatedAt     time.Time      `json:"created_at" format:"date-time"`
	UpdatedAt     time.Time      `json:"updated_at" format:"date-time"`
}

// PayPlanCreateRequest is the request body for creating a pay plan.
type PayPlanCreateRequest struct {
	Name string `json:"name" binding:"required" example:"TVöD-SuE"`
}

// PayPlanUpdateRequest is the request body for updating a pay plan.
type PayPlanUpdateRequest struct {
	Name *string `json:"name" binding:"omitempty,max=255" example:"TVöD-SuE"`
}

// PayPlanResponse is the response for a pay plan.
type PayPlanResponse struct {
	ID             uint   `json:"id" example:"1"`
	OrganizationID uint   `json:"organization_id" example:"1"`
	Name           string `json:"name" example:"TVöD-SuE"`
	// PeriodsCount is the total number of periods belonging to this
	// pay plan, included so the list view can render the count
	// without fetching the full detail response per row. Populated
	// by PayPlanService.List via a separate GROUP BY query.
	PeriodsCount int       `json:"periods_count" example:"4"`
	CreatedAt    time.Time `json:"created_at" format:"date-time"`
	UpdatedAt    time.Time `json:"updated_at" format:"date-time"`
}

// PayPlanDetailResponse includes periods for detail view.
// YAML tags allow the same struct to be used for YAML import/export.
type PayPlanDetailResponse struct {
	ID             uint                    `json:"id" yaml:"id" example:"1"`
	OrganizationID uint                    `json:"organization_id" yaml:"organization_id" example:"1"`
	Name           string                  `json:"name" yaml:"name" example:"TVöD-SuE"`
	Periods        []PayPlanPeriodResponse `json:"periods" yaml:"periods"`
	CreatedAt      time.Time               `json:"created_at" format:"date-time" yaml:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at" format:"date-time" yaml:"updated_at"`
}

// PayPlanPeriodCreateRequest is the request body for creating a period.
type PayPlanPeriodCreateRequest struct {
	From                     time.Time  `json:"from" format:"date-time" binding:"required" example:"2024-01-01T00:00:00Z"`
	To                       *time.Time `json:"to,omitempty" format:"date-time" example:"2024-12-31T00:00:00Z"`
	WeeklyHours              float64    `json:"weekly_hours" binding:"required,gt=0" example:"39.0"`
	EmployerContributionRate int        `json:"employer_contribution_rate" binding:"min=0,max=10000" example:"2200"` // hundredths of percent: 2200 = 22.00%
}

// PayPlanPeriodUpdateRequest is the request body for updating a period.
type PayPlanPeriodUpdateRequest struct {
	From                     time.Time  `json:"from" format:"date-time" binding:"required" example:"2024-01-01T00:00:00Z"`
	To                       *time.Time `json:"to,omitempty" format:"date-time" example:"2024-12-31T00:00:00Z"`
	WeeklyHours              float64    `json:"weekly_hours" binding:"required,gt=0" example:"39.0"`
	EmployerContributionRate int        `json:"employer_contribution_rate" binding:"min=0,max=10000" example:"2200"` // hundredths of percent: 2200 = 22.00%
}

// PayPlanPeriodResponse is the response for a period.
type PayPlanPeriodResponse struct {
	ID                       uint                   `json:"id" yaml:"id" example:"1"`
	PayPlanID                uint                   `json:"payplan_id" yaml:"payplan_id" example:"1"`
	From                     time.Time              `json:"from" format:"date-time" yaml:"from" example:"2024-01-01T00:00:00Z"`
	To                       *time.Time             `json:"to,omitempty" format:"date-time" yaml:"to" example:"2024-12-31T00:00:00Z"`
	WeeklyHours              float64                `json:"weekly_hours" yaml:"weekly_hours" example:"39.0"`
	EmployerContributionRate int                    `json:"employer_contribution_rate" yaml:"employer_contribution_rate" example:"2200"` // hundredths of percent: 2200 = 22.00%
	Entries                  []PayPlanEntryResponse `json:"entries,omitempty" yaml:"entries"`
	CreatedAt                time.Time              `json:"created_at" format:"date-time" yaml:"created_at"`
	UpdatedAt                time.Time              `json:"updated_at" format:"date-time" yaml:"updated_at"`
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
	ID            uint      `json:"id" yaml:"id" example:"1"`
	PeriodID      uint      `json:"period_id" yaml:"period_id" example:"1"`
	Grade         string    `json:"grade" yaml:"grade" example:"S8a"`
	Step          int       `json:"step" yaml:"step" example:"3"`
	MonthlyAmount int       `json:"monthly_amount" yaml:"monthly_amount" example:"350000"`
	StepMinYears  *int      `json:"step_min_years,omitempty" yaml:"step_min_years" example:"3"`
	CreatedAt     time.Time `json:"created_at" format:"date-time" yaml:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" format:"date-time" yaml:"updated_at"`
}

// ToResponse converts a PayPlan to PayPlanResponse.
func (p *PayPlan) ToResponse() PayPlanResponse {
	return PayPlanResponse{
		ID:             p.ID,
		OrganizationID: p.OrganizationID,
		Name:           p.Name,
		PeriodsCount:   p.PeriodsCount,
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
	entries := make([]PayPlanEntryResponse, len(p.Entries))
	for i, entry := range p.Entries {
		entries[i] = entry.ToResponse()
	}
	return PayPlanPeriodResponse{
		ID:                       p.ID,
		PayPlanID:                p.PayPlanID,
		From:                     p.From,
		To:                       p.To,
		WeeklyHours:              p.WeeklyHours,
		EmployerContributionRate: p.EmployerContributionRate,
		Entries:                  entries,
		CreatedAt:                p.CreatedAt,
		UpdatedAt:                p.UpdatedAt,
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
