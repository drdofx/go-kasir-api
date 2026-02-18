package service

import (
	"errors"
	"fmt"
	"time"

	"go-kasir-api/internal/model"
	"go-kasir-api/internal/repository"
)

// TransactionService handles checkout logic.
type TransactionService struct {
	repo *repository.TransactionRepository
}

func NewTransactionService(repo *repository.TransactionRepository) *TransactionService {
	return &TransactionService{repo: repo}
}

func (s *TransactionService) Checkout(items []model.CheckoutItem) (*model.Transaction, error) {
	if len(items) == 0 {
		return nil, errors.New("items are required")
	}

	// Validate quantities and collect product IDs
	productIDs := make([]int, len(items))
	for i, item := range items {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("invalid quantity for product %d", item.ProductID)
		}
		productIDs[i] = item.ProductID
	}

	tx, err := s.repo.BeginTx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Batch fetch all products in a single query (with FOR UPDATE lock)
	snapshots, err := s.repo.GetProductSnapshots(tx, productIDs)
	if err != nil {
		return nil, err
	}

	// 2. Validate stock and compute totals
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
		totalAmount += subtotal
		details = append(details, model.TransactionDetail{
			ProductID:   item.ProductID,
			ProductName: snap.Name,
			Quantity:    item.Quantity,
			Subtotal:    subtotal,
		})
	}

	// 3. Batch update stock in a single statement
	if err := s.repo.BatchUpdateStock(tx, items); err != nil {
		return nil, err
	}

	// 4. Insert transaction header
	transactionID, createdAt, err := s.repo.InsertTransaction(tx, totalAmount)
	if err != nil {
		return nil, err
	}

	// 5. Set transaction ID on details and batch insert
	for i := range details {
		details[i].TransactionID = transactionID
	}
	detailIDs, err := s.repo.BatchInsertDetails(tx, details)
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

func (s *TransactionService) GetSalesSummary(startDate, endDate *time.Time) (*model.SalesSummary, error) {
	return s.repo.GetSalesSummary(startDate, endDate)
}
