package service

import "time"

type transferAction int

const (
	transferNone    transferAction = iota // no active contract
	transferUpdate                        // same-day: update section on existing contract
	transferReplace                       // close existing, create new
)

// decideSectionTransfer determines how to handle a section change on a contract.
// If the contract started today, we update it in place (same-day correction).
// Otherwise we close the old contract and create a new one.
func decideSectionTransfer(contractFrom time.Time) transferAction {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if contractFrom.Truncate(24 * time.Hour).Equal(today) {
		return transferUpdate
	}
	return transferReplace
}

// sameSectionID returns true if both section IDs are equal (including both nil).
func sameSectionID(a, b *uint) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
