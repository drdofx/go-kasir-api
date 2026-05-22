package report

import (
	"net/http"
	"time"

	"go-kasir-api/internal/pkg/helpers"
)

type ReportHandler struct {
	service *ReportService
}

func NewReportHandler(service *ReportService) *ReportHandler {
	return &ReportHandler{service: service}
}

func (h *ReportHandler) HandleTodayReport(w http.ResponseWriter, r *http.Request) {
	today := time.Now().Format("2006-01-02")
	summary, err := h.service.GetSalesSummary(today, today)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, summary)
}

func (h *ReportHandler) HandleReport(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		helpers.WriteError(w, http.StatusBadRequest, "start_date and end_date are required")
		return
	}
	summary, err := h.service.GetSalesSummary(startDate, endDate)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, summary)
}
