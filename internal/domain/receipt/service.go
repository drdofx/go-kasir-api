package receipt

type ReceiptService struct {
	repo ReceiptRepository
}

func NewReceiptService(repo ReceiptRepository) *ReceiptService {
	return &ReceiptService{repo: repo}
}

func (s *ReceiptService) GetReceipt(transactionID int) (*Receipt, error) {
	return s.repo.FindByTransactionID(transactionID)
}
