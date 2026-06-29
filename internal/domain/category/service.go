package category

import "errors"

var (
	ErrCategoryNameRequired = errors.New("category name is required")
	ErrCategoryDescRequired = errors.New("category description is required")
	ErrCategoryNotFound     = errors.New("category not found")
)

type CategoryService struct {
	repo CategoryRepository
}

func NewCategoryService(repo CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) FindAll() ([]Category, error) {
	return s.repo.FindAll()
}

func (s *CategoryService) FindAllForOrg(orgID int) ([]Category, error) {
	return s.repo.FindAllForOrg(orgID)
}

func (s *CategoryService) FindByID(id int) (*Category, error) {
	return s.repo.FindByID(id)
}

func (s *CategoryService) FindByIDForOrg(orgID, id int) (*Category, error) {
	return s.repo.FindByIDForOrg(orgID, id)
}

func (s *CategoryService) Create(c *Category) error {
	if c.Name == "" {
		return ErrCategoryNameRequired
	}
	if c.Description == "" {
		return ErrCategoryDescRequired
	}
	return s.repo.Create(c)
}

func (s *CategoryService) CreateForOrg(orgID int, c *Category) error {
	if c.Name == "" {
		return ErrCategoryNameRequired
	}
	if c.Description == "" {
		return ErrCategoryDescRequired
	}
	return s.repo.CreateForOrg(orgID, c)
}

func (s *CategoryService) Update(c *Category) error {
	if c.Name == "" {
		return ErrCategoryNameRequired
	}
	if c.Description == "" {
		return ErrCategoryDescRequired
	}
	existing, err := s.repo.FindByID(c.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrCategoryNotFound
	}
	return s.repo.Update(c)
}

func (s *CategoryService) UpdateForOrg(orgID int, c *Category) error {
	if c.Name == "" {
		return ErrCategoryNameRequired
	}
	if c.Description == "" {
		return ErrCategoryDescRequired
	}
	existing, err := s.repo.FindByIDForOrg(orgID, c.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrCategoryNotFound
	}
	return s.repo.UpdateForOrg(orgID, c)
}

func (s *CategoryService) Delete(id int) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrCategoryNotFound
	}
	return s.repo.Delete(id)
}

func (s *CategoryService) DeleteForOrg(orgID, id int) error {
	existing, err := s.repo.FindByIDForOrg(orgID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrCategoryNotFound
	}
	return s.repo.DeleteForOrg(orgID, id)
}
