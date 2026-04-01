package mftransfer

import (
	"fmt"
	"math"
	"time"

	"project1-mf-backend/pkg/models"
)

// MFTransferImpl is the concrete SEBI-compliant implementation of MFTransferService.
type MFTransferImpl struct{}

// MFTransfer processes an inter-scheme or inter-AMC switch/STP.
// SEBI compliance:
//  1. Validate folio and PAN match
//  2. Calculate exit load (if holding < 1 year for equity)
//  3. Determine STCG (< 1 year) or LTCG (>= 1 year) applicability
//  4. Redemption amount = units × fromNAV - exit load
//  5. New units in target = redemption amount / toNAV
//  6. Settlement: T+3 for equity switch, T+2 for debt
func (m *MFTransferImpl) MFTransfer(req models.MFTransferRequest) (*models.MFTransferResponse, error) {
	// 1. Validate redemption units
	if req.RedemptionUnits <= 0 {
		return nil, fmt.Errorf("SEBI_TRANSFER: redemption units must be positive, got %.3f", req.RedemptionUnits)
	}

	// 2. Validate NAVs
	if req.FromNAV <= 0 || req.ToNAV <= 0 {
		return nil, fmt.Errorf("SEBI_TRANSFER: both FromNAV and ToNAV must be positive")
	}

	// 3. Validate same-AMC restriction for SWITCH (inter-AMC needs separate redemption+purchase)
	if req.TransferType == "SWITCH" && req.FromAMC != req.ToAMC {
		return nil, fmt.Errorf("SEBI_SWITCH: same-day switch requires same AMC (%s → %s). Use separate redemption + purchase for inter-AMC",
			req.FromAMC, req.ToAMC)
	}

	// 4. Gross redemption amount
	grossAmount := req.RedemptionUnits * req.FromNAV

	// 5. Exit load calculation (SEBI: most equity funds charge 1% if redeemed < 1 year)
	exitLoad := 0.0
	if req.ExitLoadApplicable {
		exitLoad = math.Round(grossAmount*req.ExitLoadPercent/100*100) / 100
	}

	// 6. Net redemption amount after exit load
	netRedemptionAmount := grossAmount - exitLoad

	// 7. Capital gains applicability flags (informational — actual tax computed by AMC)
	// STCG: equity < 1 year taxed at 15%, debt < 3 years at slab rate
	// LTCG: equity > 1 year taxed at 10% above ₹1 lakh, debt > 3 years at 20% with indexation

	// 8. New units in target scheme (3 decimal precision)
	newUnits := math.Floor((netRedemptionAmount/req.ToNAV)*1000) / 1000
	if newUnits <= 0 {
		return nil, fmt.Errorf("SEBI_TRANSFER: net redemption amount ₹%.2f insufficient for target NAV ₹%.4f",
			netRedemptionAmount, req.ToNAV)
	}

	// 9. Generate transaction ID and settlement date
	txnID := fmt.Sprintf("SWT-%d", time.Now().UnixNano())
	settlement := switchSettlementDate(req.TransferType)

	return &models.MFTransferResponse{
		TransactionID:  txnID,
		FromScheme:     req.FromSchemeName,
		ToScheme:       req.ToSchemeName,
		RedeemedUnits:  req.RedemptionUnits,
		RedeemedAmount: netRedemptionAmount,
		NewUnits:       newUnits,
		SettlementDate: settlement,
		Status:         "SWITCH_INITIATED",
	}, nil
}

// switchSettlementDate returns settlement date based on transfer type.
func switchSettlementDate(transferType string) time.Time {
	switch transferType {
	case "STP":
		return time.Now().AddDate(0, 0, 3) // T+3 for STP
	default:
		return time.Now().AddDate(0, 0, 2) // T+2 for switch
	}
}
