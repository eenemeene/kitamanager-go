package models

import "time"

// BudgetItemCategory represents the category type for a budget item.
type BudgetItemCategory string

const (
	BudgetItemCategoryIncome  BudgetItemCategory = "income"
	BudgetItemCategoryExpense BudgetItemCategory = "expense"
)

// ValidBudgetItemCategory checks if a category string is valid.
func ValidBudgetItemCategory(category string) bool {
	switch BudgetItemCategory(category) {
	case BudgetItemCategoryIncome, BudgetItemCategoryExpense:
		return true
	}
	return false
}

// BudgetItem represents an income or expense category for an organization (e.g., "Rent", "Elternbeiträge", "Essensgeld").
type BudgetItem struct {
	ID             uint              `gorm:"primaryKey" json:"id" example:"1"`
	OrganizationID uint              `gorm:"not null;index;uniqueIndex:idx_budget_item_org_name" json:"organization_id" example:"1"`
	Organization   *Organization     `gorm:"foreignKey:OrganizationID" json:"-"`
	Name           string            `gorm:"size:255;not null;uniqueIndex:idx_budget_item_org_name" json:"name" example:"Elternbeiträge"`
	Category       string            `gorm:"size:50;not null" json:"category" example:"income"`
	PerChild       bool              `gorm:"default:false;not null" json:"per_child" example:"true"`
	Entries        []BudgetItemEntry `gorm:"foreignKey:BudgetItemID" json:"entries,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// GetOrganizationID returns the organization ID for the OrgOwned interface.
func (b *BudgetItem) GetOrganizationID() uint {
	return b.OrganizationID
}

// BudgetItemEntry represents a time-bound amount for a BudgetItem.
// Entries for the same budget item cannot overlap in time.
type BudgetItemEntry struct {
	ID           uint        `gorm:"primaryKey" json:"id" example:"1"`
	BudgetItemID uint        `gorm:"not null;index" json:"budget_item_id" example:"1"`
	BudgetItem   *BudgetItem `gorm:"foreignKey:BudgetItemID" json:"-"`
	Period                   // From, To (embedded)
	AmountCents  int         `gorm:"not null" json:"amount_cents" example:"50000"` // cents, always positive
	Notes        string      `gorm:"size:500" json:"notes,omitempty" example:"Monthly co-payment"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// GetOwnerID returns the budget item ID for the PeriodRecord interface.
func (e BudgetItemEntry) GetOwnerID() uint {
	return e.BudgetItemID
}

// BudgetItemCreateRequest is the request body for creating a budget item.
type BudgetItemCreateRequest struct {
	Name     string `json:"name" binding:"required" example:"Elternbeiträge"`
	Category string `json:"category" binding:"required" example:"income"`
	PerChild bool   `json:"per_child" example:"true"`
}

// BudgetItemUpdateRequest is the request body for updating a budget item.
type BudgetItemUpdateRequest struct {
	Name     *string `json:"name" binding:"omitempty,max=255" example:"Elternbeiträge"`
	Category *string `json:"category" binding:"omitempty" example:"income"`
	PerChild *bool   `json:"per_child" example:"true"`
}

// BudgetItemResponse is the response for a budget item.
type BudgetItemResponse struct {
	ID             uint      `json:"id" example:"1"`
	OrganizationID uint      `json:"organization_id" example:"1"`
	Name           string    `json:"name" example:"Elternbeiträge"`
	Category       string    `json:"category" example:"income"`
	PerChild       bool      `json:"per_child" example:"true"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// BudgetItemDetailResponse includes entries for detail view.
type BudgetItemDetailResponse struct {
	ID             uint                      `json:"id" example:"1"`
	OrganizationID uint                      `json:"organization_id" example:"1"`
	Name           string                    `json:"name" example:"Elternbeiträge"`
	Category       string                    `json:"category" example:"income"`
	PerChild       bool                      `json:"per_child" example:"true"`
	Entries        []BudgetItemEntryResponse `json:"entries"`
	CreatedAt      time.Time                 `json:"created_at"`
	UpdatedAt      time.Time                 `json:"updated_at"`
}

// BudgetItemEntryCreateRequest is the request body for creating a budget item entry.
type BudgetItemEntryCreateRequest struct {
	From        time.Time  `json:"from" binding:"required" example:"2024-01-01T00:00:00Z"`
	To          *time.Time `json:"to,omitempty" example:"2024-12-31T00:00:00Z"`
	AmountCents int        `json:"amount_cents" binding:"required,min=0" example:"50000"`
	Notes       string     `json:"notes,omitempty" binding:"max=500" example:"Monthly co-payment"`
}

// BudgetItemEntryUpdateRequest is the request body for updating a budget item entry.
type BudgetItemEntryUpdateRequest struct {
	From        time.Time  `json:"from" binding:"required" example:"2024-01-01T00:00:00Z"`
	To          *time.Time `json:"to,omitempty" example:"2024-12-31T00:00:00Z"`
	AmountCents int        `json:"amount_cents" binding:"required,min=0" example:"50000"`
	Notes       string     `json:"notes,omitempty" binding:"max=500" example:"Monthly co-payment"`
}

// BudgetItemEntryResponse is the response for a budget item entry.
type BudgetItemEntryResponse struct {
	ID           uint       `json:"id" example:"1"`
	BudgetItemID uint       `json:"budget_item_id" example:"1"`
	From         time.Time  `json:"from" example:"2024-01-01T00:00:00Z"`
	To           *time.Time `json:"to,omitempty" example:"2024-12-31T00:00:00Z"`
	AmountCents  int        `json:"amount_cents" example:"50000"`
	Notes        string     `json:"notes,omitempty" example:"Monthly co-payment"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ToResponse converts a BudgetItem to BudgetItemResponse.
func (b *BudgetItem) ToResponse() BudgetItemResponse {
	return BudgetItemResponse{
		ID:             b.ID,
		OrganizationID: b.OrganizationID,
		Name:           b.Name,
		Category:       b.Category,
		PerChild:       b.PerChild,
		CreatedAt:      b.CreatedAt,
		UpdatedAt:      b.UpdatedAt,
	}
}

// ToDetailResponse converts a BudgetItem to BudgetItemDetailResponse.
func (b *BudgetItem) ToDetailResponse() BudgetItemDetailResponse {
	entries := make([]BudgetItemEntryResponse, len(b.Entries))
	for i, entry := range b.Entries {
		entries[i] = entry.ToResponse()
	}
	return BudgetItemDetailResponse{
		ID:             b.ID,
		OrganizationID: b.OrganizationID,
		Name:           b.Name,
		Category:       b.Category,
		PerChild:       b.PerChild,
		Entries:        entries,
		CreatedAt:      b.CreatedAt,
		UpdatedAt:      b.UpdatedAt,
	}
}

// ToResponse converts a BudgetItemEntry to BudgetItemEntryResponse.
func (e *BudgetItemEntry) ToResponse() BudgetItemEntryResponse {
	return BudgetItemEntryResponse{
		ID:           e.ID,
		BudgetItemID: e.BudgetItemID,
		From:         e.From,
		To:           e.To,
		AmountCents:  e.AmountCents,
		Notes:        e.Notes,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}
