package controller

import (
	"errors"
	"net/http"

	"inventory-manage/internal/model"
	"inventory-manage/internal/service"
	"inventory-manage/pkg/response"

	"github.com/gin-gonic/gin"
)

// CalibrationController handles HTTP requests for calibration configs.
type CalibrationController struct {
	calibService service.ICalibrationService
}

// NewCalibrationController creates a CalibrationController.
func NewCalibrationController(calibService service.ICalibrationService) *CalibrationController {
	return &CalibrationController{calibService: calibService}
}

// CreateCalibration godoc
//
//	@Summary		Register calibration config for a device
//	@Description	Creates a new calibration profile for an IoT scale
//	@Tags			calibration
//	@Accept			json
//	@Produce		json
//	@Param			config	body		model.CalibrationConfig	true	"Calibration config payload"
//	@Success		200		{object}	response.ResponseData
//	@Failure		200		{object}	response.ErrorResponseData
//	@Router			/api/v1/calibrations [post]
func (cc *CalibrationController) CreateCalibration(c *gin.Context) {
	var cfg model.CalibrationConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.ErrorResponse(c, response.ErrCodeParamInvalid, err.Error())
		return
	}

	if err := cc.calibService.RegisterCalibration(c.Request.Context(), &cfg); err != nil {
		response.ErrorResponse(c, response.ErrCodeCalibrationInvalid, err.Error())
		return
	}

	response.SuccessResponse(c, response.ErrCodeSuccess, cfg)
}

// GetActiveCalibration godoc
//
//	@Summary		Get the active calibration config for a device
//	@Tags			calibration
//	@Produce		json
//	@Param			device_id	path		string	true	"Device ID"
//	@Success		200			{object}	response.ResponseData
//	@Failure		404			{object}	response.ErrorResponseData
//	@Router			/api/v1/calibrations/{device_id}/active [get]
func (cc *CalibrationController) GetActiveCalibration(c *gin.Context) {
	deviceID := c.Param("device_id")

	cfg, err := cc.calibService.GetActiveCalibration(c.Request.Context(), deviceID)
	if err != nil {
		if errors.Is(err, model.ErrDeviceNotFound) {
			response.ErrorResponseWithHTTP(c, http.StatusNotFound,
				response.ErrCodeCalibrationNotFound, err.Error())
			return
		}
		response.ErrorResponse(c, response.ErrCodeInternalServer, err.Error())
		return
	}

	response.SuccessResponse(c, response.ErrCodeSuccess, cfg)
}

// UpdateCalibration godoc
//
//	@Summary		Update / Add a new active calibration config for a device
//	@Description	Deactivates the current config and creates a new one in a single transaction (initial, periodic, drift_correction)
//	@Tags			calibration
//	@Accept			json
//	@Produce		json
//	@Param			device_id	path		string							true	"Device ID"
//	@Param			config		body		model.UpdateCalibrationParams	true	"Calibration update payload"
//	@Success		200			{object}	response.ResponseData
//	@Failure		400			{object}	response.ErrorResponseData
//	@Router			/api/v1/calibrations/{device_id}/update [post]
func (cc *CalibrationController) UpdateCalibration(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		response.ErrorResponse(c, response.ErrCodeParamInvalid, "device_id is required")
		return
	}

	var params model.UpdateCalibrationParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.ErrorResponse(c, response.ErrCodeParamInvalid, err.Error())
		return
	}

	if err := cc.calibService.UpdateCalibration(c.Request.Context(), deviceID, &params); err != nil {
		response.ErrorResponse(c, response.ErrCodeCalibrationInvalid, err.Error())
		return
	}

	response.SuccessResponse(c, response.ErrCodeSuccess, "Calibration updated successfully")
}
