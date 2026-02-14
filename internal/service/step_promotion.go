package service

import (
	"context"
	"math"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// StepPromotionService handles step promotion calculations.
type StepPromotionService struct {
	payPlanStore  store.PayPlanStorer
	employeeStore store.EmployeeStorer
}

// NewStepPromotionService creates a new StepPromotionService.
func NewStepPromotionService(payPlanStore store.PayPlanStorer, employeeStore store.EmployeeStorer) *StepPromotionService {
	return &StepPromotionService{payPlanStore: payPlanStore, employeeStore: employeeStore}
}

// EarliestContractStart returns the earliest contract From date, or zero time if empty.
func EarliestContractStart(contracts []models.EmployeeContract) time.Time {
	if len(contracts) == 0 {
		return time.Time{}
	}
	earliest := contracts[0].From
	for _, c := range contracts[1:] {
		if c.From.Before(earliest) {
			earliest = c.From
		}
	}
	return earliest
}

// CalculateYearsOfService returns years since the earliest contract From date.
func CalculateYearsOfService(contracts []models.EmployeeContract, asOf time.Time) float64 {
	earliest := EarliestContractStart(contracts)
	if earliest.IsZero() {
		return 0
	}

	years := asOf.Sub(earliest).Hours() / (24 * 365.25)
	if years < 0 {
		return 0
	}
	return years
}

// DetermineEligibleStep returns the highest step where StepMinYears <= yearsOfService
// for the given grade. Returns 0 if no entries have step rules.
func DetermineEligibleStep(yearsOfService float64, entries []models.PayPlanEntry, grade string) int {
	bestStep := 0
	for _, e := range entries {
		if e.Grade != grade || e.StepMinYears == nil {
			continue
		}
		if float64(*e.StepMinYears) <= yearsOfService {
			if e.Step > bestStep {
				bestStep = e.Step
			}
		}
	}
	return bestStep
}

// GetStepPromotions computes employees eligible for step promotions in an organization.
func (s *StepPromotionService) GetStepPromotions(ctx context.Context, orgID uint, date time.Time) (*models.StepPromotionsResponse, error) {
	employees, err := s.employeeStore.FindByOrganizationWithContracts(ctx, orgID, date)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch employees for step promotions")
	}

	// Collect unique payplan IDs from active contracts
	payplanIDs := make(map[uint]bool)
	for _, emp := range employees {
		for _, c := range emp.Contracts {
			if isActiveOn(c, date) {
				payplanIDs[c.PayPlanID] = true
			}
		}
	}

	// Fetch active periods with entries for each payplan
	type periodInfo struct {
		period  *models.PayPlanPeriod
		payplan *models.PayPlan
	}
	periodsByPayPlan := make(map[uint]*periodInfo)
	for ppID := range payplanIDs {
		period, err := s.payPlanStore.FindActivePeriod(ctx, ppID, date)
		if err != nil {
			continue // skip payplans without active period
		}
		pp, err := s.payPlanStore.FindByID(ctx, ppID)
		if err != nil {
			continue
		}
		periodsByPayPlan[ppID] = &periodInfo{period: period, payplan: pp}
	}

	var promotions []models.StepPromotionResponse
	totalDelta := 0

	for _, emp := range employees {
		// Find the active contract on date
		var activeContract *models.EmployeeContract
		for i := range emp.Contracts {
			if isActiveOn(emp.Contracts[i], date) {
				activeContract = &emp.Contracts[i]
				break
			}
		}
		if activeContract == nil {
			continue
		}

		pi, ok := periodsByPayPlan[activeContract.PayPlanID]
		if !ok {
			continue
		}

		serviceStart := EarliestContractStart(emp.Contracts)
		yearsOfService := CalculateYearsOfService(emp.Contracts, date)
		eligibleStep := DetermineEligibleStep(yearsOfService, pi.period.Entries, activeContract.Grade)

		if eligibleStep <= activeContract.Step {
			continue
		}

		// Look up current and new amounts
		var currentAmount, newAmount int
		for _, e := range pi.period.Entries {
			if e.Grade == activeContract.Grade && e.Step == activeContract.Step {
				currentAmount = int(math.Round(float64(e.MonthlyAmount) * activeContract.WeeklyHours / pi.period.WeeklyHours))
			}
			if e.Grade == activeContract.Grade && e.Step == eligibleStep {
				newAmount = int(math.Round(float64(e.MonthlyAmount) * activeContract.WeeklyHours / pi.period.WeeklyHours))
			}
		}

		delta := newAmount - currentAmount
		totalDelta += delta

		promotions = append(promotions, models.StepPromotionResponse{
			EmployeeID:       emp.ID,
			EmployeeName:     emp.FirstName + " " + emp.LastName,
			CurrentStep:      activeContract.Step,
			EligibleStep:     eligibleStep,
			YearsOfService:   yearsOfService,
			ServiceStart:     serviceStart.Format(models.DateFormat),
			Grade:            activeContract.Grade,
			CurrentAmount:    currentAmount,
			NewAmount:        newAmount,
			MonthlyCostDelta: delta,
			PayPlanID:        activeContract.PayPlanID,
			PayPlanName:      pi.payplan.Name,
		})
	}

	if promotions == nil {
		promotions = []models.StepPromotionResponse{}
	}

	return &models.StepPromotionsResponse{
		Date:                  date.Format(models.DateFormat),
		TotalMonthlyCostDelta: totalDelta,
		Promotions:            promotions,
	}, nil
}

// isActiveOn checks if a contract is active on the given date.
func isActiveOn(c models.EmployeeContract, date time.Time) bool {
	if c.From.After(date) {
		return false
	}
	if c.To != nil && c.To.Before(date) {
		return false
	}
	return true
}
