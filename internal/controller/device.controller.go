package controller

import (
	"errors"
	"net/http"

	"inventory-manage/internal/model"
	"inventory-manage/internal/service"
	"inventory-manage/pkg/response"

	"github.com/gin-gonic/gin"
)

// DeviceController handles HTTP requests for the device resource.
type DeviceController struct {
	deviceService service.IDeviceService
}

// NewDeviceController creates a DeviceController with the injected service.
func NewDeviceController(deviceService service.IDeviceService) *DeviceController {
	return &DeviceController{deviceService: deviceService}
}

// CreateDevice godoc
//
//	@Summary		Register a new IoT scale device
//	@Description	Creates a new device record in the inventory system
//	@Tags			devices
//	@Accept			json
//	@Produce		json
//	@Param			device	body		model.Device			true	"Device payload"
//	@Success		200		{object}	response.ResponseData
//	@Failure		200		{object}	response.ErrorResponseData
//	@Router			/api/v1/devices [post]
func (dc *DeviceController) CreateDevice(c *gin.Context) {
	var d model.Device
	if err := c.ShouldBindJSON(&d); err != nil {
		response.ErrorResponse(c, response.ErrCodeParamInvalid, err.Error())
		return
	}

	if err := dc.deviceService.RegisterDevice(c.Request.Context(), &d); err != nil {
		if errors.Is(err, model.ErrDuplicateDevice) {
			response.ErrorResponse(c, response.ErrCodeDeviceDuplicate, err.Error())
			return
		}
		response.ErrorResponse(c, response.ErrCodeDeviceInvalid, err.Error())
		return
	}

	response.SuccessResponse(c, response.ErrCodeSuccess, d)
}

// ListDevices godoc
//
//	@Summary		List all devices
//	@Description	Returns all registered devices, optionally filtered by status or sku_code
//	@Tags			devices
//	@Produce		json
//	@Param			status		query		string	false	"Device status filter (active|inactive|maintenance)"
//	@Param			sku_code	query		string	false	"SKU code filter"
//	@Success		200			{object}	response.ResponseData
//	@Router			/api/v1/devices [get]
func (dc *DeviceController) ListDevices(c *gin.Context) {
	q := model.DeviceQuery{}
	if s := c.Query("status"); s != "" {
		st := model.DeviceStatus(s)
		q.Status = &st
	}
	if sku := c.Query("sku_code"); sku != "" {
		q.SKUCode = &sku
	}

	devices, err := dc.deviceService.ListDevices(c.Request.Context(), q)
	if err != nil {
		response.ErrorResponse(c, response.ErrCodeInternalServer, err.Error())
		return
	}
	if devices == nil {
		devices = []*model.Device{}
	}

	response.SuccessResponse(c, response.ErrCodeSuccess, devices)
}

// GetDevice godoc
//
//	@Summary		Get a device by ID
//	@Tags			devices
//	@Produce		json
//	@Param			id	path		string	true	"Device ID"
//	@Success		200	{object}	response.ResponseData
//	@Router			/api/v1/devices/{id} [get]
func (dc *DeviceController) GetDevice(c *gin.Context) {
	id := c.Param("id")

	d, err := dc.deviceService.GetDevice(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, model.ErrDeviceNotFound) {
			response.ErrorResponseWithHTTP(c, http.StatusNotFound, response.ErrCodeDeviceNotFound, err.Error())
			return
		}
		response.ErrorResponse(c, response.ErrCodeInternalServer, err.Error())
		return
	}

	response.SuccessResponse(c, response.ErrCodeSuccess, d)
}

// UpdateDevice godoc
//
//	@Summary		Update a device
//	@Tags			devices
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string			true	"Device ID"
//	@Param			device	body		model.Device	true	"Updated device data"
//	@Success		200		{object}	response.ResponseData
//	@Router			/api/v1/devices/{id} [put]
func (dc *DeviceController) UpdateDevice(c *gin.Context) {
	id := c.Param("id")

	var d model.Device
	if err := c.ShouldBindJSON(&d); err != nil {
		response.ErrorResponse(c, response.ErrCodeParamInvalid, err.Error())
		return
	}
	d.DeviceID = id

	if err := dc.deviceService.UpdateDevice(c.Request.Context(), &d); err != nil {
		if errors.Is(err, model.ErrDeviceNotFound) {
			response.ErrorResponseWithHTTP(c, http.StatusNotFound, response.ErrCodeDeviceNotFound, err.Error())
			return
		}
		response.ErrorResponse(c, response.ErrCodeDeviceInvalid, err.Error())
		return
	}

	updated, err := dc.deviceService.GetDevice(c.Request.Context(), id)
	if err != nil {
		response.ErrorResponse(c, response.ErrCodeInternalServer, "failed to fetch updated device")
		return
	}

	response.SuccessResponse(c, response.ErrCodeSuccess, updated)
}

// DeleteDevice godoc
//
//	@Summary		Delete a device
//	@Tags			devices
//	@Produce		json
//	@Param			id	path		string	true	"Device ID"
//	@Success		200	{object}	response.ResponseData
//	@Router			/api/v1/devices/{id} [delete]
func (dc *DeviceController) DeleteDevice(c *gin.Context) {
	id := c.Param("id")

	if err := dc.deviceService.RemoveDevice(c.Request.Context(), id); err != nil {
		if errors.Is(err, model.ErrDeviceNotFound) {
			response.ErrorResponseWithHTTP(c, http.StatusNotFound, response.ErrCodeDeviceNotFound, err.Error())
			return
		}
		response.ErrorResponse(c, response.ErrCodeInternalServer, err.Error())
		return
	}

	response.SuccessResponse(c, response.ErrCodeSuccess, nil)
}
