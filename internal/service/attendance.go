package service

import (
	"context"
	"strings"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// AttendanceService handles business logic for attendance operations
type AttendanceService struct {
	store      store.AttendanceStorer
	childStore store.ChildStorer
}

// NewAttendanceService creates a new attendance service
func NewAttendanceService(store store.AttendanceStorer, childStore store.ChildStorer) *AttendanceService {
	return &AttendanceService{
		store:      store,
		childStore: childStore,
	}
}

// CheckIn creates an attendance record for a child (check-in).
func (s *AttendanceService) CheckIn(ctx context.Context, orgID uint, req *models.AttendanceCheckInRequest, recordedBy uint) (*models.AttendanceResponse, error) {
	// Verify child exists and belongs to org
	child, err := s.childStore.FindByIDMinimal(req.ChildID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if child.OrganizationID != orgID {
		return nil, apperror.NotFound("child")
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Check if attendance record already exists for today
	existing, err := s.store.FindByChildAndDate(req.ChildID, today)
	if err == nil && existing != nil {
		return nil, apperror.Conflict("attendance record already exists for this child today")
	}

	checkInTime := &now
	if req.CheckInTime != nil {
		checkInTime = req.CheckInTime
	}

	attendance := &models.Attendance{
		ChildID:        req.ChildID,
		OrganizationID: orgID,
		Date:           today,
		CheckInTime:    checkInTime,
		Status:         models.AttendanceStatusPresent,
		Note:           strings.TrimSpace(req.Note),
		RecordedBy:     recordedBy,
	}

	if err := s.store.Create(attendance); err != nil {
		return nil, apperror.Internal("failed to create attendance record")
	}

	// Reload with child info
	attendance, err = s.store.FindByID(attendance.ID)
	if err != nil {
		return nil, apperror.Internal("failed to reload attendance record")
	}

	resp := attendance.ToResponse()
	return &resp, nil
}

// CheckOut updates an attendance record with check-out time.
func (s *AttendanceService) CheckOut(ctx context.Context, id, orgID uint, req *models.AttendanceCheckOutRequest) (*models.AttendanceResponse, error) {
	attendance, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("attendance record")
	}
	if attendance.OrganizationID != orgID {
		return nil, apperror.NotFound("attendance record")
	}

	if attendance.CheckOutTime != nil {
		return nil, apperror.BadRequest("child is already checked out")
	}

	now := time.Now()
	checkOutTime := &now
	if req.CheckOutTime != nil {
		checkOutTime = req.CheckOutTime
	}

	attendance.CheckOutTime = checkOutTime
	if req.Note != "" {
		if attendance.Note != "" {
			attendance.Note = attendance.Note + "; " + strings.TrimSpace(req.Note)
		} else {
			attendance.Note = strings.TrimSpace(req.Note)
		}
	}

	if err := s.store.Update(attendance); err != nil {
		return nil, apperror.Internal("failed to update attendance record")
	}

	resp := attendance.ToResponse()
	return &resp, nil
}

// MarkAbsent creates an attendance record marking a child absent.
func (s *AttendanceService) MarkAbsent(ctx context.Context, orgID uint, req *models.AttendanceMarkAbsentRequest, recordedBy uint) (*models.AttendanceResponse, error) {
	// Verify child exists and belongs to org
	child, err := s.childStore.FindByIDMinimal(req.ChildID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if child.OrganizationID != orgID {
		return nil, apperror.NotFound("child")
	}

	if !models.IsValidAttendanceStatus(req.Status) {
		return nil, apperror.BadRequest("invalid status, must be one of: absent, sick, vacation")
	}
	if req.Status == models.AttendanceStatusPresent {
		return nil, apperror.BadRequest("use check-in endpoint for marking present")
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, apperror.BadRequest("invalid date format, expected YYYY-MM-DD")
	}

	// Check if attendance record already exists
	existing, findErr := s.store.FindByChildAndDate(req.ChildID, date)
	if findErr == nil && existing != nil {
		return nil, apperror.Conflict("attendance record already exists for this child on this date")
	}

	attendance := &models.Attendance{
		ChildID:        req.ChildID,
		OrganizationID: orgID,
		Date:           date,
		Status:         req.Status,
		Note:           strings.TrimSpace(req.Note),
		RecordedBy:     recordedBy,
	}

	if err := s.store.Create(attendance); err != nil {
		return nil, apperror.Internal("failed to create attendance record")
	}

	// Reload with child info
	attendance, err = s.store.FindByID(attendance.ID)
	if err != nil {
		return nil, apperror.Internal("failed to reload attendance record")
	}

	resp := attendance.ToResponse()
	return &resp, nil
}

// GetByID returns an attendance record by ID, validating it belongs to the organization.
func (s *AttendanceService) GetByID(ctx context.Context, id, orgID uint) (*models.AttendanceResponse, error) {
	attendance, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("attendance record")
	}
	if attendance.OrganizationID != orgID {
		return nil, apperror.NotFound("attendance record")
	}

	resp := attendance.ToResponse()
	return &resp, nil
}

// Update updates an existing attendance record.
func (s *AttendanceService) Update(ctx context.Context, id, orgID uint, req *models.AttendanceUpdateRequest) (*models.AttendanceResponse, error) {
	attendance, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("attendance record")
	}
	if attendance.OrganizationID != orgID {
		return nil, apperror.NotFound("attendance record")
	}

	if req.CheckInTime != nil {
		attendance.CheckInTime = req.CheckInTime
	}
	if req.CheckOutTime != nil {
		attendance.CheckOutTime = req.CheckOutTime
	}
	if req.Status != nil {
		if !models.IsValidAttendanceStatus(*req.Status) {
			return nil, apperror.BadRequest("invalid status")
		}
		attendance.Status = *req.Status
	}
	if req.Note != nil {
		attendance.Note = strings.TrimSpace(*req.Note)
	}

	if err := s.store.Update(attendance); err != nil {
		return nil, apperror.Internal("failed to update attendance record")
	}

	resp := attendance.ToResponse()
	return &resp, nil
}

// Delete deletes an attendance record.
func (s *AttendanceService) Delete(ctx context.Context, id, orgID uint) error {
	attendance, err := s.store.FindByID(id)
	if err != nil {
		return apperror.NotFound("attendance record")
	}
	if attendance.OrganizationID != orgID {
		return apperror.NotFound("attendance record")
	}

	if err := s.store.Delete(id); err != nil {
		return apperror.Internal("failed to delete attendance record")
	}
	return nil
}

// ListByDate returns attendance records for an organization on a given date.
func (s *AttendanceService) ListByDate(ctx context.Context, orgID uint, date time.Time, limit, offset int) ([]models.AttendanceResponse, int64, error) {
	records, total, err := s.store.FindByOrganizationAndDate(orgID, date, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch attendance records")
	}

	responses := make([]models.AttendanceResponse, len(records))
	for i, r := range records {
		responses[i] = r.ToResponse()
	}
	return responses, total, nil
}

// ListByChild returns attendance records for a specific child in a date range.
func (s *AttendanceService) ListByChild(ctx context.Context, childID, orgID uint, from, to time.Time, limit, offset int) ([]models.AttendanceResponse, int64, error) {
	// Verify child belongs to org
	child, err := s.childStore.FindByIDMinimal(childID)
	if err != nil {
		return nil, 0, apperror.NotFound("child")
	}
	if child.OrganizationID != orgID {
		return nil, 0, apperror.NotFound("child")
	}

	records, total, err := s.store.FindByChildAndDateRange(childID, from, to, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch attendance records")
	}

	responses := make([]models.AttendanceResponse, len(records))
	for i, r := range records {
		responses[i] = r.ToResponse()
	}
	return responses, total, nil
}

// GetDailySummary returns attendance summary for a given date.
func (s *AttendanceService) GetDailySummary(ctx context.Context, orgID uint, date time.Time) (*models.DailyAttendanceSummaryResponse, error) {
	summary, err := s.store.GetDailySummary(orgID, date)
	if err != nil {
		return nil, apperror.Internal("failed to get daily summary")
	}
	return summary, nil
}
