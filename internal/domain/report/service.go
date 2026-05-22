package report

import (
	"database/sql"
)

type SalesSummary struct {
	TotalRevenue      int         `json:"total_revenue"`
	TotalTransactions int         `json:"total_transactions"`
	TopProduct        *TopProduct `json:"top_product,omitempty"`
}

type TopProduct struct {
	Name    string `json:"name"`
	QtySold int    `json:"qty_sold"`
}

type ReportService struct {
	db *sql.DB
}

func NewReportService(db *sql.DB) *ReportService {
	return &ReportService{db: db}
}

func (s *ReportService) GetSalesSummary(startDate, endDate string) (*SalesSummary, error) {
	summary := &SalesSummary{}
	err := s.db.QueryRow(`
		SELECT COALESCE(SUM(total_amount), 0), COUNT(*)
		FROM transactions WHERE DATE(created_at) >= $1 AND DATE(created_at) <= $2`,
		startDate, endDate).Scan(&summary.TotalRevenue, &summary.TotalTransactions)
	if err != nil {
		return nil, err
	}
	top := &TopProduct{}
	err = s.db.QueryRow(`
		SELECT COALESCE(p.name, ''), COALESCE(SUM(td.quantity), 0)
		FROM transaction_details td
		JOIN products p ON td.product_id = p.id
		JOIN transactions t ON td.transaction_id = t.id
		WHERE DATE(t.created_at) >= $1 AND DATE(t.created_at) <= $2
		GROUP BY p.name
		ORDER BY SUM(td.quantity) DESC LIMIT 1`,
		startDate, endDate).Scan(&top.Name, &top.QtySold)
	if err == sql.ErrNoRows || top.Name == "" {
		summary.TopProduct = nil
	} else if err != nil {
		return nil, err
	} else {
		summary.TopProduct = top
	}
	return summary, nil
}
