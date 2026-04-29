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

// FinancialBudgetItemDetail provides a breakdown of a single budget item's contribution.
// For per-child items, AmountCents is the aggregate (UnitAmountCents × child count);
// UnitAmountCents carries the unit price so the UI can render both without re-fetching.
type FinancialBudgetItemDetail struct {
	Name            string `json:"name" example:"Elternbeiträge"`
	Category        string `json:"category" example:"income"`
	AmountCents     int    `json:"amount_cents" example:"50000"`
	PerChild        bool   `json:"per_child" example:"true"`
	UnitAmountCents int    `json:"unit_amount_cents" example:"5000"`
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
	ActualFunding           *int `json:"actual_funding,omitempty" example:"5100000"`           // cents, nil if no bill for this month
	ActualFundingRegular    *int `json:"actual_funding_regular,omitempty" example:"5000000"`   // cents, regular billing only
	ActualFundingCorrection *int `json:"actual_funding_correction,omitempty" example:"100000"` // cents, correction rows only
	// Counts
	ChildCount int `json:"child_count" example:"45"`
	StaffCount int `json:"staff_count" example:"12"`
	// Breakdowns
	BudgetItemDetails []FinancialBudgetItemDetail `json:"budget_item_details,omitempty"`
	FundingDetails    []FinancialFundingDetail    `json:"funding_details,omitempty"`
	SalaryDetails     []FinancialSalaryDetail     `json:"salary_details,omitempty"`
}

// FinancialResponse represents the response for financial statistics.
//
// Warnings collects per-row, non-fatal data-quality issues hit during the
// month-by-month walk: an employee contract that references an unknown pay
// plan, a contract date that no period covers, a (grade,step) combination
// missing from the pay plan's entries. The salary for that employee in
// that month is excluded from the totals (so the numbers stay
// deterministic instead of silently zero-padding) and a warning entry is
// appended so the caller can show "1 employee's salary was excluded — pay
// plan X has no period for 2026-03." Empty when the dataset is clean.
type FinancialResponse struct {
	DataPoints []FinancialDataPoint `json:"data_points"`
	Warnings   []CalculationWarning `json:"warnings,omitempty"`
}

// CalculationWarning is a per-row, non-fatal data-quality issue surfaced
// from a statistics or forecast calculation. Encoded so the frontend can
// group/translate by Code rather than by string-matching Message. Numeric
// metadata is omitted (json `omitempty`) so warnings unrelated to a
// specific entity stay compact.
//
// Codes are stable strings, owned by the service layer; new codes can be
// added freely but existing ones MUST keep their semantics. Current set:
//
//   - missing_pay_plan         — contract.PayPlanID has no row in the
//     loaded pay plan map (data references a
//     row that no longer exists).
//   - no_pay_plan_period       — pay plan exists but no period covers the
//     contract date.
//   - no_pay_plan_entry        — period exists but the (grade, step)
//     combination is not in its entries.
//   - budget_items_load_failed — budget items query failed; expense
//     breakdown excludes operating costs for
//     this response. Distinct from "no budget
//     items configured" (which is silent).
//   - funding_bills_load_failed — actual government funding bill totals
//     could not be loaded; ActualFunding /
//     ActualFundingRegular / ActualFundingCorrection
//     fields are absent on data points that would
//     otherwise have had them. Distinct from
//     "no bill recorded for this month" (silent).
type CalculationWarning struct {
	Code       string `json:"code" example:"missing_pay_plan"`
	Message    string `json:"message" example:"employee contract references unknown pay plan"`
	EmployeeID uint   `json:"employee_id,omitempty" example:"42"`
	ContractID uint   `json:"contract_id,omitempty" example:"99"`
	PayPlanID  uint   `json:"payplan_id,omitempty" example:"7"`
	Grade      string `json:"grade,omitempty" example:"S8a"`
	Step       int    `json:"step,omitempty" example:"3"`
	Date       string `json:"date,omitempty" example:"2026-03-01"`
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
