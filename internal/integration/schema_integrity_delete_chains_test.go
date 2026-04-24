//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// Migration 000014 wired up ON DELETE CASCADE / SET NULL on FKs that
// were silently blocking org-delete and user-delete in production.
// These tests exercise the two delete chains end-to-end against a
// real Postgres and assert that the cascade actually propagates —
// before 000014 each of these scenarios would fail with a foreign-key
// violation at the first missing clause.

// TestDeleteChain_OrgWithPayPlansCascades covers fix (1): deleting an
// organization that owns a pay_plan must cascade through the whole
// pay_plan subtree (plan → period → entry) without blocking.
func TestDeleteChain_OrgWithPayPlansCascades(t *testing.T) {
	cleanupDatabase()
	ctx := context.Background()

	org := &models.Organization{Name: "DeleteChain Org", Active: true, State: "berlin"}
	if err := testDB.Create(org).Error; err != nil {
		t.Fatalf("create org: %v", err)
	}
	plan := &models.PayPlan{OrganizationID: org.ID, Name: "Standard"}
	if err := testDB.Create(plan).Error; err != nil {
		t.Fatalf("create pay_plan: %v", err)
	}
	period := &models.PayPlanPeriod{
		PayPlanID:                plan.ID,
		Period:                   models.Period{From: time.Now().UTC(), To: nil},
		WeeklyHours:              39.0,
		EmployerContributionRate: 21,
	}
	if err := testDB.Create(period).Error; err != nil {
		t.Fatalf("create pay_plan_period: %v", err)
	}
	entry := &models.PayPlanEntry{
		PeriodID:      period.ID,
		Grade:         "E6",
		Step:          1,
		MonthlyAmount: 300000,
	}
	if err := testDB.Create(entry).Error; err != nil {
		t.Fatalf("create pay_plan_entry: %v", err)
	}

	// Delete the org. Pre-000014 this would fail with
	//   ERROR: update or delete on table "organizations" violates foreign
	//   key constraint "pay_plans_organization_id_fkey" on table "pay_plans"
	// Hard-delete via Unscoped() — soft-delete (migration 000015)
	// doesn't fire the FK CASCADE, which is the invariant this
	// test verifies. Soft-delete behaviour is covered separately
	// in the service-layer soft_delete_edge_cases_test.go.
	if err := testDB.Unscoped().Delete(&models.Organization{}, org.ID).Error; err != nil {
		t.Fatalf("org hard-delete must succeed post-000014; got %v", err)
	}

	// Verify the cascade did its job: plan, period and entry are all
	// gone. Any row left behind would be an orphan.
	var left int64
	_ = testDB.Model(&models.PayPlan{}).Where("organization_id = ?", org.ID).Count(&left)
	if left != 0 {
		t.Errorf("pay_plans: %d rows left after org delete", left)
	}
	_ = testDB.Model(&models.PayPlanPeriod{}).Where("pay_plan_id = ?", plan.ID).Count(&left)
	if left != 0 {
		t.Errorf("pay_plan_periods: %d rows left after cascade", left)
	}
	_ = testDB.Model(&models.PayPlanEntry{}).Where("period_id = ?", period.ID).Count(&left)
	if left != 0 {
		t.Errorf("pay_plan_entries: %d rows left after cascade", left)
	}
	_ = ctx
}

// TestDeleteChain_OrgWithBillPeriodsCascades covers fix (2): deleting
// an organization that has a government_funding_bill_periods row must
// cascade. Pre-000014 any org that had ever uploaded a funding bill
// could not be deleted — the org-delete rolled back at the bills FK.
func TestDeleteChain_OrgWithBillPeriodsCascades(t *testing.T) {
	cleanupDatabase()

	// Seed an uploader. The bill_period.created_by FK used to block
	// user deletion too; this test implicitly covers that invariant
	// as well by deleting the org (which CASCADEs users via
	// user_organizations) while a bill exists.
	user := createBasicUser(t, "uploader@example.com")
	org := &models.Organization{Name: "BillOrg", Active: true, State: "berlin"}
	if err := testDB.Create(org).Error; err != nil {
		t.Fatalf("create org: %v", err)
	}

	billPeriod := &models.GovernmentFundingBillPeriod{
		OrganizationID:    org.ID,
		Period:            models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)},
		FileName:          "bill.xlsx",
		FileSha256:        "cafeBABE" + "00000000000000000000000000000000000000000000000000000000",
		FacilityName:      "Kita",
		FacilityTotal:     100000,
		ContractBooking:   90000,
		CorrectionBooking: 10000,
		CreatedBy:         &user.ID,
	}
	if err := testDB.Create(billPeriod).Error; err != nil {
		t.Fatalf("create bill period: %v", err)
	}

	// Hard-delete via Unscoped() — soft-delete (migration 000015)
	// doesn't fire the FK CASCADE, which is the invariant this
	// test verifies. Soft-delete behaviour is covered separately
	// in the service-layer soft_delete_edge_cases_test.go.
	if err := testDB.Unscoped().Delete(&models.Organization{}, org.ID).Error; err != nil {
		t.Fatalf("org hard-delete must succeed post-000014; got %v", err)
	}
	var left int64
	_ = testDB.Model(&models.GovernmentFundingBillPeriod{}).Where("organization_id = ?", org.ID).Count(&left)
	if left != 0 {
		t.Errorf("bill_periods: %d rows left after org delete", left)
	}
}

// TestDeleteChain_UserWithAttendanceSetsNull covers fix (3): deleting
// a user who has recorded attendance must succeed. The attendance
// rows themselves are preserved (audit / billing evidence) with
// recorded_by set to NULL.
func TestDeleteChain_UserWithAttendanceSetsNull(t *testing.T) {
	cleanupDatabase()

	user := createBasicUser(t, "recorder@example.com")
	org := &models.Organization{Name: "AttOrg", Active: true, State: "berlin"}
	if err := testDB.Create(org).Error; err != nil {
		t.Fatalf("create org: %v", err)
	}
	child := &models.Child{
		Person: models.Person{
			FirstName:      "Emma",
			LastName:       "Schmidt",
			Birthdate:      time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
			OrganizationID: org.ID,
		},
	}
	if err := testDB.Create(child).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}
	attendance := &models.ChildAttendance{
		ChildID:        child.ID,
		OrganizationID: org.ID,
		Date:           time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		Status:         models.ChildAttendanceStatusPresent,
		RecordedBy:     &user.ID,
	}
	if err := testDB.Create(attendance).Error; err != nil {
		t.Fatalf("create attendance: %v", err)
	}

	// Hard-delete the user — Unscoped() bypasses the soft-delete
	// tombstone introduced in migration 000015. The FK ON DELETE
	// SET NULL cascade only fires on real DELETE, which is the
	// behaviour we're testing. The soft-delete path is covered in
	// internal/service/soft_delete_edge_cases_test.go.
	if err := testDB.Unscoped().Delete(&models.User{}, user.ID).Error; err != nil {
		t.Fatalf("user hard-delete must succeed post-000014; got %v", err)
	}

	// Attendance row survives; recorded_by is NULL.
	var reloaded models.ChildAttendance
	if err := testDB.First(&reloaded, attendance.ID).Error; err != nil {
		t.Fatalf("attendance must still exist; got %v", err)
	}
	if reloaded.RecordedBy != nil {
		t.Errorf("recorded_by must be NULL after user delete, got %v", *reloaded.RecordedBy)
	}
}

// TestDeleteChain_UserWithBillPeriodSetsNull covers fix (4): same
// semantic for government_funding_bill_periods.created_by.
func TestDeleteChain_UserWithBillPeriodSetsNull(t *testing.T) {
	cleanupDatabase()

	user := createBasicUser(t, "billuploader@example.com")
	org := &models.Organization{Name: "BillUserOrg", Active: true, State: "berlin"}
	if err := testDB.Create(org).Error; err != nil {
		t.Fatalf("create org: %v", err)
	}
	billPeriod := &models.GovernmentFundingBillPeriod{
		OrganizationID:    org.ID,
		Period:            models.Period{From: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)},
		FileName:          "bill-user.xlsx",
		FileSha256:        "deadbeef" + "00000000000000000000000000000000000000000000000000000000",
		FacilityName:      "Kita",
		FacilityTotal:     100000,
		ContractBooking:   90000,
		CorrectionBooking: 10000,
		CreatedBy:         &user.ID,
	}
	if err := testDB.Create(billPeriod).Error; err != nil {
		t.Fatalf("create bill period: %v", err)
	}

	if err := testDB.Unscoped().Delete(&models.User{}, user.ID).Error; err != nil {
		t.Fatalf("user hard-delete must succeed post-000014; got %v", err)
	}

	var reloaded models.GovernmentFundingBillPeriod
	if err := testDB.First(&reloaded, billPeriod.ID).Error; err != nil {
		t.Fatalf("bill period must still exist; got %v", err)
	}
	if reloaded.CreatedBy != nil {
		t.Errorf("created_by must be NULL after user delete, got %v", *reloaded.CreatedBy)
	}
}

// TestChildAttendanceStatusCheck covers fix (5): direct SQL inserts
// of invalid status values are now rejected by the CHECK constraint,
// not only by the service-layer enum validation.
func TestChildAttendanceStatusCheck(t *testing.T) {
	cleanupDatabase()

	org := &models.Organization{Name: "CheckOrg", Active: true, State: "berlin"}
	if err := testDB.Create(org).Error; err != nil {
		t.Fatalf("create org: %v", err)
	}
	child := &models.Child{
		Person: models.Person{
			FirstName:      "X",
			LastName:       "Y",
			Birthdate:      time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
			OrganizationID: org.ID,
		},
	}
	if err := testDB.Create(child).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}

	// Bypass the service's enum validation by writing directly with
	// SQL. The CHECK must kick in.
	err := testDB.Exec(
		"INSERT INTO child_attendances (child_id, organization_id, date, status, created_at, updated_at) VALUES (?, ?, ?, ?, NOW(), NOW())",
		child.ID, org.ID, time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC), "definitely-invalid",
	).Error
	if err == nil {
		t.Fatal("INSERT with invalid status must fail the CHECK constraint")
	}
	// Valid values must still go through.
	for _, ok := range []string{"present", "absent", "sick", "vacation"} {
		err := testDB.Exec(
			"INSERT INTO child_attendances (child_id, organization_id, date, status, created_at, updated_at) VALUES (?, ?, ?, ?, NOW(), NOW())",
			child.ID, org.ID, time.Date(2025, 7, statusDay(ok), 0, 0, 0, 0, time.UTC), ok,
		).Error
		if err != nil {
			t.Errorf("valid status %q must be accepted; got %v", ok, err)
		}
	}
}

// statusDay maps the four valid statuses to distinct days so the
// uniqueness constraint (child_id, date) doesn't get in the way.
func statusDay(s string) int {
	switch s {
	case "present":
		return 1
	case "absent":
		return 2
	case "sick":
		return 3
	case "vacation":
		return 4
	}
	return 0
}

// createBasicUser is a minimal helper — the integration package has
// existing user-creation patterns scattered across auth tests; this
// local helper keeps this file self-contained.
func createBasicUser(t *testing.T, email string) *models.User {
	t.Helper()
	u := &models.User{
		Name:     "Test " + email,
		Email:    email,
		Password: "$2a$12$0000000000000000000000000000000000000000000000000000w", // placeholder hash
		Active:   true,
	}
	if err := store.NewUserStore(testDB).Create(context.Background(), u); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}
