package models

import (
	"testing"
	"time"
)

func TestPerson_FullName(t *testing.T) {
	tests := []struct {
		name      string
		firstName string
		lastName  string
		want      string
	}{
		{
			name:      "normal name",
			firstName: "John",
			lastName:  "Doe",
			want:      "John Doe",
		},
		{
			name:      "empty first name",
			firstName: "",
			lastName:  "Doe",
			want:      " Doe",
		},
		{
			name:      "empty last name",
			firstName: "John",
			lastName:  "",
			want:      "John ",
		},
		{
			name:      "both empty",
			firstName: "",
			lastName:  "",
			want:      " ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Person{
				FirstName: tt.firstName,
				LastName:  tt.lastName,
			}
			if got := p.FullName(); got != tt.want {
				t.Errorf("Person.FullName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPeriod_GetFrom(t *testing.T) {
	from := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	p := Period{From: from}

	if got := p.GetFrom(); !got.Equal(from) {
		t.Errorf("Period.GetFrom() = %v, want %v", got, from)
	}
}

func TestPeriod_GetTo(t *testing.T) {
	t.Run("with end date", func(t *testing.T) {
		to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
		p := Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   &to,
		}

		got := p.GetTo()
		if got == nil {
			t.Fatal("Period.GetTo() = nil, want non-nil")
		}
		if !got.Equal(to) {
			t.Errorf("Period.GetTo() = %v, want %v", *got, to)
		}
	})

	t.Run("ongoing period (nil end date)", func(t *testing.T) {
		p := Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   nil,
		}

		if got := p.GetTo(); got != nil {
			t.Errorf("Period.GetTo() = %v, want nil", *got)
		}
	})
}

func TestPeriod_IsOngoing(t *testing.T) {
	t.Run("ongoing when To is nil", func(t *testing.T) {
		p := Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   nil,
		}

		if !p.IsOngoing() {
			t.Error("Period.IsOngoing() = false, want true")
		}
	})

	t.Run("not ongoing when To is set", func(t *testing.T) {
		to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
		p := Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   &to,
		}

		if p.IsOngoing() {
			t.Error("Period.IsOngoing() = true, want false")
		}
	})
}

func TestPeriod_IsActiveOn(t *testing.T) {
	tests := []struct {
		name   string
		period Period
		date   time.Time
		want   bool
	}{
		{
			name: "date before period start",
			period: Period{
				From: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
				To:   nil,
			},
			date: time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "date on period start",
			period: Period{
				From: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
				To:   nil,
			},
			date: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "date after period start (ongoing)",
			period: Period{
				From: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
				To:   nil,
			},
			date: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "date within bounded period",
			period: func() Period {
				to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
				return Period{
					From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					To:   &to,
				}
			}(),
			date: time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "date on period end",
			period: func() Period {
				to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
				return Period{
					From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					To:   &to,
				}
			}(),
			date: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "date after period end",
			period: func() Period {
				to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
				return Period{
					From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					To:   &to,
				}
			}(),
			date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.period.IsActiveOn(tt.date); got != tt.want {
				t.Errorf("Period.IsActiveOn(%v) = %v, want %v", tt.date, got, tt.want)
			}
		})
	}
}

func TestPeriod_IsActiveOn_ConsecutiveContracts(t *testing.T) {
	// Scenario: Contract A ends on day 10, Contract B starts on day 11.
	// Query on day 9 → A active, B not active
	// Query on day 10 → A active (last day inclusive), B not active
	// Query on day 11 → A not active, B active (first day inclusive)

	day := func(d int) time.Time {
		return time.Date(2024, 1, d, 0, 0, 0, 0, time.UTC)
	}
	dayPtr := func(d int) *time.Time {
		t := day(d)
		return &t
	}

	contractA := Period{From: day(1), To: dayPtr(10)}
	contractB := Period{From: day(11), To: nil} // ongoing

	tests := []struct {
		name  string
		date  time.Time
		wantA bool
		wantB bool
	}{
		{"day before contract A", day(0), false, false},
		{"day 9 (within A)", day(9), true, false},
		{"day 10 (last day of A, inclusive)", day(10), true, false},
		{"day 11 (first day of B, inclusive)", day(11), false, true},
		{"day 12 (within B)", day(12), false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotA := contractA.IsActiveOn(tt.date)
			gotB := contractB.IsActiveOn(tt.date)
			if gotA != tt.wantA {
				t.Errorf("contractA.IsActiveOn(%v) = %v, want %v", tt.date, gotA, tt.wantA)
			}
			if gotB != tt.wantB {
				t.Errorf("contractB.IsActiveOn(%v) = %v, want %v", tt.date, gotB, tt.wantB)
			}
		})
	}
}

func TestPeriod_IsActiveOn_GapBetweenContracts(t *testing.T) {
	// Scenario: Contract A from day 1 to day 2, Contract B from day 4 (ongoing).
	// Query on day 3 → neither contract is active (gap).

	day := func(d int) time.Time {
		return time.Date(2024, 1, d, 0, 0, 0, 0, time.UTC)
	}
	dayPtr := func(d int) *time.Time {
		t := day(d)
		return &t
	}

	contractA := Period{From: day(1), To: dayPtr(2)}
	contractB := Period{From: day(4), To: nil}

	tests := []struct {
		name  string
		date  time.Time
		wantA bool
		wantB bool
	}{
		{"day 1 (first day of A)", day(1), true, false},
		{"day 2 (last day of A)", day(2), true, false},
		{"day 3 (gap - no active contract)", day(3), false, false},
		{"day 4 (first day of B)", day(4), false, true},
		{"day 5 (within B)", day(5), false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotA := contractA.IsActiveOn(tt.date)
			gotB := contractB.IsActiveOn(tt.date)
			if gotA != tt.wantA {
				t.Errorf("contractA.IsActiveOn(%v) = %v, want %v", tt.date, gotA, tt.wantA)
			}
			if gotB != tt.wantB {
				t.Errorf("contractB.IsActiveOn(%v) = %v, want %v", tt.date, gotB, tt.wantB)
			}
		})
	}
}

func TestPeriod_IsActiveOn_TimeComponentIgnored(t *testing.T) {
	// Verify that time-of-day components are stripped (truncateToDate).
	to := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	p := Period{
		From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   &to,
	}

	// Query at 23:59:59 on the last day — should still be active.
	dateWithTime := time.Date(2024, 1, 10, 23, 59, 59, 0, time.UTC)
	if !p.IsActiveOn(dateWithTime) {
		t.Error("expected period to be active on last day even with non-zero time component")
	}

	// Query at 00:00:01 on the day after — should NOT be active.
	dayAfterWithTime := time.Date(2024, 1, 11, 0, 0, 1, 0, time.UTC)
	if p.IsActiveOn(dayAfterWithTime) {
		t.Error("expected period to NOT be active on day after end date")
	}
}

func TestEmployeeContract_GetOwnerID(t *testing.T) {
	contract := EmployeeContract{
		EmployeeID: 42,
	}

	if got := contract.GetOwnerID(); got != 42 {
		t.Errorf("EmployeeContract.GetOwnerID() = %d, want 42", got)
	}
}

func TestChildContract_GetOwnerID(t *testing.T) {
	contract := ChildContract{
		ChildID: 99,
	}

	if got := contract.GetOwnerID(); got != 99 {
		t.Errorf("ChildContract.GetOwnerID() = %d, want 99", got)
	}
}

func TestUser_ToResponse(t *testing.T) {
	now := time.Now()
	lastLogin := now.Add(-time.Hour)

	user := User{
		ID:        1,
		Name:      "John Doe",
		Email:     "john@example.com",
		Password:  "secret123",
		Active:    true,
		LastLogin: &lastLogin,
		CreatedAt: now,
		CreatedBy: "admin@example.com",
		UpdatedAt: now,
		Groups: []Group{
			{ID: 1, Name: "Group 1"},
		},
	}

	response := user.ToResponse()

	if response.ID != user.ID {
		t.Errorf("ToResponse().ID = %d, want %d", response.ID, user.ID)
	}
	if response.Name != user.Name {
		t.Errorf("ToResponse().Name = %q, want %q", response.Name, user.Name)
	}
	if response.Email != user.Email {
		t.Errorf("ToResponse().Email = %q, want %q", response.Email, user.Email)
	}
	if response.Active != user.Active {
		t.Errorf("ToResponse().Active = %v, want %v", response.Active, user.Active)
	}
	if response.LastLogin == nil || !response.LastLogin.Equal(*user.LastLogin) {
		t.Errorf("ToResponse().LastLogin = %v, want %v", response.LastLogin, user.LastLogin)
	}
	if !response.CreatedAt.Equal(user.CreatedAt) {
		t.Errorf("ToResponse().CreatedAt = %v, want %v", response.CreatedAt, user.CreatedAt)
	}
	if response.CreatedBy != user.CreatedBy {
		t.Errorf("ToResponse().CreatedBy = %q, want %q", response.CreatedBy, user.CreatedBy)
	}
	if len(response.Groups) != 1 {
		t.Errorf("ToResponse().Groups length = %d, want 1", len(response.Groups))
	}
}

func TestGroup_ToResponse(t *testing.T) {
	now := time.Now()
	org := &Organization{ID: 1, Name: "Test Org"}

	group := Group{
		ID:             1,
		Name:           "Administrators",
		OrganizationID: 1,
		Organization:   org,
		Active:         true,
		CreatedAt:      now,
		CreatedBy:      "admin@example.com",
		UpdatedAt:      now,
		Users: []User{
			{ID: 1, Name: "User 1"},
		},
	}

	response := group.ToResponse()

	if response.ID != group.ID {
		t.Errorf("ToResponse().ID = %d, want %d", response.ID, group.ID)
	}
	if response.Name != group.Name {
		t.Errorf("ToResponse().Name = %q, want %q", response.Name, group.Name)
	}
	if response.OrganizationID != group.OrganizationID {
		t.Errorf("ToResponse().OrganizationID = %d, want %d", response.OrganizationID, group.OrganizationID)
	}
	if response.Organization == nil || response.Organization.ID != org.ID {
		t.Errorf("ToResponse().Organization = %v, want %v", response.Organization, org)
	}
	if response.Active != group.Active {
		t.Errorf("ToResponse().Active = %v, want %v", response.Active, group.Active)
	}
	if response.CreatedBy != group.CreatedBy {
		t.Errorf("ToResponse().CreatedBy = %q, want %q", response.CreatedBy, group.CreatedBy)
	}
	if len(response.Users) != 1 {
		t.Errorf("ToResponse().Users length = %d, want 1", len(response.Users))
	}
}
