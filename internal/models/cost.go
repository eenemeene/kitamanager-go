package models

import "time"

// Cost represents a recurring cost category for an organization (e.g., "Rent", "Insurance").
type Cost struct {
	ID             uint          `gorm:"primaryKey" json:"id" example:"1"`
	OrganizationID uint          `gorm:"not null;index;uniqueIndex:idx_cost_org_name" json:"organization_id" example:"1"`
	Organization   *Organization `gorm:"foreignKey:OrganizationID" json:"-"`
	Name           string        `gorm:"size:255;not null;uniqueIndex:idx_cost_org_name" json:"name" example:"Rent"`
	Entries        []CostEntry   `gorm:"foreignKey:CostID" json:"entries,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// GetOrganizationID returns the organization ID for the OrgOwned interface.
func (c *Cost) GetOrganizationID() uint {
	return c.OrganizationID
}

// CostEntry represents a time-bound cost amount for a Cost category.
// Entries for the same cost cannot overlap in time.
type CostEntry struct {
	ID          uint      `gorm:"primaryKey" json:"id" example:"1"`
	CostID      uint      `gorm:"not null;index" json:"cost_id" example:"1"`
	Cost        *Cost     `gorm:"foreignKey:CostID" json:"-"`
	Period                // From, To (embedded)
	AmountCents int       `gorm:"not null" json:"amount_cents" example:"150000"` // cents
	Notes       string    `gorm:"size:500" json:"notes,omitempty" example:"Monthly office rent"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GetOwnerID returns the cost ID for the PeriodRecord interface.
func (e CostEntry) GetOwnerID() uint {
	return e.CostID
}

// CostCreateRequest is the request body for creating a cost.
type CostCreateRequest struct {
	Name string `json:"name" binding:"required" example:"Rent"`
}

// CostUpdateRequest is the request body for updating a cost.
type CostUpdateRequest struct {
	Name string `json:"name" binding:"required" example:"Rent"`
}

// CostResponse is the response for a cost.
type CostResponse struct {
	ID             uint      `json:"id" example:"1"`
	OrganizationID uint      `json:"organization_id" example:"1"`
	Name           string    `json:"name" example:"Rent"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CostDetailResponse includes entries for detail view.
type CostDetailResponse struct {
	ID             uint                `json:"id" example:"1"`
	OrganizationID uint                `json:"organization_id" example:"1"`
	Name           string              `json:"name" example:"Rent"`
	Entries        []CostEntryResponse `json:"entries"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

// CostEntryCreateRequest is the request body for creating a cost entry.
type CostEntryCreateRequest struct {
	From        time.Time  `json:"from" binding:"required" example:"2024-01-01T00:00:00Z"`
	To          *time.Time `json:"to,omitempty" example:"2024-12-31T00:00:00Z"`
	AmountCents int        `json:"amount_cents" binding:"required,min=0" example:"150000"`
	Notes       string     `json:"notes,omitempty" binding:"max=500" example:"Monthly office rent"`
}

// CostEntryUpdateRequest is the request body for updating a cost entry.
type CostEntryUpdateRequest struct {
	From        time.Time  `json:"from" binding:"required" example:"2024-01-01T00:00:00Z"`
	To          *time.Time `json:"to,omitempty" example:"2024-12-31T00:00:00Z"`
	AmountCents int        `json:"amount_cents" binding:"required,min=0" example:"150000"`
	Notes       string     `json:"notes,omitempty" binding:"max=500" example:"Monthly office rent"`
}

// CostEntryResponse is the response for a cost entry.
type CostEntryResponse struct {
	ID          uint       `json:"id" example:"1"`
	CostID      uint       `json:"cost_id" example:"1"`
	From        time.Time  `json:"from" example:"2024-01-01T00:00:00Z"`
	To          *time.Time `json:"to,omitempty" example:"2024-12-31T00:00:00Z"`
	AmountCents int        `json:"amount_cents" example:"150000"`
	Notes       string     `json:"notes,omitempty" example:"Monthly office rent"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ToResponse converts a Cost to CostResponse.
func (c *Cost) ToResponse() CostResponse {
	return CostResponse{
		ID:             c.ID,
		OrganizationID: c.OrganizationID,
		Name:           c.Name,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}

// ToDetailResponse converts a Cost to CostDetailResponse.
func (c *Cost) ToDetailResponse() CostDetailResponse {
	entries := make([]CostEntryResponse, len(c.Entries))
	for i, entry := range c.Entries {
		entries[i] = entry.ToResponse()
	}
	return CostDetailResponse{
		ID:             c.ID,
		OrganizationID: c.OrganizationID,
		Name:           c.Name,
		Entries:        entries,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}

// ToResponse converts a CostEntry to CostEntryResponse.
func (e *CostEntry) ToResponse() CostEntryResponse {
	return CostEntryResponse{
		ID:          e.ID,
		CostID:      e.CostID,
		From:        e.From,
		To:          e.To,
		AmountCents: e.AmountCents,
		Notes:       e.Notes,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}
