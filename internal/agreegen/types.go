package agreegen

import "time"

// All inputs in dhivehi
type documentRequest struct {
	TenantInfo        string    `json:"tenant_info" validate:"required"`        // format: "name (ID) (address, island)"
	RentAmountStr     string    `json:"rent_amount" validate:"required"`        // format: amount/- (amountWords dhivehi rufiya)
	FloorNum          string    `json:"floor_number" validate:"required"`       // in dhivehi words, non-numeric
	SingleDeposit     string    `json:"single_deposit" validate:"required"`     // format: amount/- (amountWords dhivehi rufiya). TODO: multiple deposits
	AgreementStart    time.Time `json:"agreement_start"`                        // optional, datetime
	AgreementDuration int       `json:"agreement_duration" validate:"required"` // years

	// optional
	SigFieldTenantName    string `json:"sig_tenant_name"`
	SigFieldTenantId      string `json:"sig_tenant_id"`
	SigFieldTenantAddress string `json:"sig_tenant_address"`
	TenantPhoneNumber     string `json:"tenant_phone_number"` // no country code required. format: "7723123"
}

type agreementStartEndDates struct { // in dhivehi
	Start time.Time
	End   time.Time
}
