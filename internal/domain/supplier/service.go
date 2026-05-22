package supplier

import "errors"

var (
	ErrSupplierNameRequired = errors.New("supplier name is required")
	ErrSupplierNotFound     = errors.New("supplier not found")
)

type SupplierService struct {
	repo SupplierRepository
}

func NewSupplierService(repo SupplierRepository) *SupplierService {
	return &SupplierService{repo: repo}
}

func (s *SupplierService) FindAll(search string) ([]Supplier, error) { return s.repo.FindAll(search) }
func (s *SupplierService) FindByID(id int) (*Supplier, error)        { return s.repo.FindByID(id) }

func (s *SupplierService) Create(sup *Supplier) error {
	if sup.Name == "" {
		return ErrSupplierNameRequired
	}
	return s.repo.Create(sup)
}

func (s *SupplierService) Update(sup *Supplier) error {
	if sup.Name == "" {
		return ErrSupplierNameRequired
	}
	existing, err := s.repo.FindByID(sup.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrSupplierNotFound
	}
	return s.repo.Update(sup)
}

func (s *SupplierService) Delete(id int) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrSupplierNotFound
	}
	return s.repo.Delete(id)
}
