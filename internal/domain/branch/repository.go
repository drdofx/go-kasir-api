package branch

import (
	"database/sql"
	"time"
)

type Branch struct {
	ID             int
	OrganizationID int
	Name           string
	Code           string
	Address        string
	CreatedAt      time.Time
}

type branchRepository struct {
	db *sql.DB
}

type BranchRepository interface {
	FindByOrgID(orgID int) ([]Branch, error)
	FindByID(id int) (*Branch, error)
	BelongsToOrg(branchID, orgID int) (bool, error)
	Create(b *Branch) error
	Update(b *Branch) error
}

func NewBranchRepository(db *sql.DB) BranchRepository {
	return &branchRepository{db: db}
}

func (r *branchRepository) FindByOrgID(orgID int) ([]Branch, error) {
	rows, err := r.db.Query("SELECT id, organization_id, name, code, address, created_at FROM branches WHERE organization_id = $1 ORDER BY id", orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bs []Branch
	for rows.Next() {
		var b Branch
		if err := rows.Scan(&b.ID, &b.OrganizationID, &b.Name, &b.Code, &b.Address, &b.CreatedAt); err != nil {
			return nil, err
		}
		bs = append(bs, b)
	}
	return bs, rows.Err()
}

func (r *branchRepository) FindByID(id int) (*Branch, error) {
	row := r.db.QueryRow("SELECT id, organization_id, name, code, address, created_at FROM branches WHERE id = $1", id)
	b := &Branch{}
	if err := row.Scan(&b.ID, &b.OrganizationID, &b.Name, &b.Code, &b.Address, &b.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return b, nil
}

func (r *branchRepository) BelongsToOrg(branchID, orgID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM branches WHERE id = $1 AND organization_id = $2)", branchID, orgID).Scan(&exists)
	return exists, err
}

func (r *branchRepository) Create(b *Branch) error {
	return r.db.QueryRow("INSERT INTO branches (organization_id, name, code, address) VALUES ($1,$2,$3,$4) RETURNING id, created_at",
		b.OrganizationID, b.Name, b.Code, b.Address).Scan(&b.ID, &b.CreatedAt)
}

func (r *branchRepository) Update(b *Branch) error {
	_, err := r.db.Exec("UPDATE branches SET name=$1, code=$2, address=$3 WHERE id=$4", b.Name, b.Code, b.Address, b.ID)
	return err
}
