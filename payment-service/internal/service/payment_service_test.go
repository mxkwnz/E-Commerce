package service

import (
	"fmt"
	"testing"

	"payment-service/internal/models"
)

type mockPaymentRepo struct {
	payments []models.Payment
}

func (m *mockPaymentRepo) Create(p *models.Payment) error {
	m.payments = append(m.payments, *p)
	return nil
}

func (m *mockPaymentRepo) GetByID(id string) (*models.Payment, error) {
	for _, p := range m.payments {
		if p.ID == id && !p.IsDeleted {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockPaymentRepo) GetByUserID(userID string) ([]models.Payment, error) {
	var res []models.Payment
	for _, p := range m.payments {
		if p.UserID == userID && !p.IsDeleted {
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *mockPaymentRepo) GetAll(orderID string) ([]models.Payment, error) {
	var res []models.Payment
	for _, p := range m.payments {
		if !p.IsDeleted {
			if orderID == "" || p.OrderID == orderID {
				res = append(res, p)
			}
		}
	}
	return res, nil
}

func (m *mockPaymentRepo) UpdateStatus(id, status, txnID string) error {
	for i, p := range m.payments {
		if p.ID == id && !p.IsDeleted {
			if status != "" {
				m.payments[i].Status = status
			}
			if txnID != "" {
				m.payments[i].TransactionID = txnID
			}
			return nil
		}
	}
	return fmt.Errorf("not found")
}

func (m *mockPaymentRepo) Delete(id string) error {
	for i, p := range m.payments {
		if p.ID == id {
			m.payments[i].IsDeleted = true
			return nil
		}
	}
	return nil
}

func TestCreatePayment_Success(t *testing.T) {
	repo := &mockPaymentRepo{}
	svc := NewPaymentService(repo)

	p, err := svc.CreatePayment("u1", "o1", 100.0, "USD", "credit_card")
	if err != nil {
		t.Fatalf("CreatePayment failed: %v", err)
	}

	if p.Amount != 100.0 {
		t.Errorf("Expected amount 100.0, got %.2f", p.Amount)
	}
	if len(repo.payments) != 1 {
		t.Errorf("Expected 1 payment in repo, got %d", len(repo.payments))
	}
}

func TestProcessPaymentForOrder_Success(t *testing.T) {
	repo := &mockPaymentRepo{}
	svc := NewPaymentService(repo)

	err := svc.ProcessPaymentForOrder("o1", "u1", 50.0, "USD")
	if err != nil {
		t.Fatalf("ProcessPaymentForOrder failed: %v", err)
	}

	if len(repo.payments) != 1 {
		t.Errorf("Expected 1 payment, got %d", len(repo.payments))
	}
	if repo.payments[0].Status != "paid" {
		t.Errorf("Expected status paid, got %s", repo.payments[0].Status)
	}
}

func TestGetByUser_Filtering(t *testing.T) {
	repo := &mockPaymentRepo{
		payments: []models.Payment{
			{ID: "p1", UserID: "u1", OrderID: "o1"},
			{ID: "p2", UserID: "u2", OrderID: "o2"},
		},
	}
	svc := NewPaymentService(repo)

	res := svc.GetByUser("u1")
	if len(res) != 1 {
		t.Errorf("Expected 1 payment for u1, got %d", len(res))
	}
}
