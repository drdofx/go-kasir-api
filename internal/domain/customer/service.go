package customer

import "errors"

var (
	ErrCustomerNameRequired = errors.New("customer name is required")
	ErrCustomerNotFound     = errors.New("customer not found")
)

type CustomerService struct {
	repo CustomerRepository
}

func NewCustomerService(repo CustomerRepository) *CustomerService {
	return &CustomerService{repo: repo}
}

func (s *CustomerService) FindAll(search string) ([]Customer, error) {
	return s.repo.FindAll(search)
}

func (s *CustomerService) FindAllForOrg(orgID int, search string) ([]Customer, error) {
	return s.repo.FindAllForOrg(orgID, search)
}

func (s *CustomerService) FindByID(id int) (*Customer, error) {
	return s.repo.FindByID(id)
}

func (s *CustomerService) FindByIDForOrg(orgID, id int) (*Customer, error) {
	return s.repo.FindByIDForOrg(orgID, id)
}

func (s *CustomerService) Create(c *Customer) error {
	if c.Name == "" {
		return ErrCustomerNameRequired
	}
	return s.repo.Create(c)
}

func (s *CustomerService) CreateForOrg(orgID int, c *Customer) error {
	if c.Name == "" {
		return ErrCustomerNameRequired
	}
	return s.repo.CreateForOrg(orgID, c)
}

func (s *CustomerService) Update(c *Customer) error {
	if c.Name == "" {
		return ErrCustomerNameRequired
	}
	existing, err := s.repo.FindByID(c.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrCustomerNotFound
	}
	return s.repo.Update(c)
}

func (s *CustomerService) UpdateForOrg(orgID int, c *Customer) error {
	if c.Name == "" {
		return ErrCustomerNameRequired
	}
	existing, err := s.repo.FindByIDForOrg(orgID, c.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrCustomerNotFound
	}
	return s.repo.UpdateForOrg(orgID, c)
}

func (s *CustomerService) Delete(id int) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrCustomerNotFound
	}
	return s.repo.Delete(id)
}

func (s *CustomerService) DeleteForOrg(orgID, id int) error {
	existing, err := s.repo.FindByIDForOrg(orgID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrCustomerNotFound
	}
	return s.repo.DeleteForOrg(orgID, id)
}

func (s *CustomerService) GetPurchaseHistory(customerID int) ([]PurchaseRecord, error) {
	return s.repo.GetPurchaseHistory(customerID)
}

func (s *CustomerService) GetPurchaseHistoryForOrg(orgID, customerID int) ([]PurchaseRecord, error) {
	return s.repo.GetPurchaseHistoryForOrg(orgID, customerID)
}
