package grpc

import (
	"context"

	"payment-service/internal/models"
	"payment-service/internal/service"
	pb "payment-service/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer
	svc *service.PaymentService
}

func NewPaymentServer(svc *service.PaymentService) *PaymentServer {
	return &PaymentServer{svc: svc}
}

func (s *PaymentServer) CreatePayment(ctx context.Context, req *pb.CreatePaymentRequest) (*pb.PaymentResponse, error) {
	p, err := s.svc.CreatePayment(req.GetUserId(), req.GetOrderId(), req.GetAmount(), req.GetCurrency(), req.GetPaymentMethod())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	return &pb.PaymentResponse{Payment: paymentToProto(p)}, nil
}

func (s *PaymentServer) GetPayment(ctx context.Context, req *pb.GetPaymentRequest) (*pb.PaymentResponse, error) {
	p, err := s.svc.GetPayment(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "%v", err)
	}
	return &pb.PaymentResponse{Payment: paymentToProto(p)}, nil
}

func (s *PaymentServer) ListUserPayments(ctx context.Context, req *pb.ListUserPaymentsRequest) (*pb.ListPaymentsResponse, error) {
	all := s.svc.GetByUser(req.GetUserId())
	page := int(req.GetPage())
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize <= 0 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	if start > len(all) {
		start = len(all)
	}
	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}
	slice := all[start:end]
	out := make([]*pb.Payment, 0, len(slice))
	for i := range slice {
		cp := slice[i]
		out = append(out, paymentToProto(&cp))
	}
	return &pb.ListPaymentsResponse{
		Payments:   out,
		TotalCount: int32(len(all)),
	}, nil
}

func (s *PaymentServer) UpdatePaymentStatus(ctx context.Context, req *pb.UpdatePaymentStatusRequest) (*pb.PaymentResponse, error) {
	p, err := s.svc.UpdateStatus(req.GetId(), req.GetStatus(), req.GetTransactionId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "%v", err)
	}
	return &pb.PaymentResponse{Payment: paymentToProto(p)}, nil
}

func paymentToProto(p *models.Payment) *pb.Payment {
	if p == nil {
		return nil
	}
	return &pb.Payment{
		Id:              p.ID,
		UserId:          p.UserID,
		OrderId:         p.OrderID,
		Amount:          p.Amount,
		Currency:        p.Currency,
		Status:          p.Status,
		PaymentMethod:   p.PaymentMethod,
		TransactionId:   p.TransactionID,
		CreatedAt:       timestamppb.New(p.CreatedAt),
		UpdatedAt:       timestamppb.New(p.UpdatedAt),
	}
}
