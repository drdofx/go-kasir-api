package product

import (
    "database/sql"
)

type Product struct {
    ID           int
    Name         string
    Price        int
    CategoryID   *int
    CategoryName string
}

type ProductStock struct {
    ProductID int
    BranchID  int
    Stock     int
}

type productRepository struct {
    db *sql.DB
}

type ProductRepository interface {
    FindAll(name string) ([]Product, error)
    FindByID(id int) (*Product, error)
    Create(p *Product) error
    Update(p *Product) error
    Delete(id int) error
    GetStocks(productID int) ([]ProductStock, error)
}

func NewProductRepository(db *sql.DB) ProductRepository {
    return &productRepository{db: db}
}

func (r *productRepository) FindAll(name string) ([]Product, error) {
    query := `SELECT p.id, p.name, p.price, p.category_id, COALESCE(c.name, '') as category_name
              FROM products p LEFT JOIN categories c ON p.category_id = c.id`
    var args []interface{}
    if name != "" {
        query += " WHERE p.name ILIKE $1"
        args = append(args, "%"+name+"%")
    }
    query += " ORDER BY p.id"
    rows, err := r.db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var products []Product
    for rows.Next() {
        var p Product
        if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.CategoryID, &p.CategoryName); err != nil {
            return nil, err
        }
        products = append(products, p)
    }
    return products, rows.Err()
}

func (r *productRepository) FindByID(id int) (*Product, error) {
    row := r.db.QueryRow(`SELECT p.id, p.name, p.price, p.category_id, COALESCE(c.name, '')
        FROM products p LEFT JOIN categories c ON p.category_id = c.id WHERE p.id = $1`, id)
    p := &Product{}
    if err := row.Scan(&p.ID, &p.Name, &p.Price, &p.CategoryID, &p.CategoryName); err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        return nil, err
    }
    return p, nil
}

func (r *productRepository) Create(p *Product) error {
    return r.db.QueryRow(
        "INSERT INTO products (name, price, category_id) VALUES ($1, $2, $3) RETURNING id",
        p.Name, p.Price, p.CategoryID,
    ).Scan(&p.ID)
}

func (r *productRepository) Update(p *Product) error {
    _, err := r.db.Exec("UPDATE products SET name=$1, price=$2, category_id=$3 WHERE id=$4",
        p.Name, p.Price, p.CategoryID, p.ID)
    return err
}

func (r *productRepository) GetStocks(productID int) ([]ProductStock, error) {
	rows, err := r.db.Query(`SELECT ps.product_id, ps.branch_id, ps.stock
		FROM product_stocks ps WHERE ps.product_id = $1 ORDER BY ps.branch_id`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var stocks []ProductStock
	for rows.Next() {
		var s ProductStock
		if err := rows.Scan(&s.ProductID, &s.BranchID, &s.Stock); err != nil {
			return nil, err
		}
		stocks = append(stocks, s)
	}
	return stocks, rows.Err()
}

func (r *productRepository) Delete(id int) error {
    _, err := r.db.Exec("DELETE FROM products WHERE id = $1", id)
    return err
}
