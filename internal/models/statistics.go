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

// FinancialBudgetItemDetail provides a breakdown of a single budget item's contribution
type FinancialBudgetItemDetail struct {
	Name        string `json:"name" example:"Elternbeiträge"`
	Category    string `json:"category" example:"income"`
	AmountCents int    `json:"amount_cents" example:"50000"`
}

// FinancialFundingDetail provides a breakdown of a single funding property's contribution
type FinancialFundingDetail struct {
	Key         string `json:"key" example:"care_type"`
	Value       string `json:"value" example:"ganztag"`
	Label       string `json:"label" example:"Ganztag"`
	AmountCents int    `json:"amount_cents" example:"166847"`
}

// FinancialSalaryDetail provides a breakdown of salary costs by staff category
type FinancialSalaryDetail struct {
	StaffCategory string `json:"staff_category" example:"qualified"`
	GrossSalary   int    `json:"gross_salary" example:"300000"`
	EmployerCosts int    `json:"employer_costs" example:"66000"`
}

// FinancialDataPoint represents a single monthly data point for financial overview
type FinancialDataPoint struct {
	Date string `json:"date" example:"2025-01-01"`
	// Income
	FundingIncome int `json:"funding_income" example:"5000000"` // cents
	// Expenses
	GrossSalary    int `json:"gross_salary" example:"3500000"`   // cents
	EmployerCosts  int `json:"employer_costs" example:"770000"`  // cents
	BudgetIncome   int `json:"budget_income" example:"200000"`   // cents
	BudgetExpenses int `json:"budget_expenses" example:"300000"` // cents
	// Totals
	TotalIncome   int `json:"total_income" example:"5000000"`   // cents
	TotalExpenses int `json:"total_expenses" example:"4770000"` // cents
	Balance       int `json:"balance" example:"230000"`         // cents (income - expenses)
	// Actual funding from government funding bills
	ActualFunding *int `json:"actual_funding,omitempty" example:"5100000"` // cents, nil if no bill for this month
	// Counts
	ChildCount int `json:"child_count" example:"45"`
	StaffCount int `json:"staff_count" example:"12"`
	// Breakdowns
	BudgetItemDetails []FinancialBudgetItemDetail `json:"budget_item_details,omitempty"`
	FundingDetails    []FinancialFundingDetail    `json:"funding_details,omitempty"`
	SalaryDetails     []FinancialSalaryDetail     `json:"salary_details,omitempty"`
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

// OccupancyCareType describes a care_type funding property (e.g. ganztag, halbtag)
type OccupancyCareType struct {
	Value string `json:"value" example:"ganztag"`
	Label string `json:"label" example:"Ganztag (bis 9h)"`
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
	CareTypes       []OccupancyCareType       `json:"care_types"`
	SupplementTypes []OccupancySupplementType `json:"supplement_types"`
	DataPoints      []OccupancyDataPoint      `json:"data_points"`
}

// ContractPropertyCount represents the count of a specific property key-value pair across children
type ContractPropertyCount struct {
	Key   string `json:"key" example:"care_type"`
	Value string `json:"value" example:"ganztag"`
	Label string `json:"label" example:"Ganztag (bis 9h)"`
	Count int    `json:"count" example:"20"`
}

// ContractPropertiesDistributionResponse represents the distribution of contract properties
type ContractPropertiesDistributionResponse struct {
	Date          string                  `json:"date" example:"2026-02-15"`
	TotalChildren int                     `json:"total_children" example:"45"`
	Properties    []ContractPropertyCount `json:"properties"`
}

// EmployeeStaffingHoursRow represents a single employee's monthly hours in the staffing grid
type EmployeeStaffingHoursRow struct {
	EmployeeID    uint      `json:"employee_id" example:"1"`
	FirstName     string    `json:"first_name" example:"Max"`
	LastName      string    `json:"last_name" example:"Mustermann"`
	StaffCategory string    `json:"staff_category" example:"qualified"`
	MonthlyHours  []float64 `json:"monthly_hours"`
}

// EmployeeStaffingHoursResponse represents the response for per-employee staffing hours
type EmployeeStaffingHoursResponse struct {
	Dates     []string                   `json:"dates"`
	Employees []EmployeeStaffingHoursRow `json:"employees"`
}
