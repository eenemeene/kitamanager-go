package models

import (
	"testing"
	"time"
)

func TestValidBudgetItemCategory(t *testing.T) {
	tests := []struct {
		name     string
		category string
		want     bool
	}{
		{"valid income", "income", true},
		{"valid expense", "expense", true},
		{"invalid empty", "", false},
		{"invalid other", "other", false},
		{"invalid uppercase", "Income", false},
		{"invalid whitespace", " income", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidBudgetItemCategory(tt.category); got != tt.want {
				t.Errorf("ValidBudgetItemCategory(%q) = %v, want %v", tt.category, got, tt.want)
			}
		})
	}
}

func TestBudgetItem_GetOrganizationID(t *testing.T) {
	b := BudgetItem{OrganizationID: 42}
	if got := b.GetOrganizationID(); got != 42 {
		t.Errorf("GetOrganizationID() = %d, want 42", got)
	}
}

func TestBudgetItemEntry_GetOwnerID(t *testing.T) {
	e := BudgetItemEntry{BudgetItemID: 7}
	if got := e.GetOwnerID(); got != 7 {
		t.Errorf("GetOwnerID() = %d, want 7", got)
	}
}

func TestBudgetItem_ToResponse_NoActiveEntry(t *testing.T) {
	now := time.Now()
	b := BudgetItem{
		ID:             1,
		OrganizationID: 2,
		Name:           "Elternbeiträge",
		Category:       "income",
		PerChild:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	resp := b.ToResponse(time.Now().UTC())

	if resp.ID != 1 {
		t.Errorf("ID = %d, want 1", resp.ID)
	}
	if resp.OrganizationID != 2 {
		t.Errorf("OrganizationID = %d, want 2", resp.OrganizationID)
	}
	if resp.Name != "Elternbeiträge" {
		t.Errorf("Name = %q, want %q", resp.Name, "Elternbeiträge")
	}
	if resp.Category != "income" {
		t.Errorf("Category = %q, want %q", resp.Category, "income")
	}
	if resp.PerChild != true {
		t.Errorf("PerChild = %v, want true", resp.PerChild)
	}
	if resp.ActiveAmountCents != nil {
		t.Errorf("ActiveAmountCents = %v, want nil", resp.ActiveAmountCents)
	}
}

func TestBudgetItem_ToResponse_WithActiveEntry(t *testing.T) {
	now := time.Now().UTC()
	pastDate := now.AddDate(0, -1, 0)

	b := BudgetItem{
		ID:             1,
		OrganizationID: 2,
		Name:           "Rent",
		Category:       "expense",
		Entries: []BudgetItemEntry{
			{
				ID:           10,
				BudgetItemID: 1,
				Period:       Period{From: pastDate},
				AmountCents:  50000,
			},
		},
	}

	resp := b.ToResponse(time.Now().UTC())

	if resp.ActiveAmountCents == nil {
		t.Fatal("ActiveAmountCents = nil, want non-nil")
	}
	if *resp.ActiveAmountCents != 50000 {
		t.Errorf("ActiveAmountCents = %d, want 50000", *resp.ActiveAmountCents)
	}
}

// ============================================================
// B5: ToResponse(asOf) — caller controls "active when?"
// ============================================================

func TestBudgetItem_ToResponse_AsOfPicksMatchingEntry(t *testing.T) {
	// Two non-overlapping entries — one active in 2024, one in 2025.
	// ToResponse(asOf=2024-06-01) must pick the 2024 entry;
	// ToResponse(asOf=2025-06-01) must pick the 2025 one. The
	// previous time.Now()-by-default version meant a list-page
	// rendered today would always pick "today" regardless of the
	// caller's intent (e.g. a future "as-of-date" filter).
	endOf2024 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	b := BudgetItem{
		ID: 1, OrganizationID: 2, Name: "Rent", Category: "expense",
		Entries: []BudgetItemEntry{
			{
				ID: 1, BudgetItemID: 1,
				Period:      Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &endOf2024},
				AmountCents: 30000,
			},
			{
				ID: 2, BudgetItemID: 1,
				Period:      Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
				AmountCents: 50000,
			},
		},
	}

	resp24 := b.ToResponse(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	if resp24.ActiveAmountCents == nil || *resp24.ActiveAmountCents != 30000 {
		t.Errorf("asOf 2024-06-01: ActiveAmountCents = %v, want 30000",
			derefOrNil(resp24.ActiveAmountCents))
	}

	resp25 := b.ToResponse(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC))
	if resp25.ActiveAmountCents == nil || *resp25.ActiveAmountCents != 50000 {
		t.Errorf("asOf 2025-06-01: ActiveAmountCents = %v, want 50000",
			derefOrNil(resp25.ActiveAmountCents))
	}
}

func TestBudgetItem_ToResponse_AsOfBeforeAnyEntry_ReturnsNil(t *testing.T) {
	// asOf before the first entry's From: no match. Important for
	// historical-snapshot views where the date predates the budget
	// item's existence in the books.
	b := BudgetItem{
		ID: 1, Name: "X", Category: "income",
		Entries: []BudgetItemEntry{
			{
				ID: 1, BudgetItemID: 1,
				Period:      Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
				AmountCents: 10000,
			},
		},
	}
	resp := b.ToResponse(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	if resp.ActiveAmountCents != nil {
		t.Errorf("expected nil ActiveAmountCents for asOf-before-from, got %d",
			*resp.ActiveAmountCents)
	}
}

// derefOrNil returns the int the pointer points to, or 0 if nil. Lets
// asserting test names say "got 0" instead of "got <nil>" when
// formatting %v.
func derefOrNil(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func TestBudgetItem_ToResponse_InactiveEntry(t *testing.T) {
	// Entry with end date in the past should not be active
	pastFrom := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	pastTo := time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)

	b := BudgetItem{
		ID:             1,
		OrganizationID: 2,
		Name:           "Old Item",
		Category:       "income",
		Entries: []BudgetItemEntry{
			{
				ID:           10,
				BudgetItemID: 1,
				Period:       Period{From: pastFrom, To: &pastTo},
				AmountCents:  30000,
			},
		},
	}

	resp := b.ToResponse(time.Now().UTC())

	if resp.ActiveAmountCents != nil {
		t.Errorf("ActiveAmountCents = %d, want nil (entry ended in the past)", *resp.ActiveAmountCents)
	}
}

func TestBudgetItem_ToDetailResponse(t *testing.T) {
	now := time.Now()
	to := now.AddDate(0, 6, 0)

	b := BudgetItem{
		ID:             1,
		OrganizationID: 2,
		Name:           "Essensgeld",
		Category:       "income",
		PerChild:       true,
		Entries: []BudgetItemEntry{
			{
				ID:           10,
				BudgetItemID: 1,
				Period:       Period{From: now, To: &to},
				AmountCents:  5000,
				Notes:        "Monthly",
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			{
				ID:           11,
				BudgetItemID: 1,
				Period:       Period{From: to},
				AmountCents:  5500,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	resp := b.ToDetailResponse()

	if resp.ID != 1 {
		t.Errorf("ID = %d, want 1", resp.ID)
	}
	if resp.Name != "Essensgeld" {
		t.Errorf("Name = %q, want %q", resp.Name, "Essensgeld")
	}
	if resp.PerChild != true {
		t.Errorf("PerChild = %v, want true", resp.PerChild)
	}
	if len(resp.Entries) != 2 {
		t.Fatalf("len(Entries) = %d, want 2", len(resp.Entries))
	}
	if resp.Entries[0].AmountCents != 5000 {
		t.Errorf("Entries[0].AmountCents = %d, want 5000", resp.Entries[0].AmountCents)
	}
	if resp.Entries[0].Notes != "Monthly" {
		t.Errorf("Entries[0].Notes = %q, want %q", resp.Entries[0].Notes, "Monthly")
	}
	if resp.Entries[1].AmountCents != 5500 {
		t.Errorf("Entries[1].AmountCents = %d, want 5500", resp.Entries[1].AmountCents)
	}
}

func TestBudgetItem_ToDetailResponse_EmptyEntries(t *testing.T) {
	b := BudgetItem{
		ID:   1,
		Name: "Empty",
	}

	resp := b.ToDetailResponse()

	if len(resp.Entries) != 0 {
		t.Errorf("len(Entries) = %d, want 0", len(resp.Entries))
	}
}

func TestBudgetItemEntry_ToResponse(t *testing.T) {
	now := time.Now()
	to := now.AddDate(1, 0, 0)

	entry := BudgetItemEntry{
		ID:           10,
		BudgetItemID: 1,
		Period:       Period{From: now, To: &to},
		AmountCents:  50000,
		Notes:        "Test note",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	resp := entry.ToResponse()

	if resp.ID != 10 {
		t.Errorf("ID = %d, want 10", resp.ID)
	}
	if resp.BudgetItemID != 1 {
		t.Errorf("BudgetItemID = %d, want 1", resp.BudgetItemID)
	}
	if resp.AmountCents != 50000 {
		t.Errorf("AmountCents = %d, want 50000", resp.AmountCents)
	}
	if resp.Notes != "Test note" {
		t.Errorf("Notes = %q, want %q", resp.Notes, "Test note")
	}
	if resp.To == nil {
		t.Fatal("To = nil, want non-nil")
	}
}

func TestBudgetItemEntry_ToResponse_NilTo(t *testing.T) {
	entry := BudgetItemEntry{
		ID:          10,
		Period:      Period{From: time.Now()},
		AmountCents: 30000,
	}

	resp := entry.ToResponse()

	if resp.To != nil {
		t.Errorf("To = %v, want nil", resp.To)
	}
}
