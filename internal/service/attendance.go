package service

import (
	"context"
	"strings"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// ChildAttendanceService handles business logic for child attendance operations
type ChildAttendanceService struct {
	store      store.ChildAttendanceStorer
	childStore store.ChildStorer
}

// NewChildAttendanceService creates a new child attendance service
func NewChildAttendanceService(store store.ChildAttendanceStorer, childStore store.ChildStorer) *ChildAttendanceService {
	return &ChildAttendanceService{
		store:      store,
		childStore: childStore,
	}
}

// Create creates an attendance record for a child.
// For status "present": date defaults to today, check_in_time defaults to now.
// For status "absent", "sick", "vacation": date is required, check_in_time is ignored.
func (s *ChildAttendanceService) Create(ctx context.Context, orgID, childID uint, req *models.ChildAttendanceCreateRequest, recordedBy uint) (*models.ChildAttendanceResponse, error) {
	// Verify child exists and belongs to org
	child, err := s.childStore.FindByIDMinimal(ctx, childID)
	if err != nil {
		return nil, classifyStoreError(err, "child")
	}
	if child.OrganizationID != orgID {
		return nil, apperror.NotFound("child")
	}

	if !models.IsValidChildAttendanceStatus(req.Status) {
		return nil, apperror.BadRequest("invalid status, must be one of: present, absent, sick, vacation")
	}

	now := time.Now()
	var date time.Time
	var checkInTime *time.Time

	if req.Status == models.ChildAttendanceStatusPresent {
		// For present: date defaults to today, check_in_time defaults to now
		if req.Date != "" {
			date, err = time.Parse(models.DateFormat, req.Date)
			if err != nil {
				return nil, apperror.BadRequest("invalid date format, expected YYYY-MM-DD")
			}
		} else {
			date = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		}
		if req.CheckInTime != nil {
			checkInTime = req.CheckInTime
		} else {
			checkInTime = &now
		}
	} else {
		// For absent/sick/vacation: date is required, check_in_time is ignored
		if req.Date == "" {
			return nil, apperror.BadRequest("date is required for non-present status")
		}
		date, err = time.Parse(models.DateFormat, req.Date)
		if err != nil {
			return nil, apperror.BadRequest("invalid date format, expected YYYY-MM-DD")
		}
	}

	// Check if attendance record already exists for this child on this date
	existing, findErr := s.store.FindByChildAndDate(ctx, childID, date)
	if findErr == nil && existing != nil {
		return nil, apperror.Conflict("attendance record already exists for this child on this date")
	}

	attendance := &models.ChildAttendance{
		ChildID:        childID,
		OrganizationID: orgID,
		Date:           date,
		CheckInTime:    checkInTime,
		Status:         req.Status,
		Note:           strings.TrimSpace(req.Note),
		RecordedBy:     recordedBy,
	}

	if err := s.store.Create(ctx, attendance); err != nil {
		return nil, apperror.InternalWrap(err, "failed to create attendance record")
	}

	// Reload with child info
	attendance, err = s.store.FindByID(ctx, attendance.ID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to reload attendance record")
	}

	resp := attendance.ToResponse()
	return &resp, nil
}

// GetByID returns an attendance record by ID, validating it belongs to the organization and child.
func (s *ChildAttendanceService) GetByID(ctx context.Context, id, orgID, childID uint) (*models.ChildAttendanceResponse, error) {
	attendance, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, classifyStoreError(err, "attendance record")
	}
	if attendance.OrganizationID != orgID {
		return nil, apperror.NotFound("attendance record")
	}
	if attendance.ChildID != childID {
		return nil, apperror.NotFound("attendance record")
	}

	resp := attendance.ToResponse()
	return &resp, nil
}

// Update updates an existing attendance record.
func (s *ChildAttendanceService) Update(ctx context.Context, id, orgID, childID uint, req *models.ChildAttendanceUpdateRequest) (*models.ChildAttendanceResponse, error) {
	attendance, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, classifyStoreError(err, "attendance record")
	}
	if attendance.OrganizationID != orgID {
		return nil, apperror.NotFound("attendance record")
	}
	if attendance.ChildID != childID {
		return nil, apperror.NotFound("attendance record")
	}

	if req.CheckInTime != nil {
		attendance.CheckInTime = req.CheckInTime
	}
	if req.CheckOutTime != nil {
		attendance.CheckOutTime = req.CheckOutTime
	}
	if req.Status != nil {
		if !models.IsValidChildAttendanceStatus(*req.Status) {
			return nil, apperror.BadRequest("invalid status")
		}
		attendance.Status = *req.Status
	}
	if req.Note != nil {
		attendance.Note = strings.TrimSpace(*req.Note)
	}

	if err := s.store.Update(ctx, attendance); err != nil {
		return nil, apperror.InternalWrap(err, "failed to update attendance record")
	}

	resp := attendance.ToResponse()
	return &resp, nil
}

// Delete deletes an attendance record.
func (s *ChildAttendanceService) Delete(ctx context.Context, id, orgID, childID uint) error {
	attendance, err := s.store.FindByID(ctx, id)
	if err != nil {
		return classifyStoreError(err, "attendance record")
	}
	if attendance.OrganizationID != orgID {
		return apperror.NotFound("attendance record")
	}
	if attendance.ChildID != childID {
		return apperror.NotFound("attendance record")
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return apperror.InternalWrap(err, "failed to delete attendance record")
	}
	return nil
}

// ListByChild returns attendance records for a specific child in a date range.
func (s *ChildAttendanceService) ListByChild(ctx context.Context, childID, orgID uint, from, to time.Time, limit, offset int) ([]models.ChildAttendanceResponse, int64, error) {
	// Verify child belongs to org
	child, err := s.childStore.FindByIDMinimal(ctx, childID)
	if err != nil {
		return nil, 0, classifyStoreError(err, "child")
	}
	if child.OrganizationID != orgID {
		return nil, 0, apperror.NotFound("child")
	}

	records, total, err := s.store.FindByChildAndDateRange(ctx, childID, from, to, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch attendance records")
	}

	return toResponseList(records, (*models.ChildAttendance).ToResponse), total, nil
}

// ListByDate returns attendance records for an organization on a given date.
func (s *ChildAttendanceService) ListByDate(ctx context.Context, orgID uint, date time.Time, limit, offset int) ([]models.ChildAttendanceResponse, int64, error) {
	records, total, err := s.store.FindByOrganizationAndDate(ctx, orgID, date, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch attendance records")
	}

	return toResponseList(records, (*models.ChildAttendance).ToResponse), total, nil
}

// GetDailySummary returns attendance summary for a given date.
func (s *ChildAttendanceService) GetDailySummary(ctx context.Context, orgID uint, date time.Time) (*models.ChildAttendanceDailySummaryResponse, error) {
	summary, err := s.store.GetDailySummary(ctx, orgID, date)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to get daily summary")
	}
	return summary, nil
}
