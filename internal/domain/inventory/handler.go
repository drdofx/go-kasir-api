package inventory

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-kasir-api/internal/pkg/helpers"
)

type InventoryHandler struct {
	service *InventoryService
}

func NewInventoryHandler(service *InventoryService) *InventoryHandler {
	return &InventoryHandler{service: service}
}

type thresholdRequest struct {
	MinStock int  `json:"min_stock"`
	MaxStock int  `json:"max_stock"`
	Enabled  bool `json:"enabled"`
}

func (h *InventoryHandler) HandleAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.service.GetAlerts()
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if alerts == nil {
		alerts = []Alert{}
	}
	helpers.WriteJSON(w, http.StatusOK, alerts)
}

func (h *InventoryHandler) HandleSetThreshold(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	productID, err := strconv.Atoi(idStr)
	if err != nil || productID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "invalid product id")
		return
	}
	var req thresholdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.service.SetThreshold(Threshold{
		ProductID: productID,
		MinStock:  req.MinStock,
		MaxStock:  req.MaxStock,
		Enabled:   req.Enabled,
	}); err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"message": "threshold updated"})
}
