package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"inventory-manage/global"
	"inventory-manage/internal/model"
	"inventory-manage/internal/service"
	"inventory-manage/pkg/response"
)

type InventoryController struct {
	repo service.IInventoryRepository
}

func NewInventoryController(repo service.IInventoryRepository) *InventoryController {
	return &InventoryController{repo: repo}
}

// GetCurrentInventory godoc
//
//  @Summary      Get current inventory snapshots across all SKUs
//  @Description  Returns the latest inventory snapshots across all connected devices
//  @Tags         inventory
//  @Produce      json
//  @Success      200  {object}  response.ResponseData
//  @Failure      500  {object}  response.ErrorResponseData
//  @Router       /api/v1/inventory/current [get]
func (ic *InventoryController) GetCurrentInventory(c *gin.Context) {
	traceID := c.GetString("trace_id")

	snapshots, err := ic.repo.GetCurrentSnapshots(c.Request.Context())
	if err != nil {
		global.Logger.Error("GetCurrentSnapshots failed",
			zap.String("trace_id", traceID),
			zap.Error(err),
		)
		response.ErrorResponseWithHTTP(c, http.StatusInternalServerError, response.ErrCodeInternalServer, err.Error())
		return
	}

	response.SuccessResponse(c, response.ErrCodeSuccess, snapshots)
}

// GetInventoryBySKU godoc
//
//  @Summary      Get current inventory for a specific SKU
//  @Description  Returns the latest inventory snapshots for devices holding the specific SKU
//  @Tags         inventory
//  @Produce      json
//  @Param        sku_code   path      string  true  "SKU Code"
//  @Success      200  {object}  response.ResponseData
//  @Failure      500  {object}  response.ErrorResponseData
//  @Router       /api/v1/inventory/{sku_code}/current [get]
func (ic *InventoryController) GetInventoryBySKU(c *gin.Context) {
	skuCode := c.Param("sku_code")
	traceID := c.GetString("trace_id")

	if skuCode == "" {
		response.ErrorResponseWithHTTP(c, http.StatusBadRequest, response.ErrCodeParamInvalid, "sku_code is required")
		return
	}

	// First verify SKU exists just in case
	_, err := ic.repo.GetSKUConfig(c.Request.Context(), skuCode)
	if err != nil {
		if errors.Is(err, model.ErrSKUNotFound) {
			response.ErrorResponseWithHTTP(c, http.StatusNotFound, response.ErrCodeDeviceNotFound, "sku config not found")
			return
		}
		global.Logger.Error("GetSKUConfig failed",
			zap.String("sku_code", skuCode),
			zap.String("trace_id", traceID),
			zap.Error(err),
		)
		response.ErrorResponseWithHTTP(c, http.StatusInternalServerError, response.ErrCodeInternalServer, err.Error())
		return
	}

	snapshots, err := ic.repo.GetSnapshotBySKU(c.Request.Context(), skuCode)
	if err != nil {
		global.Logger.Error("GetSnapshotBySKU failed",
			zap.String("sku_code", skuCode),
			zap.String("trace_id", traceID),
			zap.Error(err),
		)
		response.ErrorResponseWithHTTP(c, http.StatusInternalServerError, response.ErrCodeInternalServer, err.Error())
		return
	}

	response.SuccessResponse(c, response.ErrCodeSuccess, snapshots)
}
