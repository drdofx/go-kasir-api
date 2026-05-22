package inventory

type InventoryService struct {
	repo InventoryRepository
}

func NewInventoryService(repo InventoryRepository) *InventoryService {
	return &InventoryService{repo: repo}
}

func (s *InventoryService) GetAlerts() ([]Alert, error) {
	return s.repo.FindAlerts()
}

func (s *InventoryService) SetThreshold(t Threshold) error {
	return s.repo.UpsertThreshold(t)
}
