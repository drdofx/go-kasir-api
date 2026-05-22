package purchaseorder

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type PurchaseOrder struct {
	ID          int
	SupplierID  int
	SupplierName string
	Status      string
	TotalAmount int
	CreatedAt   time.Time
	ReceivedAt  *time.Time
	Items       []POItem
}

type POItem struct {
	ID             int
	PurchaseOrderID int
	ProductID      int
	ProductName    string
	Quantity       int
	UnitPrice      int
	Subtotal       int
}

type poRepository struct {
	db *sql.DB
}

type PendingProduct struct {
	ProductID int
	Quantity  int
}

type PurchaseOrderRepository interface {
	FindAll() ([]PurchaseOrder, error)
	FindByID(id int) (*PurchaseOrder, error)
	BeginTx() (*sql.Tx, error)
	InsertPO(tx *sql.Tx, supplierID, totalAmount int) (int, error)
	InsertPOItems(tx *sql.Tx, poID int, items []POItemRequest, products []PricedProduct) error
	LockProducts(tx *sql.Tx, ids []int) ([]PricedProduct, error)
	UpdateStock(tx *sql.Tx, productID, quantity int) error
	MarkReceived(tx *sql.Tx, poID int) error
}

type PricedProduct struct {
	ID    int
	Name  string
	Price int
}

type POItemRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
	UnitPrice int `json:"unit_price"`
}

func NewPurchaseOrderRepository(db *sql.DB) PurchaseOrderRepository {
	return &poRepository{db: db}
}

func (r *poRepository) FindAll() ([]PurchaseOrder, error) {
	rows, err := r.db.Query(`SELECT po.id, po.supplier_id, COALESCE(s.name, ''), po.status, po.total_amount, po.created_at, po.received_at
		FROM purchase_orders po LEFT JOIN suppliers s ON po.supplier_id = s.id ORDER BY po.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pos []PurchaseOrder
	for rows.Next() {
		var po PurchaseOrder
		if err := rows.Scan(&po.ID, &po.SupplierID, &po.SupplierName, &po.Status, &po.TotalAmount, &po.CreatedAt, &po.ReceivedAt); err != nil {
			return nil, err
		}
		pos = append(pos, po)
	}
	for i := range pos {
		items, err := r.getItems(pos[i].ID)
		if err != nil {
			return nil, err
		}
		pos[i].Items = items
	}
	return pos, nil
}

func (r *poRepository) FindByID(id int) (*PurchaseOrder, error) {
	row := r.db.QueryRow(`SELECT po.id, po.supplier_id, COALESCE(s.name, ''), po.status, po.total_amount, po.created_at, po.received_at
		FROM purchase_orders po LEFT JOIN suppliers s ON po.supplier_id = s.id WHERE po.id = $1`, id)
	po := &PurchaseOrder{}
	if err := row.Scan(&po.ID, &po.SupplierID, &po.SupplierName, &po.Status, &po.TotalAmount, &po.CreatedAt, &po.ReceivedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	items, err := r.getItems(po.ID)
	if err != nil {
		return nil, err
	}
	po.Items = items
	return po, nil
}

func (r *poRepository) getItems(poID int) ([]POItem, error) {
	rows, err := r.db.Query(`SELECT poi.id, poi.purchase_order_id, poi.product_id, COALESCE(p.name, ''), poi.quantity, poi.unit_price, poi.subtotal
		FROM purchase_order_items poi LEFT JOIN products p ON poi.product_id = p.id WHERE poi.purchase_order_id = $1 ORDER BY poi.id`, poID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []POItem
	for rows.Next() {
		var it POItem
		if err := rows.Scan(&it.ID, &it.PurchaseOrderID, &it.ProductID, &it.ProductName, &it.Quantity, &it.UnitPrice, &it.Subtotal); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (r *poRepository) BeginTx() (*sql.Tx, error) { return r.db.Begin() }

func (r *poRepository) InsertPO(tx *sql.Tx, supplierID, totalAmount int) (int, error) {
	var id int
	err := tx.QueryRow("INSERT INTO purchase_orders (supplier_id, total_amount) VALUES ($1, $2) RETURNING id", supplierID, totalAmount).Scan(&id)
	return id, err
}

func (r *poRepository) InsertPOItems(tx *sql.Tx, poID int, items []POItemRequest, products []PricedProduct) error {
	if len(items) == 0 {
		return nil
	}
	pm := make(map[int]PricedProduct)
	for _, p := range products {
		pm[p.ID] = p
	}
	var vs []string
	var args []interface{}
	ai := 1
	for _, item := range items {
		p, ok := pm[item.ProductID]
		if !ok {
			continue
		}
		subtotal := item.UnitPrice * item.Quantity
		if subtotal < 0 {
			subtotal = p.Price * item.Quantity
		}
		vs = append(vs, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d)", ai, ai+1, ai+2, ai+3, ai+4))
		args = append(args, poID, item.ProductID, item.Quantity, item.UnitPrice, subtotal)
		ai += 5
	}
	if len(vs) == 0 {
		return nil
	}
	_, err := tx.Exec(fmt.Sprintf("INSERT INTO purchase_order_items (purchase_order_id, product_id, quantity, unit_price, subtotal) VALUES %s", strings.Join(vs, ",")), args...)
	return err
}

func (r *poRepository) LockProducts(tx *sql.Tx, ids []int) ([]PricedProduct, error) {
	phs := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		phs[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	rows, err := tx.Query(fmt.Sprintf("SELECT id, name, price FROM products WHERE id IN (%s)", strings.Join(phs, ",")), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ps []PricedProduct
	for rows.Next() {
		var p PricedProduct
		if err := rows.Scan(&p.ID, &p.Name, &p.Price); err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return ps, rows.Err()
}

func (r *poRepository) UpdateStock(tx *sql.Tx, productID, quantity int) error {
	_, err := tx.Exec("UPDATE products SET stock = stock + $1 WHERE id = $2", quantity, productID)
	return err
}

func (r *poRepository) MarkReceived(tx *sql.Tx, poID int) error {
	_, err := tx.Exec("UPDATE purchase_orders SET status = 'received', received_at = NOW() WHERE id = $1", poID)
	return err
}
