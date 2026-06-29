package category

import "database/sql"

type Category struct {
	ID             int
	Name           string
	Description    string
	OrganizationID int
}

type categoryRepository struct {
	db *sql.DB
}

type CategoryRepository interface {
	FindAll() ([]Category, error)
	FindAllForOrg(orgID int) ([]Category, error)
	FindByID(id int) (*Category, error)
	FindByIDForOrg(orgID, id int) (*Category, error)
	ExistsForOrg(orgID, id int) (bool, error)
	Create(c *Category) error
	CreateForOrg(orgID int, c *Category) error
	Update(c *Category) error
	UpdateForOrg(orgID int, c *Category) error
	Delete(id int) error
	DeleteForOrg(orgID, id int) error
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

func (r *categoryRepository) FindAllForOrg(orgID int) ([]Category, error) {
	rows, err := r.db.Query("SELECT id, name, description, COALESCE(organization_id, 0) FROM categories WHERE organization_id = $1 ORDER BY id", orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.OrganizationID); err != nil {
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

func (r *categoryRepository) FindByIDForOrg(orgID, id int) (*Category, error) {
	row := r.db.QueryRow("SELECT id, name, description, COALESCE(organization_id, 0) FROM categories WHERE organization_id = $1 AND id = $2", orgID, id)
	c := &Category{}
	if err := row.Scan(&c.ID, &c.Name, &c.Description, &c.OrganizationID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return c, nil
}

func (r *categoryRepository) ExistsForOrg(orgID, id int) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE organization_id = $1 AND id = $2)", orgID, id).Scan(&exists)
	return exists, err
}

func (r *categoryRepository) Create(c *Category) error {
	return r.db.QueryRow("INSERT INTO categories (name, description) VALUES ($1, $2) RETURNING id",
		c.Name, c.Description).Scan(&c.ID)
}

func (r *categoryRepository) CreateForOrg(orgID int, c *Category) error {
	c.OrganizationID = orgID
	return r.db.QueryRow("INSERT INTO categories (name, description, organization_id) VALUES ($1, $2, $3) RETURNING id",
		c.Name, c.Description, orgID).Scan(&c.ID)
}

func (r *categoryRepository) Update(c *Category) error {
	_, err := r.db.Exec("UPDATE categories SET name=$1, description=$2 WHERE id=$3",
		c.Name, c.Description, c.ID)
	return err
}

func (r *categoryRepository) UpdateForOrg(orgID int, c *Category) error {
	_, err := r.db.Exec("UPDATE categories SET name=$1, description=$2 WHERE organization_id=$3 AND id=$4",
		c.Name, c.Description, orgID, c.ID)
	return err
}

func (r *categoryRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM categories WHERE id = $1", id)
	return err
}

func (r *categoryRepository) DeleteForOrg(orgID, id int) error {
	_, err := r.db.Exec("DELETE FROM categories WHERE organization_id = $1 AND id = $2", orgID, id)
	return err
}
