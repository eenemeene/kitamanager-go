package models

// SettlementAmount represents a single financial line item in a settlement.
type SettlementAmount struct {
	Key    string `json:"key" example:"care_type"`
	Value  string `json:"value" example:"ganztag"`
	Amount int    `json:"amount" example:"166847"`
}

// SettlementChildResponse represents one child from the parsed settlement, enriched with match info.
type SettlementChildResponse struct {
	VoucherNumber string             `json:"voucher_number" example:"GB-12345678901-02"`
	ChildName     string             `json:"child_name" example:"Mustermann, Max"`
	BirthDate     string             `json:"birth_date" example:"01.20"`
	District      int64              `json:"district" example:"1"`
	TotalAmount   int                `json:"total_amount" example:"166847"`
	Amounts       []SettlementAmount `json:"amounts"`
	ChildID       *uint              `json:"child_id,omitempty" example:"42"`
	ContractID    *uint              `json:"contract_id,omitempty" example:"99"`
	Matched       bool               `json:"matched" example:"true"`
}

// SettlementUploadResponse is the full response for the ISBJ settlement upload endpoint.
type SettlementUploadResponse struct {
	FacilityName      string                    `json:"facility_name" example:"Kita Sonnenschein"`
	FacilityTotal     int                       `json:"facility_total" example:"500000"`
	ContractBooking   int                       `json:"contract_booking" example:"480000"`
	CorrectionBooking int                       `json:"correction_booking" example:"20000"`
	ChildrenCount     int                       `json:"children_count" example:"25"`
	MatchedCount      int                       `json:"matched_count" example:"23"`
	UnmatchedCount    int                       `json:"unmatched_count" example:"2"`
	Surcharges        []SettlementAmount        `json:"surcharges"`
	Children          []SettlementChildResponse `json:"children"`
}
