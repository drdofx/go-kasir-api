package transaction

import (
	"database/sql"
	"testing"

	"go-kasir-api/internal/domain/customer"

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

func (m *mockTransactionRepo) InsertTransaction(tx *sql.Tx, total int, customerID *int) (int, error) {
	args := m.Called(tx, total, customerID)
	return args.Int(0), args.Error(1)
}

func (m *mockTransactionRepo) InsertDetails(tx *sql.Tx, transactionID int, items []CheckoutItem, products []LockedProduct) error {
	args := m.Called(tx, transactionID, items, products)
	return args.Error(0)
}

func (m *mockTransactionRepo) ExistsByID(id int) (bool, error) {
	args := m.Called(id)
	return args.Bool(0), args.Error(1)
}

func (m *mockTransactionRepo) InsertPayment(tx *sql.Tx, transactionID, paymentTypeID, amount int) error {
	args := m.Called(tx, transactionID, paymentTypeID, amount)
	return args.Error(0)
}

type mockCustomerRepoForTransaction struct {
	mock.Mock
}

func (m *mockCustomerRepoForTransaction) FindByID(id int) (*customer.Customer, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*customer.Customer), args.Error(1)
}

type mockPaymentSvc struct {
	mock.Mock
}

func (m *mockPaymentSvc) GetPaymentTypeIDByName(name string) (int, error) {
	args := m.Called(name)
	return args.Int(0), args.Error(1)
}

func (m *mockPaymentSvc) InsertPayment(tx *sql.Tx, transactionID, paymentTypeID, amount int) error {
	args := m.Called(tx, transactionID, paymentTypeID, amount)
	return args.Error(0)
}

type mockReceiptGen struct {
	mock.Mock
}

func (m *mockReceiptGen) GenerateReceiptNumber(tx *sql.Tx) (string, error) {
	args := m.Called(tx)
	return args.String(0), args.Error(1)
}

func (m *mockReceiptGen) InsertReceipt(tx *sql.Tx, transactionID int, receiptNumber string) error {
	args := m.Called(tx, transactionID, receiptNumber)
	return args.Error(0)
}

func TestCheckout_EmptyItems(t *testing.T) {
	repo := new(mockTransactionRepo)
	customerRepo := new(mockCustomerRepoForTransaction)
	paymentSvc := new(mockPaymentSvc)
	receiptGen := new(mockReceiptGen)
	svc := NewTransactionService(repo, customerRepo, paymentSvc, receiptGen)
	_, err := svc.Checkout(CheckoutRequest{Items: []CheckoutItem{}})
	assert.ErrorIs(t, err, ErrCheckoutEmpty)
}

func TestCheckout_TooManyItems(t *testing.T) {
	repo := new(mockTransactionRepo)
	customerRepo := new(mockCustomerRepoForTransaction)
	paymentSvc := new(mockPaymentSvc)
	receiptGen := new(mockReceiptGen)
	svc := NewTransactionService(repo, customerRepo, paymentSvc, receiptGen)
	items := make([]CheckoutItem, 101)
	for i := range items {
		items[i] = CheckoutItem{ProductID: i + 1, Quantity: 1}
	}
	_, err := svc.Checkout(CheckoutRequest{Items: items})
	assert.ErrorIs(t, err, ErrCheckoutTooMany)
}

func TestCheckout_InvalidQuantity(t *testing.T) {
	repo := new(mockTransactionRepo)
	customerRepo := new(mockCustomerRepoForTransaction)
	paymentSvc := new(mockPaymentSvc)
	receiptGen := new(mockReceiptGen)
	svc := NewTransactionService(repo, customerRepo, paymentSvc, receiptGen)
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
	customerRepo := new(mockCustomerRepoForTransaction)
	paymentSvc := new(mockPaymentSvc)
	receiptGen := new(mockReceiptGen)
	svc := NewTransactionService(repo, customerRepo, paymentSvc, receiptGen)
	products := []LockedProduct{{ID: 1, Name: "Kopi", Price: 10000, Stock: 50}}
	repo.On("BeginTx").Return(realTx, nil)
	repo.On("LockProducts", realTx, []int{1}).Return(products, nil)
	repo.On("UpdateStock", realTx, 1, 2).Return(nil)
	repo.On("InsertTransaction", realTx, 20000, mock.MatchedBy(func(p *int) bool { return p == nil })).Return(1, nil)
	repo.On("InsertDetails", realTx, 1, []CheckoutItem{{ProductID: 1, Quantity: 2}}, products).Return(nil)
	receiptGen.On("GenerateReceiptNumber", realTx).Return("INV-20260522-0001", nil)
	receiptGen.On("InsertReceipt", realTx, 1, "INV-20260522-0001").Return(nil)
	repo.On("FindByID", 1).Return(&Transaction{ID: 1, TotalAmount: 20000}, nil)
	txn, err := svc.Checkout(CheckoutRequest{Items: []CheckoutItem{{ProductID: 1, Quantity: 2}}})
	assert.NoError(t, err)
	assert.Equal(t, 20000, txn.TotalAmount)
	repo.AssertExpectations(t)
}

func TestCheckout_WithCustomer(t *testing.T) {
	db, mockSql, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mockSql.ExpectBegin()
	mockSql.ExpectCommit()
	realTx, err := db.Begin()
	require.NoError(t, err)

	repo := new(mockTransactionRepo)
	customerRepo := new(mockCustomerRepoForTransaction)
	paymentSvc := new(mockPaymentSvc)
	receiptGen := new(mockReceiptGen)
	svc := NewTransactionService(repo, customerRepo, paymentSvc, receiptGen)
	customerID := 1
	products := []LockedProduct{{ID: 1, Name: "Kopi", Price: 10000, Stock: 50}}
	customerRepo.On("FindByID", 1).Return(&customer.Customer{ID: 1, Name: "Budi"}, nil)
	repo.On("BeginTx").Return(realTx, nil)
	repo.On("LockProducts", realTx, []int{1}).Return(products, nil)
	repo.On("UpdateStock", realTx, 1, 2).Return(nil)
	repo.On("InsertTransaction", realTx, 20000, &customerID).Return(1, nil)
	repo.On("InsertDetails", realTx, 1, []CheckoutItem{{ProductID: 1, Quantity: 2}}, products).Return(nil)
	receiptGen.On("GenerateReceiptNumber", realTx).Return("INV-20260522-0002", nil)
	receiptGen.On("InsertReceipt", realTx, 1, "INV-20260522-0002").Return(nil)
	repo.On("FindByID", 1).Return(&Transaction{ID: 1, TotalAmount: 20000}, nil)
	txn, err := svc.Checkout(CheckoutRequest{Items: []CheckoutItem{{ProductID: 1, Quantity: 2}}, CustomerID: &customerID})
	assert.NoError(t, err)
	assert.Equal(t, 20000, txn.TotalAmount)
	repo.AssertExpectations(t)
}

func TestCheckout_CustomerNotFound(t *testing.T) {
	repo := new(mockTransactionRepo)
	customerRepo := new(mockCustomerRepoForTransaction)
	paymentSvc := new(mockPaymentSvc)
	receiptGen := new(mockReceiptGen)
	svc := NewTransactionService(repo, customerRepo, paymentSvc, receiptGen)
	customerID := 999
	customerRepo.On("FindByID", 999).Return(nil, nil)
	_, err := svc.Checkout(CheckoutRequest{Items: []CheckoutItem{{ProductID: 1, Quantity: 1}}, CustomerID: &customerID})
	assert.ErrorIs(t, err, ErrCustomerNotFound)
}

func TestCheckout_WithPayments(t *testing.T) {
	db, mockSql, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mockSql.ExpectBegin()
	mockSql.ExpectCommit()
	realTx, err := db.Begin()
	require.NoError(t, err)

	repo := new(mockTransactionRepo)
	customerRepo := new(mockCustomerRepoForTransaction)
	paymentSvc := new(mockPaymentSvc)
	receiptGen := new(mockReceiptGen)
	svc := NewTransactionService(repo, customerRepo, paymentSvc, receiptGen)
	products := []LockedProduct{{ID: 1, Name: "Kopi", Price: 10000, Stock: 50}}
	repo.On("BeginTx").Return(realTx, nil)
	repo.On("LockProducts", realTx, []int{1}).Return(products, nil)
	repo.On("UpdateStock", realTx, 1, 2).Return(nil)
	repo.On("InsertTransaction", realTx, 20000, mock.MatchedBy(func(p *int) bool { return p == nil })).Return(1, nil)
	repo.On("InsertDetails", realTx, 1, []CheckoutItem{{ProductID: 1, Quantity: 2}}, products).Return(nil)
	repo.On("FindByID", 1).Return(&Transaction{ID: 1, TotalAmount: 20000}, nil)
	paymentSvc.On("GetPaymentTypeIDByName", "cash").Return(1, nil)
	repo.On("InsertPayment", realTx, 1, 1, 20000).Return(nil)
	receiptGen.On("GenerateReceiptNumber", realTx).Return("INV-20260522-0003", nil)
	receiptGen.On("InsertReceipt", realTx, 1, "INV-20260522-0003").Return(nil)
	repo.On("FindByID", 1).Return(&Transaction{ID: 1, TotalAmount: 20000}, nil)
	txn, err := svc.Checkout(CheckoutRequest{
		Items:    []CheckoutItem{{ProductID: 1, Quantity: 2}},
		Payments: []CheckoutPayment{{Type: "cash", Amount: 20000}},
	})
	assert.NoError(t, err)
	assert.Equal(t, 20000, txn.TotalAmount)
	repo.AssertExpectations(t)
}

func TestCheckout_PaymentMismatch(t *testing.T) {
	db, mockSql, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mockSql.ExpectBegin()
	mockSql.ExpectRollback()
	realTx, err := db.Begin()
	require.NoError(t, err)

	repo := new(mockTransactionRepo)
	customerRepo := new(mockCustomerRepoForTransaction)
	paymentSvc := new(mockPaymentSvc)
	receiptGen := new(mockReceiptGen)
	svc := NewTransactionService(repo, customerRepo, paymentSvc, receiptGen)
	products := []LockedProduct{{ID: 1, Name: "Kopi", Price: 10000, Stock: 50}}
	repo.On("BeginTx").Return(realTx, nil)
	repo.On("LockProducts", realTx, []int{1}).Return(products, nil)
	repo.On("UpdateStock", realTx, 1, 2).Return(nil)
	paymentSvc.On("GetPaymentTypeIDByName", "cash").Return(1, nil)
	_, err = svc.Checkout(CheckoutRequest{
		Items:    []CheckoutItem{{ProductID: 1, Quantity: 2}},
		Payments: []CheckoutPayment{{Type: "cash", Amount: 5000}},
	})
	assert.ErrorIs(t, err, ErrPaymentMismatch)
}
