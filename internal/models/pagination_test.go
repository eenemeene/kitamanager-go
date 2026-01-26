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

func TestNewPaginatedResponse(t *testing.T) {
	tests := []struct {
		name               string
		data               []int
		page               int
		limit              int
		total              int64
		expectedTotalPages int
	}{
		{"100 items page 1 limit 20", []int{1, 2, 3}, 1, 20, 100, 5},
		{"0 items", []int{}, 1, 20, 0, 0},
		{"21 items limit 20", []int{1}, 1, 20, 21, 2},
		{"20 items limit 20", []int{1}, 1, 20, 20, 1},
		{"1 item limit 20", []int{1}, 1, 20, 1, 1},
		{"99 items limit 20", []int{1}, 1, 20, 99, 5},
		{"100 items limit 100", []int{1}, 1, 100, 100, 1},
		{"101 items limit 100", []int{1}, 1, 100, 101, 2},
		{"1 item limit 1", []int{1}, 1, 1, 1, 1},
		{"5 items limit 1", []int{1}, 1, 1, 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewPaginatedResponse(tt.data, tt.page, tt.limit, tt.total)

			if resp.Page != tt.page {
				t.Errorf("Page = %d, want %d", resp.Page, tt.page)
			}
			if resp.Limit != tt.limit {
				t.Errorf("Limit = %d, want %d", resp.Limit, tt.limit)
			}
			if resp.Total != tt.total {
				t.Errorf("Total = %d, want %d", resp.Total, tt.total)
			}
			if resp.TotalPages != tt.expectedTotalPages {
				t.Errorf("TotalPages = %d, want %d", resp.TotalPages, tt.expectedTotalPages)
			}
			if len(resp.Data) != len(tt.data) {
				t.Errorf("len(Data) = %d, want %d", len(resp.Data), len(tt.data))
			}
		})
	}
}

func TestNewPaginatedResponse_DataPreserved(t *testing.T) {
	data := []string{"a", "b", "c"}
	resp := NewPaginatedResponse(data, 1, 10, 3)

	if len(resp.Data) != 3 {
		t.Fatalf("len(Data) = %d, want 3", len(resp.Data))
	}
	if resp.Data[0] != "a" || resp.Data[1] != "b" || resp.Data[2] != "c" {
		t.Errorf("Data = %v, want [a b c]", resp.Data)
	}
}

func TestNewPaginatedResponse_EmptySlice(t *testing.T) {
	var data []string
	resp := NewPaginatedResponse(data, 1, 20, 0)

	if len(resp.Data) != 0 {
		t.Errorf("Data should be empty, got %v", resp.Data)
	}
	if resp.TotalPages != 0 {
		t.Errorf("TotalPages = %d, want 0", resp.TotalPages)
	}
}

func TestNewPaginatedResponse_GenericType(t *testing.T) {
	type CustomType struct {
		ID   int
		Name string
	}

	data := []CustomType{{ID: 1, Name: "First"}, {ID: 2, Name: "Second"}}
	resp := NewPaginatedResponse(data, 2, 10, 25)

	if len(resp.Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2", len(resp.Data))
	}
	if resp.Data[0].ID != 1 || resp.Data[0].Name != "First" {
		t.Errorf("Data[0] = %v, want {1 First}", resp.Data[0])
	}
	if resp.TotalPages != 3 {
		t.Errorf("TotalPages = %d, want 3", resp.TotalPages)
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
	resp := NewPaginatedResponse([]int{}, 1, 25, 100)
	if resp.TotalPages != 4 {
		t.Errorf("100/25 should give 4 total pages, got %d", resp.TotalPages)
	}

	// Edge case: not exactly divisible (ceiling)
	resp = NewPaginatedResponse([]int{}, 1, 25, 101)
	if resp.TotalPages != 5 {
		t.Errorf("101/25 should give 5 total pages (ceiling), got %d", resp.TotalPages)
	}

	// Edge case: single item
	resp = NewPaginatedResponse([]int{}, 1, 20, 1)
	if resp.TotalPages != 1 {
		t.Errorf("1 item should give 1 total page, got %d", resp.TotalPages)
	}
}
