package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"go-kasir-api/internal/model"
	"go-kasir-api/internal/service"

	"github.com/rs/zerolog/log"
)

type TransactionHandler struct {
	service *service.TransactionService
}

func NewTransactionHandler(service *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{service: service}
}

func (h *TransactionHandler) HandleCheckout(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.Checkout(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *TransactionHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	var req model.CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error().Err(err).Msg("checkout: invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	transaction, err := h.service.Checkout(r.Context(), req.Items)
	if err != nil {
		log.Error().Err(err).Msg("checkout: failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info().Int("transaction_id", transaction.ID).Int("total", transaction.TotalAmount).Msg("checkout: success")
	jsonResponse(w, http.StatusOK, transaction)
}

func (h *TransactionHandler) HandleTodayReport(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.TodayReport(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *TransactionHandler) TodayReport(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.GetSalesSummary(r.Context(), nil, nil)
	if err != nil {
		log.Error().Err(err).Msg("report: failed to get today's summary")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, summary)
}

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

	summary, err := h.service.GetSalesSummary(r.Context(), &startDate, &endDate)
	if err != nil {
		log.Error().Err(err).Msg("report: failed to get summary")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, summary)
}
