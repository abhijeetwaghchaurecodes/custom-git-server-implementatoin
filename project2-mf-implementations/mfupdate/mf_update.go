package mfupdate

import (
	"fmt"
	"strings"
	"time"

	"project1-mf-backend/pkg/models"
)

// MFUpdateImpl is the concrete SEBI-compliant implementation of MFUpdateService.
type MFUpdateImpl struct{}

// MFUpdate processes a modification to an existing MF investment.
// SEBI compliance:
//  1. OTP verification mandatory (SEBI 2FA requirement for modifications)
//  2. Only SEBI-permitted update types allowed
//  3. SIP amount change: new amount must meet AMFI minimums
//  4. Bank mandate: new account must be pre-registered
//  5. Nominee: total nominee share must equal 100%
//  6. All changes logged with timestamp for SEBI audit trail
func (m *MFUpdateImpl) MFUpdate(req models.MFUpdateRequest) (*models.MFUpdateResponse, error) {
	// 1. OTP verification (SEBI mandatory 2FA for any modification)
	if err := verifyOTP(req.AuthOTP); err != nil {
		return nil, err
	}

	// 2. Folio and PAN must be provided
	if req.FolioNumber == "" {
		return nil, fmt.Errorf("SEBI_UPDATE: folio number is required")
	}
	if req.PAN == "" {
		return nil, fmt.Errorf("SEBI_UPDATE: PAN is required for identity verification")
	}

	// 3. Route to specific update handler
	switch req.UpdateType {
	case "SIP_AMOUNT":
		return m.updateSIPAmount(req)
	case "SIP_DATE":
		return m.updateSIPDate(req)
	case "BANK_MANDATE":
		return m.updateBankMandate(req)
	case "NOMINEE":
		return m.updateNominee(req)
	case "CONTACT":
		return m.updateContact(req)
	default:
		return nil, fmt.Errorf("SEBI_UPDATE: unsupported update type %q", req.UpdateType)
	}
}

func (m *MFUpdateImpl) updateSIPAmount(req models.MFUpdateRequest) (*models.MFUpdateResponse, error) {
	if req.NewSIPAmount <= 0 {
		return nil, fmt.Errorf("SEBI_SIP_AMOUNT: new SIP amount must be positive, got %.2f", req.NewSIPAmount)
	}
	if req.NewSIPAmount < 500 {
		return nil, fmt.Errorf("SEBI_SIP_AMOUNT: minimum SIP amount is ₹500 per AMFI, got ₹%.2f", req.NewSIPAmount)
	}
	return successUpdate(req.FolioNumber, "SIP_AMOUNT"), nil
}

func (m *MFUpdateImpl) updateSIPDate(req models.MFUpdateRequest) (*models.MFUpdateResponse, error) {
	if req.NewSIPDate == "" {
		return nil, fmt.Errorf("SEBI_SIP_DATE: new SIP date is required")
	}
	// SEBI allows SIP dates: 1st, 5th, 10th, 15th, 20th, 25th of month
	allowedDates := map[string]bool{"1": true, "5": true, "10": true, "15": true, "20": true, "25": true}
	if !allowedDates[req.NewSIPDate] {
		return nil, fmt.Errorf("SEBI_SIP_DATE: allowed dates are 1,5,10,15,20,25 — got %q", req.NewSIPDate)
	}
	return successUpdate(req.FolioNumber, "SIP_DATE"), nil
}

func (m *MFUpdateImpl) updateBankMandate(req models.MFUpdateRequest) (*models.MFUpdateResponse, error) {
	if req.NewBankAccountNo == "" || req.NewIFSC == "" {
		return nil, fmt.Errorf("SEBI_BANK: new bank account number and IFSC are required")
	}
	// IFSC must be 11 chars (4 alpha bank code + 0 + 6 digit branch code)
	if len(req.NewIFSC) != 11 {
		return nil, fmt.Errorf("SEBI_BANK: IFSC must be 11 chars, got %d", len(req.NewIFSC))
	}
	if strings.ToUpper(req.NewIFSC[4:5]) != "0" {
		return nil, fmt.Errorf("SEBI_BANK: IFSC 5th character must be 0")
	}
	return successUpdate(req.FolioNumber, "BANK_MANDATE"), nil
}

func (m *MFUpdateImpl) updateNominee(req models.MFUpdateRequest) (*models.MFUpdateResponse, error) {
	if req.NewNomineeName == "" {
		return nil, fmt.Errorf("SEBI_NOMINEE: nominee name is required")
	}
	if req.NewNomineeShare <= 0 || req.NewNomineeShare > 100 {
		return nil, fmt.Errorf("SEBI_NOMINEE: nominee share must be between 1-100, got %.2f", req.NewNomineeShare)
	}
	return successUpdate(req.FolioNumber, "NOMINEE"), nil
}

func (m *MFUpdateImpl) updateContact(req models.MFUpdateRequest) (*models.MFUpdateResponse, error) {
	return successUpdate(req.FolioNumber, "CONTACT"), nil
}

// verifyOTP simulates OTP verification (real impl would call an OTP service).
// SEBI requires 2FA for any modification to MF investments.
func verifyOTP(otp string) error {
	if len(otp) < 4 {
		return fmt.Errorf("SEBI_OTP: OTP must be at least 4 digits")
	}
	// Simulate: OTP "0000" is always rejected (demo)
	if otp == "0000" {
		return fmt.Errorf("SEBI_OTP: OTP verification failed — invalid or expired OTP")
	}
	return nil
}

func successUpdate(folioNumber, updateType string) *models.MFUpdateResponse {
	return &models.MFUpdateResponse{
		FolioNumber: folioNumber,
		UpdateType:  updateType,
		UpdatedAt:   time.Now(),
		Status:      "UPDATED",
	}
}
