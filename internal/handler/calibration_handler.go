package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"inventory-manage/internal/domain/device"

	"github.com/go-chi/chi/v5"
)

type CalibrationHandler struct {
	useCase device.CalibrationUseCase
}

func NewCalibrationHandler(useCase device.CalibrationUseCase) *CalibrationHandler {
	return &CalibrationHandler{useCase: useCase}
}

func (h *CalibrationHandler) RegisterRoutes(r chi.Router) {
	r.Post("/devices/{id}/calibration", h.Create)
	r.Get("/devices/{id}/calibration/active", h.GetActive)
}

func (h *CalibrationHandler) Create(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var c device.CalibrationConfig
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	c.DeviceID = id

	if err := h.useCase.RegisterCalibration(r.Context(), &c); err != nil {
		// Can add specific domain errors later for 404 vs 400
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, Response{Status: "success", Data: c})
}

func (h *CalibrationHandler) GetActive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	c, err := h.useCase.GetActiveCalibration(r.Context(), id)
	if err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			respondError(w, http.StatusNotFound, "No active calibration found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, Response{Status: "success", Data: c})
}
