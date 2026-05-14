package grpc

import (
	"context"
	"fmt"
	"net"
	"testing"

	"payment-service/internal/models"
	"payment-service/internal/service"
	pb "payment-service/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func dialBuf(t *testing.T, register func(*grpc.Server)) pb.PaymentServiceClient {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	register(s)
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Logf("grpc serve: %v", err)
		}
	}()
	t.Cleanup(func() { s.Stop() })
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return pb.NewPaymentServiceClient(conn)
}

type testPaymentRepo struct {
	payments []models.Payment
}

func (m *testPaymentRepo) Create(p *models.Payment) error {
	m.payments = append(m.payments, *p)
	return nil
}

func (m *testPaymentRepo) GetByID(id string) (*models.Payment, error) {
	for _, p := range m.payments {
		if p.ID == id && !p.IsDeleted {
			cp := p
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *testPaymentRepo) GetByUserID(userID string) ([]models.Payment, error) {
	var res []models.Payment
	for _, p := range m.payments {
		if p.UserID == userID && !p.IsDeleted {
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *testPaymentRepo) GetAll(orderID string) ([]models.Payment, error) {
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

func (m *testPaymentRepo) UpdateStatus(id, status, txnID string) error {
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

func (m *testPaymentRepo) Delete(id string) error {
	for i, p := range m.payments {
		if p.ID == id {
			m.payments[i].IsDeleted = true
			return nil
		}
	}
	return nil
}

func TestPaymentGRPC_CreatePayment(t *testing.T) {
	repo := &testPaymentRepo{}
	svc := service.NewPaymentService(repo)
	srv := NewPaymentServer(svc)
	client := dialBuf(t, func(gs *grpc.Server) {
		pb.RegisterPaymentServiceServer(gs, srv)
	})
	resp, err := client.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		UserId: "u1", OrderId: "o1", Amount: 10, Currency: "USD", PaymentMethod: "card",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetPayment().GetUserId() != "u1" {
		t.Fatalf("user id: %v", resp.GetPayment().GetUserId())
	}
	if len(repo.payments) != 1 {
		t.Fatalf("repo len %d", len(repo.payments))
	}
}

func TestPaymentGRPC_GetPayment_NotFound(t *testing.T) {
	repo := &testPaymentRepo{}
	svc := service.NewPaymentService(repo)
	srv := NewPaymentServer(svc)
	client := dialBuf(t, func(gs *grpc.Server) {
		pb.RegisterPaymentServiceServer(gs, srv)
	})
	_, err := client.GetPayment(context.Background(), &pb.GetPaymentRequest{Id: "missing"})
	if err == nil {
		t.Fatal("expected error")
	}
}
