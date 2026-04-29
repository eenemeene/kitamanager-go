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
//
// Uniqueness on (organization_id, name) is enforced by a functional
// index on `(organization_id, lower(trim(name)))` declared in
// migration 000017 — so "Rent", "rent", and " Rent " all collide.
// The GORM uniqueIndex tags below reference the same name as a
// breadcrumb but are NOT the source of truth (we never run
// AutoMigrate; the SQL migration is the truthful schema).
type BudgetItem struct {
	ID             uint              `gorm:"primaryKey" json:"id" example:"1"`
	OrganizationID uint              `gorm:"not null;index" json:"organization_id" example:"1"`
	Organization   *Organization     `gorm:"foreignKey:OrganizationID" json:"-"`
	Name           string            `gorm:"size:255;not null" json:"name" example:"Elternbeiträge"`
	Category       string            `gorm:"size:50;not null" json:"category" example:"income"`
	PerChild       bool              `gorm:"default:false;not null" json:"per_child" example:"true"`
	Entries        []BudgetItemEntry `gorm:"foreignKey:BudgetItemID" json:"entries,omitempty"`
	CreatedAt      time.Time         `json:"created_at" format:"date-time"`
	UpdatedAt      time.Time         `json:"updated_at" format:"date-time"`
}

// GetOrganizationID returns the organization ID for the OrgOwned interface.
func (b *BudgetItem) GetOrganizationID() uint {
	return b.OrganizationID
}

// BudgetItemEntry represents a time-bound amount for a BudgetItem.
//
// # Uniqueness / overlap
//
// Entries for the same budget item cannot overlap in time. Enforced
// at two layers: BudgetItemService.ValidateNoOverlap (friendly error
// message) AND a DB-level GIST EXCLUDE constraint added in migration
// 000016 (the truthful gate, prevents the TOCTOU window the service
// validation has under READ COMMITTED).
//
// # Mid-month semantics in financials
//
// The financials calculator (calculateFinancials) takes a first-of-
// month SNAPSHOT each iteration: an entry is counted for a given
// month only if `IsActiveOn(firstOfThatMonth)` returns true.
// Practical implication for budget-item entries:
//
//   - An entry with From=2025-01-15 contributes nothing to January
//     2025; it first counts in February.
//   - An entry that ends 2025-04-20 still counts for all of April
//     (snapshot on April 1 is within the period).
//
// Net effect: under-count in the first partial month, over-count
// in the last. For monthly-rollup planning this is acceptable; if a
// stakeholder ever asks for day-accurate pro-rating it has to be
// added explicitly. See calculateFinancials' doc-comment for the
// fuller treatment.
type BudgetItemEntry struct {
	ID           uint        `gorm:"primaryKey" json:"id" example:"1"`
	BudgetItemID uint        `gorm:"not null;index" json:"budget_item_id" example:"1"`
	BudgetItem   *BudgetItem `gorm:"foreignKey:BudgetItemID" json:"-"`
	Period                   // From, To (embedded)
	AmountCents  int         `gorm:"not null" json:"amount_cents" example:"50000"` // cents, always positive
	Notes        string      `gorm:"size:500" json:"notes,omitempty" example:"Monthly co-payment"`
	CreatedAt    time.Time   `json:"created_at" format:"date-time"`
	UpdatedAt    time.Time   `json:"updated_at" format:"date-time"`
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
	ID                uint      `json:"id" example:"1"`
	OrganizationID    uint      `json:"organization_id" example:"1"`
	Name              string    `json:"name" example:"Elternbeiträge"`
	Category          string    `json:"category" example:"income"`
	PerChild          bool      `json:"per_child" example:"true"`
	ActiveAmountCents *int      `json:"active_amount_cents,omitempty" example:"50000"`
	CreatedAt         time.Time `json:"created_at" format:"date-time"`
	UpdatedAt         time.Time `json:"updated_at" format:"date-time"`
}

// BudgetItemDetailResponse includes entries for detail view.
type BudgetItemDetailResponse struct {
	ID             uint                      `json:"id" example:"1"`
	OrganizationID uint                      `json:"organization_id" example:"1"`
	Name           string                    `json:"name" example:"Elternbeiträge"`
	Category       string                    `json:"category" example:"income"`
	PerChild       bool                      `json:"per_child" example:"true"`
	Entries        []BudgetItemEntryResponse `json:"entries"`
	CreatedAt      time.Time                 `json:"created_at" format:"date-time"`
	UpdatedAt      time.Time                 `json:"updated_at" format:"date-time"`
}

// BudgetItemEntryCreateRequest is the request body for creating a budget item entry.
type BudgetItemEntryCreateRequest struct {
	From        time.Time  `json:"from" format:"date-time" binding:"required" example:"2024-01-01T00:00:00Z"`
	To          *time.Time `json:"to,omitempty" format:"date-time" example:"2024-12-31T00:00:00Z"`
	AmountCents int        `json:"amount_cents" binding:"required,min=0" example:"50000"`
	Notes       string     `json:"notes,omitempty" binding:"max=500" example:"Monthly co-payment"`
}

// BudgetItemEntryUpdateRequest is the request body for updating a budget item entry.
type BudgetItemEntryUpdateRequest struct {
	From        time.Time  `json:"from" format:"date-time" binding:"required" example:"2024-01-01T00:00:00Z"`
	To          *time.Time `json:"to,omitempty" format:"date-time" example:"2024-12-31T00:00:00Z"`
	AmountCents int        `json:"amount_cents" binding:"required,min=0" example:"50000"`
	Notes       string     `json:"notes,omitempty" binding:"max=500" example:"Monthly co-payment"`
}

// BudgetItemEntryResponse is the response for a budget item entry.
type BudgetItemEntryResponse struct {
	ID           uint       `json:"id" example:"1"`
	BudgetItemID uint       `json:"budget_item_id" example:"1"`
	From         time.Time  `json:"from" format:"date-time" example:"2024-01-01T00:00:00Z"`
	To           *time.Time `json:"to,omitempty" format:"date-time" example:"2024-12-31T00:00:00Z"`
	AmountCents  int        `json:"amount_cents" example:"50000"`
	Notes        string     `json:"notes,omitempty" example:"Monthly co-payment"`
	CreatedAt    time.Time  `json:"created_at" format:"date-time"`
	UpdatedAt    time.Time  `json:"updated_at" format:"date-time"`
}

// ToResponse converts a BudgetItem to BudgetItemResponse, picking the
// entry active on `asOf` to populate ActiveAmountCents. Callers that
// want "active right now" pass time.Now().UTC().
//
// asOf is required (not optional) so the picked entry is never a
// hidden function of the server clock — every caller has to make a
// deliberate choice. The previous time.Now()-by-default version meant
// any caller (list view, future as-of-date filter, summary view) got
// "active right now" regardless of the user's intent.
//
// Determinism note: migration 000016's GIST exclusion constraint
// guarantees at most one entry is active on any given date for a
// single budget item, so the loop's first match is the only match.
// If that constraint is ever dropped, callers should also order
// b.Entries by from_date DESC so the "newest period that covers
// asOf" wins reproducibly — see the loader Order() clauses.
func (b *BudgetItem) ToResponse(asOf time.Time) BudgetItemResponse {
	resp := BudgetItemResponse{
		ID:             b.ID,
		OrganizationID: b.OrganizationID,
		Name:           b.Name,
		Category:       b.Category,
		PerChild:       b.PerChild,
		CreatedAt:      b.CreatedAt,
		UpdatedAt:      b.UpdatedAt,
	}

	for i := range b.Entries {
		if b.Entries[i].IsActiveOn(asOf) {
			amount := b.Entries[i].AmountCents
			resp.ActiveAmountCents = &amount
			break
		}
	}

	return resp
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
