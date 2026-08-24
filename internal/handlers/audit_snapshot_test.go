package handlers

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestAuditSnapshot_KeepsScalarsAndDropsBookkeeping(t *testing.T) {
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	got := auditSnapshot(&models.PayPlanPeriodResponse{
		ID:                       7,
		PayPlanID:                3,
		From:                     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:                       &to,
		WeeklyHours:              39,
		EmployerContributionRate: 2200,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	})

	if _, ok := got["created_at"]; ok {
		t.Error("created_at must not reach an audit row")
	}
	if _, ok := got["updated_at"]; ok {
		t.Error("updated_at must not reach an audit row")
	}
	if got["employer_contribution_rate"].(json.Number).String() != "2200" {
		t.Errorf("expected the rate recorded as written, got %v", got["employer_contribution_rate"])
	}
	if got["weekly_hours"].(json.Number).String() != "39" {
		t.Errorf("expected weekly_hours 39, got %v", got["weekly_hours"])
	}
	if got["to"] != "2024-12-31T00:00:00Z" {
		t.Errorf("expected an RFC3339 end date, got %v", got["to"])
	}
}

// The entries slice is what makes a naive snapshot unbounded: a period carries
// every salary row under it.
func TestAuditSnapshot_DropsNestedCollections(t *testing.T) {
	got := auditSnapshot(&models.PayPlanPeriodResponse{
		ID: 7,
		Entries: []models.PayPlanEntryResponse{
			{ID: 1, Grade: "S8a", Step: 3, MonthlyAmount: 350000},
			{ID: 2, Grade: "S8a", Step: 4, MonthlyAmount: 360000},
		},
	})
	if _, ok := got["entries"]; ok {
		t.Error("nested collections must not be copied into the parent's audit row")
	}
	if got["id"].(json.Number).String() != "7" {
		t.Errorf("expected the period's own fields to survive, got %v", got)
	}
}

// A child's name must not ride along into an attendance audit row.
func TestAuditSnapshot_DropsChildName(t *testing.T) {
	got := auditSnapshot(&models.ChildAttendanceResponse{
		ID:        4,
		ChildID:   9,
		ChildName: "Emma Testkind",
		Date:      "2025-06-15",
		Status:    "present",
	})
	if _, ok := got["child_name"]; ok {
		t.Error("child_name must not reach an audit row")
	}
	if got["child_id"].(json.Number).String() != "9" {
		t.Errorf("expected child_id to be kept as the identifier, got %v", got["child_id"])
	}
}

// Integer cents must survive as the digits they were written with. A float
// round trip is the classic way an audit log starts disagreeing with the ledger.
func TestAuditSnapshot_MoneyIsNotFloated(t *testing.T) {
	got := auditSnapshot(&models.BudgetItemEntryResponse{
		ID:          1,
		AmountCents: 123456789,
	})
	n, ok := got["amount_cents"].(json.Number)
	if !ok {
		t.Fatalf("expected a json.Number, got %T", got["amount_cents"])
	}
	if n.String() != "123456789" {
		t.Errorf("expected 123456789, got %s", n)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"amount_cents":123456789`) {
		t.Errorf("expected the exact integer in the marshalled details, got %s", raw)
	}
}

func TestAuditChangesOf_ReportsOnlyWhatChanged(t *testing.T) {
	before := &models.PayPlanEntryResponse{ID: 1, Grade: "S8a", Step: 3, MonthlyAmount: 350000}
	after := &models.PayPlanEntryResponse{ID: 1, Grade: "S8a", Step: 3, MonthlyAmount: 400000}

	changes := auditChangesOf(before, after)
	if len(changes) != 1 {
		t.Fatalf("expected exactly one changed field, got %v", changes)
	}
	got := changes["monthly_amount"].(map[string]any)
	if got["old"].(json.Number).String() != "350000" || got["new"].(json.Number).String() != "400000" {
		t.Errorf("expected 350000 -> 400000, got %v", got)
	}
}

func TestAuditChangesOf_NoOpUpdateProducesNothing(t *testing.T) {
	entry := models.PayPlanEntryResponse{ID: 1, Grade: "S8a", Step: 3, MonthlyAmount: 350000}
	other := entry
	if changes := auditChangesOf(&entry, &other); changes != nil {
		t.Errorf("expected no changes for an identical pair, got %v", changes)
	}
}

// created_at/updated_at moving must not on its own make an update look like a
// content change — otherwise every single update carries a changes map.
func TestAuditChangesOf_TimestampsAloneAreNotAChange(t *testing.T) {
	before := &models.PayPlanEntryResponse{ID: 1, Grade: "S8a", UpdatedAt: time.Now()}
	after := &models.PayPlanEntryResponse{ID: 1, Grade: "S8a", UpdatedAt: time.Now().Add(time.Hour)}
	if changes := auditChangesOf(before, after); changes != nil {
		t.Errorf("expected no changes when only updated_at moved, got %v", changes)
	}
}

// An omitempty pointer going to nil disappears from the JSON entirely. Treating
// that as "no change" would silently lose the most consequential edit a funding
// period can have: closing an open-ended one, or reopening a closed one.
func TestAuditChangesOf_OmitemptyFieldClearedIsAChange(t *testing.T) {
	before := &models.PayPlanEntryResponse{ID: 1, StepMinYears: intPtrForAudit(3)}
	after := &models.PayPlanEntryResponse{ID: 1}

	changes := auditChangesOf(before, after)
	got, ok := changes["step_min_years"].(map[string]any)
	if !ok {
		t.Fatalf("expected step_min_years to be reported as changed, got %v", changes)
	}
	if got["old"].(json.Number).String() != "3" || got["new"] != nil {
		t.Errorf("expected 3 -> nil, got %v", got)
	}

	// And the reverse direction.
	reversed := auditChangesOf(after, before)
	got2 := reversed["step_min_years"].(map[string]any)
	if got2["old"] != nil || got2["new"].(json.Number).String() != "3" {
		t.Errorf("expected nil -> 3, got %v", got2)
	}
}

func intPtrForAudit(v int) *int { return &v }

func TestAuditChanges_EmptyMapsProduceNil(t *testing.T) {
	if changes := auditChanges(nil, nil); changes != nil {
		t.Errorf("expected nil for two empty snapshots, got %v", changes)
	}
}
