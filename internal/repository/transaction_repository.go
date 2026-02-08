package repository

import (
	"database/sql"
	"time"

	"go-kasir-api/internal/model"
)

// TransactionRepository handles checkout persistence.
type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (repo *TransactionRepository) BeginTx() (*sql.Tx, error) {
	return repo.db.Begin()
}

func (repo *TransactionRepository) GetProductSnapshot(tx *sql.Tx, productID int) (string, int, int, error) {
	var productName string
	var productPrice, stock int
	err := tx.QueryRow("SELECT name, price, stock FROM products WHERE id = $1", productID).
		Scan(&productName, &productPrice, &stock)
	if err != nil {
		return "", 0, 0, err
	}
	return productName, productPrice, stock, nil
}

func (repo *TransactionRepository) UpdateProductStock(tx *sql.Tx, productID int, quantity int) error {
	_, err := tx.Exec("UPDATE products SET stock = stock - $1 WHERE id = $2", quantity, productID)
	return err
}

func (repo *TransactionRepository) InsertTransaction(tx *sql.Tx, totalAmount int) (int, time.Time, error) {
	var transactionID int
	var createdAt time.Time
	err := tx.QueryRow("INSERT INTO transactions (total_amount) VALUES ($1) RETURNING id, created_at", totalAmount).
		Scan(&transactionID, &createdAt)
	if err != nil {
		return 0, time.Time{}, err
	}
	return transactionID, createdAt, nil
}

func (repo *TransactionRepository) InsertTransactionDetail(tx *sql.Tx, detail model.TransactionDetail) (int, error) {
	var id int
	err := tx.QueryRow(
		"INSERT INTO transaction_details (transaction_id, product_id, quantity, subtotal) VALUES ($1, $2, $3, $4) RETURNING id",
		detail.TransactionID, detail.ProductID, detail.Quantity, detail.Subtotal,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (repo *TransactionRepository) GetSalesSummary(startDate, endDate *time.Time) (*model.SalesSummary, error) {
	var totalRevenue int
	var totalTransactions int

	if startDate == nil || endDate == nil {
		err := repo.db.QueryRow(
			"SELECT COALESCE(SUM(total_amount), 0), COALESCE(COUNT(*), 0) FROM transactions WHERE created_at::date = CURRENT_DATE",
		).Scan(&totalRevenue, &totalTransactions)
		if err != nil {
			return nil, err
		}
	} else {
		err := repo.db.QueryRow(
			"SELECT COALESCE(SUM(total_amount), 0), COALESCE(COUNT(*), 0) FROM transactions WHERE created_at::date BETWEEN $1 AND $2",
			startDate.Format("2006-01-02"), endDate.Format("2006-01-02"),
		).Scan(&totalRevenue, &totalTransactions)
		if err != nil {
			return nil, err
		}
	}

	var topName sql.NullString
	var topQty sql.NullInt64
	if startDate == nil || endDate == nil {
		err := repo.db.QueryRow(
			`SELECT p.name, COALESCE(SUM(td.quantity), 0) AS qty
			 FROM transaction_details td
			 JOIN transactions t ON t.id = td.transaction_id
			 JOIN products p ON p.id = td.product_id
			 WHERE t.created_at::date = CURRENT_DATE
			 GROUP BY p.name
			 ORDER BY qty DESC
			 LIMIT 1`,
		).Scan(&topName, &topQty)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	} else {
		err := repo.db.QueryRow(
			`SELECT p.name, COALESCE(SUM(td.quantity), 0) AS qty
			 FROM transaction_details td
			 JOIN transactions t ON t.id = td.transaction_id
			 JOIN products p ON p.id = td.product_id
			 WHERE t.created_at::date BETWEEN $1 AND $2
			 GROUP BY p.name
			 ORDER BY qty DESC
			 LIMIT 1`,
			startDate.Format("2006-01-02"), endDate.Format("2006-01-02"),
		).Scan(&topName, &topQty)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}

	summary := &model.SalesSummary{
		TotalRevenue:     totalRevenue,
		TotalTransactions: totalTransactions,
	}
	if topName.Valid {
		summary.TopProduct = &model.TopProduct{
			Name:    topName.String,
			QtySold: int(topQty.Int64),
		}
	}

	return summary, nil
}
