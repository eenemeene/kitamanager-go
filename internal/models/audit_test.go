package models

import (
	"testing"
	"time"
)

func TestAuditLog_ToResponse(t *testing.T) {
	now := time.Now()
	userID := uint(5)
	resourceID := uint(42)

	log := AuditLog{
		ID:           1,
		Timestamp:    now,
		UserID:       &userID,
		UserEmail:    "admin@example.com",
		Action:       AuditActionLogin,
		ResourceType: "user",
		ResourceID:   &resourceID,
		IPAddress:    "192.168.1.1",
		UserAgent:    "Mozilla/5.0",
		Details:      `{"key":"value"}`,
		Success:      true,
	}

	resp := log.ToResponse()

	if resp.ID != 1 {
		t.Errorf("ID = %d, want 1", resp.ID)
	}
	if !resp.Timestamp.Equal(now) {
		t.Errorf("Timestamp = %v, want %v", resp.Timestamp, now)
	}
	if resp.UserID == nil || *resp.UserID != 5 {
		t.Errorf("UserID = %v, want 5", resp.UserID)
	}
	if resp.UserEmail != "admin@example.com" {
		t.Errorf("UserEmail = %q, want %q", resp.UserEmail, "admin@example.com")
	}
	if resp.Action != AuditActionLogin {
		t.Errorf("Action = %q, want %q", resp.Action, AuditActionLogin)
	}
	if resp.ResourceType != "user" {
		t.Errorf("ResourceType = %q, want %q", resp.ResourceType, "user")
	}
	if resp.ResourceID == nil || *resp.ResourceID != 42 {
		t.Errorf("ResourceID = %v, want 42", resp.ResourceID)
	}
	if resp.IPAddress != "192.168.1.1" {
		t.Errorf("IPAddress = %q, want %q", resp.IPAddress, "192.168.1.1")
	}
	if resp.Details != `{"key":"value"}` {
		t.Errorf("Details = %q, want %q", resp.Details, `{"key":"value"}`)
	}
	if resp.Success != true {
		t.Errorf("Success = %v, want true", resp.Success)
	}
}

func TestAuditLog_ToResponse_NilOptionalFields(t *testing.T) {
	log := AuditLog{
		ID:        2,
		Timestamp: time.Now(),
		Action:    AuditActionChildDelete,
		Success:   false,
	}

	resp := log.ToResponse()

	if resp.UserID != nil {
		t.Errorf("UserID = %v, want nil", resp.UserID)
	}
	if resp.ResourceID != nil {
		t.Errorf("ResourceID = %v, want nil", resp.ResourceID)
	}
	if resp.UserEmail != "" {
		t.Errorf("UserEmail = %q, want empty", resp.UserEmail)
	}
	if resp.Success != false {
		t.Errorf("Success = %v, want false", resp.Success)
	}
}
