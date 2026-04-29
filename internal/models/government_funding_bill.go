package models

import (
	"time"

	"gorm.io/gorm"
)

// ============================================================
// GORM models (stored in database)
// ============================================================

// RowType values for GovernmentFundingBillPayment.
const (
	RowTypeRegular    = "regular"    // Normal monthly billing (Typ="A" in Excel)
	RowTypeCorrection = "correction" // Rate correction for a prior month (Typ="K" in Excel)
)

// GovernmentFundingBillPayment represents a single financial line item for a child in a bill.
type GovernmentFundingBillPayment struct {
	ID       uint   `gorm:"primaryKey" json:"-"`
	ChildID  uint   `gorm:"not null;index" json:"-"`
	Key      string `gorm:"size:100;not null" json:"key" example:"care_type"`
	Value    string `gorm:"size:255;not null" json:"value" example:"ganztag"`
	Amount   int    `gorm:"not null" json:"amount" example:"166847"`
	RowIndex int    `gorm:"not null;default:0" json:"-"`
	RowType  string `gorm:"size:20;not null;default:'regular'" json:"row_type" example:"regular"` // "regular" or "correction"
}

// BeforeCreate sets default RowType to "regular" when not explicitly set.
func (p *GovernmentFundingBillPayment) BeforeCreate(tx *gorm.DB) error {
	if p.RowType == "" {
		p.RowType = RowTypeRegular
	}
	return nil
}

// GovernmentFundingBillChild represents one child row in a bill period.
type GovernmentFundingBillChild struct {
	ID            uint                           `gorm:"primaryKey" json:"-"`
	PeriodID      uint                           `gorm:"not null;index" json:"-"`
	VoucherNumber string                         `gorm:"size:20;not null" json:"voucher_number" example:"GB-12345678901-02"`
	ChildName     string                         `gorm:"size:255;not null" json:"child_name" example:"Mustermann, Max"`
	BirthDate     string                         `gorm:"size:10;not null" json:"birth_date" example:"01.20"`
	District      int64                          `gorm:"not null" json:"district" example:"1"`
	Payments      []GovernmentFundingBillPayment `gorm:"foreignKey:ChildID;constraint:OnDelete:CASCADE" json:"payments"`
}

// GovernmentFundingBillPeriod represents a single uploaded government funding bill.
type GovernmentFundingBillPeriod struct {
	ID                uint   `gorm:"primaryKey" json:"id" example:"1"`
	OrganizationID    uint   `gorm:"not null;index" json:"organization_id" example:"1"`
	Period                   // from_date, to_date
	FileName          string `gorm:"size:255;not null" json:"file_name" example:"Abrechnung_11-25.xlsx"`
	FileSha256        string `gorm:"size:64;not null" json:"file_sha256" example:"a1b2c3d4..."`
	FacilityName      string `gorm:"size:255;not null" json:"facility_name" example:"Kita Sonnenschein"`
	FacilityTotal     int    `gorm:"not null" json:"facility_total" example:"500000"`
	ContractBooking   int    `gorm:"not null" json:"contract_booking" example:"480000"`
	CorrectionBooking int    `gorm:"not null" json:"correction_booking" example:"20000"`
	// CreatedBy became nullable in migration 000014: the FK to
	// users(id) is now ON DELETE SET NULL so deleting the uploader
	// doesn't block user deletion and doesn't destroy the bill
	// record. Fresh inserts always have a non-nil value.
	CreatedBy *uint                        `json:"created_by,omitempty" example:"1"`
	CreatedAt time.Time                    `json:"created_at" format:"date-time"`
	UpdatedAt time.Time                    `json:"updated_at" format:"date-time"`
	Children  []GovernmentFundingBillChild `gorm:"foreignKey:PeriodID;constraint:OnDelete:CASCADE" json:"children,omitempty"`
}

// ============================================================
// Response DTOs (enriched at read time, not stored)
// ============================================================

// GovernmentFundingBillAmount represents a single financial line item in a bill response.
type GovernmentFundingBillAmount struct {
	Key    string `json:"key" example:"care_type"`
	Value  string `json:"value" example:"ganztag"`
	Amount int    `json:"amount" example:"166847"`
}

// GovernmentFundingBillRowResponse represents one billing row (Excel line) with its amounts.
type GovernmentFundingBillRowResponse struct {
	TotalRowAmount int                           `json:"total_row_amount" example:"141331"`
	Amounts        []GovernmentFundingBillAmount `json:"amounts"`
}

// GovernmentFundingBillChildResponse represents one child from a bill, enriched with match info.
type GovernmentFundingBillChildResponse struct {
	VoucherNumber string                             `json:"voucher_number" example:"GB-12345678901-02"`
	ChildName     string                             `json:"child_name" example:"Mustermann, Max"`
	BirthDate     string                             `json:"birth_date" example:"01.20"`
	District      int64                              `json:"district" example:"1"`
	TotalAmount   int                                `json:"total_amount" example:"166847"`
	Rows          []GovernmentFundingBillRowResponse `json:"rows"`
	ChildID       *uint                              `json:"child_id,omitempty" example:"42"`
	ContractID    *uint                              `json:"contract_id,omitempty" example:"99"`
	Matched       bool                               `json:"matched" example:"true"`
}

// GovernmentFundingBillPeriodResponse is the full detail response for a single bill period.
type GovernmentFundingBillPeriodResponse struct {
	ID                uint                                 `json:"id" example:"1"`
	OrganizationID    uint                                 `json:"organization_id" example:"1"`
	From              string                               `json:"from" example:"2025-11-01"`
	To                string                               `json:"to" example:"2025-11-30"`
	FileName          string                               `json:"file_name" example:"Abrechnung_11-25.xlsx"`
	FileSha256        string                               `json:"file_sha256" example:"a1b2c3d4..."`
	FacilityName      string                               `json:"facility_name" example:"Kita Sonnenschein"`
	FacilityTotal     int                                  `json:"facility_total" example:"500000"`
	ContractBooking   int                                  `json:"contract_booking" example:"480000"`
	CorrectionBooking int                                  `json:"correction_booking" example:"20000"`
	ChildrenCount     int                                  `json:"children_count" example:"25"`
	MatchedCount      int                                  `json:"matched_count" example:"23"`
	UnmatchedCount    int                                  `json:"unmatched_count" example:"2"`
	Surcharges        []GovernmentFundingBillAmount        `json:"surcharges"`
	Children          []GovernmentFundingBillChildResponse `json:"children"`
	// CreatedBy is nil when the uploader has since been deleted
	// (migration 000014 ON DELETE SET NULL).
	CreatedBy *uint     `json:"created_by,omitempty" example:"1"`
	CreatedAt time.Time `json:"created_at" format:"date-time"`
}

// GovernmentFundingBillPeriodListResponse is the summary response for list view.
type GovernmentFundingBillPeriodListResponse struct {
	ID                uint      `json:"id" example:"1"`
	From              string    `json:"from" example:"2025-11-01"`
	To                string    `json:"to" example:"2025-11-30"`
	FileName          string    `json:"file_name" example:"Abrechnung_11-25.xlsx"`
	FacilityName      string    `json:"facility_name" example:"Kita Sonnenschein"`
	FacilityTotal     int       `json:"facility_total" example:"500000"`
	ContractBooking   int       `json:"contract_booking" example:"480000"`
	CorrectionBooking int       `json:"correction_booking" example:"20000"`
	ChildrenCount     int       `json:"children_count" example:"25"`
	CreatedAt         time.Time `json:"created_at" format:"date-time"`
}

// MismatchType classifies property discrepancies between bill and contract.
type MismatchType string

const (
	MismatchNone       MismatchType = ""           // property matches or is present on both sides with same value
	MismatchMissing    MismatchType = "missing"    // property in contract/calc but NOT in bill
	MismatchAdditional MismatchType = "additional" // property in bill but NOT in contract/calc
	MismatchDifferent  MismatchType = "different"  // same key exists in both but with different values
)

// FundingComparisonAmount represents one property's amounts in the comparison.
type FundingComparisonAmount struct {
	Key        string       `json:"key" example:"care_type"`
	Value      string       `json:"value" example:"ganztag"`
	Label      string       `json:"label" example:"Ganztag"`
	BillAmount *int         `json:"bill_amount" example:"166847"`       // nil if not in bill
	CalcAmount *int         `json:"calculated_amount" example:"166847"` // nil if not calculable
	Difference int          `json:"difference" example:"0"`             // bill - calc (0 if either nil)
	Mismatch   MismatchType `json:"mismatch,omitempty"`                 // ""|"missing"|"additional"|"different"
}

// BillAppearance represents a bill that a child appeared in.
type BillAppearance struct {
	BillID       uint   `json:"bill_id" example:"1"`
	BillFrom     string `json:"bill_from" example:"2025-11-01"`
	FacilityName string `json:"facility_name" example:"Kita Sonnenschein"`
}

// FundingComparisonChild represents the comparison for one child.
type FundingComparisonChild struct {
	VoucherNumber   string                    `json:"voucher_number" example:"GB-12345678901-02"`
	ChildName       string                    `json:"child_name" example:"Mustermann, Max"`
	BirthDate       string                    `json:"birth_date,omitempty" example:"01.20"`
	ChildID         *uint                     `json:"child_id,omitempty" example:"42"`
	Age             *int                      `json:"age,omitempty" example:"3"`
	BillTotal       int                       `json:"bill_total" example:"166847"`
	CorrectionTotal int                       `json:"correction_total" example:"0"`
	CalcTotal       *int                      `json:"calculated_total,omitempty" example:"166847"`
	Difference      *int                      `json:"difference,omitempty" example:"0"`
	Status          string                    `json:"status" example:"match"` // match|difference|bill_only|calc_only
	Properties      []FundingComparisonAmount `json:"properties"`
	BillAppearances []BillAppearance          `json:"bill_appearances,omitempty"`
	ContractFrom    *string                   `json:"contract_from,omitempty" example:"2024-01-01"`
	ContractTo      *string                   `json:"contract_to,omitempty" example:"2025-12-31"`
}

// FundingComparisonResponse is the top-level comparison result.
type FundingComparisonResponse struct {
	BillID          uint                     `json:"bill_id" example:"1"`
	BillFrom        string                   `json:"bill_from" example:"2025-11-01"`
	BillTo          string                   `json:"bill_to" example:"2025-11-30"`
	FacilityName    string                   `json:"facility_name" example:"Kita Sonnenschein"`
	BillTotal       int                      `json:"bill_total" example:"500000"`
	CorrectionTotal int                      `json:"correction_total" example:"0"`
	CalcTotal       int                      `json:"calculated_total" example:"498000"`
	Difference      int                      `json:"difference" example:"2000"`
	ChildrenCount   int                      `json:"children_count" example:"25"`
	MatchCount      int                      `json:"match_count" example:"20"`
	DifferenceCount int                      `json:"difference_count" example:"3"`
	BillOnlyCount   int                      `json:"bill_only_count" example:"1"`
	BillOnlyAmount  int                      `json:"bill_only_amount" example:"150000"`
	CalcOnlyCount   int                      `json:"calc_only_count" example:"1"`
	CalcOnlyAmount  int                      `json:"calc_only_amount" example:"80000"`
	Children        []FundingComparisonChild `json:"children"`
}

// ============================================================
// Wrapped comparison response (always returned by /compare)
// ============================================================

// FundingComparisonCategorySummary aggregates one category of difference across all children and months.
type FundingComparisonCategorySummary struct {
	Category    string `json:"category" example:"rate_difference"`
	TotalAmount int    `json:"total_amount" example:"-60100"`
	ChildCount  int    `json:"child_count" example:"40"`
	Actionable  bool   `json:"actionable" example:"false"`
}

// FundingComparisonIssueSummary describes one per-child issue deduplicated across months.
type FundingComparisonIssueSummary struct {
	VoucherNumber  string `json:"voucher_number" example:"GB-12345678901-02"`
	ChildName      string `json:"child_name" example:"Bagus, Nathan Albert"`
	ChildID        *uint  `json:"child_id,omitempty" example:"42"`
	Category       string `json:"category" example:"property_mismatch"`
	IssueType      string `json:"issue_type,omitempty" example:"missing"`
	Description    string `json:"description" example:"integration:integration b — in contract but not billed"`
	PropertyKey    string `json:"property_key,omitempty" example:"integration"`
	CalcValue      string `json:"calc_value,omitempty" example:"integration b"`
	BillValue      string `json:"bill_value,omitempty" example:""`
	AmountPerMonth int    `json:"amount_per_month" example:"-33064"`
	MonthCount     int    `json:"month_count" example:"12"`
	TotalAmount    int    `json:"total_amount" example:"-396768"`
	Actionable     bool   `json:"actionable" example:"true"`
}

// FundingComparisonSummary provides aggregate analysis across all comparisons.
type FundingComparisonSummary struct {
	TotalBilled      int                                `json:"total_billed" example:"712387"`
	TotalCalculated  int                                `json:"total_calculated" example:"783227"`
	TotalDifference  int                                `json:"total_difference" example:"-70840"`
	TotalCorrections int                                `json:"total_corrections" example:"37114"`
	MonthCount       int                                `json:"month_count" example:"12"`
	Categories       []FundingComparisonCategorySummary `json:"categories"`
	Issues           []FundingComparisonIssueSummary    `json:"issues"`
}

// FundingComparisonWrappedResponse wraps per-bill comparisons with aggregate summary.
type FundingComparisonWrappedResponse struct {
	Comparisons []FundingComparisonResponse `json:"comparisons"`
	Summary     FundingComparisonSummary    `json:"summary"`
}

// ============================================================
// Per-child billing history DTOs
// ============================================================

// ChildBillingHistoryEntryResponse represents one month's billing data for a child.
type ChildBillingHistoryEntryResponse struct {
	BillID            uint                      `json:"bill_id" example:"1"`
	BillFrom          string                    `json:"bill_from" example:"2025-11-01"`
	BillTo            string                    `json:"bill_to" example:"2025-11-30"`
	FacilityName      string                    `json:"facility_name" example:"Kita Sonnenschein"`
	VoucherNumber     string                    `json:"voucher_number" example:"GB-12345678901-02"`
	ChildName         string                    `json:"child_name" example:"Mustermann, Max"`
	BirthDate         string                    `json:"birth_date" example:"01.20"`
	Age               *int                      `json:"age,omitempty" example:"3"`
	BillTotal         int                       `json:"bill_total" example:"166847"`
	CorrectionTotal   int                       `json:"correction_total" example:"0"`
	CalcTotal         *int                      `json:"calculated_total,omitempty" example:"166847"`
	Difference        *int                      `json:"difference,omitempty" example:"0"`
	Status            string                    `json:"status" example:"match"` // match|difference|bill_only|no_contract|no_funding_config
	RunningDifference int                       `json:"running_difference" example:"-102"`
	Properties        []FundingComparisonAmount `json:"properties"`
	ContractID        *uint                     `json:"contract_id,omitempty" example:"99"`
}

// ChildBillingHistoryResponse is the top-level billing history for a child.
type ChildBillingHistoryResponse struct {
	ChildID         uint                               `json:"child_id" example:"42"`
	ChildName       string                             `json:"child_name" example:"Max Mustermann"`
	Birthdate       string                             `json:"birthdate" example:"2020-03-10"`
	VoucherNumbers  []string                           `json:"voucher_numbers"`
	TotalBilled     int                                `json:"total_billed" example:"1500000"`
	TotalCalculated int                                `json:"total_calculated" example:"1500000"`
	TotalDifference int                                `json:"total_difference" example:"0"`
	Entries         []ChildBillingHistoryEntryResponse `json:"entries"`
}

// GovernmentFundingBillChildWithPeriod pairs a bill child with its parent period metadata.
// Used internally between store and service layers, not exposed via API.
type GovernmentFundingBillChildWithPeriod struct {
	BillPeriodID uint
	BillFrom     time.Time
	BillTo       *time.Time
	FacilityName string
	Child        GovernmentFundingBillChild
}

// ============================================================
// Bulk billing summary DTOs (org-wide, for children list)
// ============================================================

// VoucherBilledTotal holds SQL-aggregated billed totals per voucher number.
type VoucherBilledTotal struct {
	VoucherNumber string `gorm:"column:voucher_number"`
	TotalBilled   int    `gorm:"column:total_billed"`
	BillCount     int    `gorm:"column:bill_count"`
}

// BillDateVoucher is a lightweight struct for computing expected amounts: just voucher + bill date.
type BillDateVoucher struct {
	VoucherNumber string    `gorm:"column:voucher_number"`
	BillFrom      time.Time `format:"date-time" gorm:"column:bill_from"`
}

// ChildBillingSummaryEntry represents the billing summary for one child.
type ChildBillingSummaryEntry struct {
	ChildID         uint `json:"child_id" example:"42"`
	TotalBilled     int  `json:"total_billed" example:"1500000"`
	TotalCalculated int  `json:"total_calculated" example:"1500000"`
	TotalDifference int  `json:"total_difference" example:"0"`
	BillCount       int  `json:"bill_count" example:"12"`
	ContractMonths  int  `json:"contract_months" example:"18"`
}

// ChildrenBillingSummaryResponse is the response for the bulk billing summary endpoint.
type ChildrenBillingSummaryResponse struct {
	Children []ChildBillingSummaryEntry `json:"children"`
}

// VoucherSuggestion represents a fuzzy-matched voucher suggestion from an unmatched bill child.
type VoucherSuggestion struct {
	VoucherNumber string  `json:"voucher_number" example:"GB-12345678901-02"`
	BillChildName string  `json:"bill_child_name" example:"Mustermann, Max"`
	BillFirstName string  `json:"bill_first_name" example:"Max"`
	BillLastName  string  `json:"bill_last_name" example:"Mustermann"`
	BillBirthDate string  `json:"bill_birth_date" example:"03.20"`
	Similarity    float64 `json:"similarity" example:"0.85"`
	BillFrom      string  `json:"bill_from" example:"2025-11-01"`
}

// ChildWithoutVoucherResponse extends ChildResponse with fuzzy match suggestions.
type ChildWithoutVoucherResponse struct {
	ChildResponse
	Suggestions []VoucherSuggestion `json:"suggestions,omitempty"`
}

// GovernmentFundingBillResponse is the full response for the ISBJ upload endpoint (backwards compatible).
type GovernmentFundingBillResponse struct {
	ID                uint                                 `json:"id" example:"1"`
	FacilityName      string                               `json:"facility_name" example:"Kita Sonnenschein"`
	FacilityTotal     int                                  `json:"facility_total" example:"500000"`
	ContractBooking   int                                  `json:"contract_booking" example:"480000"`
	CorrectionBooking int                                  `json:"correction_booking" example:"20000"`
	ChildrenCount     int                                  `json:"children_count" example:"25"`
	MatchedCount      int                                  `json:"matched_count" example:"23"`
	UnmatchedCount    int                                  `json:"unmatched_count" example:"2"`
	Surcharges        []GovernmentFundingBillAmount        `json:"surcharges"`
	Children          []GovernmentFundingBillChildResponse `json:"children"`
}
