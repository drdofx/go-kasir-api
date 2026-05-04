package service

import (
	"context"
	"errors"
	"strings"

	"go-kasir-api/internal/model"
)

type ProductRepository interface {
	GetAll(ctx context.Context, name string) ([]model.Product, error)
	GetByID(ctx context.Context, id int) (*model.Product, error)
	Create(ctx context.Context, product *model.Product) error
	Update(ctx context.Context, product *model.Product) error
	Delete(ctx context.Context, id int) error
}

type CategoryRepository interface {
	GetAll(ctx context.Context) ([]model.Category, error)
	GetByID(ctx context.Context, id int) (*model.Category, error)
	Create(ctx context.Context, category *model.Category) error
	Update(ctx context.Context, category *model.Category) error
	Delete(ctx context.Context, id int) error
}

var (
	ErrInvalidCategoryID = errors.New("invalid category id")
	ErrInvalidProduct    = errors.New("invalid product data")
)

type ProductService struct {
	repo         ProductRepository
	categoryRepo CategoryRepository
}

func NewProductService(repo ProductRepository, categoryRepo CategoryRepository) *ProductService {
	return &ProductService{repo: repo, categoryRepo: categoryRepo}
}

func (s *ProductService) GetAll(ctx context.Context, name string) ([]model.Product, error) {
	return s.repo.GetAll(ctx, name)
}

func (s *ProductService) Create(ctx context.Context, data *model.Product) error {
	if err := validateProduct(data); err != nil {
		return err
	}
	if data.CategoryID <= 0 {
		return ErrInvalidCategoryID
	}
	if _, err := s.categoryRepo.GetByID(ctx, data.CategoryID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		}
		return err
	}
	return s.repo.Create(ctx, data)
}

func (s *ProductService) GetByID(ctx context.Context, id int) (*model.Product, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProductService) Update(ctx context.Context, product *model.Product) error {
	if err := validateProduct(product); err != nil {
		return err
	}
	if product.CategoryID <= 0 {
		return ErrInvalidCategoryID
	}
	if _, err := s.categoryRepo.GetByID(ctx, product.CategoryID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		}
		return err
	}
	return s.repo.Update(ctx, product)
}

func (s *ProductService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}

func validateProduct(p *model.Product) error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("product name is required")
	}
	if p.Price < 0 {
		return errors.New("product price cannot be negative")
	}
	if p.Stock < 0 {
		return errors.New("product stock cannot be negative")
	}
	return nil
}
