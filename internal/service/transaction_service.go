package service

import (
	"database/sql"
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

	tx, err := s.repo.BeginTx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	totalAmount := 0
	details := make([]model.TransactionDetail, 0, len(items))

	for _, item := range items {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("invalid quantity for product %d", item.ProductID)
		}

		productName, productPrice, stock, err := s.repo.GetProductSnapshot(tx, item.ProductID)
		if err == sql.ErrNoRows {
			return nil, repository.ErrProductNotFound
		}
		if err != nil {
			return nil, err
		}

		if stock < item.Quantity {
			return nil, fmt.Errorf("insufficient stock for product id %d", item.ProductID)
		}

		subtotal := productPrice * item.Quantity
		totalAmount += subtotal

		if err := s.repo.UpdateProductStock(tx, item.ProductID, item.Quantity); err != nil {
			return nil, err
		}

		details = append(details, model.TransactionDetail{
			ProductID:   item.ProductID,
			ProductName: productName,
			Quantity:    item.Quantity,
			Subtotal:    subtotal,
		})
	}

	transactionID, createdAt, err := s.repo.InsertTransaction(tx, totalAmount)
	if err != nil {
		return nil, err
	}

	for i := range details {
		details[i].TransactionID = transactionID
		detailID, err := s.repo.InsertTransactionDetail(tx, details[i])
		if err != nil {
			return nil, err
		}
		details[i].ID = detailID
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
