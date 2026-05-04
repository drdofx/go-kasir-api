package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"go-kasir-api/internal/model"
	"go-kasir-api/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockTxRepo struct {
	mock.Mock
}

type mockTx struct {
	mock.Mock
}

func (m *mockTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	// not used in tests - we mock at the repo level
	return nil, nil
}

func (m *mockTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	// not used in tests - we mock at the repo level
	return nil, nil
}

func (m *mockTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	// not used in tests - we mock at the repo level
	return nil
}

func (m *mockTx) Commit() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockTx) Rollback() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockTxRepo) BeginTx(ctx context.Context, opts *repository.TxOptions) (repository.Transactor, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(repository.Transactor), args.Error(1)
}

func (m *mockTxRepo) GetProductSnapshots(ctx context.Context, tx repository.Transactor, productIDs []int) (map[int]model.ProductSnapshot, error) {
	args := m.Called(ctx, tx, productIDs)
	return args.Get(0).(map[int]model.ProductSnapshot), args.Error(1)
}

func (m *mockTxRepo) BatchUpdateStock(ctx context.Context, tx repository.Transactor, items []model.CheckoutItem) error {
	args := m.Called(ctx, tx, items)
	return args.Error(0)
}

func (m *mockTxRepo) InsertTransaction(ctx context.Context, tx repository.Transactor, totalAmount int) (int, time.Time, error) {
	args := m.Called(ctx, tx, totalAmount)
	return args.Int(0), args.Get(1).(time.Time), args.Error(2)
}

func (m *mockTxRepo) BatchInsertDetails(ctx context.Context, tx repository.Transactor, details []model.TransactionDetail) ([]int, error) {
	args := m.Called(ctx, tx, details)
	return args.Get(0).([]int), args.Error(1)
}

func (m *mockTxRepo) GetSalesSummary(ctx context.Context, startDate, endDate *time.Time) (*model.SalesSummary, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SalesSummary), args.Error(1)
}

func TestCheckout_EmptyItems(t *testing.T) {
	svc := NewTransactionService(new(mockTxRepo))
	_, err := svc.Checkout(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "items are required")

	_, err = svc.Checkout(context.Background(), []model.CheckoutItem{})
	assert.Error(t, err)
}

func TestCheckout_TooManyItems(t *testing.T) {
	svc := NewTransactionService(new(mockTxRepo))
	items := make([]model.CheckoutItem, maxCheckoutItems+1)
	_, err := svc.Checkout(context.Background(), items)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum")
}

func TestCheckout_InvalidQuantity(t *testing.T) {
	svc := NewTransactionService(new(mockTxRepo))
	items := []model.CheckoutItem{{ProductID: 1, Quantity: 0}}
	_, err := svc.Checkout(context.Background(), items)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quantity")
}

func TestCheckout_ProductNotFound(t *testing.T) {
	mockRepo := new(mockTxRepo)
	svc := NewTransactionService(mockRepo)

	tx := new(mockTx)
	tx.On("Rollback").Return(nil)
	mockRepo.On("BeginTx", mock.Anything, (*repository.TxOptions)(nil)).Return(tx, nil)
	mockRepo.On("GetProductSnapshots", mock.Anything, tx, []int{999}).Return(map[int]model.ProductSnapshot{}, nil)

	items := []model.CheckoutItem{{ProductID: 999, Quantity: 1}}
	_, err := svc.Checkout(context.Background(), items)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCheckout_InsufficientStock(t *testing.T) {
	mockRepo := new(mockTxRepo)
	svc := NewTransactionService(mockRepo)

	tx := new(mockTx)
	tx.On("Rollback").Return(nil)
	mockRepo.On("BeginTx", mock.Anything, (*repository.TxOptions)(nil)).Return(tx, nil)
	mockRepo.On("GetProductSnapshots", mock.Anything, tx, []int{1}).Return(
		map[int]model.ProductSnapshot{1: {ID: 1, Name: "Coffee", Price: 5000, Stock: 0}}, nil,
	)

	items := []model.CheckoutItem{{ProductID: 1, Quantity: 1}}
	_, err := svc.Checkout(context.Background(), items)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient stock")
}

func TestCheckout_Success(t *testing.T) {
	mockRepo := new(mockTxRepo)
	svc := NewTransactionService(mockRepo)

	now := time.Now()
	tx := new(mockTx)
	tx.On("Commit").Return(nil)
	tx.On("Rollback").Return(sql.ErrTxDone)
	mockRepo.On("BeginTx", mock.Anything, (*repository.TxOptions)(nil)).Return(tx, nil)
	mockRepo.On("GetProductSnapshots", mock.Anything, tx, []int{1, 2}).Return(
		map[int]model.ProductSnapshot{
			1: {ID: 1, Name: "Coffee", Price: 5000, Stock: 10},
			2: {ID: 2, Name: "Milk", Price: 3000, Stock: 5},
		}, nil,
	)
	mockRepo.On("BatchUpdateStock", mock.Anything, tx, mock.Anything).Return(nil)
	mockRepo.On("InsertTransaction", mock.Anything, tx, 8000).Return(1, now, nil)
	mockRepo.On("BatchInsertDetails", mock.Anything, tx, mock.Anything).Return([]int{1, 2}, nil)

	items := []model.CheckoutItem{
		{ProductID: 1, Quantity: 1},
		{ProductID: 2, Quantity: 1},
	}
	result, err := svc.Checkout(context.Background(), items)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.ID)
	assert.Equal(t, 8000, result.TotalAmount)
	assert.Len(t, result.Details, 2)
	mockRepo.AssertExpectations(t)
}

func TestGetSalesSummary(t *testing.T) {
	mockRepo := new(mockTxRepo)
	svc := NewTransactionService(mockRepo)

	expected := &model.SalesSummary{
		TotalRevenue:      50000,
		TotalTransactions: 3,
	}
	mockRepo.On("GetSalesSummary", mock.Anything, mock.Anything, mock.Anything).Return(expected, nil)

	result, err := svc.GetSalesSummary(context.Background(), nil, nil)

	assert.NoError(t, err)
	assert.Equal(t, 50000, result.TotalRevenue)
	assert.Equal(t, 3, result.TotalTransactions)
}

func TestCheckout_BeginTxError(t *testing.T) {
	mockRepo := new(mockTxRepo)
	svc := NewTransactionService(mockRepo)

	mockRepo.On("BeginTx", mock.Anything, (*repository.TxOptions)(nil)).Return(nil, errors.New("connection failed"))

	items := []model.CheckoutItem{{ProductID: 1, Quantity: 1}}
	_, err := svc.Checkout(context.Background(), items)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection failed")
}
