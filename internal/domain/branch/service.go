package branch

import "errors"

var (
	ErrBranchNameRequired = errors.New("branch name is required")
	ErrBranchNotFound     = errors.New("branch not found")
)

type BranchService struct {
	repo BranchRepository
}

func NewBranchService(repo BranchRepository) *BranchService {
	return &BranchService{repo: repo}
}

func (s *BranchService) FindByOrgID(orgID int) ([]Branch, error) {
	return s.repo.FindByOrgID(orgID)
}

func (s *BranchService) Create(b *Branch) error {
	if b.Name == "" {
		return ErrBranchNameRequired
	}
	return s.repo.Create(b)
}

func (s *BranchService) Update(b *Branch) error {
	if b.Name == "" {
		return ErrBranchNameRequired
	}
	existing, err := s.repo.FindByID(b.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrBranchNotFound
	}
	return s.repo.Update(b)
}
