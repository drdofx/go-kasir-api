package service

import (
	"context"
	"errors"
	"strings"

	"go-kasir-api/internal/model"
)

var ErrNotFound = errors.New("not found")

type CategoryService struct {
	repo CategoryRepository
}

func NewCategoryService(repo CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) GetAll(ctx context.Context) ([]model.Category, error) {
	return s.repo.GetAll(ctx)
}

func (s *CategoryService) Create(ctx context.Context, data *model.Category) error {
	if err := validateCategory(data); err != nil {
		return err
	}
	return s.repo.Create(ctx, data)
}

func (s *CategoryService) GetByID(ctx context.Context, id int) (*model.Category, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *CategoryService) Update(ctx context.Context, category *model.Category) error {
	if err := validateCategory(category); err != nil {
		return err
	}
	return s.repo.Update(ctx, category)
}

func (s *CategoryService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}

func validateCategory(c *model.Category) error {
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("category name is required")
	}
	if strings.TrimSpace(c.Description) == "" {
		return errors.New("category description is required")
	}
	return nil
}
