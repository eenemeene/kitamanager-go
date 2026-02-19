package models

import (
	"testing"
)

func TestPaginationParams_Validate(t *testing.T) {
	tests := []struct {
		name      string
		params    PaginationParams
		wantErr   bool
		errSubstr string
	}{
		{"valid page 1 limit 20", PaginationParams{Page: 1, Limit: 20}, false, ""},
		{"valid page 0 limit 0 (defaults not set)", PaginationParams{Page: 0, Limit: 0}, false, ""},
		{"valid max limit 100", PaginationParams{Page: 1, Limit: 100}, false, ""},
		{"invalid limit exceeds max", PaginationParams{Page: 1, Limit: 101}, true, "limit must not exceed 100"},
		{"invalid negative page", PaginationParams{Page: -1, Limit: 20}, true, "page must be positive"},
		{"invalid negative limit", PaginationParams{Page: 1, Limit: -1}, true, "limit must be positive"},
		{"both negative", PaginationParams{Page: -5, Limit: -10}, true, "page must be positive"},
		{"large valid limit", PaginationParams{Page: 1, Limit: 100}, false, ""},
		{"large page number", PaginationParams{Page: 10000, Limit: 20}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errSubstr)
				} else if tt.errSubstr != "" && err.Error() != tt.errSubstr {
					t.Errorf("Validate() error = %q, want %q", err.Error(), tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPaginationParams_SetDefaults(t *testing.T) {
	tests := []struct {
		name          string
		initial       PaginationParams
		expectedPage  int
		expectedLimit int
	}{
		{"sets defaults when both zero", PaginationParams{Page: 0, Limit: 0}, 1, 20},
		{"sets page when zero", PaginationParams{Page: 0, Limit: 50}, 1, 50},
		{"sets limit when zero", PaginationParams{Page: 5, Limit: 0}, 5, 20},
		{"does not override valid values", PaginationParams{Page: 3, Limit: 30}, 3, 30},
		{"sets page when negative", PaginationParams{Page: -1, Limit: 20}, 1, 20},
		{"sets limit when negative", PaginationParams{Page: 1, Limit: -5}, 1, 20},
		{"page 1 is not changed", PaginationParams{Page: 1, Limit: 20}, 1, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tt.initial
			params.SetDefaults()

			if params.Page != tt.expectedPage {
				t.Errorf("SetDefaults() Page = %d, want %d", params.Page, tt.expectedPage)
			}
			if params.Limit != tt.expectedLimit {
				t.Errorf("SetDefaults() Limit = %d, want %d", params.Limit, tt.expectedLimit)
			}
		})
	}
}

func TestPaginationParams_Offset(t *testing.T) {
	tests := []struct {
		name     string
		params   PaginationParams
		expected int
	}{
		{"page 1 limit 20", PaginationParams{Page: 1, Limit: 20}, 0},
		{"page 2 limit 20", PaginationParams{Page: 2, Limit: 20}, 20},
		{"page 3 limit 20", PaginationParams{Page: 3, Limit: 20}, 40},
		{"page 5 limit 10", PaginationParams{Page: 5, Limit: 10}, 40},
		{"page 1 limit 100", PaginationParams{Page: 1, Limit: 100}, 0},
		{"page 10 limit 100", PaginationParams{Page: 10, Limit: 100}, 900},
		{"page 0 limit 20", PaginationParams{Page: 0, Limit: 20}, -20},
		{"page 1 limit 1", PaginationParams{Page: 1, Limit: 1}, 0},
		{"page 2 limit 1", PaginationParams{Page: 2, Limit: 1}, 1},
		{"page 100 limit 50", PaginationParams{Page: 100, Limit: 50}, 4950},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.params.Offset()
			if got != tt.expected {
				t.Errorf("Offset() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestPaginationParams_SetDefaults_Idempotent(t *testing.T) {
	params := PaginationParams{Page: 0, Limit: 0}
	params.SetDefaults()

	page1, limit1 := params.Page, params.Limit

	params.SetDefaults() // Call again

	if params.Page != page1 || params.Limit != limit1 {
		t.Error("SetDefaults should be idempotent")
	}
}

func TestPaginationParams_ValidateThenSetDefaults(t *testing.T) {
	params := PaginationParams{Page: 0, Limit: 0}

	// Zero values should pass validation (they mean "not provided")
	if err := params.Validate(); err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}

	// Then defaults should be set
	params.SetDefaults()

	if params.Page != 1 {
		t.Errorf("Page = %d after SetDefaults, want 1", params.Page)
	}
	if params.Limit != 20 {
		t.Errorf("Limit = %d after SetDefaults, want 20", params.Limit)
	}
}

func TestPaginatedResponse_TotalPagesCalculation(t *testing.T) {
	// Edge case: exactly divisible
	resp := NewPaginatedResponseWithLinks([]int{}, 1, 25, 100, "/test", "")
	if resp.TotalPages != 4 {
		t.Errorf("100/25 should give 4 total pages, got %d", resp.TotalPages)
	}

	// Edge case: not exactly divisible (ceiling)
	resp = NewPaginatedResponseWithLinks([]int{}, 1, 25, 101, "/test", "")
	if resp.TotalPages != 5 {
		t.Errorf("101/25 should give 5 total pages (ceiling), got %d", resp.TotalPages)
	}

	// Edge case: single item
	resp = NewPaginatedResponseWithLinks([]int{}, 1, 20, 1, "/test", "")
	if resp.TotalPages != 1 {
		t.Errorf("1 item should give 1 total page, got %d", resp.TotalPages)
	}
}

func TestPaginatedResponse_PreservesFilterParams(t *testing.T) {
	resp := NewPaginatedResponseWithLinks(
		[]int{1, 2, 3}, 2, 10, 50, "/api/v1/children",
		"search=alice&section_id=5&page=2&limit=10",
	)

	// Links should contain filter params
	if resp.Links == nil {
		t.Fatal("expected links to be set")
	}

	// Check that self link preserves filters
	assertContains(t, resp.Links.Self, "search=alice")
	assertContains(t, resp.Links.Self, "section_id=5")
	assertContains(t, resp.Links.Self, "page=2")
	assertContains(t, resp.Links.Self, "limit=10")

	// Check that first link preserves filters with page=1
	assertContains(t, resp.Links.First, "search=alice")
	assertContains(t, resp.Links.First, "section_id=5")
	assertContains(t, resp.Links.First, "page=1")

	// Check prev link (page 2 -> page 1)
	if resp.Links.Prev == nil {
		t.Fatal("expected prev link on page 2")
	}
	assertContains(t, *resp.Links.Prev, "search=alice")
	assertContains(t, *resp.Links.Prev, "page=1")

	// Check next link (page 2 -> page 3)
	if resp.Links.Next == nil {
		t.Fatal("expected next link on page 2 of 5")
	}
	assertContains(t, *resp.Links.Next, "search=alice")
	assertContains(t, *resp.Links.Next, "page=3")
}

func TestPaginatedResponse_NoFilters(t *testing.T) {
	resp := NewPaginatedResponseWithLinks([]int{1}, 1, 20, 1, "/api/v1/users", "")

	assertContains(t, resp.Links.Self, "page=1")
	assertContains(t, resp.Links.Self, "limit=20")
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !contains(s, substr) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
