package receipt

import (
	"database/sql"
	"fmt"
	"time"
)

type Receipt struct {
	ID            int    `json:"id"`
	TransactionID int    `json:"transaction_id"`
	ReceiptNumber string `json:"receipt_number"`
	PrintedCount  int    `json:"printed_count"`
	CreatedAt     string `json:"created_at"`
}

type receiptRepository struct {
	db *sql.DB
}

type ReceiptRepository interface {
	FindByTransactionID(transactionID int) (*Receipt, error)
	GenerateReceiptNumber(tx *sql.Tx) (string, error)
	InsertReceipt(tx *sql.Tx, transactionID int, receiptNumber string) error
}

func NewReceiptRepository(db *sql.DB) ReceiptRepository {
	return &receiptRepository{db: db}
}

func (r *receiptRepository) FindByTransactionID(transactionID int) (*Receipt, error) {
	row := r.db.QueryRow("SELECT id, transaction_id, receipt_number, printed_count, created_at FROM receipts WHERE transaction_id = $1", transactionID)
	rec := &Receipt{}
	var createdAt interface{}
	if err := row.Scan(&rec.ID, &rec.TransactionID, &rec.ReceiptNumber, &rec.PrintedCount, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	rec.CreatedAt = fmt.Sprintf("%v", createdAt)
	return rec, nil
}

func (r *receiptRepository) GenerateReceiptNumber(tx *sql.Tx) (string, error) {
	var count int
	err := tx.QueryRow("SELECT COUNT(*) FROM receipts WHERE DATE(created_at) = CURRENT_DATE").Scan(&count)
	if err != nil {
		return "", err
	}
	now := time.Now()
	return fmt.Sprintf("INV-%s-%04d", now.Format("20060102"), count+1), nil
}

func (r *receiptRepository) InsertReceipt(tx *sql.Tx, transactionID int, receiptNumber string) error {
	_, err := tx.Exec("INSERT INTO receipts (transaction_id, receipt_number) VALUES ($1, $2)", transactionID, receiptNumber)
	return err
}

