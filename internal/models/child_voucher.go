package models

import "time"

// ChildVoucher maps a Gutschein number to a child. A child can have multiple
// vouchers over time (suffix renewals, full number changes). Each voucher
// is globally unique — a given Gutschein belongs to exactly one child.
type ChildVoucher struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ChildID       uint      `gorm:"not null;index" json:"child_id"`
	VoucherNumber string    `gorm:"size:17;not null;uniqueIndex" json:"voucher_number" example:"GB-12345678901-02"`
	FirstSeen     time.Time `gorm:"type:date;not null" json:"first_seen" format:"date-time"`
	CreatedAt     time.Time `json:"created_at" format:"date-time"`
}

// ChildVoucherCreateRequest is the request body for assigning a voucher to a child.
//
// The `voucher` validator (registered in cmd/api/main.go via
// RegisterCustomValidators) enforces the Berlin Gutschein format
// `GB-DDDDDDDDDDD-NN` — 11 carrier-account digits and a 2-digit suffix.
// Closes audit finding I-M-4: previously the only validation was at
// the database `size:17` boundary, so a wildly malformed value would
// reach the audit log before being rejected.
type ChildVoucherCreateRequest struct {
	VoucherNumber string `json:"voucher_number" binding:"required,voucher" example:"GB-12345678901-02"`
}
