package contracts

import "project1-mf-backend/pkg/models"

// MFCreateService defines the contract for mutual fund creation.
// SEBI compliance: KYC check, PAN validation, scheme validation,
// allotment at applicable NAV, folio generation.
type MFCreateService interface {
	MFCreate(req models.MFCreateRequest) (*models.MFCreateResponse, error)
}

// MFTransferService defines the contract for inter-scheme / inter-AMC transfers.
// SEBI compliance: exit load calculation, STCG/LTCG check,
// T+1 settlement for equity, T+2 for debt.
type MFTransferService interface {
	MFTransfer(req models.MFTransferRequest) (*models.MFTransferResponse, error)
}

// MFUpdateService defines the contract for modifying an existing MF investment.
// SEBI compliance: 2FA via OTP, audit trail, bank mandate validation.
type MFUpdateService interface {
	MFUpdate(req models.MFUpdateRequest) (*models.MFUpdateResponse, error)
}

// MFDeleteService defines the contract for redemption / cancellation.
// SEBI compliance: lock-in period check, TDS deduction, exit load,
// payout to registered bank account only.
type MFDeleteService interface {
	MFDelete(req models.MFDeleteRequest) (*models.MFDeleteResponse, error)
}
