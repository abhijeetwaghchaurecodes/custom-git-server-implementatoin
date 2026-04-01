package service

import (
	"errors"
	"project1-mf-backend/pkg/models"
	"project1-mf-backend/internal/registry"
)

// MFCreate delegates to the injected MFCreateService.
// Actual business logic lives in project2-mf-implementations/mfcreate.
func MFCreate(req models.MFCreateRequest) (*models.MFCreateResponse, error) {
	svc := registry.GetMFCreate()
	if svc == nil {
		return nil, errors.New("MFCreate: implementation not injected — run the injector service")
	}
	return svc.MFCreate(req)
}

// MFTransfer delegates to the injected MFTransferService.
// Actual business logic lives in project2-mf-implementations/mftransfer.
func MFTransfer(req models.MFTransferRequest) (*models.MFTransferResponse, error) {
	svc := registry.GetMFTransfer()
	if svc == nil {
		return nil, errors.New("MFTransfer: implementation not injected — run the injector service")
	}
	return svc.MFTransfer(req)
}

// MFUpdate delegates to the injected MFUpdateService.
// Actual business logic lives in project2-mf-implementations/mfupdate.
func MFUpdate(req models.MFUpdateRequest) (*models.MFUpdateResponse, error) {
	svc := registry.GetMFUpdate()
	if svc == nil {
		return nil, errors.New("MFUpdate: implementation not injected — run the injector service")
	}
	return svc.MFUpdate(req)
}

// MFDelete delegates to the injected MFDeleteService.
// Actual business logic lives in project2-mf-implementations/mfdelete.
func MFDelete(req models.MFDeleteRequest) (*models.MFDeleteResponse, error) {
	svc := registry.GetMFDelete()
	if svc == nil {
		return nil, errors.New("MFDelete: implementation not injected — run the injector service")
	}
	return svc.MFDelete(req)
}
