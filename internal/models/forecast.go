package models

import "time"

// ForecastRequest is the input for the forecast endpoint.
// It combines date range/section filters with hypothetical overlay modifications.
// Only children and employees are configurable — pay plans, funding, and budgets
// use the real data as-is.
type ForecastRequest struct {
	From      *time.Time `json:"from,omitempty"`
	To        *time.Time `json:"to,omitempty"`
	SectionID *uint      `json:"section_id,omitempty"`

	// Child overlays
	AddChildren       []Child         `json:"add_children,omitempty"`
	RemoveChildIDs    []uint          `json:"remove_child_ids,omitempty"`
	AddChildContracts []ChildContract `json:"add_child_contracts,omitempty"`

	// Employee overlays
	AddEmployees         []Employee         `json:"add_employees,omitempty"`
	RemoveEmployeeIDs    []uint             `json:"remove_employee_ids,omitempty"`
	AddEmployeeContracts []EmployeeContract `json:"add_employee_contracts,omitempty"`
}

// ForecastResponse is the combined response from the forecast endpoint.
type ForecastResponse struct {
	Financials            *FinancialResponse             `json:"financials,omitempty"`
	StaffingHours         *StaffingHoursResponse         `json:"staffing_hours,omitempty"`
	Occupancy             *OccupancyResponse             `json:"occupancy,omitempty"`
	EmployeeStaffingHours *EmployeeStaffingHoursResponse `json:"employee_staffing_hours,omitempty"`
}
