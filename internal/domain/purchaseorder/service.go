package purchaseorder

import (
	"errors"
	"fmt"

	"go-kasir-api/internal/domain/supplier"
)

var (
	ErrPOCreateEmpty     = errors.New("purchase order requires at least one item")
	ErrPONotFound        = errors.New("purchase order not found")
	ErrPOAlreadyReceived = errors.New("purchase order already received")
	ErrPOProductNotFound = errors.New("purchase order product not found")
)

type SupplierRepository interface {
	FindByID(id int) (*supplier.Supplier, error)
	FindByIDForOrg(orgID, id int) (*supplier.Supplier, error)
}

type POService struct {
	repo         PurchaseOrderRepository
	supplierRepo SupplierRepository
}

func NewPOService(repo PurchaseOrderRepository, supplierRepo SupplierRepository) *POService {
	return &POService{repo: repo, supplierRepo: supplierRepo}
}

func (s *POService) FindAll() ([]PurchaseOrder, error)       { return s.repo.FindAll() }
func (s *POService) FindByID(id int) (*PurchaseOrder, error) { return s.repo.FindByID(id) }
func (s *POService) FindAllForOrg(orgID int) ([]PurchaseOrder, error) {
	return s.repo.FindAllForOrg(orgID)
}
func (s *POService) FindByIDForOrg(orgID, id int) (*PurchaseOrder, error) {
	return s.repo.FindByIDForOrg(orgID, id)
}

func (s *POService) CreatePO(supplierID int, items []POItemRequest) (*PurchaseOrder, error) {
	if len(items) == 0 {
		return nil, ErrPOCreateEmpty
	}
	for _, it := range items {
		if it.Quantity <= 0 {
			return nil, fmt.Errorf("invalid quantity for product %d", it.ProductID)
		}
	}
	sup, err := s.supplierRepo.FindByID(supplierID)
	if err != nil {
		return nil, err
	}
	if sup == nil {
		return nil, errors.New("supplier not found")
	}
	tx, err := s.repo.BeginTx()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()
	var total int
	for _, it := range items {
		if total > 1<<31-1-it.Quantity*it.UnitPrice {
			return nil, errors.New("total amount overflow")
		}
		total += it.Quantity * it.UnitPrice
	}
	poID, err := s.repo.InsertPO(tx, supplierID, total)
	if err != nil {
		return nil, fmt.Errorf("insert po: %w", err)
	}
	if err := s.repo.InsertPOItems(tx, poID, items, nil); err != nil {
		return nil, fmt.Errorf("insert items: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return s.repo.FindByID(poID)
}

func (s *POService) CreatePOForOrg(orgID, branchID, supplierID int, items []POItemRequest) (*PurchaseOrder, error) {
	if len(items) == 0 {
		return nil, ErrPOCreateEmpty
	}
	productIDs := make([]int, len(items))
	for i, it := range items {
		if it.Quantity <= 0 {
			return nil, fmt.Errorf("invalid quantity for product %d", it.ProductID)
		}
		productIDs[i] = it.ProductID
	}
	sup, err := s.supplierRepo.FindByIDForOrg(orgID, supplierID)
	if err != nil {
		return nil, err
	}
	if sup == nil {
		return nil, errors.New("supplier not found")
	}
	tx, err := s.repo.BeginTx()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()
	products, err := s.repo.LockProductsForOrg(tx, orgID, productIDs)
	if err != nil {
		return nil, fmt.Errorf("lock products: %w", err)
	}
	productMap := make(map[int]PricedProduct, len(products))
	for _, product := range products {
		productMap[product.ID] = product
	}
	var total int
	for i := range items {
		product, ok := productMap[items[i].ProductID]
		if !ok {
			return nil, fmt.Errorf("%w: %d", ErrPOProductNotFound, items[i].ProductID)
		}
		if items[i].UnitPrice <= 0 {
			items[i].UnitPrice = product.Price
		}
		if total > 1<<31-1-items[i].Quantity*items[i].UnitPrice {
			return nil, errors.New("total amount overflow")
		}
		total += items[i].Quantity * items[i].UnitPrice
	}
	poID, err := s.repo.InsertPOForOrg(tx, orgID, branchID, supplierID, total)
	if err != nil {
		return nil, fmt.Errorf("insert po: %w", err)
	}
	if err := s.repo.InsertPOItems(tx, poID, items, products); err != nil {
		return nil, fmt.Errorf("insert items: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return s.repo.FindByIDForOrg(orgID, poID)
}

func (s *POService) ReceivePO(poID int) (*PurchaseOrder, error) {
	po, err := s.repo.FindByID(poID)
	if err != nil {
		return nil, err
	}
	if po == nil {
		return nil, ErrPONotFound
	}
	if po.Status == "received" {
		return nil, ErrPOAlreadyReceived
	}
	tx, err := s.repo.BeginTx()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()
	for _, item := range po.Items {
		if err := s.repo.UpdateStock(tx, item.ProductID, item.Quantity); err != nil {
			return nil, fmt.Errorf("update stock for product %d: %w", item.ProductID, err)
		}
	}
	if err := s.repo.MarkReceived(tx, poID); err != nil {
		return nil, fmt.Errorf("mark received: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return s.repo.FindByID(poID)
}

func (s *POService) ReceivePOForOrg(orgID, branchID, poID int) (*PurchaseOrder, error) {
	po, err := s.repo.FindByIDForOrg(orgID, poID)
	if err != nil {
		return nil, err
	}
	if po == nil {
		return nil, ErrPONotFound
	}
	if po.Status == "received" {
		return nil, ErrPOAlreadyReceived
	}
	tx, err := s.repo.BeginTx()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()
	for _, item := range po.Items {
		if err := s.repo.UpdateBranchStock(tx, branchID, item.ProductID, item.Quantity); err != nil {
			return nil, fmt.Errorf("update stock for product %d: %w", item.ProductID, err)
		}
	}
	if err := s.repo.MarkReceivedForOrg(tx, orgID, poID); err != nil {
		return nil, fmt.Errorf("mark received: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return s.repo.FindByIDForOrg(orgID, poID)
}
