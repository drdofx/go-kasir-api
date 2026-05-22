package product

import "errors"

var (
    ErrProductNameRequired = errors.New("product name is required")
    ErrNegativePrice       = errors.New("price must be non-negative")
    ErrNegativeStock       = errors.New("stock must be non-negative")
    ErrCategoryNotFound    = errors.New("category not found")
    ErrProductNotFound     = errors.New("product not found")
)

type CategoryRepository interface {
    FindByID(id int) (*Category, error)
}

type Category struct {
    ID          int
    Name        string
    Description string
}

type ProductService struct {
    repo         ProductRepository
    categoryRepo CategoryRepository
}

func NewProductService(repo ProductRepository, categoryRepo CategoryRepository) *ProductService {
    return &ProductService{repo: repo, categoryRepo: categoryRepo}
}

func (s *ProductService) FindAll(name string) ([]Product, error) {
    return s.repo.FindAll(name)
}

func (s *ProductService) FindByID(id int) (*Product, error) {
    return s.repo.FindByID(id)
}

func (s *ProductService) Create(p *Product) error {
    if p.Name == "" {
        return ErrProductNameRequired
    }
    if p.Price < 0 {
        return ErrNegativePrice
    }
    if p.Stock < 0 {
        return ErrNegativeStock
    }
    if p.CategoryID != nil && *p.CategoryID > 0 {
        cat, err := s.categoryRepo.FindByID(*p.CategoryID)
        if err != nil {
            return err
        }
        if cat == nil {
            return ErrCategoryNotFound
        }
    }
    return s.repo.Create(p)
}

func (s *ProductService) Update(p *Product) error {
    if p.Name == "" {
        return ErrProductNameRequired
    }
    if p.Price < 0 {
        return ErrNegativePrice
    }
    if p.Stock < 0 {
        return ErrNegativeStock
    }
    existing, err := s.repo.FindByID(p.ID)
    if err != nil {
        return err
    }
    if existing == nil {
        return ErrProductNotFound
    }
    if p.CategoryID != nil && *p.CategoryID > 0 {
        cat, err := s.categoryRepo.FindByID(*p.CategoryID)
        if err != nil {
            return err
        }
        if cat == nil {
            return ErrCategoryNotFound
        }
    }
    return s.repo.Update(p)
}

func (s *ProductService) Delete(id int) error {
    existing, err := s.repo.FindByID(id)
    if err != nil {
        return err
    }
    if existing == nil {
        return ErrProductNotFound
    }
    return s.repo.Delete(id)
}
