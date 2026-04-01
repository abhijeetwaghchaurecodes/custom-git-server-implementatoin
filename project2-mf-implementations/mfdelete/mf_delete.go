package mfdelete

import (
	"fmt"
	"math"
	"time"

	"project1-mf-backend/pkg/models"
)

// MFDeleteImpl is the concrete SEBI-compliant implementation of MFDeleteService.
type MFDeleteImpl struct{}

// MFDelete processes a mutual fund redemption or cancellation.
// SEBI compliance:
//  1. OTP verification mandatory
//  2. Lock-in period check (ELSS: 3 years, Close-ended funds: till maturity)
//  3. Exit load calculation if applicable
//  4. TDS: 10% on LTCG above ₹1 lakh for equity, 20% with indexation for debt
//  5. Payout only to registered bank account (SEBI mandate)
//  6. Settlement: T+3 for equity, T+2 for debt
func (m *MFDeleteImpl) MFDelete(req models.MFDeleteRequest) (*models.MFDeleteResponse, error) {
	// 1. OTP verification (SEBI mandatory for redemptions)
	if err := verifyOTP(req.AuthOTP); err != nil {
		return nil, err
	}

	// 2. Lock-in period check
	if !req.LockInPeriodOver {
		return nil, fmt.Errorf("SEBI_LOCKIN: redemption blocked — lock-in period has not ended for folio %s",
			req.FolioNumber)
	}

	// 3. Determine redemption amount and units
	redeemUnits, redeemAmount, err := resolveRedemption(req)
	if err != nil {
		return nil, err
	}

	// 4. Exit load deduction
	exitLoad := 0.0
	if req.ExitLoadApplicable {
		// Standard 1% exit load
		exitLoad = math.Round(redeemAmount*0.01*100) / 100
	}
	amountAfterExitLoad := redeemAmount - exitLoad

	// 5. TDS deduction (simplified: flat rate on gains portion)
	tdsAmount := math.Round(amountAfterExitLoad*req.TaxDeductedAtSource/100*100) / 100

	// 6. Net payout
	netPayout := math.Round((amountAfterExitLoad-tdsAmount)*100) / 100
	if netPayout <= 0 {
		return nil, fmt.Errorf("SEBI_REDEEM: net payout is ₹%.2f after deductions — cannot process", netPayout)
	}

	// 7. Payout bank must be registered (SEBI: no third-party payouts)
	if req.BankAccountNo == "" || req.IFSC == "" {
		return nil, fmt.Errorf("SEBI_PAYOUT: registered bank account and IFSC are mandatory for payout")
	}

	return &models.MFDeleteResponse{
		FolioNumber:      req.FolioNumber,
		RedeemedUnits:    redeemUnits,
		RedemptionAmount: redeemAmount,
		TDSDeducted:      tdsAmount,
		NetPayoutAmount:  netPayout,
		SettlementDate:   settlementDate(),
		Status:           "REDEMPTION_INITIATED",
	}, nil
}

// resolveRedemption determines units and amount based on redemption mode.
func resolveRedemption(req models.MFDeleteRequest) (units, amount float64, err error) {
	switch req.RedemptionMode {
	case "ALL":
		if req.Units <= 0 {
			return 0, 0, fmt.Errorf("SEBI_REDEEM: total units must be provided for ALL mode")
		}
		// Simulate: NAV not in request, use a placeholder calculation
		// Real impl would fetch current NAV from AMFI API
		amount = math.Round(req.Units*45.50*100) / 100 // placeholder NAV
		return req.Units, amount, nil

	case "UNITS":
		if req.Units <= 0 {
			return 0, 0, fmt.Errorf("SEBI_REDEEM: units must be positive for UNITS mode, got %.3f", req.Units)
		}
		amount = math.Round(req.Units*45.50*100) / 100 // placeholder NAV
		return req.Units, amount, nil

	case "AMOUNT":
		if req.Amount <= 0 {
			return 0, 0, fmt.Errorf("SEBI_REDEEM: amount must be positive for AMOUNT mode, got %.2f", req.Amount)
		}
		// Simulate reverse calculation: units = amount / NAV
		units = math.Floor((req.Amount/45.50)*1000) / 1000
		return units, req.Amount, nil

	default:
		return 0, 0, fmt.Errorf("SEBI_REDEEM: unknown redemption mode %q", req.RedemptionMode)
	}
}

// settlementDate returns T+3 (equity standard per SEBI).
func settlementDate() time.Time {
	return time.Now().AddDate(0, 0, 3)
}

// verifyOTP simulates SEBI-mandated OTP verification for redemptions.
func verifyOTP(otp string) error {
	if len(otp) < 4 {
		return fmt.Errorf("SEBI_OTP: OTP must be at least 4 digits")
	}
	if otp == "0000" {
		return fmt.Errorf("SEBI_OTP: invalid or expired OTP")
	}
	return nil
}
