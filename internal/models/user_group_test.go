package models

import (
	"testing"
	"time"
)

func TestRole_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		role     Role
		expected bool
	}{
		{"valid admin", RoleAdmin, true},
		{"valid manager", RoleManager, true},
		{"valid member", RoleMember, true},
		{"empty string", Role(""), false},
		{"superadmin is not valid", Role("superadmin"), false},
		{"case sensitive - Admin", Role("Admin"), false},
		{"case sensitive - ADMIN", Role("ADMIN"), false},
		{"unknown role", Role("unknown"), false},
		{"whitespace", Role(" "), false},
		{"whitespace admin", Role(" admin"), false},
		{"admin with trailing space", Role("admin "), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.role.IsValid()
			if got != tt.expected {
				t.Errorf("Role(%q).IsValid() = %v, want %v", tt.role, got, tt.expected)
			}
		})
	}
}

func TestRole_Precedence(t *testing.T) {
	tests := []struct {
		name     string
		role     Role
		expected int
	}{
		{"admin has precedence 3", RoleAdmin, 3},
		{"manager has precedence 2", RoleManager, 2},
		{"member has precedence 1", RoleMember, 1},
		{"invalid role has precedence 0", Role("invalid"), 0},
		{"empty string has precedence 0", Role(""), 0},
		{"superadmin has precedence 0", Role("superadmin"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.role.Precedence()
			if got != tt.expected {
				t.Errorf("Role(%q).Precedence() = %v, want %v", tt.role, got, tt.expected)
			}
		})
	}
}

func TestRole_Precedence_Ordering(t *testing.T) {
	// Verify that admin > manager > member
	if RoleAdmin.Precedence() <= RoleManager.Precedence() {
		t.Error("admin should have higher precedence than manager")
	}
	if RoleManager.Precedence() <= RoleMember.Precedence() {
		t.Error("manager should have higher precedence than member")
	}
}

func TestUserGroup_ToResponse(t *testing.T) {
	now := time.Now()
	group := &Group{
		ID:             1,
		Name:           "Test Group",
		OrganizationID: 1,
		Active:         true,
	}

	ug := &UserGroup{
		UserID:    1,
		GroupID:   1,
		Role:      RoleAdmin,
		CreatedAt: now,
		CreatedBy: "admin@example.com",
		Group:     group,
	}

	resp := ug.ToResponse()

	if resp.UserID != 1 {
		t.Errorf("ToResponse().UserID = %d, want 1", resp.UserID)
	}
	if resp.GroupID != 1 {
		t.Errorf("ToResponse().GroupID = %d, want 1", resp.GroupID)
	}
	if resp.Role != RoleAdmin {
		t.Errorf("ToResponse().Role = %v, want %v", resp.Role, RoleAdmin)
	}
	if !resp.CreatedAt.Equal(now) {
		t.Errorf("ToResponse().CreatedAt = %v, want %v", resp.CreatedAt, now)
	}
	if resp.CreatedBy != "admin@example.com" {
		t.Errorf("ToResponse().CreatedBy = %v, want admin@example.com", resp.CreatedBy)
	}
	if resp.Group != group {
		t.Error("ToResponse().Group should reference the same group")
	}
}

func TestUserGroup_ToResponse_NilGroup(t *testing.T) {
	ug := &UserGroup{
		UserID:    1,
		GroupID:   1,
		Role:      RoleMember,
		CreatedBy: "test@example.com",
		Group:     nil,
	}

	resp := ug.ToResponse()

	if resp.Group != nil {
		t.Error("ToResponse().Group should be nil when UserGroup.Group is nil")
	}
	if resp.UserID != 1 {
		t.Errorf("ToResponse().UserID = %d, want 1", resp.UserID)
	}
}

func TestUserGroup_ToResponse_ZeroValues(t *testing.T) {
	ug := &UserGroup{}

	resp := ug.ToResponse()

	if resp.UserID != 0 {
		t.Errorf("ToResponse().UserID = %d, want 0", resp.UserID)
	}
	if resp.GroupID != 0 {
		t.Errorf("ToResponse().GroupID = %d, want 0", resp.GroupID)
	}
	if resp.Role != "" {
		t.Errorf("ToResponse().Role = %v, want empty", resp.Role)
	}
	if resp.CreatedBy != "" {
		t.Errorf("ToResponse().CreatedBy = %v, want empty", resp.CreatedBy)
	}
}

func TestUserGroup_TableName(t *testing.T) {
	ug := UserGroup{}
	if ug.TableName() != "user_groups" {
		t.Errorf("TableName() = %v, want user_groups", ug.TableName())
	}
}

func TestAddUserToGroupRequest(t *testing.T) {
	req := AddUserToGroupRequest{
		GroupID: 5,
		Role:    RoleManager,
	}

	if req.GroupID != 5 {
		t.Errorf("GroupID = %d, want 5", req.GroupID)
	}
	if req.Role != RoleManager {
		t.Errorf("Role = %v, want %v", req.Role, RoleManager)
	}
}

func TestUpdateUserGroupRoleRequest(t *testing.T) {
	req := UpdateUserGroupRoleRequest{
		Role: RoleAdmin,
	}

	if req.Role != RoleAdmin {
		t.Errorf("Role = %v, want %v", req.Role, RoleAdmin)
	}
}

func TestUserMembership(t *testing.T) {
	org := &Organization{ID: 1, Name: "Test Org"}
	group := &Group{ID: 1, Name: "Test Group", Organization: org}

	membership := UserMembership{
		UserID:           1,
		GroupID:          1,
		Role:             RoleMember,
		Group:            group,
		EffectiveOrgRole: RoleAdmin,
		Organization:     org,
	}

	if membership.UserID != 1 {
		t.Errorf("UserID = %d, want 1", membership.UserID)
	}
	if membership.EffectiveOrgRole != RoleAdmin {
		t.Errorf("EffectiveOrgRole = %v, want %v", membership.EffectiveOrgRole, RoleAdmin)
	}
	if membership.Organization != org {
		t.Error("Organization should reference the same org")
	}
}

func TestUserMembershipsResponse(t *testing.T) {
	memberships := []UserMembership{
		{UserID: 1, GroupID: 1, Role: RoleAdmin},
		{UserID: 1, GroupID: 2, Role: RoleMember},
	}

	resp := UserMembershipsResponse{Memberships: memberships}

	if len(resp.Memberships) != 2 {
		t.Errorf("len(Memberships) = %d, want 2", len(resp.Memberships))
	}
}

func TestUserMembershipsResponse_Empty(t *testing.T) {
	resp := UserMembershipsResponse{Memberships: []UserMembership{}}

	if len(resp.Memberships) != 0 {
		t.Errorf("len(Memberships) = %d, want 0", len(resp.Memberships))
	}
}
