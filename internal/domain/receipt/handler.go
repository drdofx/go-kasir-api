package receipt

import (
	"net/http"
	"strconv"

	"go-kasir-api/internal/pkg/helpers"
)

type ReceiptHandler struct {
	service *ReceiptService
}

func NewReceiptHandler(service *ReceiptService) *ReceiptHandler {
	return &ReceiptHandler{service: service}
}

func (h *ReceiptHandler) HandleGetReceipt(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}
	rec, err := h.service.GetReceipt(id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if rec == nil {
		helpers.WriteError(w, http.StatusNotFound, "receipt not found")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, rec)
}
