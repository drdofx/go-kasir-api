package payment

import (
	"net/http"

	"go-kasir-api/internal/pkg/helpers"
)

type PaymentHandler struct {
	service *PaymentService
}

func NewPaymentHandler(service *PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

type paymentTypeResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func toPaymentTypeResponses(types []PaymentType) []paymentTypeResponse {
	res := make([]paymentTypeResponse, len(types))
	for i, pt := range types {
		res[i] = paymentTypeResponse{ID: pt.ID, Name: pt.Name}
	}
	return res
}

func (h *PaymentHandler) HandlePaymentTypes(w http.ResponseWriter, r *http.Request) {
	types, err := h.service.GetPaymentTypes()
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if types == nil {
		types = []PaymentType{}
	}
	helpers.WriteJSON(w, http.StatusOK, toPaymentTypeResponses(types))
}
