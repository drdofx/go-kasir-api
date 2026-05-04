package service

import (
	"context"
	"testing"

	"go-kasir-api/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockCategoryRepo struct {
	mock.Mock
}

func (m *mockCategoryRepo) GetAll(ctx context.Context) ([]model.Category, error) {
	args := m.Called(ctx)
	return args.Get(0).([]model.Category), args.Error(1)
}

func (m *mockCategoryRepo) GetByID(ctx context.Context, id int) (*model.Category, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Category), args.Error(1)
}

func (m *mockCategoryRepo) Create(ctx context.Context, category *model.Category) error {
	args := m.Called(ctx, category)
	return args.Error(0)
}

func (m *mockCategoryRepo) Update(ctx context.Context, category *model.Category) error {
	args := m.Called(ctx, category)
	return args.Error(0)
}

func (m *mockCategoryRepo) Delete(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestCategoryService_GetAll(t *testing.T) {
	mockRepo := new(mockCategoryRepo)
	svc := NewCategoryService(mockRepo)

	expected := []model.Category{
		{ID: 1, Name: "Beverages", Description: "Drinks"},
	}
	mockRepo.On("GetAll", context.Background()).Return(expected, nil)

	result, err := svc.GetAll(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_Create_InvalidName(t *testing.T) {
	svc := NewCategoryService(new(mockCategoryRepo))

	err := svc.Create(context.Background(), &model.Category{Name: "", Description: "test"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestCategoryService_Create_InvalidDescription(t *testing.T) {
	svc := NewCategoryService(new(mockCategoryRepo))

	err := svc.Create(context.Background(), &model.Category{Name: "Test", Description: ""})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "description is required")
}

func TestCategoryService_Create_Success(t *testing.T) {
	mockRepo := new(mockCategoryRepo)
	svc := NewCategoryService(mockRepo)

	cat := &model.Category{Name: "Beverages", Description: "Drinks"}
	mockRepo.On("Create", context.Background(), cat).Return(nil)

	err := svc.Create(context.Background(), cat)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_GetByID_NotFound(t *testing.T) {
	mockRepo := new(mockCategoryRepo)
	svc := NewCategoryService(mockRepo)

	mockRepo.On("GetByID", context.Background(), 999).Return(nil, ErrNotFound)

	result, err := svc.GetByID(context.Background(), 999)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestCategoryService_Delete(t *testing.T) {
	mockRepo := new(mockCategoryRepo)
	svc := NewCategoryService(mockRepo)

	mockRepo.On("Delete", context.Background(), 1).Return(nil)

	err := svc.Delete(context.Background(), 1)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
