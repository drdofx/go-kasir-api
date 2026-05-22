package transaction

import (
	"database/sql"
	"fmt"
	"strings"
)

type Transaction struct {
	ID          int
	TotalAmount int
	CreatedAt   string
	Details     []DetailItem
}

type DetailItem struct {
	ID            int
	TransactionID int
	ProductID     int
	ProductName   string
	Quantity      int
	Subtotal      int
}

type LockedProduct struct {
	ID    int
	Name  string
	Price int
	Stock int
}

type transactionRepository struct {
	db *sql.DB
}

type TransactionRepository interface {
	BeginTx() (*sql.Tx, error)
	FindAll() ([]Transaction, error)
	FindByID(id int) (*Transaction, error)
	LockProducts(tx *sql.Tx, ids []int) ([]LockedProduct, error)
	UpdateStock(tx *sql.Tx, id, qty int) error
	InsertTransaction(tx *sql.Tx, total int, customerID *int) (int, error)
	InsertDetails(tx *sql.Tx, transactionID int, items []CheckoutItem, products []LockedProduct) error
	InsertPayment(tx *sql.Tx, transactionID, paymentTypeID, amount int) error
}

func NewTransactionRepository(db *sql.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) BeginTx() (*sql.Tx, error) {
	return r.db.Begin()
}

func (r *transactionRepository) FindAll() ([]Transaction, error) {
	rows, err := r.db.Query("SELECT id, total_amount, created_at FROM transactions ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var txns []Transaction
	for rows.Next() {
		var t Transaction
		var createdAt interface{}
		if err := rows.Scan(&t.ID, &t.TotalAmount, &createdAt); err != nil {
			return nil, err
		}
		t.CreatedAt = fmt.Sprintf("%v", createdAt)
		txns = append(txns, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range txns {
		details, err := r.getDetails(txns[i].ID)
		if err != nil {
			return nil, err
		}
		txns[i].Details = details
	}
	return txns, nil
}

func (r *transactionRepository) FindByID(id int) (*Transaction, error) {
	row := r.db.QueryRow("SELECT id, total_amount, created_at FROM transactions WHERE id = $1", id)
	t := &Transaction{}
	var createdAt interface{}
	if err := row.Scan(&t.ID, &t.TotalAmount, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.CreatedAt = fmt.Sprintf("%v", createdAt)
	details, err := r.getDetails(t.ID)
	if err != nil {
		return nil, err
	}
	t.Details = details
	return t, nil
}

func (r *transactionRepository) getDetails(transactionID int) ([]DetailItem, error) {
	rows, err := r.db.Query(`SELECT td.id, td.transaction_id, td.product_id, COALESCE(p.name, ''), td.quantity, td.subtotal
		FROM transaction_details td LEFT JOIN products p ON td.product_id = p.id
		WHERE td.transaction_id = $1 ORDER BY td.id`, transactionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var details []DetailItem
	for rows.Next() {
		var d DetailItem
		if err := rows.Scan(&d.ID, &d.TransactionID, &d.ProductID, &d.ProductName, &d.Quantity, &d.Subtotal); err != nil {
			return nil, err
		}
		details = append(details, d)
	}
	return details, rows.Err()
}

func (r *transactionRepository) LockProducts(tx *sql.Tx, ids []int) ([]LockedProduct, error) {
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query := fmt.Sprintf("SELECT id, name, price, stock FROM products WHERE id IN (%s) ORDER BY id", strings.Join(placeholders, ","))
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var products []LockedProduct
	for rows.Next() {
		var p LockedProduct
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Stock); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *transactionRepository) UpdateStock(tx *sql.Tx, id, qty int) error {
	_, err := tx.Exec("UPDATE products SET stock = stock - $1 WHERE id = $2 AND stock >= $1", qty, id)
	return err
}

func (r *transactionRepository) InsertTransaction(tx *sql.Tx, total int, customerID *int) (int, error) {
	var id int
	if customerID != nil && *customerID > 0 {
		err := tx.QueryRow("INSERT INTO transactions (total_amount, customer_id) VALUES ($1, $2) RETURNING id", total, *customerID).Scan(&id)
		return id, err
	}
	err := tx.QueryRow("INSERT INTO transactions (total_amount) VALUES ($1) RETURNING id", total).Scan(&id)
	return id, err
}

func (r *transactionRepository) InsertPayment(tx *sql.Tx, transactionID, paymentTypeID, amount int) error {
	_, err := tx.Exec("INSERT INTO transaction_payments (transaction_id, payment_type_id, amount) VALUES ($1, $2, $3)",
		transactionID, paymentTypeID, amount)
	return err
}

func (r *transactionRepository) InsertDetails(tx *sql.Tx, transactionID int, items []CheckoutItem, products []LockedProduct) error {
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
		args = append(args, transactionID, item.ProductID, item.Quantity, subtotal)
		argIdx += 4
	}
	query := fmt.Sprintf("INSERT INTO transaction_details (transaction_id, product_id, quantity, subtotal) VALUES %s",
		strings.Join(valueStrings, ","))
	_, err := tx.Exec(query, args...)
	return err
}
