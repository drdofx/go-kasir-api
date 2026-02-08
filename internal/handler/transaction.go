package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"go-kasir-api/internal/model"
	"go-kasir-api/internal/service"
)

// TransactionHandler handles checkout requests.
type TransactionHandler struct {
	service *service.TransactionService
}

func NewTransactionHandler(service *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{service: service}
}

// HandleCheckout handles POST /api/checkout.
func (h *TransactionHandler) HandleCheckout(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.Checkout(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *TransactionHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	var req model.CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	transaction, err := h.service.Checkout(req.Items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transaction)
}

// HandleTodayReport handles GET /api/report/hari-ini.
func (h *TransactionHandler) HandleTodayReport(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.TodayReport(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *TransactionHandler) TodayReport(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.GetSalesSummary(nil, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// HandleReport handles GET /api/report?start_date=YYYY-MM-DD&end_date=YYYY-MM-DD.
func (h *TransactionHandler) HandleReport(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.Report(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *TransactionHandler) Report(w http.ResponseWriter, r *http.Request) {
	start := r.URL.Query().Get("start_date")
	end := r.URL.Query().Get("end_date")
	if start == "" || end == "" {
		http.Error(w, "start_date and end_date are required", http.StatusBadRequest)
		return
	}

	startDate, err := time.Parse("2006-01-02", start)
	if err != nil {
		http.Error(w, "Invalid start_date", http.StatusBadRequest)
		return
	}
	endDate, err := time.Parse("2006-01-02", end)
	if err != nil {
		http.Error(w, "Invalid end_date", http.StatusBadRequest)
		return
	}

	summary, err := h.service.GetSalesSummary(&startDate, &endDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}
