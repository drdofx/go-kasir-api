package payment

import "database/sql"

type PaymentType struct {
	ID   int
	Name string
}

type TransactionPayment struct {
	TransactionID int
	PaymentTypeID int
	Amount        int
}

type paymentRepository struct {
	db *sql.DB
}

type PaymentRepository interface {
	FindAllPaymentTypes() ([]PaymentType, error)
	InsertPayment(tx *sql.Tx, transactionID, paymentTypeID, amount int) error
}

func NewPaymentRepository(db *sql.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) FindAllPaymentTypes() ([]PaymentType, error) {
	rows, err := r.db.Query("SELECT id, name FROM payment_types ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var types []PaymentType
	for rows.Next() {
		var pt PaymentType
		if err := rows.Scan(&pt.ID, &pt.Name); err != nil {
			return nil, err
		}
		types = append(types, pt)
	}
	return types, rows.Err()
}

func (r *paymentRepository) InsertPayment(tx *sql.Tx, transactionID, paymentTypeID, amount int) error {
	_, err := tx.Exec("INSERT INTO transaction_payments (transaction_id, payment_type_id, amount) VALUES ($1, $2, $3)",
		transactionID, paymentTypeID, amount)
	return err
}
