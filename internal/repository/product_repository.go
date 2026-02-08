package repository

import (
	"database/sql"
	"errors"

	"go-kasir-api/internal/model"
)

var ErrProductNotFound = errors.New("product not found")

// ProductRepository handles product persistence.
type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (repo *ProductRepository) GetAll(nameFilter string) ([]model.Product, error) {
	query := `
		SELECT p.id, p.name, p.price, p.stock, p.category_id, c.name
		FROM products p
		LEFT JOIN categories c ON c.id = p.category_id
	`
	args := []interface{}{}
	if nameFilter != "" {
		query += " WHERE p.name ILIKE $1"
		args = append(args, "%"+nameFilter+"%")
	}

	rows, err := repo.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]model.Product, 0)
	for rows.Next() {
		var p model.Product
		var categoryID sql.NullInt64
		var categoryName sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Stock, &categoryID, &categoryName); err != nil {
			return nil, err
		}
		if categoryID.Valid {
			p.CategoryID = int(categoryID.Int64)
		}
		if categoryName.Valid {
			p.CategoryName = categoryName.String
		}
		products = append(products, p)
	}

	return products, nil
}

func (repo *ProductRepository) Create(product *model.Product) error {
	query := "INSERT INTO products (name, price, stock, category_id) VALUES ($1, $2, $3, $4) RETURNING id"
	return repo.db.QueryRow(query, product.Name, product.Price, product.Stock, product.CategoryID).Scan(&product.ID)
}

func (repo *ProductRepository) GetByID(id int) (*model.Product, error) {
	query := `
		SELECT p.id, p.name, p.price, p.stock, p.category_id, c.name
		FROM products p
		LEFT JOIN categories c ON c.id = p.category_id
		WHERE p.id = $1
	`

	var p model.Product
	var categoryID sql.NullInt64
	var categoryName sql.NullString
	err := repo.db.QueryRow(query, id).Scan(&p.ID, &p.Name, &p.Price, &p.Stock, &categoryID, &categoryName)
	if err == sql.ErrNoRows {
		return nil, ErrProductNotFound
	}
	if err != nil {
		return nil, err
	}
	if categoryID.Valid {
		p.CategoryID = int(categoryID.Int64)
	}
	if categoryName.Valid {
		p.CategoryName = categoryName.String
	}

	return &p, nil
}

func (repo *ProductRepository) Update(product *model.Product) error {
	query := "UPDATE products SET name = $1, price = $2, stock = $3, category_id = $4 WHERE id = $5"
	result, err := repo.db.Exec(query, product.Name, product.Price, product.Stock, product.CategoryID, product.ID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrProductNotFound
	}

	return nil
}

func (repo *ProductRepository) Delete(id int) error {
	query := "DELETE FROM products WHERE id = $1"
	result, err := repo.db.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrProductNotFound
	}

	return nil
}
