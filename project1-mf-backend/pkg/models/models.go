package models

import "time"

// MFCreateRequest represents the SEBI-compliant mutual fund creation request.
type MFCreateRequest struct {
	// Investor details
	InvestorID    string `json:"investor_id" validate:"required"`
	InvestorName  string `json:"investor_name" validate:"required"`
	PAN           string `json:"pan" validate:"required,len=10"`
	KYCStatus     string `json:"kyc_status" validate:"required,oneof=VERIFIED PENDING REJECTED"`
	DematAccountNo string `json:"demat_account_no" validate:"required"`

	// Fund details
	SchemeCode    string  `json:"scheme_code" validate:"required"`    // AMFI scheme code
	SchemeName    string  `json:"scheme_name" validate:"required"`
	AMC           string  `json:"amc" validate:"required"`            // Asset Management Company
	FundCategory  string  `json:"fund_category" validate:"required,oneof=EQUITY DEBT HYBRID LIQUID ETF"`
	FundType      string  `json:"fund_type" validate:"required,oneof=GROWTH DIVIDEND_PAYOUT DIVIDEND_REINVEST"`

	// Investment details
	InvestmentMode string  `json:"investment_mode" validate:"required,oneof=LUMPSUM SIP"`
	Amount         float64 `json:"amount" validate:"required,gt=0"`
	SIPFrequency   string  `json:"sip_frequency,omitempty" validate:"omitempty,oneof=DAILY WEEKLY MONTHLY QUARTERLY"`
	SIPStartDate   string  `json:"sip_start_date,omitempty"`
	SIPEndDate     string  `json:"sip_end_date,omitempty"`
	NAV            float64 `json:"nav" validate:"required,gt=0"`        // Net Asset Value at time of purchase
	Units          float64 `json:"units" validate:"required,gt=0"`

	// SEBI Compliance
	RiskProfile    string `json:"risk_profile" validate:"required,oneof=LOW MODERATE HIGH"`
	NomineeID      string `json:"nominee_id,omitempty"`
	BankAccountNo  string `json:"bank_account_no" validate:"required"`
	IFSC           string `json:"ifsc" validate:"required"`
	FolioNumber    string `json:"folio_number,omitempty"`

	// Metadata
	IPAddress      string `json:"ip_address"`
	Platform       string `json:"platform" validate:"required,oneof=WEB MOBILE API"`
}

// MFTransferRequest represents an inter-scheme or inter-AMC switch/transfer.
type MFTransferRequest struct {
	InvestorID       string  `json:"investor_id" validate:"required"`
	PAN              string  `json:"pan" validate:"required,len=10"`
	FolioNumber      string  `json:"folio_number" validate:"required"`

	// Source fund
	FromSchemeCode   string  `json:"from_scheme_code" validate:"required"`
	FromSchemeName   string  `json:"from_scheme_name" validate:"required"`
	FromAMC          string  `json:"from_amc" validate:"required"`
	RedemptionUnits  float64 `json:"redemption_units" validate:"required,gt=0"`
	FromNAV          float64 `json:"from_nav" validate:"required,gt=0"`

	// Target fund
	ToSchemeCode     string  `json:"to_scheme_code" validate:"required"`
	ToSchemeName     string  `json:"to_scheme_name" validate:"required"`
	ToAMC            string  `json:"to_amc" validate:"required"`
	ToNAV            float64 `json:"to_nav" validate:"required,gt=0"`

	// Transfer type
	TransferType     string  `json:"transfer_type" validate:"required,oneof=SWITCH SWITCH_PARTIAL STP"` // STP = Systematic Transfer Plan
	STAFrequency     string  `json:"sta_frequency,omitempty"`

	// SEBI compliance
	ExitLoadApplicable bool   `json:"exit_load_applicable"`
	ExitLoadPercent    float64 `json:"exit_load_percent,omitempty"`
	STCGApplicable     bool   `json:"stcg_applicable"`     // Short Term Capital Gains
	LTCGApplicable     bool   `json:"ltcg_applicable"`     // Long Term Capital Gains
	SEBITransactionID  string `json:"sebi_transaction_id,omitempty"`
}

// MFUpdateRequest represents a modification to an existing MF investment.
type MFUpdateRequest struct {
	InvestorID     string  `json:"investor_id" validate:"required"`
	PAN            string  `json:"pan" validate:"required,len=10"`
	FolioNumber    string  `json:"folio_number" validate:"required"`
	SchemeCode     string  `json:"scheme_code" validate:"required"`

	// What can be updated
	UpdateType     string  `json:"update_type" validate:"required,oneof=SIP_AMOUNT SIP_DATE BANK_MANDATE NOMINEE CONTACT"`

	// SIP modification
	NewSIPAmount   float64 `json:"new_sip_amount,omitempty"`
	NewSIPDate     string  `json:"new_sip_date,omitempty"`
	NewSIPFrequency string `json:"new_sip_frequency,omitempty"`

	// Bank mandate change
	NewBankAccountNo string `json:"new_bank_account_no,omitempty"`
	NewIFSC          string `json:"new_ifsc,omitempty"`

	// Nominee update
	NewNomineeID     string `json:"new_nominee_id,omitempty"`
	NewNomineeName   string `json:"new_nominee_name,omitempty"`
	NewNomineeShare  float64 `json:"new_nominee_share,omitempty"`

	// Audit
	Reason           string `json:"reason" validate:"required"`
	AuthOTP          string `json:"auth_otp" validate:"required"` // OTP for SEBI-mandated 2FA
}

// MFDeleteRequest represents a redemption or cancellation.
type MFDeleteRequest struct {
	InvestorID       string  `json:"investor_id" validate:"required"`
	PAN              string  `json:"pan" validate:"required,len=10"`
	FolioNumber      string  `json:"folio_number" validate:"required"`
	SchemeCode       string  `json:"scheme_code" validate:"required"`

	// Redemption type
	RedemptionType   string  `json:"redemption_type" validate:"required,oneof=FULL PARTIAL SIP_CANCEL"`
	Units            float64 `json:"units,omitempty"`       // for PARTIAL
	Amount           float64 `json:"amount,omitempty"`      // alternative: amount-based redemption
	RedemptionMode   string  `json:"redemption_mode" validate:"required,oneof=UNITS AMOUNT ALL"`

	// Payout
	BankAccountNo    string  `json:"bank_account_no" validate:"required"`
	IFSC             string  `json:"ifsc" validate:"required"`

	// SEBI compliance
	ExitLoadApplicable bool   `json:"exit_load_applicable"`
	LockInPeriodOver   bool   `json:"lock_in_period_over"`
	TaxDeductedAtSource float64 `json:"tds_percent,omitempty"`
	AuthOTP            string  `json:"auth_otp" validate:"required"`
}

// --- Responses ---

// MFResponse is the standard SEBI-compliant API response envelope.
type MFResponse struct {
	Success       bool        `json:"success"`
	TransactionID string      `json:"transaction_id"`
	SEBIRefNo     string      `json:"sebi_ref_no"`
	Timestamp     time.Time   `json:"timestamp"`
	Data          interface{} `json:"data,omitempty"`
	Error         *MFError    `json:"error,omitempty"`
}

// MFError is the structured error response.
type MFError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

// MFCreateResponse is the data payload for a successful creation.
type MFCreateResponse struct {
	FolioNumber       string    `json:"folio_number"`
	SchemeCode        string    `json:"scheme_code"`
	SchemeName        string    `json:"scheme_name"`
	Units             float64   `json:"units"`
	NAV               float64   `json:"nav"`
	InvestedAmount    float64   `json:"invested_amount"`
	AllotmentDate     time.Time `json:"allotment_date"`
	Status            string    `json:"status"`
}

// MFTransferResponse is the data payload for a successful transfer.
type MFTransferResponse struct {
	TransactionID   string    `json:"transaction_id"`
	FromScheme      string    `json:"from_scheme"`
	ToScheme        string    `json:"to_scheme"`
	RedeemedUnits   float64   `json:"redeemed_units"`
	RedeemedAmount  float64   `json:"redeemed_amount"`
	NewUnits        float64   `json:"new_units"`
	SettlementDate  time.Time `json:"settlement_date"`
	Status          string    `json:"status"`
}

// MFUpdateResponse is the data payload for a successful update.
type MFUpdateResponse struct {
	FolioNumber string    `json:"folio_number"`
	UpdateType  string    `json:"update_type"`
	UpdatedAt   time.Time `json:"updated_at"`
	Status      string    `json:"status"`
}

// MFDeleteResponse is the data payload for a successful redemption.
type MFDeleteResponse struct {
	FolioNumber       string    `json:"folio_number"`
	RedeemedUnits     float64   `json:"redeemed_units"`
	RedemptionAmount  float64   `json:"redemption_amount"`
	TDSDeducted       float64   `json:"tds_deducted"`
	NetPayoutAmount   float64   `json:"net_payout_amount"`
	SettlementDate    time.Time `json:"settlement_date"`
	Status            string    `json:"status"`
}
