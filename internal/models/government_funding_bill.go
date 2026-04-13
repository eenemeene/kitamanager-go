package models

import "time"

// ============================================================
// GORM models (stored in database)
// ============================================================

// GovernmentFundingBillPayment represents a single financial line item for a child in a bill.
type GovernmentFundingBillPayment struct {
	ID       uint   `gorm:"primaryKey" json:"-"`
	ChildID  uint   `gorm:"not null;index" json:"-"`
	Key      string `gorm:"size:100;not null" json:"key" example:"care_type"`
	Value    string `gorm:"size:255;not null" json:"value" example:"ganztag"`
	Amount   int    `gorm:"not null" json:"amount" example:"166847"`
	RowIndex int    `gorm:"not null;default:0" json:"-"`
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
	ID                uint                         `gorm:"primaryKey" json:"id" example:"1"`
	OrganizationID    uint                         `gorm:"not null;index" json:"organization_id" example:"1"`
	Period                                         // from_date, to_date
	FileName          string                       `gorm:"size:255;not null" json:"file_name" example:"Abrechnung_11-25.xlsx"`
	FileSha256        string                       `gorm:"size:64;not null" json:"file_sha256" example:"a1b2c3d4..."`
	FacilityName      string                       `gorm:"size:255;not null" json:"facility_name" example:"Kita Sonnenschein"`
	FacilityTotal     int                          `gorm:"not null" json:"facility_total" example:"500000"`
	ContractBooking   int                          `gorm:"not null" json:"contract_booking" example:"480000"`
	CorrectionBooking int                          `gorm:"not null" json:"correction_booking" example:"20000"`
	CreatedBy         uint                         `gorm:"not null" json:"created_by" example:"1"`
	CreatedAt         time.Time                    `json:"created_at"`
	UpdatedAt         time.Time                    `json:"updated_at"`
	Children          []GovernmentFundingBillChild `gorm:"foreignKey:PeriodID;constraint:OnDelete:CASCADE" json:"children,omitempty"`
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
	CreatedBy         uint                                 `json:"created_by" example:"1"`
	CreatedAt         time.Time                            `json:"created_at"`
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
	CreatedAt         time.Time `json:"created_at"`
}

// FundingComparisonAmount represents one property's amounts in the comparison.
type FundingComparisonAmount struct {
	Key        string `json:"key" example:"care_type"`
	Value      string `json:"value" example:"ganztag"`
	Label      string `json:"label" example:"Ganztag"`
	BillAmount *int   `json:"bill_amount" example:"166847"`       // nil if not in bill
	CalcAmount *int   `json:"calculated_amount" example:"166847"` // nil if not calculable
	Difference int    `json:"difference" example:"0"`             // bill - calc (0 if either nil)
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
	CalcTotal       int                      `json:"calculated_total" example:"498000"`
	Difference      int                      `json:"difference" example:"2000"`
	ChildrenCount   int                      `json:"children_count" example:"25"`
	MatchCount      int                      `json:"match_count" example:"20"`
	DifferenceCount int                      `json:"difference_count" example:"3"`
	BillOnlyCount   int                      `json:"bill_only_count" example:"1"`
	CalcOnlyCount   int                      `json:"calc_only_count" example:"1"`
	Children        []FundingComparisonChild `json:"children"`
}

// ============================================================
// Per-child billing history DTOs
// ============================================================

// ChildBillingHistoryEntryResponse represents one month's billing data for a child.
type ChildBillingHistoryEntryResponse struct {
	BillID        uint                      `json:"bill_id" example:"1"`
	BillFrom      string                    `json:"bill_from" example:"2025-11-01"`
	BillTo        string                    `json:"bill_to" example:"2025-11-30"`
	FacilityName  string                    `json:"facility_name" example:"Kita Sonnenschein"`
	VoucherNumber string                    `json:"voucher_number" example:"GB-12345678901-02"`
	ChildName     string                    `json:"child_name" example:"Mustermann, Max"`
	BirthDate     string                    `json:"birth_date" example:"01.20"`
	Age           *int                      `json:"age,omitempty" example:"3"`
	BillTotal     int                       `json:"bill_total" example:"166847"`
	CalcTotal     *int                      `json:"calculated_total,omitempty" example:"166847"`
	Difference    *int                      `json:"difference,omitempty" example:"0"`
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
