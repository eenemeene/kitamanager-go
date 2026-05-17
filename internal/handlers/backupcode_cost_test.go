package handlers

import (
	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/service"
)

// See internal/service/backupcode_cost_test.go for rationale —
// handler tests instantiate FactorService via service.NewFactorService
// and pay the same enrol-cost tax, so they need the same override.
func init() {
	service.BackupCodeBcryptCost = bcrypt.MinCost
}
