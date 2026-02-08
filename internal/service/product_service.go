package service

import (
	"errors"

	"go-kasir-api/internal/model"
	"go-kasir-api/internal/repository"
)

var ErrInvalidCategoryID = errors.New("invalid category id")

// ProductService holds product business logic.
type ProductService struct {
	repo         *repository.ProductRepository
	categoryRepo *repository.CategoryRepository
}

func NewProductService(repo *repository.ProductRepository, categoryRepo *repository.CategoryRepository) *ProductService {
	return &ProductService{repo: repo, categoryRepo: categoryRepo}
}

func (s *ProductService) GetAll(name string) ([]model.Product, error) {
	return s.repo.GetAll(name)
}

func (s *ProductService) Create(data *model.Product) error {
	if data.CategoryID <= 0 {
		return ErrInvalidCategoryID
	}
	if _, err := s.categoryRepo.GetByID(data.CategoryID); err != nil {
		if errors.Is(err, repository.ErrCategoryNotFound) {
			return repository.ErrCategoryNotFound
		}
		return err
	}
	return s.repo.Create(data)
}

func (s *ProductService) GetByID(id int) (*model.Product, error) {
	return s.repo.GetByID(id)
}

func (s *ProductService) Update(product *model.Product) error {
	if product.CategoryID <= 0 {
		return ErrInvalidCategoryID
	}
	if _, err := s.categoryRepo.GetByID(product.CategoryID); err != nil {
		if errors.Is(err, repository.ErrCategoryNotFound) {
			return repository.ErrCategoryNotFound
		}
		return err
	}
	return s.repo.Update(product)
}

func (s *ProductService) Delete(id int) error {
	return s.repo.Delete(id)
}
