package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"project1-mf-backend/pkg/models"
	"project1-mf-backend/internal/service"
)

// MFHandler holds all HTTP handlers for the /mutualfund route group.
type MFHandler struct{}

func NewMFHandler() *MFHandler { return &MFHandler{} }

// RegisterRoutes wires all MF endpoints:
//
//	POST   /mutualfund          → MFCreate
//	POST   /mutualfund/transfer → MFTransfer
//	PUT    /mutualfund          → MFUpdate
//	DELETE /mutualfund          → MFDelete
func (h *MFHandler) RegisterRoutes(e *echo.Echo) {
	mf := e.Group("/mutualfund")
	mf.POST("", h.Create)
	mf.POST("/transfer", h.Transfer)
	mf.PUT("", h.Update)
	mf.DELETE("", h.Delete)
}

// Create handles POST /mutualfund
func (h *MFHandler) Create(c echo.Context) error {
	var req models.MFCreateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errResp("INVALID_REQUEST", err.Error()))
	}
	resp, err := service.MFCreate(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResp("MF_CREATE_FAILED", err.Error()))
	}
	return c.JSON(http.StatusCreated, okResp("CRE", resp))
}

// Transfer handles POST /mutualfund/transfer
func (h *MFHandler) Transfer(c echo.Context) error {
	var req models.MFTransferRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errResp("INVALID_REQUEST", err.Error()))
	}
	resp, err := service.MFTransfer(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResp("MF_TRANSFER_FAILED", err.Error()))
	}
	return c.JSON(http.StatusOK, okResp("TRF", resp))
}

// Update handles PUT /mutualfund
func (h *MFHandler) Update(c echo.Context) error {
	var req models.MFUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errResp("INVALID_REQUEST", err.Error()))
	}
	resp, err := service.MFUpdate(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResp("MF_UPDATE_FAILED", err.Error()))
	}
	return c.JSON(http.StatusOK, okResp("UPD", resp))
}

// Delete handles DELETE /mutualfund
func (h *MFHandler) Delete(c echo.Context) error {
	var req models.MFDeleteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errResp("INVALID_REQUEST", err.Error()))
	}
	resp, err := service.MFDelete(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResp("MF_DELETE_FAILED", err.Error()))
	}
	return c.JSON(http.StatusOK, okResp("DEL", resp))
}

func okResp(prefix string, data interface{}) models.MFResponse {
	return models.MFResponse{
		Success:       true,
		TransactionID: fmt.Sprintf("SEBI-%s-%d", prefix, time.Now().UnixNano()),
		SEBIRefNo:     fmt.Sprintf("SEBIMF%s", time.Now().Format("20060102150405")),
		Timestamp:     time.Now(),
		Data:          data,
	}
}

func errResp(code, msg string) models.MFResponse {
	return models.MFResponse{
		Success:   false,
		Timestamp: time.Now(),
		Error:     &models.MFError{Code: code, Message: msg},
	}
}
