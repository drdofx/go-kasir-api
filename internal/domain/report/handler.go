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

func (h *ReportHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.GetDashboard()
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, data)
}

func (h *ReportHandler) HandleWeeklyReport(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := now.AddDate(0, 0, -(weekday - 1)).Format("2006-01-02")
	end := now.Format("2006-01-02")
	summary, err := h.service.GetSalesSummary(start, end)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, summary)
}

func (h *ReportHandler) HandleMonthlyReport(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	end := now.Format("2006-01-02")
	summary, err := h.service.GetSalesSummary(start, end)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, summary)
}

func (h *ReportHandler) HandleSalesByCategory(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		helpers.WriteError(w, http.StatusBadRequest, "start_date and end_date are required")
		return
	}
	data, err := h.service.GetSalesByCategory(startDate, endDate)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if data == nil {
		data = []CategorySales{}
	}
	helpers.WriteJSON(w, http.StatusOK, data)
}

func (h *ReportHandler) HandleSalesByProduct(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		helpers.WriteError(w, http.StatusBadRequest, "start_date and end_date are required")
		return
	}
	data, err := h.service.GetSalesByProduct(startDate, endDate)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if data == nil {
		data = []ProductSales{}
	}
	helpers.WriteJSON(w, http.StatusOK, data)
}

func (h *ReportHandler) HandleExportCSV(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		helpers.WriteError(w, http.StatusBadRequest, "start_date and end_date are required")
		return
	}
	csv, err := h.service.ExportCSV(startDate, endDate)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=report.csv")
	w.Write([]byte(csv))
}
