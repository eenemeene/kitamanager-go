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

func TestEmployeeContract_GetPersonID(t *testing.T) {
	contract := EmployeeContract{
		EmployeeID: 42,
	}

	if got := contract.GetPersonID(); got != 42 {
		t.Errorf("EmployeeContract.GetPersonID() = %d, want 42", got)
	}
}

func TestChildContract_GetPersonID(t *testing.T) {
	contract := ChildContract{
		ChildID: 99,
	}

	if got := contract.GetPersonID(); got != 99 {
		t.Errorf("ChildContract.GetPersonID() = %d, want 99", got)
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
