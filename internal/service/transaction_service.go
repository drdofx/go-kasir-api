package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-kasir-api/internal/model"
	"go-kasir-api/internal/repository"
)

type TransactionRepository interface {
	BeginTx(ctx context.Context, opts *repository.TxOptions) (repository.Transactor, error)
	GetProductSnapshots(ctx context.Context, tx repository.Transactor, productIDs []int) (map[int]model.ProductSnapshot, error)
	BatchUpdateStock(ctx context.Context, tx repository.Transactor, items []model.CheckoutItem) error
	InsertTransaction(ctx context.Context, tx repository.Transactor, totalAmount int) (int, time.Time, error)
	BatchInsertDetails(ctx context.Context, tx repository.Transactor, details []model.TransactionDetail) ([]int, error)
	GetSalesSummary(ctx context.Context, startDate, endDate *time.Time) (*model.SalesSummary, error)
}



const maxCheckoutItems = 100

type TransactionService struct {
	repo TransactionRepository
}

func NewTransactionService(repo TransactionRepository) *TransactionService {
	return &TransactionService{repo: repo}
}

func (s *TransactionService) Checkout(ctx context.Context, items []model.CheckoutItem) (*model.Transaction, error) {
	if len(items) == 0 {
		return nil, errors.New("items are required")
	}
	if len(items) > maxCheckoutItems {
		return nil, fmt.Errorf("maximum %d items per checkout", maxCheckoutItems)
	}

	productIDs := make([]int, len(items))
	for i, item := range items {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("invalid quantity for product %d", item.ProductID)
		}
		productIDs[i] = item.ProductID
	}

	tx, err := s.repo.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	snapshots, err := s.repo.GetProductSnapshots(ctx, tx, productIDs)
	if err != nil {
		return nil, err
	}

	totalAmount := 0
	details := make([]model.TransactionDetail, 0, len(items))
	for _, item := range items {
		snap, ok := snapshots[item.ProductID]
		if !ok {
			return nil, fmt.Errorf("product %d not found", item.ProductID)
		}
		if snap.Stock < item.Quantity {
			return nil, fmt.Errorf("insufficient stock for %s (have %d, need %d)", snap.Name, snap.Stock, item.Quantity)
		}
		subtotal := snap.Price * item.Quantity
		if subtotal < 0 || subtotal > 1_000_000_000 {
			return nil, fmt.Errorf("subtotal exceeds maximum allowed for product %d", item.ProductID)
		}
		if totalAmount > 1_000_000_000-subtotal {
			return nil, errors.New("total amount exceeds maximum allowed")
		}
		totalAmount += subtotal
		details = append(details, model.TransactionDetail{
			ProductID:   item.ProductID,
			ProductName: snap.Name,
			Quantity:    item.Quantity,
			Subtotal:    subtotal,
		})
	}

	if err := s.repo.BatchUpdateStock(ctx, tx, items); err != nil {
		return nil, err
	}

	transactionID, createdAt, err := s.repo.InsertTransaction(ctx, tx, totalAmount)
	if err != nil {
		return nil, err
	}

	for i := range details {
		details[i].TransactionID = transactionID
	}
	detailIDs, err := s.repo.BatchInsertDetails(ctx, tx, details)
	if err != nil {
		return nil, err
	}
	for i, id := range detailIDs {
		details[i].ID = id
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &model.Transaction{
		ID:          transactionID,
		TotalAmount: totalAmount,
		CreatedAt:   createdAt,
		Details:     details,
	}, nil
}

func (s *TransactionService) GetSalesSummary(ctx context.Context, startDate, endDate *time.Time) (*model.SalesSummary, error) {
	return s.repo.GetSalesSummary(ctx, startDate, endDate)
}
