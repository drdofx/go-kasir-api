package returns

import (
	"errors"
	"fmt"
)

var (
	ErrReturnEmpty     = errors.New("return requires at least one item")
	ErrReturnNotFound  = errors.New("return not found")
	ErrInvalidQuantity = errors.New("invalid quantity")
)

type ReturnService struct {
	repo          ReturnRepository
	transactionRepo TransactionRepository
}

type TransactionRepository interface {
	ExistsByID(id int) (bool, error)
}

func NewReturnService(repo ReturnRepository, transactionRepo TransactionRepository) *ReturnService {
	return &ReturnService{repo: repo, transactionRepo: transactionRepo}
}

func (s *ReturnService) ProcessReturn(transactionID int, reason string, items []ReturnItemRequest) (*Return, error) {
	if len(items) == 0 {
		return nil, ErrReturnEmpty
	}
	for _, item := range items {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("%w for product %d", ErrInvalidQuantity, item.ProductID)
		}
	}
	exists, err := s.transactionRepo.ExistsByID(transactionID)
	if err != nil {
		return nil, fmt.Errorf("check transaction: %w", err)
	}
	if !exists {
		return nil, errors.New("transaction not found")
	}
	tx, err := s.repo.BeginTx()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()
	products, err := s.repo.FindTransactionDetails(tx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("find transaction details: %w", err)
	}
	productMap := make(map[int]LockedProduct)
	for _, p := range products {
		productMap[p.ID] = p
	}
	var totalRefund int
	for _, item := range items {
		p, ok := productMap[item.ProductID]
		if !ok {
			return nil, fmt.Errorf("product %d not found in transaction", item.ProductID)
		}
		subtotal := p.Price * item.Quantity
		if totalRefund > 1<<31-1-subtotal {
			return nil, errors.New("refund amount overflow")
		}
		totalRefund += subtotal
		if err := s.repo.UpdateStock(tx, item.ProductID, item.Quantity); err != nil {
			return nil, fmt.Errorf("restore stock for product %d: %w", item.ProductID, err)
		}
	}
	returnID, err := s.repo.InsertReturn(tx, transactionID, totalRefund, reason)
	if err != nil {
		return nil, fmt.Errorf("insert return: %w", err)
	}
	if err := s.repo.InsertReturnItems(tx, returnID, items, products); err != nil {
		return nil, fmt.Errorf("insert return items: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit return: %w", err)
	}
	return s.repo.FindByID(returnID)
}

func (s *ReturnService) FindAll() ([]Return, error) {
	return s.repo.FindAll()
}

func (s *ReturnService) FindByID(id int) (*Return, error) {
	return s.repo.FindByID(id)
}
