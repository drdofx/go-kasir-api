package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go-kasir-api/internal/model"
)

type TxOptions = sql.TxOptions

type Transactor interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Commit() error
	Rollback() error
}

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (repo *TransactionRepository) BeginTx(ctx context.Context, opts *sql.TxOptions) (Transactor, error) {
	return repo.db.BeginTx(ctx, opts)
}

func (repo *TransactionRepository) GetProductSnapshots(ctx context.Context, tx Transactor, productIDs []int) (map[int]model.ProductSnapshot, error) {
	if len(productIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(productIDs))
	args := make([]any, len(productIDs))
	for i, id := range productIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(
		"SELECT id, name, price, stock FROM products WHERE id IN (%s) FOR UPDATE",
		strings.Join(placeholders, ","),
	)

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]model.ProductSnapshot, len(productIDs))
	for rows.Next() {
		var s model.ProductSnapshot
		if err := rows.Scan(&s.ID, &s.Name, &s.Price, &s.Stock); err != nil {
			return nil, err
		}
		result[s.ID] = s
	}
	return result, rows.Err()
}

func (repo *TransactionRepository) BatchUpdateStock(ctx context.Context, tx Transactor, items []model.CheckoutItem) error {
	if len(items) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("UPDATE products SET stock = stock - (CASE id ")
	args := make([]any, 0, len(items)*3)
	idx := 1
	ids := make([]string, len(items))
	for i, item := range items {
		sb.WriteString(fmt.Sprintf("WHEN $%d THEN $%d::int ", idx, idx+1))
		args = append(args, item.ProductID, item.Quantity)
		ids[i] = fmt.Sprintf("$%d", idx)
		idx += 2
	}
	sb.WriteString("END) WHERE id IN (")
	sb.WriteString(strings.Join(ids, ","))
	sb.WriteString(")")

	_, err := tx.ExecContext(ctx, sb.String(), args...)
	return err
}

func (repo *TransactionRepository) InsertTransaction(ctx context.Context, tx Transactor, totalAmount int) (int, time.Time, error) {
	var transactionID int
	var createdAt time.Time
	err := tx.QueryRowContext(ctx, "INSERT INTO transactions (total_amount) VALUES ($1) RETURNING id, created_at", totalAmount).
		Scan(&transactionID, &createdAt)
	if err != nil {
		return 0, time.Time{}, err
	}
	return transactionID, createdAt, nil
}

func (repo *TransactionRepository) BatchInsertDetails(ctx context.Context, tx Transactor, details []model.TransactionDetail) ([]int, error) {
	if len(details) == 0 {
		return nil, nil
	}

	var sb strings.Builder
	sb.WriteString("INSERT INTO transaction_details (transaction_id, product_id, quantity, subtotal) VALUES ")
	args := make([]any, 0, len(details)*4)
	idx := 1
	for i, d := range details {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("($%d,$%d,$%d,$%d)", idx, idx+1, idx+2, idx+3))
		args = append(args, d.TransactionID, d.ProductID, d.Quantity, d.Subtotal)
		idx += 4
	}
	sb.WriteString(" RETURNING id")

	rows, err := tx.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int, 0, len(details))
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (repo *TransactionRepository) GetSalesSummary(ctx context.Context, startDate, endDate *time.Time) (*model.SalesSummary, error) {
	var totalRevenue int
	var totalTransactions int

	if startDate == nil || endDate == nil {
		err := repo.db.QueryRowContext(ctx,
			"SELECT COALESCE(SUM(total_amount), 0), COALESCE(COUNT(*), 0) FROM transactions WHERE created_at::date = CURRENT_DATE",
		).Scan(&totalRevenue, &totalTransactions)
		if err != nil {
			return nil, err
		}
	} else {
		err := repo.db.QueryRowContext(ctx,
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
		err := repo.db.QueryRowContext(ctx,
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
		err := repo.db.QueryRowContext(ctx,
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
		TotalRevenue:      totalRevenue,
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
