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

func (s *InventoryService) GetAlertsForOrg(orgID int) ([]Alert, error) {
	return s.repo.FindAlertsForOrg(orgID)
}

func (s *InventoryService) SetThreshold(t Threshold) error {
	return s.repo.UpsertThreshold(t)
}

func (s *InventoryService) SetThresholdForOrg(orgID int, t Threshold) error {
	return s.repo.UpsertThresholdForOrg(orgID, t)
}
