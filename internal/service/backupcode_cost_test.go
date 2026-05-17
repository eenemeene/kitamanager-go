package service

import "golang.org/x/crypto/bcrypt"

// Lower the backup-code bcrypt cost for the duration of the test
// binary. Production keeps DefaultCost (10); paying that on every
// MFA-enrol test pushes the suite past the pre-commit go-test wall
// (10 min/package, doubled by -race). MinCost (4) still exercises
// the full bcrypt code path — generation, format, verify — without
// the 100ms-per-code latency that DefaultCost imposes.
func init() {
	BackupCodeBcryptCost = bcrypt.MinCost
}
