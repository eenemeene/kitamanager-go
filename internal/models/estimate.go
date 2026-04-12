package models

import "time"

// ChildFundingEstimateRequest is the input for estimating government funding for a hypothetical child.
type ChildFundingEstimateRequest struct {
	Birthdate  time.Time          `json:"birthdate" binding:"required" example:"2022-05-15T00:00:00Z"`
	Date       *time.Time         `json:"date,omitempty" example:"2026-04-01T00:00:00Z"`
	Properties ContractProperties `json:"properties" binding:"required"`
}

// EmployeeCostEstimateRequest is the input for estimating monthly cost of a hypothetical employee.
type EmployeeCostEstimateRequest struct {
	PayPlanID     uint       `json:"pay_plan_id" binding:"required" example:"1"`
	Grade         string     `json:"grade" binding:"required,max=20" example:"S8a"`
	Step          int        `json:"step" binding:"required,gte=1,lte=10" example:"3"`
	WeeklyHours   float64    `json:"weekly_hours" binding:"required,gt=0" example:"39.4"`
	Date          *time.Time `json:"date,omitempty" example:"2026-04-01T00:00:00Z"`
	StaffCategory string     `json:"staff_category,omitempty" example:"qualified"`
}

// EmployeeCostEstimateResponse is the response from the employee cost estimate endpoint.
type EmployeeCostEstimateResponse struct {
	Date                     string  `json:"date" example:"2026-04-01"`
	StaffCategory            string  `json:"staff_category,omitempty" example:"qualified"`
	Grade                    string  `json:"grade" example:"S8a"`
	Step                     int     `json:"step" example:"3"`
	WeeklyHours              float64 `json:"weekly_hours" example:"39.4"`
	PayPlanWeeklyHours       float64 `json:"pay_plan_weekly_hours" example:"39.4"`
	FullTimeMonthlyAmount    int     `json:"full_time_monthly_amount" example:"380000"`
	GrossSalary              int     `json:"gross_salary" example:"380000"`
	EmployerContributionRate int     `json:"employer_contribution_rate" example:"2200"`
	EmployerCosts            int     `json:"employer_costs" example:"83600"`
	TotalMonthlyCost         int     `json:"total_monthly_cost" example:"463600"`
}
