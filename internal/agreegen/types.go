package agreegen

import "time"

// All inputs in dhivehi
type documentRequest struct {
	TenantInfo        string `json:"tenant_info" validate:"required"`                 // format: "name (ID) (address, island)"
	TenantPhoneNumber string `json:"tenant_phone_number" validate:"required,numeric"` // no country code required. format: "7723123"
	RentAmountStr     string `json:"rent_amount_string" validate:"required"`
	RentAmountNumStr  string `json:"rent_amount_number" validate:"required,numeric"`
	FloorNum          string `json:"floor_number" validate:"required"`
	AgreementDuration int    `json:"agreement_duration" validate:"required"` // years
}

type agreementStartEndDates struct { // in dhivehi
	Start time.Time
	End   time.Time
}
