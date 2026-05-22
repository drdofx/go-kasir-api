package transaction

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockTransactionRepo struct {
	mock.Mock
}

func (m *mockTransactionRepo) BeginTx() (*sql.Tx, error) {
	args := m.Called()
	return args.Get(0).(*sql.Tx), args.Error(1)
}

func (m *mockTransactionRepo) FindAll() ([]Transaction, error) {
	args := m.Called()
	return args.Get(0).([]Transaction), args.Error(1)
}

func (m *mockTransactionRepo) FindByID(id int) (*Transaction, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Transaction), args.Error(1)
}

func (m *mockTransactionRepo) LockProducts(tx *sql.Tx, ids []int) ([]LockedProduct, error) {
	args := m.Called(tx, ids)
	return args.Get(0).([]LockedProduct), args.Error(1)
}

func (m *mockTransactionRepo) UpdateStock(tx *sql.Tx, id, qty int) error {
	args := m.Called(tx, id, qty)
	return args.Error(0)
}

func (m *mockTransactionRepo) InsertTransaction(tx *sql.Tx, total int) (int, error) {
	args := m.Called(tx, total)
	return args.Int(0), args.Error(1)
}

func (m *mockTransactionRepo) InsertDetails(tx *sql.Tx, transactionID int, items []CheckoutItem, products []LockedProduct) error {
	args := m.Called(tx, transactionID, items, products)
	return args.Error(0)
}

func TestCheckout_EmptyItems(t *testing.T) {
	repo := new(mockTransactionRepo)
	svc := NewTransactionService(repo)
	_, err := svc.Checkout(CheckoutRequest{Items: []CheckoutItem{}})
	assert.ErrorIs(t, err, ErrCheckoutEmpty)
}

func TestCheckout_TooManyItems(t *testing.T) {
	repo := new(mockTransactionRepo)
	svc := NewTransactionService(repo)
	items := make([]CheckoutItem, 101)
	for i := range items {
		items[i] = CheckoutItem{ProductID: i + 1, Quantity: 1}
	}
	_, err := svc.Checkout(CheckoutRequest{Items: items})
	assert.ErrorIs(t, err, ErrCheckoutTooMany)
}

func TestCheckout_InvalidQuantity(t *testing.T) {
	repo := new(mockTransactionRepo)
	svc := NewTransactionService(repo)
	_, err := svc.Checkout(CheckoutRequest{Items: []CheckoutItem{{ProductID: 1, Quantity: 0}}})
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidQuantity)
}

func TestCheckout_Success(t *testing.T) {
	db, mockSql, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mockSql.ExpectBegin()
	mockSql.ExpectCommit()
	realTx, err := db.Begin()
	require.NoError(t, err)

	repo := new(mockTransactionRepo)
	svc := NewTransactionService(repo)
	products := []LockedProduct{{ID: 1, Name: "Kopi", Price: 10000, Stock: 50}}
	repo.On("BeginTx").Return(realTx, nil)
	repo.On("LockProducts", realTx, []int{1}).Return(products, nil)
	repo.On("UpdateStock", realTx, 1, 2).Return(nil)
	repo.On("InsertTransaction", realTx, 20000).Return(1, nil)
	repo.On("InsertDetails", realTx, 1, []CheckoutItem{{ProductID: 1, Quantity: 2}}, products).Return(nil)
	repo.On("FindByID", 1).Return(&Transaction{ID: 1, TotalAmount: 20000}, nil)
	txn, err := svc.Checkout(CheckoutRequest{Items: []CheckoutItem{{ProductID: 1, Quantity: 2}}})
	assert.NoError(t, err)
	assert.Equal(t, 20000, txn.TotalAmount)
	repo.AssertExpectations(t)
}
