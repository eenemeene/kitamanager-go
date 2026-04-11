package models

import "time"

// ForecastRequest is the input for the forecast endpoint.
// It combines date range/section filters with hypothetical overlay modifications.
type ForecastRequest struct {
	From      *time.Time `json:"from,omitempty"`
	To        *time.Time `json:"to,omitempty"`
	SectionID *uint      `json:"section_id,omitempty"`

	// Employee overlays
	AddEmployees         []ForecastAddEmployee         `json:"add_employees,omitempty"`
	RemoveEmployeeIDs    []uint                        `json:"remove_employee_ids,omitempty"`
	AddEmployeeContracts []ForecastAddEmployeeContract `json:"add_employee_contracts,omitempty"`
	EndEmployeeContracts []ForecastEndContract         `json:"end_employee_contracts,omitempty"`

	// Child overlays
	AddChildren       []ForecastAddChild         `json:"add_children,omitempty"`
	RemoveChildIDs    []uint                     `json:"remove_child_ids,omitempty"`
	AddChildContracts []ForecastAddChildContract `json:"add_child_contracts,omitempty"`
	EndChildContracts []ForecastEndContract      `json:"end_child_contracts,omitempty"`

	// Pay plan overlays
	AddPayPlanPeriods []ForecastAddPayPlanPeriod `json:"add_pay_plan_periods,omitempty"`

	// Government funding overlays
	AddFundingPeriods []ForecastAddFundingPeriod `json:"add_funding_periods,omitempty"`

	// Budget item overlays
	AddBudgetItems      []ForecastAddBudgetItem `json:"add_budget_items,omitempty"`
	RemoveBudgetItemIDs []uint                  `json:"remove_budget_item_ids,omitempty"`
}

// ForecastAddEmployee adds a virtual employee with inline contracts.
type ForecastAddEmployee struct {
	FirstName string                        `json:"first_name" binding:"required" example:"Maria"`
	LastName  string                        `json:"last_name" binding:"required" example:"Musterfrau"`
	Gender    string                        `json:"gender" binding:"required" example:"female"`
	Birthdate time.Time                     `json:"birthdate" binding:"required" example:"1990-05-15"`
	Contracts []ForecastAddEmployeeContract `json:"contracts" binding:"required,min=1"`
}

// ForecastAddEmployeeContract adds a contract to an existing or new employee.
// When nested inside ForecastAddEmployee, EmployeeID is ignored.
// When used standalone in AddEmployeeContracts, EmployeeID must reference an existing employee.
type ForecastAddEmployeeContract struct {
	EmployeeID    uint       `json:"employee_id,omitempty" example:"0"`
	From          time.Time  `json:"from" binding:"required" example:"2026-08-01"`
	To            *time.Time `json:"to,omitempty" example:"2027-07-31"`
	SectionID     uint       `json:"section_id" binding:"required" example:"1"`
	StaffCategory string     `json:"staff_category" binding:"required" example:"qualified"`
	Grade         string     `json:"grade" example:"S8a"`
	Step          int        `json:"step" example:"3"`
	WeeklyHours   float64    `json:"weekly_hours" binding:"required" example:"39.0"`
	PayPlanID     uint       `json:"pay_plan_id" binding:"required" example:"1"`
}

// ForecastEndContract sets an end date on an existing contract (employee or child).
type ForecastEndContract struct {
	ContractID uint      `json:"contract_id" binding:"required" example:"5"`
	EndDate    time.Time `json:"end_date" binding:"required" example:"2026-07-31"`
}

// ForecastAddChild adds a virtual child with inline contracts.
type ForecastAddChild struct {
	FirstName string                     `json:"first_name" binding:"required" example:"Emma"`
	LastName  string                     `json:"last_name" binding:"required" example:"Schmidt"`
	Gender    string                     `json:"gender" binding:"required" example:"female"`
	Birthdate time.Time                  `json:"birthdate" binding:"required" example:"2023-03-10"`
	Contracts []ForecastAddChildContract `json:"contracts" binding:"required,min=1"`
}

// ForecastAddChildContract adds a contract to an existing or new child.
// When nested inside ForecastAddChild, ChildID is ignored.
// When used standalone in AddChildContracts, ChildID must reference an existing child.
type ForecastAddChildContract struct {
	ChildID    uint               `json:"child_id,omitempty" example:"0"`
	From       time.Time          `json:"from" binding:"required" example:"2026-08-01"`
	To         *time.Time         `json:"to,omitempty" example:"2027-07-31"`
	SectionID  uint               `json:"section_id" binding:"required" example:"1"`
	Properties ContractProperties `json:"properties,omitempty"`
}

// ForecastAddPayPlanPeriod adds a new period to an existing pay plan.
// This models salary increases: copy existing entries with new amounts.
type ForecastAddPayPlanPeriod struct {
	PayPlanID                uint                   `json:"pay_plan_id" binding:"required" example:"1"`
	From                     time.Time              `json:"from" binding:"required" example:"2027-01-01"`
	To                       *time.Time             `json:"to,omitempty" example:"2027-12-31"`
	WeeklyHours              float64                `json:"weekly_hours" binding:"required" example:"39.0"`
	EmployerContributionRate int                    `json:"employer_contribution_rate" binding:"required" example:"2200"`
	Entries                  []ForecastPayPlanEntry `json:"entries" binding:"required,min=1"`
}

// ForecastPayPlanEntry represents a single grade/step salary amount in a forecast pay plan period.
type ForecastPayPlanEntry struct {
	Grade         string `json:"grade" binding:"required" example:"S8a"`
	Step          int    `json:"step" binding:"required" example:"3"`
	MonthlyAmount int    `json:"monthly_amount" binding:"required" example:"380000"`
}

// ForecastAddFundingPeriod adds a hypothetical government funding period.
type ForecastAddFundingPeriod struct {
	From                time.Time                 `json:"from" binding:"required" example:"2027-08-01"`
	To                  *time.Time                `json:"to,omitempty" example:"2028-07-31"`
	FullTimeWeeklyHours float64                   `json:"full_time_weekly_hours" binding:"required" example:"39.0"`
	Properties          []ForecastFundingProperty `json:"properties" binding:"required,min=1"`
}

// ForecastFundingProperty represents a single funding property within a forecast funding period.
type ForecastFundingProperty struct {
	Key                 string  `json:"key" binding:"required" example:"care_type"`
	Value               string  `json:"value" binding:"required" example:"ganztag"`
	Label               string  `json:"label" binding:"required" example:"Ganztag"`
	Payment             int     `json:"payment" binding:"required" example:"166847"`
	Requirement         float64 `json:"requirement" example:"0.261"`
	MinAge              *int    `json:"min_age,omitempty" example:"0"`
	MaxAge              *int    `json:"max_age,omitempty" example:"3"`
	ApplyToAllContracts bool    `json:"apply_to_all_contracts" example:"false"`
}

// ForecastAddBudgetItem adds a virtual budget item with entries.
type ForecastAddBudgetItem struct {
	Name     string                    `json:"name" binding:"required" example:"Elternbeiträge"`
	Category string                    `json:"category" binding:"required" example:"income"`
	PerChild bool                      `json:"per_child" example:"true"`
	Entries  []ForecastBudgetItemEntry `json:"entries" binding:"required,min=1"`
}

// ForecastBudgetItemEntry represents a time-bound amount for a forecast budget item.
type ForecastBudgetItemEntry struct {
	From        time.Time  `json:"from" binding:"required" example:"2027-01-01"`
	To          *time.Time `json:"to,omitempty" example:"2027-12-31"`
	AmountCents int        `json:"amount_cents" binding:"required" example:"50000"`
}

// ForecastResponse is the combined response from the forecast endpoint.
type ForecastResponse struct {
	Financials            *FinancialResponse             `json:"financials,omitempty"`
	StaffingHours         *StaffingHoursResponse         `json:"staffing_hours,omitempty"`
	Occupancy             *OccupancyResponse             `json:"occupancy,omitempty"`
	EmployeeStaffingHours *EmployeeStaffingHoursResponse `json:"employee_staffing_hours,omitempty"`
}
