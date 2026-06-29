package customer

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-kasir-api/internal/pkg/helpers"
	"go-kasir-api/internal/pkg/middleware"
)

type CustomerHandler struct {
	service *CustomerService
}

func NewCustomerHandler(service *CustomerService) *CustomerHandler {
	return &CustomerHandler{service: service}
}

type customerRequest struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Address string `json:"address"`
}

type customerResponse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Address   string `json:"address"`
	CreatedAt string `json:"created_at"`
}

type purchaseItemResponse struct {
	TransactionID int    `json:"transaction_id"`
	TotalAmount   int    `json:"total_amount"`
	CreatedAt     string `json:"created_at"`
}

type purchaseHistoryResponse struct {
	Customer  customerResponse       `json:"customer"`
	Purchases []purchaseItemResponse `json:"purchases"`
}

func toCustomerResponse(c Customer) customerResponse {
	return customerResponse{
		ID:        c.ID,
		Name:      c.Name,
		Phone:     c.Phone,
		Email:     c.Email,
		Address:   c.Address,
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func toCustomerResponses(customers []Customer) []customerResponse {
	res := make([]customerResponse, len(customers))
	for i, c := range customers {
		res[i] = toCustomerResponse(c)
	}
	return res
}

func (h *CustomerHandler) HandleCustomers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPost:
		h.create(w, r)
	default:
		helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *CustomerHandler) HandleCustomerByID(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	idStr := r.PathValue("id")

	if idStr == "" {
		helpers.WriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	// Detect /history suffix
	if len(path) > 0 && path[len(path)-8:] == "/history" {
		h.HandleCustomerHistory(w, r)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.getByID(w, r, id)
	case http.MethodPut:
		h.update(w, r, id)
	case http.MethodDelete:
		h.delete(w, r, id)
	default:
		helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *CustomerHandler) list(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	search := r.URL.Query().Get("search")
	customers, err := h.service.FindAllForOrg(user.OrgID, search)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if customers == nil {
		customers = []Customer{}
	}
	helpers.WriteJSON(w, http.StatusOK, toCustomerResponses(customers))
}

func (h *CustomerHandler) create(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req customerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c := &Customer{Name: req.Name, Phone: req.Phone, Email: req.Email, Address: req.Address}
	if err := h.service.CreateForOrg(user.OrgID, c); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, toCustomerResponse(*c))
}

func (h *CustomerHandler) getByID(w http.ResponseWriter, r *http.Request, id int) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	c, err := h.service.FindByIDForOrg(user.OrgID, id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if c == nil {
		helpers.WriteError(w, http.StatusNotFound, "customer not found")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, toCustomerResponse(*c))
}

func (h *CustomerHandler) update(w http.ResponseWriter, r *http.Request, id int) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req customerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c := &Customer{ID: id, Name: req.Name, Phone: req.Phone, Email: req.Email, Address: req.Address}
	if err := h.service.UpdateForOrg(user.OrgID, c); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	helpers.WriteJSON(w, http.StatusOK, toCustomerResponse(*c))
}

func (h *CustomerHandler) delete(w http.ResponseWriter, r *http.Request, id int) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.service.DeleteForOrg(user.OrgID, id); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CustomerHandler) HandleCustomerHistory(w http.ResponseWriter, r *http.Request) {
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
	customer, err := h.service.FindByIDForOrg(user.OrgID, id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if customer == nil {
		helpers.WriteError(w, http.StatusNotFound, "customer not found")
		return
	}
	purchases, err := h.service.GetPurchaseHistoryForOrg(user.OrgID, id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if purchases == nil {
		purchases = []PurchaseRecord{}
	}
	purchaseRes := make([]purchaseItemResponse, len(purchases))
	for i, p := range purchases {
		purchaseRes[i] = purchaseItemResponse{
			TransactionID: p.TransactionID,
			TotalAmount:   p.TotalAmount,
			CreatedAt:     p.CreatedAt,
		}
	}
	helpers.WriteJSON(w, http.StatusOK, purchaseHistoryResponse{
		Customer:  toCustomerResponse(*customer),
		Purchases: purchaseRes,
	})
}
