package store

import (
	"strings"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// voucherBases converts a slice of full voucher numbers to their unique base forms.
func voucherBases(vouchers []string) []string {
	seen := make(map[string]bool, len(vouchers))
	bases := make([]string, 0, len(vouchers))
	for _, v := range vouchers {
		b := models.VoucherBase(v)
		if !seen[b] {
			seen[b] = true
			bases = append(bases, b)
		}
	}
	return bases
}

// scopeVoucherBaseMatch adds a WHERE clause that matches a voucher column
// against a list of voucher numbers using base (prefix) matching.
// For standard vouchers (GB-XXXXXXXXXXX-XX), this matches all suffix revisions.
// For non-standard vouchers, falls back to exact match.
func scopeVoucherBaseMatch(column string, voucherNumbers []string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		bases := voucherBases(voucherNumbers)
		if len(bases) == 0 {
			return db
		}
		if len(bases) == 1 {
			return db.Where(column+" LIKE ?", bases[0]+"%")
		}
		conditions := make([]string, len(bases))
		args := make([]any, len(bases))
		for i, b := range bases {
			conditions[i] = column + " LIKE ?"
			args[i] = b + "%"
		}
		return db.Where("("+strings.Join(conditions, " OR ")+")", args...)
	}
}
