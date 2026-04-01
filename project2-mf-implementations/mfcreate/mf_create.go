package mfcreate

import (
	"fmt"
	"math"
	"strings"
	"time"

	"project1-mf-backend/pkg/models"
)

// MFCreateImpl is the concrete SEBI-compliant implementation of MFCreateService.
type MFCreateImpl struct{}

// MFCreate processes a new mutual fund investment.
// SEBI compliance:
//  1. KYC must be VERIFIED
//  2. PAN must be valid 10-char format
//  3. Amount >= AMFI minimum investment
//  4. Units = floor(amount/NAV * 1000) / 1000  (3 decimal precision)
//  5. Folio generated if not provided
//  6. Allotment: T+0 liquid, T+1 equity, T+2 debt
func (m *MFCreateImpl) MFCreate(req models.MFCreateRequest) (*models.MFCreateResponse, error) {
	// 1. KYC check — SEBI mandatory before any investment
	if req.KYCStatus != "VERIFIED" {
		return nil, fmt.Errorf("SEBI_KYC_FAILED: investor %s KYC status is %q, only VERIFIED investors can invest",
			req.InvestorID, req.KYCStatus)
	}

	// 2. PAN validation (AAAAA9999A format)
	if err := validatePAN(req.PAN); err != nil {
		return nil, err
	}

	// 3. Minimum investment amount per AMFI guidelines
	minAmt := minimumAmount(req.FundCategory, req.InvestmentMode)
	if req.Amount < minAmt {
		return nil, fmt.Errorf("SEBI_MIN_AMOUNT: minimum for %s %s is ₹%.2f, received ₹%.2f",
			req.FundCategory, req.InvestmentMode, minAmt, req.Amount)
	}

	// 4. Calculate allotted units (3 decimal precision, SEBI norm)
	units := math.Floor((req.Amount/req.NAV)*1000) / 1000

	// 5. Folio number (generate if not provided)
	folio := req.FolioNumber
	if folio == "" {
		folio = buildFolioNumber(req.InvestorID, req.SchemeCode)
	}

	// 6. SIP-specific validations
	if req.InvestmentMode == "SIP" {
		if req.SIPFrequency == "" {
			return nil, fmt.Errorf("SEBI_SIP: frequency required for SIP mode")
		}
		if req.SIPStartDate == "" {
			return nil, fmt.Errorf("SEBI_SIP: start date required for SIP mode")
		}
	}

	return &models.MFCreateResponse{
		FolioNumber:    folio,
		SchemeCode:     req.SchemeCode,
		SchemeName:     req.SchemeName,
		Units:          units,
		NAV:            req.NAV,
		InvestedAmount: req.Amount,
		AllotmentDate:  settlementDate(req.FundCategory),
		Status:         "ALLOTTED",
	}, nil
}

func validatePAN(pan string) error {
	pan = strings.ToUpper(pan)
	if len(pan) != 10 {
		return fmt.Errorf("SEBI_PAN: must be 10 chars, got %d", len(pan))
	}
	for i, ch := range pan {
		if i < 5 && !(ch >= 'A' && ch <= 'Z') {
			return fmt.Errorf("SEBI_PAN: positions 1-5 must be alpha")
		}
		if i >= 5 && i <= 8 && !(ch >= '0' && ch <= '9') {
			return fmt.Errorf("SEBI_PAN: positions 6-9 must be numeric")
		}
		if i == 9 && !(ch >= 'A' && ch <= 'Z') {
			return fmt.Errorf("SEBI_PAN: position 10 must be alpha")
		}
	}
	return nil
}

func minimumAmount(category, mode string) float64 {
	if mode == "SIP" {
		if category == "LIQUID" {
			return 100
		}
		return 500
	}
	if category == "ETF" {
		return 1000
	}
	return 5000
}

func settlementDate(category string) time.Time {
	switch category {
	case "LIQUID":
		return time.Now()
	case "DEBT", "HYBRID":
		return time.Now().AddDate(0, 0, 2)
	default:
		return time.Now().AddDate(0, 0, 1)
	}
}

func buildFolioNumber(investorID, schemeCode string) string {
	inv := investorID
	if len(inv) > 6 {
		inv = inv[:6]
	}
	sch := schemeCode
	if len(sch) > 4 {
		sch = sch[:4]
	}
	return fmt.Sprintf("INV-%s-%s-%d",
		strings.ToUpper(inv),
		strings.ToUpper(sch),
		time.Now().UnixMilli()%1000000,
	)
}
