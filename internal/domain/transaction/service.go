package transaction

import (
	"database/sql"
	"errors"
	"fmt"

	"go-kasir-api/internal/domain/customer"
)

var (
	ErrCheckoutEmpty      = errors.New("checkout requires at least one item")
	ErrCheckoutTooMany    = fmt.Errorf("maximum %d items per checkout", maxItemsPerCheckout)
	ErrInvalidQuantity    = errors.New("invalid quantity")
	ErrProductNotFound    = errors.New("product not found")
	ErrInsufficientStock  = errors.New("insufficient stock")
	ErrAmountOverflow     = errors.New("total amount overflow")
	ErrPaymentMismatch    = errors.New("payment total does not match total amount")
	ErrInvalidPaymentType = errors.New("invalid payment type")
	ErrCustomerNotFound   = errors.New("customer not found")
)

type CheckoutItem struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type CheckoutPayment struct {
	Type   string
	Amount int
}

type CheckoutRequest struct {
	Items      []CheckoutItem   `json:"items"`
	CustomerID *int             `json:"customer_id"`
	Payments   []CheckoutPayment `json:"payments"`
}

type CustomerRepository interface {
	FindByID(id int) (*customer.Customer, error)
}

type PaymentService interface {
	GetPaymentTypeIDByName(name string) (int, error)
	InsertPayment(tx *sql.Tx, transactionID, paymentTypeID, amount int) error
}

const maxItemsPerCheckout = 100

type TransactionService struct {
	repo         TransactionRepository
	customerRepo CustomerRepository
	paymentSvc   PaymentService
}

func NewTransactionService(repo TransactionRepository, customerRepo CustomerRepository, paymentSvc PaymentService) *TransactionService {
	return &TransactionService{repo: repo, customerRepo: customerRepo, paymentSvc: paymentSvc}
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
	if req.CustomerID != nil && *req.CustomerID > 0 {
		customer, err := s.customerRepo.FindByID(*req.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("find customer: %w", err)
		}
		if customer == nil {
			return nil, ErrCustomerNotFound
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
	if len(req.Payments) > 0 {
		var paymentTotal int
		for _, p := range req.Payments {
			if p.Amount <= 0 {
				return nil, fmt.Errorf("invalid payment amount: %d", p.Amount)
			}
			_, err := s.paymentSvc.GetPaymentTypeIDByName(p.Type)
			if err != nil {
				return nil, fmt.Errorf("%w: %s", ErrInvalidPaymentType, p.Type)
			}
			if paymentTotal > 1<<31-1-p.Amount {
				return nil, ErrAmountOverflow
			}
			paymentTotal += p.Amount
		}
		if paymentTotal != totalAmount {
			return nil, fmt.Errorf("%w: got %d, expected %d", ErrPaymentMismatch, paymentTotal, totalAmount)
		}
	}
	transactionID, err := s.repo.InsertTransaction(tx, totalAmount, req.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}
	if err := s.repo.InsertDetails(tx, transactionID, req.Items, products); err != nil {
		return nil, fmt.Errorf("failed to insert transaction details: %w", err)
	}
	for _, p := range req.Payments {
		typeID, err := s.paymentSvc.GetPaymentTypeIDByName(p.Type)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidPaymentType, p.Type)
		}
		if err := s.repo.InsertPayment(tx, transactionID, typeID, p.Amount); err != nil {
			return nil, fmt.Errorf("failed to insert payment: %w", err)
		}
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
