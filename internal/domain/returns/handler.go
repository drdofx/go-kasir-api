package returns

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-kasir-api/internal/pkg/helpers"
	"go-kasir-api/internal/pkg/middleware"
)

type ReturnHandler struct {
	service *ReturnService
}

func NewReturnHandler(service *ReturnService) *ReturnHandler {
	return &ReturnHandler{service: service}
}

type returnRequest struct {
	TransactionID int                 `json:"transaction_id"`
	Reason        string              `json:"reason"`
	Items         []ReturnItemRequest `json:"items"`
}

type returnItemResponse struct {
	ID          int    `json:"id"`
	ProductID   int    `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	Subtotal    int    `json:"subtotal"`
}

type returnResponse struct {
	ID            int                  `json:"id"`
	TransactionID int                  `json:"transaction_id"`
	TotalRefund   int                  `json:"total_refund"`
	Reason        string               `json:"reason"`
	CreatedAt     string               `json:"created_at"`
	Items         []returnItemResponse `json:"items"`
}

func toReturnItemResponse(item ReturnItem) returnItemResponse {
	return returnItemResponse{
		ID:          item.ID,
		ProductID:   item.ProductID,
		ProductName: item.ProductName,
		Quantity:    item.Quantity,
		Subtotal:    item.Subtotal,
	}
}

func toReturnResponse(r Return) returnResponse {
	items := make([]returnItemResponse, len(r.Items))
	for i, item := range r.Items {
		items[i] = toReturnItemResponse(item)
	}
	return returnResponse{
		ID:            r.ID,
		TransactionID: r.TransactionID,
		TotalRefund:   r.TotalRefund,
		Reason:        r.Reason,
		CreatedAt:     r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Items:         items,
	}
}

func toReturnResponses(returns []Return) []returnResponse {
	res := make([]returnResponse, len(returns))
	for i, r := range returns {
		res[i] = toReturnResponse(r)
	}
	return res
}

func (h *ReturnHandler) HandleReturns(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.HandleListReturns(w, r)
	case http.MethodPost:
		h.HandleCreateReturn(w, r)
	default:
		helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ReturnHandler) HandleCreateReturn(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if user.BranchID == nil {
		helpers.WriteError(w, http.StatusBadRequest, "no active branch")
		return
	}
	var req returnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TransactionID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "transaction_id is required")
		return
	}
	ret, err := h.service.ProcessReturnForOrg(user.OrgID, *user.BranchID, req.TransactionID, req.Reason, req.Items)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, toReturnResponse(*ret))
}

func (h *ReturnHandler) HandleListReturns(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	returns, err := h.service.FindAllForOrg(user.OrgID)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if returns == nil {
		returns = []Return{}
	}
	pagination, err := helpers.ParsePagination(r)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	helpers.WritePaginationHeaders(w, pagination, len(returns))
	returns = helpers.Paginate(returns, pagination)
	helpers.WriteJSON(w, http.StatusOK, toReturnResponses(returns))
}

func (h *ReturnHandler) HandleGetReturn(w http.ResponseWriter, r *http.Request) {
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
	ret, err := h.service.FindByIDForOrg(user.OrgID, id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if ret == nil {
		helpers.WriteError(w, http.StatusNotFound, "return not found")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, toReturnResponse(*ret))
}
