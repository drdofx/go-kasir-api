package purchaseorder

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-kasir-api/internal/pkg/helpers"
	"go-kasir-api/internal/pkg/middleware"
)

type POHandler struct {
	service *POService
}

func NewPOHandler(service *POService) *POHandler {
	return &POHandler{service: service}
}

type poRequest struct {
	SupplierID int             `json:"supplier_id"`
	Items      []POItemRequest `json:"items"`
}

type poItemResponse struct {
	ID          int    `json:"id"`
	ProductID   int    `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int    `json:"unit_price"`
	Subtotal    int    `json:"subtotal"`
}

type poResponse struct {
	ID           int              `json:"id"`
	SupplierID   int              `json:"supplier_id"`
	SupplierName string           `json:"supplier_name"`
	Status       string           `json:"status"`
	TotalAmount  int              `json:"total_amount"`
	CreatedAt    string           `json:"created_at"`
	Items        []poItemResponse `json:"items"`
}

func toPOItemResponse(it POItem) poItemResponse {
	return poItemResponse{
		ID: it.ID, ProductID: it.ProductID, ProductName: it.ProductName,
		Quantity: it.Quantity, UnitPrice: it.UnitPrice, Subtotal: it.Subtotal,
	}
}

func toPOResponse(po PurchaseOrder) poResponse {
	items := make([]poItemResponse, len(po.Items))
	for i, it := range po.Items {
		items[i] = toPOItemResponse(it)
	}
	return poResponse{
		ID: po.ID, SupplierID: po.SupplierID, SupplierName: po.SupplierName,
		Status: po.Status, TotalAmount: po.TotalAmount,
		CreatedAt: po.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Items:     items,
	}
}

func toPOResponses(pos []PurchaseOrder) []poResponse {
	res := make([]poResponse, len(pos))
	for i, po := range pos {
		res[i] = toPOResponse(po)
	}
	return res
}

func (h *POHandler) HandlePOs(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	switch r.Method {
	case http.MethodGet:
		pos, err := h.service.FindAllForOrg(user.OrgID)
		if err != nil {
			helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if pos == nil {
			pos = []PurchaseOrder{}
		}
		helpers.WriteJSON(w, http.StatusOK, toPOResponses(pos))
	case http.MethodPost:
		if user.BranchID == nil {
			helpers.WriteError(w, http.StatusBadRequest, "no active branch")
			return
		}
		var req poRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.SupplierID <= 0 {
			helpers.WriteError(w, http.StatusBadRequest, "supplier_id is required")
			return
		}
		po, err := h.service.CreatePOForOrg(user.OrgID, *user.BranchID, req.SupplierID, req.Items)
		if err != nil {
			helpers.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		helpers.WriteJSON(w, http.StatusCreated, toPOResponse(*po))
	default:
		helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *POHandler) HandlePOByID(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	path := r.URL.Path
	idStr := r.PathValue("id")
	if idStr == "" {
		helpers.WriteError(w, http.StatusBadRequest, "id is required")
		return
	}
	if len(path) > 8 && path[len(path)-8:] == "/receive" {
		h.HandleReceivePO(w, r)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	po, err := h.service.FindByIDForOrg(user.OrgID, id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if po == nil {
		helpers.WriteError(w, http.StatusNotFound, "purchase order not found")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, toPOResponse(*po))
}

func (h *POHandler) HandleReceivePO(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if user.BranchID == nil {
		helpers.WriteError(w, http.StatusBadRequest, "no active branch")
		return
	}
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	po, err := h.service.ReceivePOForOrg(user.OrgID, *user.BranchID, id)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	helpers.WriteJSON(w, http.StatusOK, toPOResponse(*po))
}
