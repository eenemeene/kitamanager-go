package handlers

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func TestGovernmentFundingHandler_CreateEntry_ValidAgeRange(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	orgStore := store.NewOrganizationStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, orgStore)
	handler := NewGovernmentFundingHandler(svc)

	// Create test funding and period
	funding := &models.GovernmentFunding{Name: "Test Funding"}
	db.Create(funding)
	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		From:                time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	db.Create(period)

	r := setupTestRouter()
	r.POST("/fundings/:id/periods/:periodId/entries", handler.CreateEntry)

	tests := []struct {
		name           string
		minAge         int
		maxAge         int
		expectedStatus int
		description    string
	}{
		{
			name:           "valid range 0-2 (covers ages 0 and 1)",
			minAge:         0,
			maxAge:         2,
			expectedStatus: http.StatusCreated,
			description:    "Children from birth up to but not including 2nd birthday",
		},
		{
			name:           "valid range 2-3 (covers age 2 only)",
			minAge:         2,
			maxAge:         3,
			expectedStatus: http.StatusCreated,
			description:    "Children from 2nd birthday up to but not including 3rd birthday",
		},
		{
			name:           "valid range 3-7 (covers ages 3, 4, 5, 6)",
			minAge:         3,
			maxAge:         7,
			expectedStatus: http.StatusCreated,
			description:    "Children from 3rd birthday up to but not including 7th birthday",
		},
		{
			name:           "invalid: min equals max",
			minAge:         2,
			maxAge:         2,
			expectedStatus: http.StatusBadRequest,
			description:    "Empty range - no children would qualify",
		},
		{
			name:           "invalid: min greater than max",
			minAge:         5,
			maxAge:         3,
			expectedStatus: http.StatusBadRequest,
			description:    "Inverted range is invalid",
		},
		{
			name:           "valid: single year range",
			minAge:         1,
			maxAge:         2,
			expectedStatus: http.StatusCreated,
			description:    "Children who are exactly 1 year old (up to but not including 2nd birthday)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := models.GovernmentFundingEntryCreateRequest{
				MinAge: tt.minAge,
				MaxAge: tt.maxAge,
			}

			w := performRequest(r, "POST", "/fundings/1/periods/1/entries", body)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d: %s",
					tt.description, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestGovernmentFundingHandler_UpdateEntry_AgeRangeValidation(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	orgStore := store.NewOrganizationStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, orgStore)
	handler := NewGovernmentFundingHandler(svc)

	// Create test funding, period, and entry
	funding := &models.GovernmentFunding{Name: "Test Funding"}
	db.Create(funding)
	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		From:                time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	db.Create(period)
	entry := &models.GovernmentFundingEntry{
		PeriodID: period.ID,
		MinAge:   0,
		MaxAge:   3,
	}
	db.Create(entry)

	r := setupTestRouter()
	r.PUT("/fundings/:id/periods/:periodId/entries/:entryId", handler.UpdateEntry)

	tests := []struct {
		name           string
		minAge         *int
		maxAge         *int
		expectedStatus int
		description    string
	}{
		{
			name:           "update only max_age to valid value",
			minAge:         nil,
			maxAge:         intPtr(5),
			expectedStatus: http.StatusOK,
			description:    "Extending max age should succeed",
		},
		{
			name:           "update max_age to equal min_age",
			minAge:         nil,
			maxAge:         intPtr(0),
			expectedStatus: http.StatusBadRequest,
			description:    "max_age equal to current min_age (0) is invalid",
		},
		{
			name:           "update min_age to exceed max_age",
			minAge:         intPtr(10),
			maxAge:         nil,
			expectedStatus: http.StatusBadRequest,
			description:    "min_age greater than current max_age is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset entry to known state
			db.Model(&models.GovernmentFundingEntry{}).Where("id = ?", entry.ID).Updates(map[string]interface{}{
				"min_age": 0,
				"max_age": 3,
			})

			body := models.GovernmentFundingEntryUpdateRequest{
				MinAge: tt.minAge,
				MaxAge: tt.maxAge,
			}

			w := performRequest(r, "PUT", "/fundings/1/periods/1/entries/1", body)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d: %s",
					tt.description, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestGovernmentFundingHandler_Entry_AgeRangeBoundarySemantics(t *testing.T) {
	// This test documents the expected behavior of age ranges:
	// - MinAge is INCLUSIVE: a child whose age >= MinAge qualifies
	// - MaxAge is EXCLUSIVE: a child whose age < MaxAge qualifies
	//
	// Example: MinAge=0, MaxAge=2 means:
	// - Age 0 qualifies (0 >= 0 AND 0 < 2)
	// - Age 1 qualifies (1 >= 0 AND 1 < 2)
	// - Age 2 does NOT qualify (2 >= 0 BUT 2 < 2 is FALSE)

	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	orgStore := store.NewOrganizationStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, orgStore)
	handler := NewGovernmentFundingHandler(svc)

	funding := &models.GovernmentFunding{Name: "Test Funding"}
	db.Create(funding)
	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		From:                time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	db.Create(period)

	r := setupTestRouter()
	r.POST("/fundings/:id/periods/:periodId/entries", handler.CreateEntry)

	// Create an entry for ages 0-2 (covers ages 0 and 1)
	body := models.GovernmentFundingEntryCreateRequest{
		MinAge: 0,
		MaxAge: 2,
	}

	w := performRequest(r, "POST", "/fundings/1/periods/1/entries", body)

	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create entry: %s", w.Body.String())
	}

	var result models.GovernmentFundingEntry
	parseResponse(t, w, &result)

	// Verify the entry was created with correct values
	if result.MinAge != 0 {
		t.Errorf("expected min_age 0, got %d", result.MinAge)
	}
	if result.MaxAge != 2 {
		t.Errorf("expected max_age 2, got %d", result.MaxAge)
	}

	// Document the semantics in test output
	t.Logf("Entry created: MinAge=%d (inclusive), MaxAge=%d (exclusive)", result.MinAge, result.MaxAge)
	t.Logf("This entry covers children aged 0 and 1 (from birth up to but not including 2nd birthday)")
}

func TestGovernmentFundingHandler_CRUD(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	orgStore := store.NewOrganizationStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, orgStore)
	handler := NewGovernmentFundingHandler(svc)

	r := setupTestRouter()
	r.GET("/fundings", handler.List)
	r.GET("/fundings/:id", handler.Get)
	r.POST("/fundings", handler.Create)
	r.PUT("/fundings/:id", handler.Update)
	r.DELETE("/fundings/:id", handler.Delete)

	// Test Create
	t.Run("Create", func(t *testing.T) {
		body := GovernmentFundingCreateRequest{Name: "Berlin"}
		w := performRequest(r, "POST", "/fundings", body)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var result models.GovernmentFunding
		parseResponse(t, w, &result)
		if result.Name != "Berlin" {
			t.Errorf("expected name 'Berlin', got '%s'", result.Name)
		}
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		w := performRequest(r, "GET", "/fundings", nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response models.PaginatedResponse[models.GovernmentFunding]
		parseResponse(t, w, &response)
		if len(response.Data) != 1 {
			t.Errorf("expected 1 funding, got %d", len(response.Data))
		}
	})

	// Test Get
	t.Run("Get", func(t *testing.T) {
		w := performRequest(r, "GET", "/fundings/1", nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var result models.GovernmentFunding
		parseResponse(t, w, &result)
		if result.Name != "Berlin" {
			t.Errorf("expected name 'Berlin', got '%s'", result.Name)
		}
	})

	// Test Update
	t.Run("Update", func(t *testing.T) {
		name := "Berlin Updated"
		body := GovernmentFundingUpdateRequest{Name: &name}
		w := performRequest(r, "PUT", "/fundings/1", body)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var result models.GovernmentFunding
		parseResponse(t, w, &result)
		if result.Name != "Berlin Updated" {
			t.Errorf("expected name 'Berlin Updated', got '%s'", result.Name)
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		w := performRequest(r, "DELETE", "/fundings/1", nil)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
		}
	})
}

func TestGovernmentFundingHandler_CreatePeriod_NoOverlap(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	orgStore := store.NewOrganizationStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, orgStore)
	handler := NewGovernmentFundingHandler(svc)

	// Create test funding
	funding := &models.GovernmentFunding{Name: "Test Funding"}
	db.Create(funding)

	r := setupTestRouter()
	r.POST("/fundings/:id/periods", handler.CreatePeriod)

	tests := []struct {
		name           string
		existingFrom   string
		existingTo     *string
		newFrom        string
		newTo          *string
		expectedStatus int
		description    string
	}{
		{
			name:           "no overlap: new period after existing",
			existingFrom:   "2024-01-01",
			existingTo:     strPtr("2024-06-30"),
			newFrom:        "2024-07-01",
			newTo:          strPtr("2024-12-31"),
			expectedStatus: http.StatusCreated,
			description:    "New period starts after existing ends",
		},
		{
			name:           "no overlap: new period before existing",
			existingFrom:   "2024-07-01",
			existingTo:     strPtr("2024-12-31"),
			newFrom:        "2024-01-01",
			newTo:          strPtr("2024-06-30"),
			expectedStatus: http.StatusCreated,
			description:    "New period ends before existing starts",
		},
		{
			name:           "overlap: new period inside existing",
			existingFrom:   "2024-01-01",
			existingTo:     strPtr("2024-12-31"),
			newFrom:        "2024-03-01",
			newTo:          strPtr("2024-06-30"),
			expectedStatus: http.StatusBadRequest,
			description:    "New period is entirely within existing period",
		},
		{
			name:           "overlap: new period spans existing",
			existingFrom:   "2024-03-01",
			existingTo:     strPtr("2024-06-30"),
			newFrom:        "2024-01-01",
			newTo:          strPtr("2024-12-31"),
			expectedStatus: http.StatusBadRequest,
			description:    "New period completely covers existing period",
		},
		{
			name:           "overlap: partial overlap at start",
			existingFrom:   "2024-06-01",
			existingTo:     strPtr("2024-12-31"),
			newFrom:        "2024-01-01",
			newTo:          strPtr("2024-07-31"),
			expectedStatus: http.StatusBadRequest,
			description:    "New period overlaps at the start of existing",
		},
		{
			name:           "overlap: partial overlap at end",
			existingFrom:   "2024-01-01",
			existingTo:     strPtr("2024-06-30"),
			newFrom:        "2024-05-01",
			newTo:          strPtr("2024-12-31"),
			expectedStatus: http.StatusBadRequest,
			description:    "New period overlaps at the end of existing",
		},
		{
			name:           "overlap: existing has no end date",
			existingFrom:   "2024-01-01",
			existingTo:     nil,
			newFrom:        "2024-06-01",
			newTo:          strPtr("2024-12-31"),
			expectedStatus: http.StatusBadRequest,
			description:    "Existing period is ongoing, new period overlaps",
		},
		{
			name:           "overlap: new has no end date",
			existingFrom:   "2024-06-01",
			existingTo:     strPtr("2024-12-31"),
			newFrom:        "2024-01-01",
			newTo:          nil,
			expectedStatus: http.StatusBadRequest,
			description:    "New period is ongoing and overlaps existing",
		},
		{
			name:           "no overlap: new period before ongoing",
			existingFrom:   "2024-07-01",
			existingTo:     nil,
			newFrom:        "2024-01-01",
			newTo:          strPtr("2024-06-30"),
			expectedStatus: http.StatusCreated,
			description:    "New period ends before ongoing period starts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up periods from previous test
			db.Where("government_funding_id = ?", funding.ID).Delete(&models.GovernmentFundingPeriod{})

			// Create existing period
			existingFrom, _ := time.Parse("2006-01-02", tt.existingFrom)
			existingPeriod := &models.GovernmentFundingPeriod{
				GovernmentFundingID: funding.ID,
				From:                existingFrom,
			}
			if tt.existingTo != nil {
				to, _ := time.Parse("2006-01-02", *tt.existingTo)
				existingPeriod.To = &to
			}
			db.Create(existingPeriod)

			// Try to create new period
			newFrom, _ := time.Parse("2006-01-02", tt.newFrom)
			body := map[string]interface{}{
				"from": newFrom.Format(time.RFC3339),
			}
			if tt.newTo != nil {
				newTo, _ := time.Parse("2006-01-02", *tt.newTo)
				body["to"] = newTo.Format(time.RFC3339)
			}

			w := performRequest(r, "POST", "/fundings/1/periods", body)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d: %s",
					tt.description, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestGovernmentFundingHandler_UpdatePeriod_NoOverlap(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	orgStore := store.NewOrganizationStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, orgStore)
	handler := NewGovernmentFundingHandler(svc)

	// Create test funding
	funding := &models.GovernmentFunding{Name: "Test Funding"}
	db.Create(funding)

	r := setupTestRouter()
	r.PUT("/fundings/:id/periods/:periodId", handler.UpdatePeriod)

	t.Run("update period to overlap with another", func(t *testing.T) {
		// Clean up
		db.Where("government_funding_id = ?", funding.ID).Delete(&models.GovernmentFundingPeriod{})

		// Create two non-overlapping periods
		from1, _ := time.Parse("2006-01-02", "2024-01-01")
		to1, _ := time.Parse("2006-01-02", "2024-06-30")
		period1 := &models.GovernmentFundingPeriod{GovernmentFundingID: funding.ID, From: from1, To: &to1}
		db.Create(period1)

		from2, _ := time.Parse("2006-01-02", "2024-07-01")
		to2, _ := time.Parse("2006-01-02", "2024-12-31")
		period2 := &models.GovernmentFundingPeriod{GovernmentFundingID: funding.ID, From: from2, To: &to2}
		db.Create(period2)

		// Try to update period2 to overlap with period1
		newFrom, _ := time.Parse("2006-01-02", "2024-05-01")
		body := map[string]interface{}{
			"from": newFrom.Format(time.RFC3339),
		}

		w := performRequest(r, "PUT", "/fundings/1/periods/"+itoa(int(period2.ID)), body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d for overlapping update, got %d: %s",
				http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("update period without causing overlap", func(t *testing.T) {
		// Clean up
		db.Where("government_funding_id = ?", funding.ID).Delete(&models.GovernmentFundingPeriod{})

		// Create two non-overlapping periods
		from1, _ := time.Parse("2006-01-02", "2024-01-01")
		to1, _ := time.Parse("2006-01-02", "2024-06-30")
		period1 := &models.GovernmentFundingPeriod{GovernmentFundingID: funding.ID, From: from1, To: &to1}
		db.Create(period1)

		from2, _ := time.Parse("2006-01-02", "2024-07-01")
		to2, _ := time.Parse("2006-01-02", "2024-12-31")
		period2 := &models.GovernmentFundingPeriod{GovernmentFundingID: funding.ID, From: from2, To: &to2}
		db.Create(period2)

		// Update period2's end date (no overlap)
		newTo, _ := time.Parse("2006-01-02", "2025-06-30")
		body := map[string]interface{}{
			"to": newTo.Format(time.RFC3339),
		}

		w := performRequest(r, "PUT", "/fundings/1/periods/"+itoa(int(period2.ID)), body)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d for valid update, got %d: %s",
				http.StatusOK, w.Code, w.Body.String())
		}
	})
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}
