package payment

import (
	"database/sql"
	"errors"
)

var ErrPaymentTypeNotFound = errors.New("payment type not found")

type PaymentService struct {
	repo PaymentRepository
}

func NewPaymentService(repo PaymentRepository) *PaymentService {
	return &PaymentService{repo: repo}
}

func (s *PaymentService) GetPaymentTypes() ([]PaymentType, error) {
	return s.repo.FindAllPaymentTypes()
}

func (s *PaymentService) GetPaymentTypeIDByName(name string) (int, error) {
	types, err := s.repo.FindAllPaymentTypes()
	if err != nil {
		return 0, err
	}
	for _, pt := range types {
		if pt.Name == name {
			return pt.ID, nil
		}
	}
	return 0, ErrPaymentTypeNotFound
}

func (s *PaymentService) InsertPayment(tx *sql.Tx, transactionID, paymentTypeID, amount int) error {
	return s.repo.InsertPayment(tx, transactionID, paymentTypeID, amount)
}
