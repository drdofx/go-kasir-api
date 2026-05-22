package customer

import (
	"database/sql"
	"fmt"
	"time"
)

type Customer struct {
	ID        int
	Name      string
	Phone     string
	Email     string
	Address   string
	CreatedAt time.Time
}

type PurchaseRecord struct {
	TransactionID int
	TotalAmount   int
	CreatedAt     string
}

type customerRepository struct {
	db *sql.DB
}

type CustomerRepository interface {
	FindAll(search string) ([]Customer, error)
	FindByID(id int) (*Customer, error)
	Create(c *Customer) error
	Update(c *Customer) error
	Delete(id int) error
	GetPurchaseHistory(customerID int) ([]PurchaseRecord, error)
}

func NewCustomerRepository(db *sql.DB) CustomerRepository {
	return &customerRepository{db: db}
}

func (r *customerRepository) FindAll(search string) ([]Customer, error) {
	query := "SELECT id, name, phone, email, address, created_at FROM customers"
	var args []interface{}
	if search != "" {
		query += " WHERE name ILIKE $1 OR phone ILIKE $1"
		args = append(args, "%"+search+"%")
	}
	query += " ORDER BY id"
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var customers []Customer
	for rows.Next() {
		var c Customer
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.Address, &c.CreatedAt); err != nil {
			return nil, err
		}
		customers = append(customers, c)
	}
	return customers, rows.Err()
}

func (r *customerRepository) FindByID(id int) (*Customer, error) {
	row := r.db.QueryRow("SELECT id, name, phone, email, address, created_at FROM customers WHERE id = $1", id)
	c := &Customer{}
	if err := row.Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.Address, &c.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return c, nil
}

func (r *customerRepository) Create(c *Customer) error {
	return r.db.QueryRow(
		"INSERT INTO customers (name, phone, email, address) VALUES ($1, $2, $3, $4) RETURNING id, created_at",
		c.Name, c.Phone, c.Email, c.Address,
	).Scan(&c.ID, &c.CreatedAt)
}

func (r *customerRepository) Update(c *Customer) error {
	_, err := r.db.Exec("UPDATE customers SET name=$1, phone=$2, email=$3, address=$4 WHERE id=$5",
		c.Name, c.Phone, c.Email, c.Address, c.ID)
	return err
}

func (r *customerRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM customers WHERE id = $1", id)
	return err
}

func (r *customerRepository) GetPurchaseHistory(customerID int) ([]PurchaseRecord, error) {
	rows, err := r.db.Query("SELECT id, total_amount, created_at FROM transactions WHERE customer_id = $1 ORDER BY created_at DESC", customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []PurchaseRecord
	for rows.Next() {
		var pr PurchaseRecord
		var createdAt interface{}
		if err := rows.Scan(&pr.TransactionID, &pr.TotalAmount, &createdAt); err != nil {
			return nil, err
		}
		pr.CreatedAt = fmt.Sprintf("%v", createdAt)
		records = append(records, pr)
	}
	return records, rows.Err()
}
