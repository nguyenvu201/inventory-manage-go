package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"inventory-manage/internal/domain/device"

	"github.com/go-chi/chi/v5"
)

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type DeviceHandler struct {
	useCase device.UseCase
}

func NewDeviceHandler(useCase device.UseCase) *DeviceHandler {
	return &DeviceHandler{useCase: useCase}
}

func (h *DeviceHandler) RegisterRoutes(r chi.Router) {
	r.Post("/devices", h.Create)
	r.Get("/devices", h.List)
	r.Get("/devices/{id}", h.Get)
	r.Put("/devices/{id}", h.Update)
	r.Delete("/devices/{id}", h.Delete)
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, Response{Status: "error", Message: msg})
}

func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var d device.Device
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if err := h.useCase.RegisterDevice(r.Context(), &d); err != nil {
		if errors.Is(err, device.ErrDuplicateDevice) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, Response{Status: "success", Data: d})
}

func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	q := device.DeviceQuery{}
	
	if status := r.URL.Query().Get("status"); status != "" {
		st := device.DeviceStatus(status)
		q.Status = &st
	}
	if sku := r.URL.Query().Get("sku_code"); sku != "" {
		q.SKUCode = &sku
	}

	devices, err := h.useCase.ListDevices(r.Context(), q)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch devices")
		return
	}

	if devices == nil {
		devices = []*device.Device{}
	}

	respondJSON(w, http.StatusOK, Response{Status: "success", Data: devices})
}

func (h *DeviceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	d, err := h.useCase.GetDevice(r.Context(), id)
	if err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			respondError(w, http.StatusNotFound, "Device not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, Response{Status: "success", Data: d})
}

func (h *DeviceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var d device.Device
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	// ensure ID matches URL param
	d.DeviceID = id

	if err := h.useCase.UpdateDevice(r.Context(), &d); err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			respondError(w, http.StatusNotFound, "Device not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Fetch updated to return complete object
	updated, err := h.useCase.GetDevice(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to retrieve updated device")
		return
	}

	respondJSON(w, http.StatusOK, Response{Status: "success", Data: updated})
}

func (h *DeviceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.useCase.RemoveDevice(r.Context(), id); err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			respondError(w, http.StatusNotFound, "Device not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, Response{Status: "success"})
}
