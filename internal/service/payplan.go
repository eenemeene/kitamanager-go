package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// PayPlanService handles business logic for pay plans.
type PayPlanService struct {
	store *store.PayPlanStore
}

// NewPayPlanService creates a new PayPlanService.
func NewPayPlanService(store *store.PayPlanStore) *PayPlanService {
	return &PayPlanService{store: store}
}

// Create creates a new pay plan.
func (s *PayPlanService) Create(ctx context.Context, orgID uint, req models.PayPlanCreateRequest) (*models.PayPlanResponse, error) {
	payplan := &models.PayPlan{
		OrganizationID: orgID,
		Name:           req.Name,
	}

	if err := s.store.Create(ctx, payplan); err != nil {
		return nil, apperror.Internal("failed to create pay plan")
	}

	resp := payplan.ToResponse()
	return &resp, nil
}

// GetByID retrieves a pay plan by ID.
func (s *PayPlanService) GetByID(ctx context.Context, id, orgID uint) (*models.PayPlanDetailResponse, error) {
	payplan, err := s.store.GetByIDWithPeriods(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("pay plan")
		}
		return nil, apperror.Internal("failed to fetch pay plan")
	}

	if payplan.OrganizationID != orgID {
		return nil, apperror.NotFound("pay plan")
	}

	resp := payplan.ToDetailResponse()
	return &resp, nil
}

// List retrieves all pay plans for an organization.
func (s *PayPlanService) List(ctx context.Context, orgID uint, limit, offset int) ([]models.PayPlanResponse, int64, error) {
	payplans, total, err := s.store.GetByOrganization(ctx, orgID, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch pay plans")
	}

	responses := make([]models.PayPlanResponse, len(payplans))
	for i, p := range payplans {
		responses[i] = p.ToResponse()
	}

	return responses, total, nil
}

// Update updates a pay plan.
func (s *PayPlanService) Update(ctx context.Context, id, orgID uint, req models.PayPlanUpdateRequest) (*models.PayPlanResponse, error) {
	payplan, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("pay plan")
		}
		return nil, apperror.Internal("failed to fetch pay plan")
	}

	if payplan.OrganizationID != orgID {
		return nil, apperror.NotFound("pay plan")
	}

	payplan.Name = req.Name

	if err := s.store.Update(ctx, payplan); err != nil {
		return nil, apperror.Internal("failed to update pay plan")
	}

	resp := payplan.ToResponse()
	return &resp, nil
}

// Delete deletes a pay plan.
func (s *PayPlanService) Delete(ctx context.Context, id, orgID uint) error {
	payplan, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("pay plan")
		}
		return apperror.Internal("failed to fetch pay plan")
	}

	if payplan.OrganizationID != orgID {
		return apperror.NotFound("pay plan")
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return apperror.Internal("failed to delete pay plan")
	}
	return nil
}

// Period operations

// CreatePeriod creates a new period for a pay plan.
func (s *PayPlanService) CreatePeriod(ctx context.Context, payplanID, orgID uint, req models.PayPlanPeriodCreateRequest) (*models.PayPlanPeriodResponse, error) {
	// Verify pay plan exists and belongs to org
	payplan, err := s.store.GetByID(ctx, payplanID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("pay plan")
		}
		return nil, apperror.Internal("failed to fetch pay plan")
	}
	if payplan.OrganizationID != orgID {
		return nil, apperror.NotFound("pay plan")
	}

	from, err := time.Parse("2006-01-02", req.From)
	if err != nil {
		return nil, apperror.BadRequest("invalid from date format")
	}

	var to *time.Time
	if req.To != nil && *req.To != "" {
		toDate, err := time.Parse("2006-01-02", *req.To)
		if err != nil {
			return nil, apperror.BadRequest("invalid to date format")
		}
		to = &toDate
	}

	period := &models.PayPlanPeriod{
		PayPlanID:   payplanID,
		From:        from,
		To:          to,
		WeeklyHours: req.WeeklyHours,
	}

	if err := s.store.CreatePeriod(ctx, period); err != nil {
		return nil, apperror.Internal("failed to create period")
	}

	resp := period.ToResponse()
	return &resp, nil
}

// GetPeriod retrieves a period by ID.
func (s *PayPlanService) GetPeriod(ctx context.Context, periodID, payplanID, orgID uint) (*models.PayPlanPeriodResponse, error) {
	// Verify pay plan exists and belongs to org
	payplan, err := s.store.GetByID(ctx, payplanID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("pay plan")
		}
		return nil, apperror.Internal("failed to fetch pay plan")
	}
	if payplan.OrganizationID != orgID {
		return nil, apperror.NotFound("pay plan")
	}

	period, err := s.store.GetPeriodByIDWithEntries(ctx, periodID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("period")
		}
		return nil, apperror.Internal("failed to fetch period")
	}

	if period.PayPlanID != payplanID {
		return nil, apperror.NotFound("period")
	}

	resp := period.ToResponse()
	return &resp, nil
}

// UpdatePeriod updates a period.
func (s *PayPlanService) UpdatePeriod(ctx context.Context, periodID, payplanID, orgID uint, req models.PayPlanPeriodUpdateRequest) (*models.PayPlanPeriodResponse, error) {
	// Verify pay plan exists and belongs to org
	payplan, err := s.store.GetByID(ctx, payplanID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("pay plan")
		}
		return nil, apperror.Internal("failed to fetch pay plan")
	}
	if payplan.OrganizationID != orgID {
		return nil, apperror.NotFound("pay plan")
	}

	period, err := s.store.GetPeriodByID(ctx, periodID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("period")
		}
		return nil, apperror.Internal("failed to fetch period")
	}

	if period.PayPlanID != payplanID {
		return nil, apperror.NotFound("period")
	}

	from, err := time.Parse("2006-01-02", req.From)
	if err != nil {
		return nil, apperror.BadRequest("invalid from date format")
	}

	var to *time.Time
	if req.To != nil && *req.To != "" {
		toDate, err := time.Parse("2006-01-02", *req.To)
		if err != nil {
			return nil, apperror.BadRequest("invalid to date format")
		}
		to = &toDate
	}

	period.From = from
	period.To = to
	period.WeeklyHours = req.WeeklyHours

	if err := s.store.UpdatePeriod(ctx, period); err != nil {
		return nil, apperror.Internal("failed to update period")
	}

	resp := period.ToResponse()
	return &resp, nil
}

// DeletePeriod deletes a period.
func (s *PayPlanService) DeletePeriod(ctx context.Context, periodID, payplanID, orgID uint) error {
	// Verify pay plan exists and belongs to org
	payplan, err := s.store.GetByID(ctx, payplanID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("pay plan")
		}
		return apperror.Internal("failed to fetch pay plan")
	}
	if payplan.OrganizationID != orgID {
		return apperror.NotFound("pay plan")
	}

	period, err := s.store.GetPeriodByID(ctx, periodID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("period")
		}
		return apperror.Internal("failed to fetch period")
	}

	if period.PayPlanID != payplanID {
		return apperror.NotFound("period")
	}

	if err := s.store.DeletePeriod(ctx, periodID); err != nil {
		return apperror.Internal("failed to delete period")
	}
	return nil
}

// Entry operations

// CreateEntry creates a new entry for a period.
func (s *PayPlanService) CreateEntry(ctx context.Context, entryReq models.PayPlanEntryCreateRequest, periodID, payplanID, orgID uint) (*models.PayPlanEntryResponse, error) {
	// Verify pay plan exists and belongs to org
	payplan, err := s.store.GetByID(ctx, payplanID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("pay plan")
		}
		return nil, apperror.Internal("failed to fetch pay plan")
	}
	if payplan.OrganizationID != orgID {
		return nil, apperror.NotFound("pay plan")
	}

	// Verify period belongs to pay plan
	period, err := s.store.GetPeriodByID(ctx, periodID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("period")
		}
		return nil, apperror.Internal("failed to fetch period")
	}
	if period.PayPlanID != payplanID {
		return nil, apperror.NotFound("period")
	}

	entry := &models.PayPlanEntry{
		PeriodID:      periodID,
		Grade:         entryReq.Grade,
		Step:          entryReq.Step,
		MonthlyAmount: entryReq.MonthlyAmount,
	}

	if err := s.store.CreateEntry(ctx, entry); err != nil {
		return nil, apperror.Internal("failed to create entry")
	}

	resp := entry.ToResponse()
	return &resp, nil
}

// GetEntry retrieves an entry by ID.
func (s *PayPlanService) GetEntry(ctx context.Context, entryID, periodID, payplanID, orgID uint) (*models.PayPlanEntryResponse, error) {
	// Verify pay plan exists and belongs to org
	payplan, err := s.store.GetByID(ctx, payplanID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("pay plan")
		}
		return nil, apperror.Internal("failed to fetch pay plan")
	}
	if payplan.OrganizationID != orgID {
		return nil, apperror.NotFound("pay plan")
	}

	// Verify period belongs to pay plan
	period, err := s.store.GetPeriodByID(ctx, periodID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("period")
		}
		return nil, apperror.Internal("failed to fetch period")
	}
	if period.PayPlanID != payplanID {
		return nil, apperror.NotFound("period")
	}

	entry, err := s.store.GetEntryByID(ctx, entryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("entry")
		}
		return nil, apperror.Internal("failed to fetch entry")
	}

	if entry.PeriodID != periodID {
		return nil, apperror.NotFound("entry")
	}

	resp := entry.ToResponse()
	return &resp, nil
}

// UpdateEntry updates an entry.
func (s *PayPlanService) UpdateEntry(ctx context.Context, entryID, periodID, payplanID, orgID uint, req models.PayPlanEntryUpdateRequest) (*models.PayPlanEntryResponse, error) {
	// Verify pay plan exists and belongs to org
	payplan, err := s.store.GetByID(ctx, payplanID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("pay plan")
		}
		return nil, apperror.Internal("failed to fetch pay plan")
	}
	if payplan.OrganizationID != orgID {
		return nil, apperror.NotFound("pay plan")
	}

	// Verify period belongs to pay plan
	period, err := s.store.GetPeriodByID(ctx, periodID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("period")
		}
		return nil, apperror.Internal("failed to fetch period")
	}
	if period.PayPlanID != payplanID {
		return nil, apperror.NotFound("period")
	}

	entry, err := s.store.GetEntryByID(ctx, entryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("entry")
		}
		return nil, apperror.Internal("failed to fetch entry")
	}

	if entry.PeriodID != periodID {
		return nil, apperror.NotFound("entry")
	}

	entry.Grade = req.Grade
	entry.Step = req.Step
	entry.MonthlyAmount = req.MonthlyAmount

	if err := s.store.UpdateEntry(ctx, entry); err != nil {
		return nil, apperror.Internal("failed to update entry")
	}

	resp := entry.ToResponse()
	return &resp, nil
}

// DeleteEntry deletes an entry.
func (s *PayPlanService) DeleteEntry(ctx context.Context, entryID, periodID, payplanID, orgID uint) error {
	// Verify pay plan exists and belongs to org
	payplan, err := s.store.GetByID(ctx, payplanID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("pay plan")
		}
		return apperror.Internal("failed to fetch pay plan")
	}
	if payplan.OrganizationID != orgID {
		return apperror.NotFound("pay plan")
	}

	// Verify period belongs to pay plan
	period, err := s.store.GetPeriodByID(ctx, periodID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("period")
		}
		return apperror.Internal("failed to fetch period")
	}
	if period.PayPlanID != payplanID {
		return apperror.NotFound("period")
	}

	entry, err := s.store.GetEntryByID(ctx, entryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("entry")
		}
		return apperror.Internal("failed to fetch entry")
	}

	if entry.PeriodID != periodID {
		return apperror.NotFound("entry")
	}

	if err := s.store.DeleteEntry(ctx, entryID); err != nil {
		return apperror.Internal("failed to delete entry")
	}
	return nil
}

// CalculateSalary calculates the monthly salary based on pay plan, grade, step, and working hours.
func (s *PayPlanService) CalculateSalary(ctx context.Context, payplanID uint, grade string, step int, weeklyHours float64, date time.Time) (int, error) {
	period, err := s.store.GetActivePeriod(ctx, payplanID, date)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, apperror.NotFound("no active pay plan period found for the given date")
		}
		return 0, apperror.Internal("failed to fetch pay plan period")
	}

	entry, err := s.store.GetEntry(ctx, period.ID, grade, step)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, apperror.NotFound("no pay plan entry found for the given grade and step")
		}
		return 0, apperror.Internal("failed to fetch pay plan entry")
	}

	// Calculate salary: MonthlyAmount * (employeeWeeklyHours / periodWeeklyHours)
	salary := float64(entry.MonthlyAmount) * (weeklyHours / period.WeeklyHours)
	return int(salary), nil
}
