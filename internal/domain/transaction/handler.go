package transaction

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-kasir-api/internal/pkg/helpers"
	"go-kasir-api/internal/pkg/middleware"
)

type TransactionHandler struct {
	service *TransactionService
}

func NewTransactionHandler(service *TransactionService) *TransactionHandler {
	return &TransactionHandler{service: service}
}

type checkoutPayment struct {
	Type   string `json:"type"`
	Amount int    `json:"amount"`
}

type checkoutRequest struct {
	Items      []CheckoutItem    `json:"items"`
	CustomerID *int              `json:"customer_id"`
	Payments   []checkoutPayment `json:"payments"`
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
	user := middleware.UserFromContext(r.Context())
	if user == nil || user.BranchID == nil {
		helpers.WriteError(w, http.StatusBadRequest, "no active branch")
		return
	}
	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	checkoutReq := CheckoutRequest{
		Items:      req.Items,
		CustomerID: req.CustomerID,
		OrgID:      user.OrgID,
		BranchID:   *user.BranchID,
	}
	for _, p := range req.Payments {
		checkoutReq.Payments = append(checkoutReq.Payments, CheckoutPayment{
			Type:   p.Type,
			Amount: p.Amount,
		})
	}
	txn, err := h.service.Checkout(checkoutReq)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, toTransactionResponse(*txn))
}

func (h *TransactionHandler) HandleTransactions(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	txns, err := h.service.FindAllForOrg(user.OrgID)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if txns == nil {
		txns = []Transaction{}
	}
	pagination, err := helpers.ParsePagination(r)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	helpers.WritePaginationHeaders(w, pagination, len(txns))
	txns = helpers.Paginate(txns, pagination)
	helpers.WriteJSON(w, http.StatusOK, toTransactionResponses(txns))
}

func (h *TransactionHandler) HandleTransactionByID(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
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
	txn, err := h.service.FindByIDForOrg(user.OrgID, id)
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
