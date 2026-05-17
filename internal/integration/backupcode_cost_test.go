//go:build integration

package integration

import (
	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/service"
)

// See internal/service/backupcode_cost_test.go for rationale.
// Integration MFA flows enrol factors end-to-end and would otherwise
// pay production bcrypt cost on every test.
func init() {
	service.BackupCodeBcryptCost = bcrypt.MinCost
}
