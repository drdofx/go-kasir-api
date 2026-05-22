package transaction

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-kasir-api/internal/pkg/helpers"
)

type TransactionHandler struct {
	service *TransactionService
}

func NewTransactionHandler(service *TransactionService) *TransactionHandler {
	return &TransactionHandler{service: service}
}

type checkoutRequest struct {
	Items []CheckoutItem `json:"items"`
}

type transactionResponse struct {
	ID          int              `json:"id"`
	TotalAmount int              `json:"total_amount"`
	CreatedAt   string           `json:"created_at"`
	Details     []detailResponse `json:"details"`
}

type detailResponse struct {
	ID            int    `json:"id"`
	TransactionID int    `json:"transaction_id"`
	ProductID     int    `json:"product_id"`
	ProductName   string `json:"product_name"`
	Quantity      int    `json:"quantity"`
	Subtotal      int    `json:"subtotal"`
}

func toTransactionResponse(t Transaction) transactionResponse {
	details := make([]detailResponse, len(t.Details))
	for i, d := range t.Details {
		details[i] = detailResponse{
			ID:            d.ID,
			TransactionID: d.TransactionID,
			ProductID:     d.ProductID,
			ProductName:   d.ProductName,
			Quantity:      d.Quantity,
			Subtotal:      d.Subtotal,
		}
	}
	return transactionResponse{
		ID:          t.ID,
		TotalAmount: t.TotalAmount,
		CreatedAt:   t.CreatedAt,
		Details:     details,
	}
}

func toTransactionResponses(txns []Transaction) []transactionResponse {
	res := make([]transactionResponse, len(txns))
	for i, t := range txns {
		res[i] = toTransactionResponse(t)
	}
	return res
}

func (h *TransactionHandler) HandleCheckout(w http.ResponseWriter, r *http.Request) {
	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	checkoutReq := CheckoutRequest{Items: req.Items}
	txn, err := h.service.Checkout(checkoutReq)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, toTransactionResponse(*txn))
}

func (h *TransactionHandler) HandleTransactions(w http.ResponseWriter, r *http.Request) {
	txns, err := h.service.FindAll()
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if txns == nil {
		txns = []Transaction{}
	}
	helpers.WriteJSON(w, http.StatusOK, toTransactionResponses(txns))
}

func (h *TransactionHandler) HandleTransactionByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		helpers.WriteError(w, http.StatusBadRequest, "id is required")
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	txn, err := h.service.FindByID(id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if txn == nil {
		helpers.WriteError(w, http.StatusNotFound, "transaction not found")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, toTransactionResponse(*txn))
}
