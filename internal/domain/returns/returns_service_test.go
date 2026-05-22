package returns

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockReturnRepo struct {
	mock.Mock
}

func (m *mockReturnRepo) BeginTx() (*sql.Tx, error) {
	args := m.Called()
	return args.Get(0).(*sql.Tx), args.Error(1)
}

func (m *mockReturnRepo) FindAll() ([]Return, error) {
	args := m.Called()
	return args.Get(0).([]Return), args.Error(1)
}

func (m *mockReturnRepo) FindByID(id int) (*Return, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Return), args.Error(1)
}

func (m *mockReturnRepo) FindTransactionDetails(tx *sql.Tx, transactionID int) ([]LockedProduct, error) {
	args := m.Called(tx, transactionID)
	return args.Get(0).([]LockedProduct), args.Error(1)
}

func (m *mockReturnRepo) UpdateStock(tx *sql.Tx, productID, quantity int) error {
	args := m.Called(tx, productID, quantity)
	return args.Error(0)
}

func (m *mockReturnRepo) InsertReturn(tx *sql.Tx, transactionID, totalRefund int, reason string) (int, error) {
	args := m.Called(tx, transactionID, totalRefund, reason)
	return args.Int(0), args.Error(1)
}

func (m *mockReturnRepo) InsertReturnItems(tx *sql.Tx, returnID int, items []ReturnItemRequest, products []LockedProduct) error {
	args := m.Called(tx, returnID, items, products)
	return args.Error(0)
}

type mockTxRepo struct {
	mock.Mock
}

func (m *mockTxRepo) ExistsByID(id int) (bool, error) {
	args := m.Called(id)
	return args.Bool(0), args.Error(1)
}

func TestProcessReturn_EmptyItems(t *testing.T) {
	returnRepo := new(mockReturnRepo)
	txRepo := new(mockTxRepo)
	svc := NewReturnService(returnRepo, txRepo)
	_, err := svc.ProcessReturn(1, "", nil)
	assert.ErrorIs(t, err, ErrReturnEmpty)
}

func TestProcessReturn_TransactionNotFound(t *testing.T) {
	returnRepo := new(mockReturnRepo)
	txRepo := new(mockTxRepo)
	svc := NewReturnService(returnRepo, txRepo)
	txRepo.On("ExistsByID", 999).Return(false, nil)
	_, err := svc.ProcessReturn(999, "", []ReturnItemRequest{{ProductID: 1, Quantity: 1}})
	assert.Error(t, err)
}
