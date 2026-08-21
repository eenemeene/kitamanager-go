package service

import (
	"context"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/testutil"
)

// An org admin reading their organization's audit feed must not be handed a
// colleague's exact IP address.
//
// The authorization middleware has always stashed ctxkeys.IsSuperAdmin with a
// comment saying it existed so the audit handler need not repeat the lookup for
// "audit-log IP redaction". Nothing read that key and no redaction existed, so
// every org admin could read the home address of everyone who had ever touched
// a record in their org.

func seedOneOrgAuditRow(t *testing.T, ip string) (*AuditService, uint) {
	t.Helper()
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	org := testutil.CreateTestOrganization(t, db, "Kita Sonnenschein")

	svc := NewAuditService(auditStore)
	svc.LogResourceCreate(context.Background(), 1, "actor@example.com", "child", 42, "Anna", ip, &org.ID)
	// Drain the async worker so the read below is deterministic.
	svc.Shutdown()

	return NewAuditService(auditStore), org.ID
}

func TestGetLogsByOrganization_AnonymizesIPForNonSuperAdmin(t *testing.T) {
	readSvc, orgID := seedOneOrgAuditRow(t, "203.0.113.147")
	defer readSvc.Shutdown()

	logs, _, err := readSvc.GetLogsByOrganization(context.Background(), orgID, "", nil, nil, nil, 100, 0, IPAnonymized)
	if err != nil {
		t.Fatalf("GetLogsByOrganization: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 row, got %d", len(logs))
	}
	if logs[0].IPAddress != "203.0.113.0" {
		t.Errorf("IPAddress = %q, want the /24 prefix %q", logs[0].IPAddress, "203.0.113.0")
	}
	if !logs[0].IPAnonymized {
		t.Error("IPAnonymized must be set so the client knows it is looking at a prefix")
	}
}

func TestGetLogsByOrganization_SuperAdminSeesTheRecordedAddress(t *testing.T) {
	readSvc, orgID := seedOneOrgAuditRow(t, "203.0.113.147")
	defer readSvc.Shutdown()

	logs, _, err := readSvc.GetLogsByOrganization(context.Background(), orgID, "", nil, nil, nil, 100, 0, IPFull)
	if err != nil {
		t.Fatalf("GetLogsByOrganization: %v", err)
	}
	if logs[0].IPAddress != "203.0.113.147" {
		t.Errorf("IPAddress = %q, want the address as recorded", logs[0].IPAddress)
	}
	if logs[0].IPAnonymized {
		t.Error("IPAnonymized must stay unset when nothing was reduced")
	}
}

// The zero value of IPVisibility has to be the safe one: an endpoint added
// later, or a refactor that drops the argument, must fail closed.
func TestIPVisibility_ZeroValueAnonymizes(t *testing.T) {
	var unset IPVisibility
	if unset != IPAnonymized {
		t.Fatal("the zero value of IPVisibility must be IPAnonymized so forgetting to choose protects the data")
	}

	readSvc, orgID := seedOneOrgAuditRow(t, "203.0.113.147")
	defer readSvc.Shutdown()

	logs, _, err := readSvc.GetLogsByOrganization(context.Background(), orgID, "", nil, nil, nil, 100, 0, unset)
	if err != nil {
		t.Fatalf("GetLogsByOrganization: %v", err)
	}
	if logs[0].IPAddress != "203.0.113.0" {
		t.Errorf("an unset visibility must anonymize; got %q", logs[0].IPAddress)
	}
}

func TestGetLogByID_AnonymizesForNonSuperAdmin(t *testing.T) {
	readSvc, orgID := seedOneOrgAuditRow(t, "203.0.113.147")
	defer readSvc.Shutdown()

	logs, _, err := readSvc.GetLogsByOrganization(context.Background(), orgID, "", nil, nil, nil, 100, 0, IPFull)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	single, err := readSvc.GetLogByID(context.Background(), logs[0].ID, IPAnonymized)
	if err != nil {
		t.Fatalf("GetLogByID: %v", err)
	}
	if single.IPAddress != "203.0.113.0" || !single.IPAnonymized {
		t.Errorf("single-row read must anonymize too: got %q anonymized=%v", single.IPAddress, single.IPAnonymized)
	}
}
