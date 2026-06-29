package inventory

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"go-kasir-api/internal/pkg/helpers"
	"go-kasir-api/internal/pkg/middleware"
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
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	alerts, err := h.service.GetAlertsForOrg(user.OrgID)
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
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
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
	if err := h.service.SetThresholdForOrg(user.OrgID, Threshold{
		ProductID: productID,
		MinStock:  req.MinStock,
		MaxStock:  req.MaxStock,
		Enabled:   req.Enabled,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helpers.WriteError(w, http.StatusNotFound, "product not found")
			return
		}
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"message": "threshold updated"})
}
