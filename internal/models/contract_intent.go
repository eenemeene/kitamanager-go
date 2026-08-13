package models

import "time"

// Contract writes, expressed as intents.
//
// The old surface had one PUT whose meaning depended on stored data: if the
// contract started before today it closed that row and created a successor
// (ignoring the request's own `from`), otherwise it edited the row in place. A
// nil field meant "inherit" on the first path and "clear" on the second, so the
// same JSON body did different things to different contracts and the caller
// could not tell which it would get. That ambiguity is where the funding-data
// losses came from: a dates-only timeline drag stripped care_type and every
// supplement off a neighbouring contract, silently recomputing months of
// funding at the base rate.
//
// Each type here names exactly one intent, so a given body always means one
// thing:
//
//	correct  — the recorded facts for this period were wrong; fix them in place
//	amend    — the facts changed as of a date; close this period, open a successor
//	end      — this period stops on a date, or stops being open-ended
//	boundary — the seam between two adjacent periods moves
//
// Correct and amend use Opt[T] for every field, so omitting a field cannot
// destroy it: absent means "leave alone" (correct) or "inherit from the
// predecessor" (amend), and an explicit null is the only way to clear.

// ChildContractCorrectRequest corrects a child contract in place.
//
// Every field is optional and omitting one leaves it untouched — this is the
// endpoint the kanban board and the age-alerts widget use to move a child
// between sections by sending `{"section_id": N}` alone.
type ChildContractCorrectRequest struct {
	From       Opt[time.Time]          `json:"from,omitzero" swaggertype:"string" format:"date-time" example:"2025-01-01"`
	To         Opt[time.Time]          `json:"to,omitzero" swaggertype:"string" format:"date-time" extensions:"x-nullable" example:"2025-12-31"`
	SectionID  Opt[uint]               `json:"section_id,omitzero" swaggertype:"integer" example:"2"`
	Properties Opt[ContractProperties] `json:"properties,omitzero" swaggertype:"object" extensions:"x-nullable"`

	// ExpectedVersion is set from the If-Match header, not from JSON.
	ExpectedVersion *int64 `json:"-" swaggerignore:"true"`
}

// ChildContractAmendRequest records a change to a child's contract effective
// from a date: the addressed contract is closed the day before, and a successor
// carrying the changes starts on EffectiveFrom.
//
// EffectiveFrom is honoured rather than forced to today, so a Bescheid that
// arrives late is one call instead of an amend followed by a boundary drag. It
// also anchors the auto-applied funding properties, which the old path resolved
// at today — wrong for any backdated change that crosses a funding period.
type ChildContractAmendRequest struct {
	EffectiveFrom time.Time               `json:"effective_from" binding:"required" format:"date-time" example:"2025-08-01"`
	To            Opt[time.Time]          `json:"to,omitzero" swaggertype:"string" format:"date-time" extensions:"x-nullable" example:"2025-12-31"`
	SectionID     Opt[uint]               `json:"section_id,omitzero" swaggertype:"integer" example:"2"`
	Properties    Opt[ContractProperties] `json:"properties,omitzero" swaggertype:"object" extensions:"x-nullable"`

	// ExpectedVersion is set from the If-Match header, not from JSON.
	ExpectedVersion *int64 `json:"-" swaggerignore:"true"`
}

// EmployeeContractCorrectRequest corrects an employee contract in place.
//
// WeeklyHours carries no `required` binding: 0 is a legitimate value (parental
// leave keeps the contract with no hours) and `required` rejects zero values,
// which is why the old update request could not express it.
type EmployeeContractCorrectRequest struct {
	From          Opt[time.Time]          `json:"from,omitzero" swaggertype:"string" format:"date-time" example:"2025-01-01"`
	To            Opt[time.Time]          `json:"to,omitzero" swaggertype:"string" format:"date-time" extensions:"x-nullable" example:"2025-12-31"`
	SectionID     Opt[uint]               `json:"section_id,omitzero" swaggertype:"integer" example:"2"`
	StaffCategory Opt[string]             `json:"staff_category,omitzero" swaggertype:"string" example:"qualified"`
	Grade         Opt[string]             `json:"grade,omitzero" swaggertype:"string" example:"S8a"`
	Step          Opt[int]                `json:"step,omitzero" swaggertype:"integer" example:"3"`
	WeeklyHours   Opt[float64]            `json:"weekly_hours,omitzero" swaggertype:"number" example:"40"`
	PayPlanID     Opt[uint]               `json:"payplan_id,omitzero" swaggertype:"integer" example:"1"`
	Properties    Opt[ContractProperties] `json:"properties,omitzero" swaggertype:"object" extensions:"x-nullable"`

	// ExpectedVersion is set from the If-Match header, not from JSON.
	ExpectedVersion *int64 `json:"-" swaggerignore:"true"`
}

// EmployeeContractAmendRequest records a change to an employee's contract
// effective from a date — a raise, a change in weekly hours, a move to another
// section. Unspecified fields inherit from the contract being amended.
//
// EffectiveFrom also anchors the pay-plan coverage check, which the old path ran
// at today: amending to a grade/step that the pay plan only gained later used to
// pass, and one that it had back then used to fail.
type EmployeeContractAmendRequest struct {
	EffectiveFrom time.Time               `json:"effective_from" binding:"required" format:"date-time" example:"2025-08-01"`
	To            Opt[time.Time]          `json:"to,omitzero" swaggertype:"string" format:"date-time" extensions:"x-nullable" example:"2025-12-31"`
	SectionID     Opt[uint]               `json:"section_id,omitzero" swaggertype:"integer" example:"2"`
	StaffCategory Opt[string]             `json:"staff_category,omitzero" swaggertype:"string" example:"qualified"`
	Grade         Opt[string]             `json:"grade,omitzero" swaggertype:"string" example:"S8a"`
	Step          Opt[int]                `json:"step,omitzero" swaggertype:"integer" example:"3"`
	WeeklyHours   Opt[float64]            `json:"weekly_hours,omitzero" swaggertype:"number" example:"40"`
	PayPlanID     Opt[uint]               `json:"payplan_id,omitzero" swaggertype:"integer" example:"1"`
	Properties    Opt[ContractProperties] `json:"properties,omitzero" swaggertype:"object" extensions:"x-nullable"`

	// ExpectedVersion is set from the If-Match header, not from JSON.
	ExpectedVersion *int64 `json:"-" swaggerignore:"true"`
}

// ContractEndRequest sets or clears a contract's end date. Shared by children
// and employees: a departure date is a date either way.
//
// `to` must be present. A null ends nothing — it reopens the contract as
// open-ended, which the old surface could only express by omitting the field,
// indistinguishable from "don't touch it".
type ContractEndRequest struct {
	To Opt[time.Time] `json:"to,omitzero" swaggertype:"string" format:"date-time" extensions:"x-nullable" example:"2025-12-31"`

	// ExpectedVersion is set from the If-Match header, not from JSON.
	ExpectedVersion *int64 `json:"-" swaggerignore:"true"`
}

// ContractBoundaryMoveRequest moves the seam between two adjacent contracts:
// the later contract starts on At and the earlier one is closed the day before.
//
// Both sides are derived server-side from one date, which is what makes this
// safe. The old client had to send both dates for both contracts and got it
// wrong twice — once clearing `to` on the neighbour (a 409 for every child with
// three or more contracts) and once wiping its properties.
type ContractBoundaryMoveRequest struct {
	EarlierID uint      `json:"earlier_id" binding:"required" example:"5"`
	LaterID   uint      `json:"later_id" binding:"required" example:"6"`
	At        time.Time `json:"at" binding:"required" format:"date-time" example:"2025-09-01"`
	// The versions of both contracts, as read. These live in the body rather than
	// in an If-Match header because this request changes two resources and is
	// addressed to the collection: one header cannot name a version per contract,
	// and If-Match on a collection URL means something else entirely.
	//
	// Required, like If-Match is on the single-contract writes: a seam move
	// rewrites dates on a contract the user did not open, so it is the operation
	// where a lost update is least likely to be noticed. Versions start at 1, so
	// `required` rejecting 0 is correct here.
	EarlierVersion int64 `json:"earlier_version" binding:"required" example:"3"`
	LaterVersion   int64 `json:"later_version" binding:"required" example:"1"`
}

// ChildContractAmendResponse returns both contracts an amend touched, so the
// caller never has to guess which id it now holds. The old PUT returned the
// successor under the addressed contract's URL, which is what forced the audit
// log to reconstruct the update/create pair after the fact.
type ChildContractAmendResponse struct {
	Closed  ChildContractResponse `json:"closed"`
	Created ChildContractResponse `json:"created"`
}

// EmployeeContractAmendResponse returns both contracts an amend touched.
type EmployeeContractAmendResponse struct {
	Closed  EmployeeContractResponse `json:"closed"`
	Created EmployeeContractResponse `json:"created"`
}

// ChildContractBoundaryResponse returns both sides of a moved seam, named
// rather than positional so a caller cannot mix them up.
type ChildContractBoundaryResponse struct {
	Earlier ChildContractResponse `json:"earlier"`
	Later   ChildContractResponse `json:"later"`
}

// EmployeeContractBoundaryResponse returns both sides of a moved seam.
type EmployeeContractBoundaryResponse struct {
	Earlier EmployeeContractResponse `json:"earlier"`
	Later   EmployeeContractResponse `json:"later"`
}

// ExpectedVersion carries the If-Match precondition from the transport layer to
// the service. Never part of the JSON body: it is a header on a single-contract
// write, and putting it in the body too would give a client two ways to say the
// same thing.
//
// nil means the caller sent `*` — "any current version" — so no comparison is
// made. The zero value (a request built in Go, e.g. by the YAML importer) also
// means no comparison, which is why these types keep working for callers that
// never see an HTTP header.

// SetExpectedVersion records the If-Match expectation for this correction.
func (r *ChildContractCorrectRequest) SetExpectedVersion(v *int64) { r.ExpectedVersion = v }

// SetExpectedVersion records the If-Match expectation for this amendment.
func (r *ChildContractAmendRequest) SetExpectedVersion(v *int64) { r.ExpectedVersion = v }

// SetExpectedVersion records the If-Match expectation for this correction.
func (r *EmployeeContractCorrectRequest) SetExpectedVersion(v *int64) { r.ExpectedVersion = v }

// SetExpectedVersion records the If-Match expectation for this amendment.
func (r *EmployeeContractAmendRequest) SetExpectedVersion(v *int64) { r.ExpectedVersion = v }

// SetExpectedVersion records the If-Match expectation for this end-date change.
func (r *ContractEndRequest) SetExpectedVersion(v *int64) { r.ExpectedVersion = v }

// GetVersion exposes the optimistic-concurrency token for ETag generation.
func (r ChildContractResponse) GetVersion() int64 { return r.Version }

// GetVersion exposes the optimistic-concurrency token for ETag generation.
func (r EmployeeContractResponse) GetVersion() int64 { return r.Version }
