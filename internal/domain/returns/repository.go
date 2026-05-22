package returns

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Return struct {
	ID            int
	TransactionID int
	TotalRefund   int
	Reason        string
	CreatedAt     time.Time
	Items         []ReturnItem
}

type ReturnItem struct {
	ID        int
	ReturnID  int
	ProductID int
	ProductName string
	Quantity  int
	Subtotal  int
}

type returnRepository struct {
	db *sql.DB
}

type ReturnRepository interface {
	BeginTx() (*sql.Tx, error)
	FindAll() ([]Return, error)
	FindByID(id int) (*Return, error)
	FindTransactionDetails(tx *sql.Tx, transactionID int) ([]LockedProduct, error)
	UpdateStock(tx *sql.Tx, productID, quantity int) error
	InsertReturn(tx *sql.Tx, transactionID, totalRefund int, reason string) (int, error)
	InsertReturnItems(tx *sql.Tx, returnID int, items []ReturnItemRequest, products []LockedProduct) error
}

type LockedProduct struct {
	ID    int
	Name  string
	Price int
}

type ReturnItemRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

func NewReturnRepository(db *sql.DB) ReturnRepository {
	return &returnRepository{db: db}
}

func (r *returnRepository) BeginTx() (*sql.Tx, error) {
	return r.db.Begin()
}

func (r *returnRepository) FindAll() ([]Return, error) {
	rows, err := r.db.Query("SELECT id, transaction_id, total_refund, reason, created_at FROM returns ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var returns []Return
	for rows.Next() {
		var ret Return
		if err := rows.Scan(&ret.ID, &ret.TransactionID, &ret.TotalRefund, &ret.Reason, &ret.CreatedAt); err != nil {
			return nil, err
		}
		returns = append(returns, ret)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range returns {
		items, err := r.getItems(returns[i].ID)
		if err != nil {
			return nil, err
		}
		returns[i].Items = items
	}
	return returns, nil
}

func (r *returnRepository) FindByID(id int) (*Return, error) {
	row := r.db.QueryRow("SELECT id, transaction_id, total_refund, reason, created_at FROM returns WHERE id = $1", id)
	ret := &Return{}
	if err := row.Scan(&ret.ID, &ret.TransactionID, &ret.TotalRefund, &ret.Reason, &ret.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	items, err := r.getItems(ret.ID)
	if err != nil {
		return nil, err
	}
	ret.Items = items
	return ret, nil
}

func (r *returnRepository) getItems(returnID int) ([]ReturnItem, error) {
	rows, err := r.db.Query(`SELECT ri.id, ri.return_id, ri.product_id, COALESCE(p.name, ''), ri.quantity, ri.subtotal
		FROM return_items ri LEFT JOIN products p ON ri.product_id = p.id
		WHERE ri.return_id = $1 ORDER BY ri.id`, returnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ReturnItem
	for rows.Next() {
		var item ReturnItem
		if err := rows.Scan(&item.ID, &item.ReturnID, &item.ProductID, &item.ProductName, &item.Quantity, &item.Subtotal); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *returnRepository) FindTransactionDetails(tx *sql.Tx, transactionID int) ([]LockedProduct, error) {
	rows, err := tx.Query(`SELECT p.id, p.name, td.subtotal / td.quantity
		FROM transaction_details td JOIN products p ON td.product_id = p.id
		WHERE td.transaction_id = $1`, transactionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var products []LockedProduct
	for rows.Next() {
		var p LockedProduct
		if err := rows.Scan(&p.ID, &p.Name, &p.Price); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *returnRepository) UpdateStock(tx *sql.Tx, productID, quantity int) error {
	_, err := tx.Exec("UPDATE products SET stock = stock + $1 WHERE id = $2", quantity, productID)
	return err
}

func (r *returnRepository) InsertReturn(tx *sql.Tx, transactionID, totalRefund int, reason string) (int, error) {
	var id int
	err := tx.QueryRow("INSERT INTO returns (transaction_id, total_refund, reason) VALUES ($1, $2, $3) RETURNING id",
		transactionID, totalRefund, reason).Scan(&id)
	return id, err
}

func (r *returnRepository) InsertReturnItems(tx *sql.Tx, returnID int, items []ReturnItemRequest, products []LockedProduct) error {
	if len(items) == 0 {
		return nil
	}
	productMap := make(map[int]LockedProduct)
	for _, p := range products {
		productMap[p.ID] = p
	}
	var valueStrings []string
	var args []interface{}
	argIdx := 1
	for _, item := range items {
		p, ok := productMap[item.ProductID]
		if !ok {
			continue
		}
		subtotal := p.Price * item.Quantity
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d)", argIdx, argIdx+1, argIdx+2, argIdx+3))
		args = append(args, returnID, item.ProductID, item.Quantity, subtotal)
		argIdx += 4
	}
	if len(valueStrings) == 0 {
		return nil
	}
	query := fmt.Sprintf("INSERT INTO return_items (return_id, product_id, quantity, subtotal) VALUES %s",
		strings.Join(valueStrings, ","))
	_, err := tx.Exec(query, args...)
	return err
}
