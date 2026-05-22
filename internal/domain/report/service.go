package report

import (
	"database/sql"
	"fmt"
	"strings"
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

type DashboardData struct {
	TodayRevenue      int            `json:"today_revenue"`
	TodayTransactions int            `json:"today_transactions"`
	TotalProducts     int            `json:"total_products"`
	LowStockCount     int            `json:"low_stock_count"`
	TopProducts       []TopProduct   `json:"top_products"`
	RecentSales       []SaleItem     `json:"recent_sales"`
}

type SaleItem struct {
	ID          int    `json:"id"`
	TotalAmount int    `json:"total_amount"`
	CreatedAt   string `json:"created_at"`
}

type CategorySales struct {
	CategoryName string `json:"category_name"`
	TotalSales   int    `json:"total_sales"`
	ItemCount    int    `json:"item_count"`
}

type ProductSales struct {
	ProductName string `json:"product_name"`
	CategoryName string `json:"category_name"`
	QtySold     int    `json:"qty_sold"`
	TotalSales  int    `json:"total_sales"`
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

func (s *ReportService) GetDashboard() (*DashboardData, error) {
	d := &DashboardData{}
	s.db.QueryRow("SELECT COALESCE(SUM(total_amount),0), COUNT(*) FROM transactions WHERE DATE(created_at) = CURRENT_DATE").
		Scan(&d.TodayRevenue, &d.TodayTransactions)
	s.db.QueryRow("SELECT COUNT(*) FROM products").Scan(&d.TotalProducts)
	s.db.QueryRow(`SELECT COUNT(*) FROM products p LEFT JOIN inventory_alerts ia ON p.id = ia.product_id
		WHERE ia.enabled = true AND p.stock <= ia.min_stock`).Scan(&d.LowStockCount)
	rows, err := s.db.Query(`SELECT COALESCE(p.name,''), COALESCE(SUM(td.quantity),0)
		FROM transaction_details td JOIN products p ON td.product_id = p.id
		JOIN transactions t ON td.transaction_id = t.id WHERE DATE(t.created_at) = CURRENT_DATE
		GROUP BY p.name ORDER BY SUM(td.quantity) DESC LIMIT 5`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tp TopProduct
			if rows.Scan(&tp.Name, &tp.QtySold) == nil {
				d.TopProducts = append(d.TopProducts, tp)
			}
		}
	}
	if d.TopProducts == nil {
		d.TopProducts = []TopProduct{}
	}
	saleRows, err := s.db.Query("SELECT id, total_amount, created_at FROM transactions WHERE DATE(created_at) = CURRENT_DATE ORDER BY created_at DESC LIMIT 10")
	if err == nil {
		defer saleRows.Close()
		for saleRows.Next() {
			var si SaleItem
			var createdAt interface{}
			if saleRows.Scan(&si.ID, &si.TotalAmount, &createdAt) == nil {
				si.CreatedAt = fmt.Sprintf("%v", createdAt)
				d.RecentSales = append(d.RecentSales, si)
			}
		}
	}
	if d.RecentSales == nil {
		d.RecentSales = []SaleItem{}
	}
	return d, nil
}

func (s *ReportService) GetSalesByCategory(startDate, endDate string) ([]CategorySales, error) {
	rows, err := s.db.Query(`SELECT COALESCE(c.name, 'Uncategorized'), COALESCE(SUM(td.subtotal), 0), COUNT(td.id)
		FROM transaction_details td JOIN transactions t ON td.transaction_id = t.id
		LEFT JOIN products p ON td.product_id = p.id
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE DATE(t.created_at) >= $1 AND DATE(t.created_at) <= $2
		GROUP BY c.name ORDER BY SUM(td.subtotal) DESC`, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []CategorySales
	for rows.Next() {
		var cs CategorySales
		if err := rows.Scan(&cs.CategoryName, &cs.TotalSales, &cs.ItemCount); err != nil {
			return nil, err
		}
		res = append(res, cs)
	}
	return res, rows.Err()
}

func (s *ReportService) GetSalesByProduct(startDate, endDate string) ([]ProductSales, error) {
	rows, err := s.db.Query(`SELECT COALESCE(p.name,''), COALESCE(c.name,''), COALESCE(SUM(td.quantity),0), COALESCE(SUM(td.subtotal),0)
		FROM transaction_details td JOIN transactions t ON td.transaction_id = t.id
		JOIN products p ON td.product_id = p.id
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE DATE(t.created_at) >= $1 AND DATE(t.created_at) <= $2
		GROUP BY p.name, c.name ORDER BY SUM(td.subtotal) DESC`, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []ProductSales
	for rows.Next() {
		var ps ProductSales
		if err := rows.Scan(&ps.ProductName, &ps.CategoryName, &ps.QtySold, &ps.TotalSales); err != nil {
			return nil, err
		}
		res = append(res, ps)
	}
	return res, rows.Err()
}

func (s *ReportService) ExportCSV(startDate, endDate string) (string, error) {
	rows, err := s.db.Query(`SELECT t.id, t.total_amount, t.created_at, COALESCE(c.name, ''), COALESCE(cus.name, '')
		FROM transactions t LEFT JOIN customers cus ON t.customer_id = cus.id
		LEFT JOIN transaction_payments tp ON tp.transaction_id = t.id
		LEFT JOIN payment_types pt ON tp.payment_type_id = pt.id
		LEFT JOIN customers c ON c.id = t.customer_id
		WHERE DATE(t.created_at) >= $1 AND DATE(t.created_at) <= $2
		ORDER BY t.created_at DESC`, startDate, endDate)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var sb strings.Builder
	sb.WriteString("ID,Total,Created At,Customer,Payment\n")
	for rows.Next() {
		var id, total int
		var createdAt, customer, payment interface{}
		if err := rows.Scan(&id, &total, &createdAt, &customer, &payment); err != nil {
			return "", err
		}
		sb.WriteString(fmt.Sprintf("%d,%d,%v,%v,%v\n", id, total, createdAt, customer, payment))
	}
	return sb.String(), rows.Err()
}
