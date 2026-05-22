package category

import "database/sql"

type Category struct {
	ID          int
	Name        string
	Description string
}

type categoryRepository struct {
	db *sql.DB
}

type CategoryRepository interface {
	FindAll() ([]Category, error)
	FindByID(id int) (*Category, error)
	Create(c *Category) error
	Update(c *Category) error
	Delete(id int) error
}

func NewCategoryRepository(db *sql.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) FindAll() ([]Category, error) {
	rows, err := r.db.Query("SELECT id, name, description FROM categories ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Description); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (r *categoryRepository) FindByID(id int) (*Category, error) {
	row := r.db.QueryRow("SELECT id, name, description FROM categories WHERE id = $1", id)
	c := &Category{}
	if err := row.Scan(&c.ID, &c.Name, &c.Description); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return c, nil
}

func (r *categoryRepository) Create(c *Category) error {
	return r.db.QueryRow("INSERT INTO categories (name, description) VALUES ($1, $2) RETURNING id",
		c.Name, c.Description).Scan(&c.ID)
}

func (r *categoryRepository) Update(c *Category) error {
	_, err := r.db.Exec("UPDATE categories SET name=$1, description=$2 WHERE id=$3",
		c.Name, c.Description, c.ID)
	return err
}

func (r *categoryRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM categories WHERE id = $1", id)
	return err
}
