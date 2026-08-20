package models

import "time"

// ForecastRequest is the input for the forecast endpoint.
// It combines date range/section filters with hypothetical overlay modifications.
// Only children and employees are configurable — pay plans, funding, and budgets
// use the real data as-is.
type ForecastRequest struct {
	From      *time.Time `json:"from,omitempty" format:"date-time"`
	To        *time.Time `json:"to,omitempty" format:"date-time"`
	SectionID *uint      `json:"section_id,omitempty"`

	// Child overlays
	AddChildren       []ForecastChildInput         `json:"add_children,omitempty" binding:"dive"`
	RemoveChildIDs    []uint                       `json:"remove_child_ids,omitempty"`
	AddChildContracts []ForecastChildContractInput `json:"add_child_contracts,omitempty" binding:"dive"`

	// Employee overlays
	AddEmployees         []ForecastEmployeeInput         `json:"add_employees,omitempty" binding:"dive"`
	RemoveEmployeeIDs    []uint                          `json:"remove_employee_ids,omitempty"`
	AddEmployeeContracts []ForecastEmployeeContractInput `json:"add_employee_contracts,omitempty" binding:"dive"`
}

// ForecastResponse is the combined response from the forecast endpoint.
//
// Warnings is the union of per-row data-quality warnings emitted by all
// embedded calculations (today only Financials emits any). See
// CalculationWarning for the code set.
type ForecastResponse struct {
	Financials            *FinancialResponse             `json:"financials,omitempty"`
	StaffingHours         *StaffingHoursResponse         `json:"staffing_hours,omitempty"`
	Occupancy             *OccupancyResponse             `json:"occupancy,omitempty"`
	EmployeeStaffingHours *EmployeeStaffingHoursResponse `json:"employee_staffing_hours,omitempty"`
	Warnings              []CalculationWarning           `json:"warnings,omitempty"`
}

// Forecast overlay inputs.
//
// These describe a hypothetical child, employee or contract as a scenario
// states one: the fields the calculation reads, and nothing else.
//
// Deliberately not the persistence models. ForecastRequest used to carry
// []Child, []ChildContract, []Employee and []EmployeeContract directly, which
// put id, created_at, updated_at, version, a nested organization and a nested
// section into a request body — fields a caller cannot meaningfully supply and
// the server ignores. openapi-fixer marks response properties required, and
// these models are responses elsewhere, so the generated TypeScript demanded
// all of them. The frontend could not build a scenario child that satisfied
// that type and reached for `as unknown as ForecastRequest` instead, which
// switched off type checking on exactly the payload that most needed it.

// ForecastChildContractInput is a child contract in a forecast scenario.
type ForecastChildContractInput struct {
	// ChildID names the existing child a standalone contract attaches to. It is
	// unset for contracts nested under a new child in add_children, which have
	// no id yet.
	ChildID    uint               `json:"child_id,omitempty" example:"1"`
	From       time.Time          `json:"from" binding:"required" format:"date-time" example:"2026-08-01"`
	To         *time.Time         `json:"to,omitempty" format:"date-time" example:"2027-07-31"`
	SectionID  uint               `json:"section_id" binding:"required" example:"2"`
	Properties ContractProperties `json:"properties,omitempty"`
}

// ToModel builds the contract the calculation works on. Id and version stay
// zero: applyOverlay assigns virtual ids, and nothing here is ever persisted.
func (i ForecastChildContractInput) ToModel() ChildContract {
	return ChildContract{
		ChildID: i.ChildID,
		BaseContract: BaseContract{
			Period:     Period{From: i.From, To: i.To},
			SectionID:  i.SectionID,
			Properties: i.Properties,
		},
	}
}

// ForecastChildInput is a hypothetical child, with the contracts that make it
// count towards funding. A child with no contracts is not enrolled and would
// change nothing, so Contracts is required.
type ForecastChildInput struct {
	FirstName string `json:"first_name,omitempty" example:"Emma"`
	LastName  string `json:"last_name,omitempty" example:"Schmidt"`
	Gender    string `json:"gender,omitempty" enums:"male,female,diverse" example:"female"`
	// Birthdate drives the age bracket the funding rate is read from, so it is
	// the one person-level field the calculation actually needs.
	Birthdate time.Time                    `json:"birthdate" binding:"required" format:"date-time" example:"2023-03-10"`
	Contracts []ForecastChildContractInput `json:"contracts" binding:"required,dive"`
}

// ToModel builds the child the calculation works on.
func (i ForecastChildInput) ToModel() Child {
	c := Child{Person: Person{
		FirstName: i.FirstName,
		LastName:  i.LastName,
		Gender:    i.Gender,
		Birthdate: i.Birthdate,
	}}
	c.Contracts = make([]ChildContract, 0, len(i.Contracts))
	for _, ct := range i.Contracts {
		c.Contracts = append(c.Contracts, ct.ToModel())
	}
	return c
}

// ForecastEmployeeContractInput is an employment contract in a forecast
// scenario. The pay-plan fields are all required by the cost calculation:
// grade and step select the row, weekly hours scale it.
type ForecastEmployeeContractInput struct {
	// EmployeeID names the existing employee a standalone contract attaches to.
	// Unset for contracts nested under a new employee in add_employees.
	EmployeeID    uint               `json:"employee_id,omitempty" example:"1"`
	From          time.Time          `json:"from" binding:"required" format:"date-time" example:"2026-08-01"`
	To            *time.Time         `json:"to,omitempty" format:"date-time" example:"2027-07-31"`
	SectionID     uint               `json:"section_id" binding:"required" example:"2"`
	StaffCategory string             `json:"staff_category" binding:"required" example:"qualified"`
	Grade         string             `json:"grade" binding:"required" example:"S8a"`
	Step          int                `json:"step" binding:"min=1" example:"3"`
	WeeklyHours   float64            `json:"weekly_hours" binding:"gt=0" example:"40"`
	PayPlanID     uint               `json:"payplan_id" binding:"required" example:"1"`
	Properties    ContractProperties `json:"properties,omitempty"`
}

// ToModel builds the contract the calculation works on.
func (i ForecastEmployeeContractInput) ToModel() EmployeeContract {
	return EmployeeContract{
		EmployeeID: i.EmployeeID,
		BaseContract: BaseContract{
			Period:     Period{From: i.From, To: i.To},
			SectionID:  i.SectionID,
			Properties: i.Properties,
		},
		StaffCategory: i.StaffCategory,
		Grade:         i.Grade,
		Step:          i.Step,
		WeeklyHours:   i.WeeklyHours,
		PayPlanID:     i.PayPlanID,
	}
}

// ForecastEmployeeInput is a hypothetical employee. Unlike a child, no
// person-level field feeds the cost calculation — it all comes off the
// contracts — so the name and birthdate are carried only so the scenario reads
// as something recognisable.
type ForecastEmployeeInput struct {
	FirstName string                          `json:"first_name,omitempty" example:"Max"`
	LastName  string                          `json:"last_name,omitempty" example:"Mustermann"`
	Gender    string                          `json:"gender,omitempty" enums:"male,female,diverse" example:"male"`
	Birthdate time.Time                       `json:"birthdate,omitempty" format:"date-time" example:"1990-05-15"`
	Contracts []ForecastEmployeeContractInput `json:"contracts" binding:"required,dive"`
}

// ToModel builds the employee the calculation works on.
func (i ForecastEmployeeInput) ToModel() Employee {
	e := Employee{Person: Person{
		FirstName: i.FirstName,
		LastName:  i.LastName,
		Gender:    i.Gender,
		Birthdate: i.Birthdate,
	}}
	e.Contracts = make([]EmployeeContract, 0, len(i.Contracts))
	for _, ct := range i.Contracts {
		e.Contracts = append(e.Contracts, ct.ToModel())
	}
	return e
}
