package product

import "errors"

var (
	ErrProductNameRequired = errors.New("product name is required")
	ErrNegativePrice       = errors.New("price must be non-negative")
	ErrCategoryNotFound    = errors.New("category not found")
	ErrProductNotFound     = errors.New("product not found")
)

type CategoryRepository interface {
	FindByIDForOrg(orgID, id int) (*Category, error)
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

func (s *ProductService) FindAllForOrg(orgID int, name string) ([]Product, error) {
	return s.repo.FindAllForOrg(orgID, name)
}

func (s *ProductService) FindByID(id int) (*Product, error) {
	return s.repo.FindByID(id)
}

func (s *ProductService) FindByIDForOrg(orgID, id int) (*Product, error) {
	return s.repo.FindByIDForOrg(orgID, id)
}

func (s *ProductService) Create(p *Product) error {
	if p.Name == "" {
		return ErrProductNameRequired
	}
	if p.Price < 0 {
		return ErrNegativePrice
	}
	if p.CategoryID != nil && *p.CategoryID > 0 {
		cat, err := s.categoryRepo.FindByIDForOrg(p.OrganizationID, *p.CategoryID)
		if err != nil {
			return err
		}
		if cat == nil {
			return ErrCategoryNotFound
		}
	}
	return s.repo.Create(p)
}

func (s *ProductService) CreateForOrg(orgID int, p *Product) error {
	p.OrganizationID = orgID
	if p.Name == "" {
		return ErrProductNameRequired
	}
	if p.Price < 0 {
		return ErrNegativePrice
	}
	if p.CategoryID != nil && *p.CategoryID > 0 {
		cat, err := s.categoryRepo.FindByIDForOrg(orgID, *p.CategoryID)
		if err != nil {
			return err
		}
		if cat == nil {
			return ErrCategoryNotFound
		}
	}
	return s.repo.CreateForOrg(orgID, p)
}

func (s *ProductService) Update(p *Product) error {
	if p.Name == "" {
		return ErrProductNameRequired
	}
	if p.Price < 0 {
		return ErrNegativePrice
	}
	existing, err := s.repo.FindByID(p.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrProductNotFound
	}
	if p.CategoryID != nil && *p.CategoryID > 0 {
		cat, err := s.categoryRepo.FindByIDForOrg(p.OrganizationID, *p.CategoryID)
		if err != nil {
			return err
		}
		if cat == nil {
			return ErrCategoryNotFound
		}
	}
	return s.repo.Update(p)
}

func (s *ProductService) UpdateForOrg(orgID int, p *Product) error {
	p.OrganizationID = orgID
	if p.Name == "" {
		return ErrProductNameRequired
	}
	if p.Price < 0 {
		return ErrNegativePrice
	}
	existing, err := s.repo.FindByIDForOrg(orgID, p.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrProductNotFound
	}
	if p.CategoryID != nil && *p.CategoryID > 0 {
		cat, err := s.categoryRepo.FindByIDForOrg(orgID, *p.CategoryID)
		if err != nil {
			return err
		}
		if cat == nil {
			return ErrCategoryNotFound
		}
	}
	return s.repo.UpdateForOrg(orgID, p)
}

func (s *ProductService) GetStocks(productID int) ([]ProductStock, error) {
	return s.repo.GetStocks(productID)
}

func (s *ProductService) GetStocksForOrg(orgID, productID int) ([]ProductStock, error) {
	return s.repo.GetStocksForOrg(orgID, productID)
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

func (s *ProductService) DeleteForOrg(orgID, id int) error {
	existing, err := s.repo.FindByIDForOrg(orgID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrProductNotFound
	}
	return s.repo.DeleteForOrg(orgID, id)
}
