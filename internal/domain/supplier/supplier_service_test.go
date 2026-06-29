package supplier

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockSupplierRepo struct {
	mock.Mock
}

func (m *mockSupplierRepo) FindAll(search string) ([]Supplier, error) {
	args := m.Called(search)
	return args.Get(0).([]Supplier), args.Error(1)
}
func (m *mockSupplierRepo) FindAllForOrg(orgID int, search string) ([]Supplier, error) {
	args := m.Called(orgID, search)
	return args.Get(0).([]Supplier), args.Error(1)
}
func (m *mockSupplierRepo) FindByID(id int) (*Supplier, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Supplier), args.Error(1)
}
func (m *mockSupplierRepo) FindByIDForOrg(orgID, id int) (*Supplier, error) {
	args := m.Called(orgID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Supplier), args.Error(1)
}
func (m *mockSupplierRepo) Create(s *Supplier) error {
	args := m.Called(s)
	return args.Error(0)
}
func (m *mockSupplierRepo) CreateForOrg(orgID int, s *Supplier) error {
	args := m.Called(orgID, s)
	return args.Error(0)
}
func (m *mockSupplierRepo) Update(s *Supplier) error {
	args := m.Called(s)
	return args.Error(0)
}
func (m *mockSupplierRepo) UpdateForOrg(orgID int, s *Supplier) error {
	args := m.Called(orgID, s)
	return args.Error(0)
}
func (m *mockSupplierRepo) Delete(id int) error {
	args := m.Called(id)
	return args.Error(0)
}
func (m *mockSupplierRepo) DeleteForOrg(orgID, id int) error {
	args := m.Called(orgID, id)
	return args.Error(0)
}

func TestCreate_Valid(t *testing.T) {
	repo := new(mockSupplierRepo)
	svc := NewSupplierService(repo)
	s := &Supplier{Name: "Acme Supply"}
	repo.On("Create", s).Return(nil)
	assert.NoError(t, svc.Create(s))
}

func TestCreateForOrg_Valid(t *testing.T) {
	repo := new(mockSupplierRepo)
	svc := NewSupplierService(repo)
	s := &Supplier{Name: "Acme Supply"}
	repo.On("CreateForOrg", 10, s).Return(nil)
	assert.NoError(t, svc.CreateForOrg(10, s))
}

func TestCreate_NoName(t *testing.T) {
	repo := new(mockSupplierRepo)
	svc := NewSupplierService(repo)
	assert.ErrorIs(t, svc.Create(&Supplier{}), ErrSupplierNameRequired)
}

func TestDelete_NotFound(t *testing.T) {
	repo := new(mockSupplierRepo)
	svc := NewSupplierService(repo)
	repo.On("FindByID", 999).Return(nil, nil)
	assert.ErrorIs(t, svc.Delete(999), ErrSupplierNotFound)
}
