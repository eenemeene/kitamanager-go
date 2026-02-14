package service

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// PayPlanService handles business logic for pay plans.
type PayPlanService struct {
	store store.PayPlanStorer
}

// NewPayPlanService creates a new PayPlanService.
func NewPayPlanService(store store.PayPlanStorer) *PayPlanService {
	return &PayPlanService{store: store}
}

// verifyPayPlanOwnership verifies a pay plan exists and belongs to the organization.
func (s *PayPlanService) verifyPayPlanOwnership(ctx context.Context, payplanID, orgID uint) (*models.PayPlan, error) {
	payplan, err := s.store.FindByID(ctx, payplanID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("pay plan")
		}
		return nil, apperror.InternalWrap(err, "failed to fetch pay plan")
	}
	if payplan.OrganizationID != orgID {
		return nil, apperror.NotFound("pay plan")
	}
	return payplan, nil
}

// verifyPeriodOwnership verifies a period exists and belongs to the pay plan.
func (s *PayPlanService) verifyPeriodOwnership(ctx context.Context, periodID, payplanID uint) (*models.PayPlanPeriod, error) {
	period, err := s.store.FindPeriodByID(ctx, periodID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("period")
		}
		return nil, apperror.InternalWrap(err, "failed to fetch period")
	}
	if period.PayPlanID != payplanID {
		return nil, apperror.NotFound("period")
	}
	return period, nil
}

// verifyEntryOwnership verifies an entry exists and belongs to the period.
func (s *PayPlanService) verifyEntryOwnership(ctx context.Context, entryID, periodID uint) (*models.PayPlanEntry, error) {
	entry, err := s.store.FindEntryByID(ctx, entryID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("entry")
		}
		return nil, apperror.InternalWrap(err, "failed to fetch entry")
	}
	if entry.PeriodID != periodID {
		return nil, apperror.NotFound("entry")
	}
	return entry, nil
}

// Create creates a new pay plan.
func (s *PayPlanService) Create(ctx context.Context, orgID uint, req *models.PayPlanCreateRequest) (*models.PayPlanResponse, error) {
	payplan := &models.PayPlan{
		OrganizationID: orgID,
		Name:           req.Name,
	}

	if err := s.store.Create(ctx, payplan); err != nil {
		return nil, apperror.InternalWrap(err, "failed to create pay plan")
	}

	resp := payplan.ToResponse()
	return &resp, nil
}

// GetByID retrieves a pay plan by ID.
// activeOn filters periods to those active on the given date (nil = no filter).
func (s *PayPlanService) GetByID(ctx context.Context, id, orgID uint, activeOn *time.Time) (*models.PayPlanDetailResponse, error) {
	payplan, err := s.store.FindByIDWithPeriods(ctx, id, activeOn)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("pay plan")
		}
		return nil, apperror.InternalWrap(err, "failed to fetch pay plan")
	}

	if payplan.OrganizationID != orgID {
		return nil, apperror.NotFound("pay plan")
	}

	resp := payplan.ToDetailResponse()
	return &resp, nil
}

// List retrieves all pay plans for an organization.
func (s *PayPlanService) List(ctx context.Context, orgID uint, limit, offset int) ([]models.PayPlanResponse, int64, error) {
	payplans, total, err := s.store.FindByOrganization(ctx, orgID, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch pay plans")
	}

	return toResponseList(payplans, (*models.PayPlan).ToResponse), total, nil
}

// Update updates a pay plan.
func (s *PayPlanService) Update(ctx context.Context, id, orgID uint, req *models.PayPlanUpdateRequest) (*models.PayPlanResponse, error) {
	payplan, err := s.verifyPayPlanOwnership(ctx, id, orgID)
	if err != nil {
		return nil, err
	}

	payplan.Name = req.Name

	if err := s.store.Update(ctx, payplan); err != nil {
		return nil, apperror.InternalWrap(err, "failed to update pay plan")
	}

	resp := payplan.ToResponse()
	return &resp, nil
}

// Delete deletes a pay plan.
func (s *PayPlanService) Delete(ctx context.Context, id, orgID uint) error {
	if _, err := s.verifyPayPlanOwnership(ctx, id, orgID); err != nil {
		return err
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return apperror.InternalWrap(err, "failed to delete pay plan")
	}
	return nil
}

// Period operations

// CreatePeriod creates a new period for a pay plan.
func (s *PayPlanService) CreatePeriod(ctx context.Context, payplanID, orgID uint, req *models.PayPlanPeriodCreateRequest) (*models.PayPlanPeriodResponse, error) {
	if _, err := s.verifyPayPlanOwnership(ctx, payplanID, orgID); err != nil {
		return nil, err
	}

	period := &models.PayPlanPeriod{
		PayPlanID:   payplanID,
		From:        req.From,
		To:          req.To,
		WeeklyHours: req.WeeklyHours,
	}

	if err := s.store.CreatePeriod(ctx, period); err != nil {
		return nil, apperror.InternalWrap(err, "failed to create period")
	}

	resp := period.ToResponse()
	return &resp, nil
}

// GetPeriod retrieves a period by ID.
func (s *PayPlanService) GetPeriod(ctx context.Context, periodID, payplanID, orgID uint) (*models.PayPlanPeriodResponse, error) {
	if _, err := s.verifyPayPlanOwnership(ctx, payplanID, orgID); err != nil {
		return nil, err
	}

	period, err := s.store.FindPeriodByIDWithEntries(ctx, periodID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("period")
		}
		return nil, apperror.InternalWrap(err, "failed to fetch period")
	}

	if period.PayPlanID != payplanID {
		return nil, apperror.NotFound("period")
	}

	resp := period.ToResponse()
	return &resp, nil
}

// UpdatePeriod updates a period.
func (s *PayPlanService) UpdatePeriod(ctx context.Context, periodID, payplanID, orgID uint, req *models.PayPlanPeriodUpdateRequest) (*models.PayPlanPeriodResponse, error) {
	if _, err := s.verifyPayPlanOwnership(ctx, payplanID, orgID); err != nil {
		return nil, err
	}

	period, err := s.verifyPeriodOwnership(ctx, periodID, payplanID)
	if err != nil {
		return nil, err
	}

	period.From = req.From
	period.To = req.To
	period.WeeklyHours = req.WeeklyHours

	if err := s.store.UpdatePeriod(ctx, period); err != nil {
		return nil, apperror.InternalWrap(err, "failed to update period")
	}

	resp := period.ToResponse()
	return &resp, nil
}

// DeletePeriod deletes a period.
func (s *PayPlanService) DeletePeriod(ctx context.Context, periodID, payplanID, orgID uint) error {
	if _, err := s.verifyPayPlanOwnership(ctx, payplanID, orgID); err != nil {
		return err
	}

	if _, err := s.verifyPeriodOwnership(ctx, periodID, payplanID); err != nil {
		return err
	}

	if err := s.store.DeletePeriod(ctx, periodID); err != nil {
		return apperror.InternalWrap(err, "failed to delete period")
	}
	return nil
}

// Entry operations

// CreateEntry creates a new entry for a period.
func (s *PayPlanService) CreateEntry(ctx context.Context, periodID, payplanID, orgID uint, req *models.PayPlanEntryCreateRequest) (*models.PayPlanEntryResponse, error) {
	if _, err := s.verifyPayPlanOwnership(ctx, payplanID, orgID); err != nil {
		return nil, err
	}

	if _, err := s.verifyPeriodOwnership(ctx, periodID, payplanID); err != nil {
		return nil, err
	}

	entry := &models.PayPlanEntry{
		PeriodID:      periodID,
		Grade:         req.Grade,
		Step:          req.Step,
		MonthlyAmount: req.MonthlyAmount,
		StepMinYears:  req.StepMinYears,
	}

	if err := s.store.CreateEntry(ctx, entry); err != nil {
		return nil, apperror.InternalWrap(err, "failed to create entry")
	}

	resp := entry.ToResponse()
	return &resp, nil
}

// GetEntry retrieves an entry by ID.
func (s *PayPlanService) GetEntry(ctx context.Context, entryID, periodID, payplanID, orgID uint) (*models.PayPlanEntryResponse, error) {
	if _, err := s.verifyPayPlanOwnership(ctx, payplanID, orgID); err != nil {
		return nil, err
	}

	if _, err := s.verifyPeriodOwnership(ctx, periodID, payplanID); err != nil {
		return nil, err
	}

	entry, err := s.verifyEntryOwnership(ctx, entryID, periodID)
	if err != nil {
		return nil, err
	}

	resp := entry.ToResponse()
	return &resp, nil
}

// UpdateEntry updates an entry.
func (s *PayPlanService) UpdateEntry(ctx context.Context, entryID, periodID, payplanID, orgID uint, req *models.PayPlanEntryUpdateRequest) (*models.PayPlanEntryResponse, error) {
	if _, err := s.verifyPayPlanOwnership(ctx, payplanID, orgID); err != nil {
		return nil, err
	}

	if _, err := s.verifyPeriodOwnership(ctx, periodID, payplanID); err != nil {
		return nil, err
	}

	entry, err := s.verifyEntryOwnership(ctx, entryID, periodID)
	if err != nil {
		return nil, err
	}

	entry.Grade = req.Grade
	entry.Step = req.Step
	entry.MonthlyAmount = req.MonthlyAmount
	entry.StepMinYears = req.StepMinYears

	if err := s.store.UpdateEntry(ctx, entry); err != nil {
		return nil, apperror.InternalWrap(err, "failed to update entry")
	}

	resp := entry.ToResponse()
	return &resp, nil
}

// DeleteEntry deletes an entry.
func (s *PayPlanService) DeleteEntry(ctx context.Context, entryID, periodID, payplanID, orgID uint) error {
	if _, err := s.verifyPayPlanOwnership(ctx, payplanID, orgID); err != nil {
		return err
	}

	if _, err := s.verifyPeriodOwnership(ctx, periodID, payplanID); err != nil {
		return err
	}

	if _, err := s.verifyEntryOwnership(ctx, entryID, periodID); err != nil {
		return err
	}

	if err := s.store.DeleteEntry(ctx, entryID); err != nil {
		return apperror.InternalWrap(err, "failed to delete entry")
	}
	return nil
}

// CalculateSalary calculates the monthly salary based on pay plan, grade, step, and working hours.
func (s *PayPlanService) CalculateSalary(ctx context.Context, payplanID uint, grade string, step int, weeklyHours float64, date time.Time) (int, error) {
	period, err := s.store.FindActivePeriod(ctx, payplanID, date)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return 0, apperror.NotFound("no active pay plan period found for the given date")
		}
		return 0, apperror.InternalWrap(err, "failed to fetch pay plan period")
	}

	entry, err := s.store.FindEntry(ctx, period.ID, grade, step)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return 0, apperror.NotFound("no pay plan entry found for the given grade and step")
		}
		return 0, apperror.InternalWrap(err, "failed to fetch pay plan entry")
	}

	// Calculate salary: MonthlyAmount * (employeeWeeklyHours / periodWeeklyHours)
	salary := float64(entry.MonthlyAmount) * (weeklyHours / period.WeeklyHours)
	return int(math.Round(salary)), nil
}
