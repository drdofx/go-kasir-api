package category

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockCategoryRepo struct {
	mock.Mock
}

func (m *mockCategoryRepo) FindAll() ([]Category, error) {
	args := m.Called()
	return args.Get(0).([]Category), args.Error(1)
}

func (m *mockCategoryRepo) FindAllForOrg(orgID int) ([]Category, error) {
	args := m.Called(orgID)
	return args.Get(0).([]Category), args.Error(1)
}

func (m *mockCategoryRepo) FindByID(id int) (*Category, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Category), args.Error(1)
}

func (m *mockCategoryRepo) FindByIDForOrg(orgID, id int) (*Category, error) {
	args := m.Called(orgID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Category), args.Error(1)
}

func (m *mockCategoryRepo) ExistsForOrg(orgID, id int) (bool, error) {
	args := m.Called(orgID, id)
	return args.Bool(0), args.Error(1)
}

func (m *mockCategoryRepo) Create(c *Category) error {
	args := m.Called(c)
	return args.Error(0)
}

func (m *mockCategoryRepo) CreateForOrg(orgID int, c *Category) error {
	args := m.Called(orgID, c)
	return args.Error(0)
}

func (m *mockCategoryRepo) Update(c *Category) error {
	args := m.Called(c)
	return args.Error(0)
}

func (m *mockCategoryRepo) UpdateForOrg(orgID int, c *Category) error {
	args := m.Called(orgID, c)
	return args.Error(0)
}

func (m *mockCategoryRepo) Delete(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *mockCategoryRepo) DeleteForOrg(orgID, id int) error {
	args := m.Called(orgID, id)
	return args.Error(0)
}

func TestCategoryService_FindAll(t *testing.T) {
	repo := new(mockCategoryRepo)
	svc := NewCategoryService(repo)
	expected := []Category{{ID: 1, Name: "Minuman", Description: "Minuman"}}
	repo.On("FindAll").Return(expected, nil)
	result, err := svc.FindAll()
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestCategoryService_FindAllForOrg(t *testing.T) {
	repo := new(mockCategoryRepo)
	svc := NewCategoryService(repo)
	expected := []Category{{ID: 1, Name: "Beverages", Description: "Beverages", OrganizationID: 10}}
	repo.On("FindAllForOrg", 10).Return(expected, nil)
	result, err := svc.FindAllForOrg(10)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestCategoryService_Create_Valid(t *testing.T) {
	repo := new(mockCategoryRepo)
	svc := NewCategoryService(repo)
	c := &Category{Name: "Makanan", Description: "Makanan ringan"}
	repo.On("Create", c).Return(nil)
	err := svc.Create(c)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCategoryService_Create_Invalid(t *testing.T) {
	repo := new(mockCategoryRepo)
	svc := NewCategoryService(repo)
	err := svc.Create(&Category{Name: "", Description: ""})
	assert.Error(t, err)
	repo.AssertNotCalled(t, "Create")
}

func TestCategoryService_FindByID_NotFound(t *testing.T) {
	repo := new(mockCategoryRepo)
	svc := NewCategoryService(repo)
	repo.On("FindByID", 999).Return(nil, nil)
	result, err := svc.FindByID(999)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestCategoryService_Delete(t *testing.T) {
	repo := new(mockCategoryRepo)
	svc := NewCategoryService(repo)
	repo.On("FindByID", 1).Return(&Category{ID: 1}, nil)
	repo.On("Delete", 1).Return(nil)
	err := svc.Delete(1)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCategoryService_DeleteForOrg_NotFound(t *testing.T) {
	repo := new(mockCategoryRepo)
	svc := NewCategoryService(repo)
	repo.On("FindByIDForOrg", 10, 999).Return(nil, nil)
	err := svc.DeleteForOrg(10, 999)
	assert.ErrorIs(t, err, ErrCategoryNotFound)
}
