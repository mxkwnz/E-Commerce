package service

import (
	"fmt"
	"log"

	"payment-service/internal/models"
)

type PaymentRepository interface {
	Create(p *models.Payment) error
	GetByID(id string) (*models.Payment, error)
	GetByUserID(userID string) ([]models.Payment, error)
	GetAll(orderID string) ([]models.Payment, error)
	UpdateStatus(id, status, txnID string) error
	Delete(id string) error
}

type PaymentService struct {
	repo PaymentRepository
}

func NewPaymentService(repo PaymentRepository) *PaymentService {
	return &PaymentService{repo: repo}
}

func (s *PaymentService) ProcessPaymentForOrder(orderID, userID string, amount float64, currency string) error {
	if orderID == "" || userID == "" {
		return fmt.Errorf("orderID and userID are required")
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if currency == "" {
		currency = "USD"
	}

	p := &models.Payment{
		ID:            generateID(),
		UserID:        userID,
		OrderID:       orderID,
		Amount:        amount,
		Currency:      currency,
		Status:        "paid",
		PaymentMethod: "auto",
		TransactionID: "txn_" + generateID(),
	}

	if err := s.repo.Create(p); err != nil {
		return err
	}

	log.Printf("[PaymentService] processed payment %s for order %s: %.2f %s",
		p.ID, orderID, amount, currency)
	return nil
}

func (s *PaymentService) CreatePayment(userID, orderID string, amount float64, currency, method string) (*models.Payment, error) {
	if userID == "" || orderID == "" {
		return nil, fmt.Errorf("userId and orderId required")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if currency == "" {
		currency = "USD"
	}
	p := &models.Payment{
		ID:            generateID(),
		UserID:        userID,
		OrderID:       orderID,
		Amount:        amount,
		Currency:      currency,
		Status:        "paid",
		PaymentMethod: method,
		TransactionID: "txn_" + generateID(),
	}
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *PaymentService) GetPayment(id string) (*models.Payment, error) {
	return s.repo.GetByID(id)
}

func (s *PaymentService) GetByUser(userID string) []models.Payment {
	res, err := s.repo.GetByUserID(userID)
	if err != nil {
		return []models.Payment{}
	}
	return res
}

func (s *PaymentService) GetAll(orderID string) []models.Payment {
	res, err := s.repo.GetAll(orderID)
	if err != nil {
		return []models.Payment{}
	}
	return res
}

func (s *PaymentService) UpdateStatus(id, status, txnID string) (*models.Payment, error) {
	if err := s.repo.UpdateStatus(id, status, txnID); err != nil {
		return nil, err
	}
	return s.repo.GetByID(id)
}

func (s *PaymentService) Delete(id string) error {
	return s.repo.Delete(id)
}
