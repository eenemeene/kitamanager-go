package models

// StaffingHoursDataPoint represents a single monthly data point for staffing hours
type StaffingHoursDataPoint struct {
	Date           string  `json:"date" example:"2025-01-01"`
	RequiredHours  float64 `json:"required_hours" example:"312.5"`
	AvailableHours float64 `json:"available_hours" example:"340.0"`
	ChildCount     int     `json:"child_count" example:"45"`
	StaffCount     int     `json:"staff_count" example:"12"`
}

// StaffingHoursResponse represents the response for staffing hours statistics
type StaffingHoursResponse struct {
	DataPoints []StaffingHoursDataPoint `json:"data_points"`
}

// FinancialDataPoint represents a single monthly data point for financial overview
type FinancialDataPoint struct {
	Date string `json:"date" example:"2025-01-01"`
	// Income
	FundingIncome int `json:"funding_income" example:"5000000"` // cents
	// Expenses
	GrossSalary   int `json:"gross_salary" example:"3500000"`  // cents
	EmployerCosts int `json:"employer_costs" example:"770000"` // cents
	OperatingCost int `json:"operating_cost" example:"500000"` // cents
	// Totals
	TotalIncome   int `json:"total_income" example:"5000000"`   // cents
	TotalExpenses int `json:"total_expenses" example:"4770000"` // cents
	Balance       int `json:"balance" example:"230000"`         // cents (income - expenses)
	// Counts
	ChildCount int `json:"child_count" example:"45"`
	StaffCount int `json:"staff_count" example:"12"`
}

// FinancialResponse represents the response for financial statistics
type FinancialResponse struct {
	DataPoints []FinancialDataPoint `json:"data_points"`
}

// OccupancyAgeGroup describes an age group derived from government funding configuration
type OccupancyAgeGroup struct {
	Label  string `json:"label" example:"0/1"`
	MinAge int    `json:"min_age" example:"0"`
	MaxAge int    `json:"max_age" example:"1"`
}

// OccupancySupplementType describes a non-care_type funding property (e.g. integration, ndh)
type OccupancySupplementType struct {
	Key   string `json:"key" example:"integration"`
	Value string `json:"value" example:"integration a"`
	Label string `json:"label" example:"Integration A"`
}

// OccupancyDataPoint represents a single monthly snapshot of the occupancy matrix
type OccupancyDataPoint struct {
	Date             string                    `json:"date" example:"2026-01-01"`
	Total            int                       `json:"total" example:"45"`
	ByAgeAndCareType map[string]map[string]int `json:"by_age_and_care_type"`
	BySupplement     map[string]int            `json:"by_supplement"`
}

// OccupancyResponse represents the full occupancy matrix response
type OccupancyResponse struct {
	AgeGroups       []OccupancyAgeGroup       `json:"age_groups"`
	CareTypes       []string                  `json:"care_types"`
	SupplementTypes []OccupancySupplementType `json:"supplement_types"`
	DataPoints      []OccupancyDataPoint      `json:"data_points"`
}

// ContractPropertyCount represents the count of a specific property key-value pair across children
type ContractPropertyCount struct {
	Key   string `json:"key" example:"care_type"`
	Value string `json:"value" example:"ganztag"`
	Count int    `json:"count" example:"20"`
}

// ContractPropertiesDistributionResponse represents the distribution of contract properties
type ContractPropertiesDistributionResponse struct {
	Date          string                  `json:"date" example:"2026-02-15"`
	TotalChildren int                     `json:"total_children" example:"45"`
	Properties    []ContractPropertyCount `json:"properties"`
}
