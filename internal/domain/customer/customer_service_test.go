package customer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockCustomerRepo struct {
	mock.Mock
}

func (m *mockCustomerRepo) FindAll(search string) ([]Customer, error) {
	args := m.Called(search)
	return args.Get(0).([]Customer), args.Error(1)
}

func (m *mockCustomerRepo) FindByID(id int) (*Customer, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Customer), args.Error(1)
}

func (m *mockCustomerRepo) Create(c *Customer) error {
	args := m.Called(c)
	return args.Error(0)
}

func (m *mockCustomerRepo) Update(c *Customer) error {
	args := m.Called(c)
	return args.Error(0)
}

func (m *mockCustomerRepo) Delete(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *mockCustomerRepo) GetPurchaseHistory(customerID int) ([]PurchaseRecord, error) {
	args := m.Called(customerID)
	return args.Get(0).([]PurchaseRecord), args.Error(1)
}

func TestCustomerService_Create_Valid(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	c := &Customer{Name: "Budi"}
	repo.On("Create", c).Return(nil)
	err := svc.Create(c)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCustomerService_Create_NoName(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	err := svc.Create(&Customer{Name: ""})
	assert.ErrorIs(t, err, ErrCustomerNameRequired)
}

func TestCustomerService_GetByID_NotFound(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	repo.On("FindByID", 999).Return(nil, nil)
	result, err := svc.FindByID(999)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestCustomerService_Delete_NotFound(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	repo.On("FindByID", 999).Return(nil, nil)
	err := svc.Delete(999)
	assert.ErrorIs(t, err, ErrCustomerNotFound)
}
