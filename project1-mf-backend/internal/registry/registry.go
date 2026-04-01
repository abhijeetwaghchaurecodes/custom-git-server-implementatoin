package registry

import "project1-mf-backend/internal/contracts"

var (
	mfCreateSvc   contracts.MFCreateService
	mfTransferSvc contracts.MFTransferService
	mfUpdateSvc   contracts.MFUpdateService
	mfDeleteSvc   contracts.MFDeleteService
)

func RegisterMFCreate(s contracts.MFCreateService)    { mfCreateSvc = s }
func RegisterMFTransfer(s contracts.MFTransferService) { mfTransferSvc = s }
func RegisterMFUpdate(s contracts.MFUpdateService)    { mfUpdateSvc = s }
func RegisterMFDelete(s contracts.MFDeleteService)    { mfDeleteSvc = s }

func GetMFCreate() contracts.MFCreateService    { return mfCreateSvc }
func GetMFTransfer() contracts.MFTransferService { return mfTransferSvc }
func GetMFUpdate() contracts.MFUpdateService    { return mfUpdateSvc }
func GetMFDelete() contracts.MFDeleteService    { return mfDeleteSvc }
