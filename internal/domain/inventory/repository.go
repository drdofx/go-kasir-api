package inventory

import "database/sql"

type Alert struct {
	ProductID   int    `json:"product_id"`
	ProductName string `json:"product_name"`
	BranchID    int    `json:"branch_id"`
	BranchName  string `json:"branch_name"`
	Stock       int    `json:"stock"`
	MinStock    int    `json:"min_stock"`
	MaxStock    int    `json:"max_stock"`
	Enabled     bool   `json:"enabled"`
}

type Threshold struct {
	ProductID int  `json:"product_id"`
	MinStock  int  `json:"min_stock"`
	MaxStock  int  `json:"max_stock"`
	Enabled   bool `json:"enabled"`
}

type inventoryRepository struct {
	db *sql.DB
}

type InventoryRepository interface {
	FindAlerts() ([]Alert, error)
	FindAlertsForOrg(orgID int) ([]Alert, error)
	UpsertThreshold(t Threshold) error
	UpsertThresholdForOrg(orgID int, t Threshold) error
}

func NewInventoryRepository(db *sql.DB) InventoryRepository {
	return &inventoryRepository{db: db}
}

func (r *inventoryRepository) FindAlerts() ([]Alert, error) {
	rows, err := r.db.Query(`SELECT p.id, p.name, ps.branch_id, COALESCE(b.name, ''), ps.stock,
			COALESCE(ia.min_stock, 0), COALESCE(ia.max_stock, 0), COALESCE(ia.enabled, true)
		FROM product_stocks ps
		JOIN products p ON p.id = ps.product_id
		JOIN branches b ON b.id = ps.branch_id
		LEFT JOIN inventory_alerts ia ON p.id = ia.product_id
		WHERE (ia.enabled = true AND ps.stock <= ia.min_stock) OR (ia.enabled IS NULL AND ps.stock <= 0)
		ORDER BY ps.stock ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var alerts []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ProductID, &a.ProductName, &a.BranchID, &a.BranchName, &a.Stock, &a.MinStock, &a.MaxStock, &a.Enabled); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

func (r *inventoryRepository) FindAlertsForOrg(orgID int) ([]Alert, error) {
	rows, err := r.db.Query(`SELECT p.id, p.name, ps.branch_id, COALESCE(b.name, ''), ps.stock,
			COALESCE(ia.min_stock, 0), COALESCE(ia.max_stock, 0), COALESCE(ia.enabled, true)
		FROM product_stocks ps
		JOIN products p ON p.id = ps.product_id
		JOIN branches b ON b.id = ps.branch_id
		LEFT JOIN inventory_alerts ia ON p.id = ia.product_id
		WHERE b.organization_id = $1
			AND ((ia.enabled = true AND ps.stock <= ia.min_stock) OR (ia.enabled IS NULL AND ps.stock <= 0))
		ORDER BY ps.stock ASC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var alerts []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ProductID, &a.ProductName, &a.BranchID, &a.BranchName, &a.Stock, &a.MinStock, &a.MaxStock, &a.Enabled); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

func (r *inventoryRepository) UpsertThreshold(t Threshold) error {
	_, err := r.db.Exec(`INSERT INTO inventory_alerts (product_id, min_stock, max_stock, enabled)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (product_id) DO UPDATE SET min_stock=$2, max_stock=$3, enabled=$4`,
		t.ProductID, t.MinStock, t.MaxStock, t.Enabled)
	return err
}

func (r *inventoryRepository) UpsertThresholdForOrg(orgID int, t Threshold) error {
	res, err := r.db.Exec(`INSERT INTO inventory_alerts (product_id, min_stock, max_stock, enabled)
		SELECT id, $2, $3, $4 FROM products WHERE organization_id = $1 AND id = $5
		ON CONFLICT (product_id) DO UPDATE SET min_stock=$2, max_stock=$3, enabled=$4`,
		orgID, t.MinStock, t.MaxStock, t.Enabled, t.ProductID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
