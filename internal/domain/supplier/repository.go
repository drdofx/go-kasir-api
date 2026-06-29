package supplier

import (
	"database/sql"
	"time"
)

type Supplier struct {
	ID             int
	Name           string
	ContactPerson  string
	Phone          string
	Email          string
	Address        string
	OrganizationID int
	CreatedAt      time.Time
}

type supplierRepository struct {
	db *sql.DB
}

type SupplierRepository interface {
	FindAll(search string) ([]Supplier, error)
	FindAllForOrg(orgID int, search string) ([]Supplier, error)
	FindByID(id int) (*Supplier, error)
	FindByIDForOrg(orgID, id int) (*Supplier, error)
	Create(s *Supplier) error
	CreateForOrg(orgID int, s *Supplier) error
	Update(s *Supplier) error
	UpdateForOrg(orgID int, s *Supplier) error
	Delete(id int) error
	DeleteForOrg(orgID, id int) error
}

func NewSupplierRepository(db *sql.DB) SupplierRepository {
	return &supplierRepository{db: db}
}

func (r *supplierRepository) FindAll(search string) ([]Supplier, error) {
	q := "SELECT id, name, contact_person, phone, email, address, created_at FROM suppliers"
	var args []interface{}
	if search != "" {
		q += " WHERE name ILIKE $1"
		args = append(args, "%"+search+"%")
	}
	q += " ORDER BY id"
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ss []Supplier
	for rows.Next() {
		var s Supplier
		if err := rows.Scan(&s.ID, &s.Name, &s.ContactPerson, &s.Phone, &s.Email, &s.Address, &s.CreatedAt); err != nil {
			return nil, err
		}
		ss = append(ss, s)
	}
	return ss, rows.Err()
}

func (r *supplierRepository) FindAllForOrg(orgID int, search string) ([]Supplier, error) {
	q := "SELECT id, name, contact_person, phone, email, address, COALESCE(organization_id, 0), created_at FROM suppliers WHERE organization_id = $1"
	args := []interface{}{orgID}
	if search != "" {
		q += " AND name ILIKE $2"
		args = append(args, "%"+search+"%")
	}
	q += " ORDER BY id"
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ss []Supplier
	for rows.Next() {
		var s Supplier
		if err := rows.Scan(&s.ID, &s.Name, &s.ContactPerson, &s.Phone, &s.Email, &s.Address, &s.OrganizationID, &s.CreatedAt); err != nil {
			return nil, err
		}
		ss = append(ss, s)
	}
	return ss, rows.Err()
}

func (r *supplierRepository) FindByID(id int) (*Supplier, error) {
	row := r.db.QueryRow("SELECT id, name, contact_person, phone, email, address, created_at FROM suppliers WHERE id = $1", id)
	s := &Supplier{}
	if err := row.Scan(&s.ID, &s.Name, &s.ContactPerson, &s.Phone, &s.Email, &s.Address, &s.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

func (r *supplierRepository) FindByIDForOrg(orgID, id int) (*Supplier, error) {
	row := r.db.QueryRow("SELECT id, name, contact_person, phone, email, address, COALESCE(organization_id, 0), created_at FROM suppliers WHERE organization_id = $1 AND id = $2", orgID, id)
	s := &Supplier{}
	if err := row.Scan(&s.ID, &s.Name, &s.ContactPerson, &s.Phone, &s.Email, &s.Address, &s.OrganizationID, &s.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

func (r *supplierRepository) Create(s *Supplier) error {
	return r.db.QueryRow(
		"INSERT INTO suppliers (name, contact_person, phone, email, address) VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at",
		s.Name, s.ContactPerson, s.Phone, s.Email, s.Address,
	).Scan(&s.ID, &s.CreatedAt)
}

func (r *supplierRepository) CreateForOrg(orgID int, s *Supplier) error {
	s.OrganizationID = orgID
	return r.db.QueryRow(
		"INSERT INTO suppliers (name, contact_person, phone, email, address, organization_id) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at",
		s.Name, s.ContactPerson, s.Phone, s.Email, s.Address, orgID,
	).Scan(&s.ID, &s.CreatedAt)
}

func (r *supplierRepository) Update(s *Supplier) error {
	_, err := r.db.Exec("UPDATE suppliers SET name=$1, contact_person=$2, phone=$3, email=$4, address=$5 WHERE id=$6",
		s.Name, s.ContactPerson, s.Phone, s.Email, s.Address, s.ID)
	return err
}

func (r *supplierRepository) UpdateForOrg(orgID int, s *Supplier) error {
	_, err := r.db.Exec("UPDATE suppliers SET name=$1, contact_person=$2, phone=$3, email=$4, address=$5 WHERE organization_id=$6 AND id=$7",
		s.Name, s.ContactPerson, s.Phone, s.Email, s.Address, orgID, s.ID)
	return err
}

func (r *supplierRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM suppliers WHERE id = $1", id)
	return err
}

func (r *supplierRepository) DeleteForOrg(orgID, id int) error {
	_, err := r.db.Exec("DELETE FROM suppliers WHERE organization_id = $1 AND id = $2", orgID, id)
	return err
}
