package transaction

import (
	"errors"
	"fmt"
)

var (
	ErrCheckoutEmpty     = errors.New("checkout requires at least one item")
	ErrCheckoutTooMany   = fmt.Errorf("maximum %d items per checkout", maxItemsPerCheckout)
	ErrInvalidQuantity   = errors.New("invalid quantity")
	ErrProductNotFound   = errors.New("product not found")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrAmountOverflow    = errors.New("total amount overflow")
)

type CheckoutItem struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type CheckoutRequest struct {
	Items []CheckoutItem `json:"items"`
}

const maxItemsPerCheckout = 100

type TransactionService struct {
	repo TransactionRepository
}

func NewTransactionService(repo TransactionRepository) *TransactionService {
	return &TransactionService{repo: repo}
}

func (s *TransactionService) Checkout(req CheckoutRequest) (*Transaction, error) {
	if len(req.Items) == 0 {
		return nil, ErrCheckoutEmpty
	}
	if len(req.Items) > maxItemsPerCheckout {
		return nil, ErrCheckoutTooMany
	}
	for _, item := range req.Items {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("%w for product %d", ErrInvalidQuantity, item.ProductID)
		}
	}
	productIDs := make([]int, len(req.Items))
	for i, item := range req.Items {
		productIDs[i] = item.ProductID
	}
	tx, err := s.repo.BeginTx()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	products, err := s.repo.LockProducts(tx, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to lock products: %w", err)
	}
	productMap := make(map[int]LockedProduct)
	for _, p := range products {
		productMap[p.ID] = p
	}
	for _, item := range req.Items {
		p, ok := productMap[item.ProductID]
		if !ok {
			return nil, fmt.Errorf("%w: product %d", ErrProductNotFound, item.ProductID)
		}
		if p.Stock < item.Quantity {
			return nil, fmt.Errorf("%w for product %s (available: %d, requested: %d)",
				ErrInsufficientStock, p.Name, p.Stock, item.Quantity)
		}
	}
	var totalAmount int
	for _, item := range req.Items {
		p := productMap[item.ProductID]
		subtotal := p.Price * item.Quantity
		if totalAmount > 1<<31-1-subtotal {
			return nil, ErrAmountOverflow
		}
		totalAmount += subtotal
		if err := s.repo.UpdateStock(tx, item.ProductID, item.Quantity); err != nil {
			return nil, fmt.Errorf("failed to update stock for product %d: %w", item.ProductID, err)
		}
	}
	transactionID, err := s.repo.InsertTransaction(tx, totalAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}
	if err := s.repo.InsertDetails(tx, transactionID, req.Items, products); err != nil {
		return nil, fmt.Errorf("failed to insert transaction details: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return s.repo.FindByID(transactionID)
}

func (s *TransactionService) FindAll() ([]Transaction, error) {
	return s.repo.FindAll()
}

func (s *TransactionService) FindByID(id int) (*Transaction, error) {
	return s.repo.FindByID(id)
}
