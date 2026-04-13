package models

import "time"

// ChildVoucher maps a Gutschein number to a child. A child can have multiple
// vouchers over time (suffix renewals, full number changes). Each voucher
// is globally unique — a given Gutschein belongs to exactly one child.
type ChildVoucher struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ChildID       uint      `gorm:"not null;index" json:"child_id"`
	VoucherNumber string    `gorm:"size:17;not null;uniqueIndex" json:"voucher_number" example:"GB-12345678901-02"`
	FirstSeen     time.Time `gorm:"type:date;not null" json:"first_seen"`
	CreatedAt     time.Time `json:"created_at"`
}
