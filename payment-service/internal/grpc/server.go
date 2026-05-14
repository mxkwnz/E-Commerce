package grpc

import (
	"context"
	"fmt"
	"log"
	"time"

	"payment-service/internal/models"
	"payment-service/internal/service"
	pb "payment-service/proto"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer
	svc *service.PaymentService
}

func NewPaymentServer(svc *service.PaymentService) *PaymentServer {
	return &PaymentServer{svc: svc}
}

func (s *PaymentServer) CreatePayment(
	ctx context.Context,
	req *pb.CreatePaymentRequest,
) (*pb.PaymentResponse, error) {
	log.Printf("[gRPC] CreatePayment: user=%s order=%s amount=%.2f",
		req.UserId, req.OrderId, req.Amount)

	if req.UserId == "" || req.OrderId == "" {
		return nil, fmt.Errorf("userId and orderId are required")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	p, err := s.svc.CreatePayment(req.UserId, req.OrderId, req.Amount, req.Currency, req.PaymentMethod)
	if err != nil {
		return nil, err
	}
	return &pb.PaymentResponse{Payment: toProto(p)}, nil
}

func (s *PaymentServer) GetPayment(
	ctx context.Context,
	req *pb.GetPaymentRequest,
) (*pb.PaymentResponse, error) {
	log.Printf("[gRPC] GetPayment: id=%s", req.Id)
	if req.Id == "" {
		return nil, fmt.Errorf("id is required")
	}
	p, err := s.svc.GetPayment(req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.PaymentResponse{Payment: toProto(p)}, nil
}

func (s *PaymentServer) ListUserPayments(
	ctx context.Context,
	req *pb.ListUserPaymentsRequest,
) (*pb.ListPaymentsResponse, error) {
	log.Printf("[gRPC] ListUserPayments: user=%s", req.UserId)
	if req.UserId == "" {
		return nil, fmt.Errorf("userId is required")
	}
	payments := s.svc.GetByUser(req.UserId)
	var pbPayments []*pb.Payment
	for _, p := range payments {
		cp := p
		pbPayments = append(pbPayments, toProto(&cp))
	}
	return &pb.ListPaymentsResponse{
		Payments:   pbPayments,
		TotalCount: int32(len(pbPayments)),
	}, nil
}

func (s *PaymentServer) UpdatePaymentStatus(
	ctx context.Context,
	req *pb.UpdatePaymentStatusRequest,
) (*pb.PaymentResponse, error) {
	log.Printf("[gRPC] UpdatePaymentStatus: id=%s status=%s", req.Id, req.Status)
	if req.Id == "" {
		return nil, fmt.Errorf("id is required")
	}
	p, err := s.svc.UpdateStatus(req.Id, req.Status, req.TransactionId)
	if err != nil {
		return nil, err
	}
	return &pb.PaymentResponse{Payment: toProto(p)}, nil
}

func toProto(p *models.Payment) *pb.Payment {
	if p == nil {
		return nil
	}
	now := timestamppb.New(time.Now())
	return &pb.Payment{
		Id:            p.ID,
		UserId:        p.UserID,
		OrderId:       p.OrderID,
		Amount:        p.Amount,
		Currency:      p.Currency,
		Status:        p.Status,
		PaymentMethod: p.PaymentMethod,
		TransactionId: p.TransactionID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
