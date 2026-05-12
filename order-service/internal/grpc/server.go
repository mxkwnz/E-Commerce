package grpc

import (
	"context"
	"fmt"
	"log"

	"order-service/internal/currency"
	"order-service/internal/models"
	"order-service/internal/repository"
	"order-service/internal/service"
	pb "order-service/proto"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderServer struct {
	pb.UnimplementedOrderServiceServer
	cartService  *service.CartService
	orderService *service.OrderService
}

func NewOrderServer(nc *nats.Conn) *OrderServer {
	cartRepo := repository.NewCartRepository()
	orderRepo := repository.NewOrderRepository()

	cartService := service.NewCartService(cartRepo)
	orderService := service.NewOrderService(orderRepo, cartRepo, nc)

	return &OrderServer{
		cartService:  cartService,
		orderService: orderService,
	}
}

func (s *OrderServer) AddToCart(ctx context.Context, req *pb.AddToCartRequest) (*pb.CartItemResponse, error) {
	log.Printf("gRPC AddToCart called for user: %s, product: %s", req.UserId, req.ProductId)

	if req.UserId == "" || req.ProductId == "" {
		return nil, fmt.Errorf("user_id and product_id are required")
	}

	if req.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be positive")
	}

	item := &models.CartItem{
		UserID:    req.UserId,
		ProductID: req.ProductId,
		Quantity:  int(req.Quantity),
		UnitPrice: req.UnitPrice,
		Currency:  req.Currency,
	}

	if item.Currency == "" {
		item.Currency = "USD"
	}

	if err := s.cartService.AddToCart(item); err != nil {
		return nil, fmt.Errorf("failed to add to cart: %w", err)
	}

	return &pb.CartItemResponse{
		Item: &pb.CartItem{
			Id:        item.ID,
			UserId:    item.UserID,
			ProductId: item.ProductID,
			Quantity:  int32(item.Quantity),
			UnitPrice: item.UnitPrice,
			Currency:  item.Currency,
			CreatedAt: timestamppb.New(item.CreatedAt),
			UpdatedAt: timestamppb.New(item.UpdatedAt),
		},
	}, nil
}

func (s *OrderServer) GetCart(ctx context.Context, req *pb.GetCartRequest) (*pb.CartResponse, error) {
	log.Printf("gRPC GetCart called for user: %s", req.UserId)

	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	items, err := s.cartService.GetUserCart(req.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	ccy, err := currency.ValidateCartUniform(items)
	if err != nil {
		return nil, err
	}

	var pbItems []*pb.CartItem
	for _, item := range items {
		pbItems = append(pbItems, &pb.CartItem{
			Id:        item.ID,
			UserId:    item.UserID,
			ProductId: item.ProductID,
			Quantity:  int32(item.Quantity),
			UnitPrice: item.UnitPrice,
			Currency:  item.Currency,
			CreatedAt: timestamppb.New(item.CreatedAt),
			UpdatedAt: timestamppb.New(item.UpdatedAt),
		})
	}

	return &pb.CartResponse{
		Items:       pbItems,
		TotalAmount: currency.SumLineTotals(items),
		Currency:    ccy,
		TotalItems:  int32(len(items)),
	}, nil
}

func (s *OrderServer) GetCartItem(ctx context.Context, req *pb.GetCartItemRequest) (*pb.CartItemResponse, error) {
	log.Printf("gRPC GetCartItem called for item: %s", req.Id)

	if req.Id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	item, err := s.cartService.GetCartItemForUser(req.Id, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("cart item not found: %w", err)
	}

	return &pb.CartItemResponse{
		Item: &pb.CartItem{
			Id:        item.ID,
			UserId:    item.UserID,
			ProductId: item.ProductID,
			Quantity:  int32(item.Quantity),
			UnitPrice: item.UnitPrice,
			Currency:  item.Currency,
			CreatedAt: timestamppb.New(item.CreatedAt),
			UpdatedAt: timestamppb.New(item.UpdatedAt),
		},
	}, nil
}

func (s *OrderServer) UpdateCartItem(ctx context.Context, req *pb.UpdateCartItemRequest) (*pb.CartItemResponse, error) {
	log.Printf("gRPC UpdateCartItem called for item: %s", req.Id)

	if req.Id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	if req.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be positive")
	}

	if err := s.cartService.UpdateCartItemForUser(req.Id, req.UserId, int(req.Quantity)); err != nil {
		return nil, fmt.Errorf("failed to update cart item: %w", err)
	}

	item, err := s.cartService.GetCartItemForUser(req.Id, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to load cart item: %w", err)
	}

	return &pb.CartItemResponse{
		Item: &pb.CartItem{
			Id:        item.ID,
			UserId:    item.UserID,
			ProductId: item.ProductID,
			Quantity:  int32(item.Quantity),
			UnitPrice: item.UnitPrice,
			Currency:  item.Currency,
			CreatedAt: timestamppb.New(item.CreatedAt),
			UpdatedAt: timestamppb.New(item.UpdatedAt),
		},
	}, nil
}

func (s *OrderServer) RemoveFromCart(ctx context.Context, req *pb.RemoveFromCartRequest) (*pb.RemoveFromCartResponse, error) {
	log.Printf("gRPC RemoveFromCart called for item: %s", req.Id)

	if req.Id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	if err := s.cartService.RemoveFromCartForUser(req.Id, req.UserId); err != nil {
		return &pb.RemoveFromCartResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.RemoveFromCartResponse{
		Success: true,
		Message: "Item removed from cart",
	}, nil
}

func (s *OrderServer) ClearCart(ctx context.Context, req *pb.ClearCartRequest) (*pb.ClearCartResponse, error) {
	log.Printf("gRPC ClearCart called for user: %s", req.UserId)

	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	items, err := s.cartService.GetUserCart(req.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}
	itemCount := len(items)

	if err := s.cartService.ClearCart(req.UserId); err != nil {
		return nil, fmt.Errorf("failed to clear cart: %w", err)
	}

	return &pb.ClearCartResponse{
		Success:      true,
		ItemsRemoved: int32(itemCount),
	}, nil
}

func (s *OrderServer) Checkout(ctx context.Context, req *pb.CheckoutRequest) (*pb.CheckoutResponse, error) {
	log.Printf("gRPC Checkout called for user: %s", req.UserId)

	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	resp, err := s.orderService.Checkout(req.UserId)
	if err != nil {
		return nil, fmt.Errorf("checkout failed: %w", err)
	}

	return &pb.CheckoutResponse{
		OrderId:     resp.OrderID,
		TotalAmount: resp.TotalAmount,
		Currency:    resp.Currency,
		Status:      resp.Status,
	}, nil
}

func (s *OrderServer) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.OrderResponse, error) {
	log.Printf("gRPC GetOrder called for order: %s", req.Id)

	if req.Id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	order, err := s.orderService.GetOrderForUser(req.Id, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	return &pb.OrderResponse{
		Order: convertOrderToProto(order),
	}, nil
}

func (s *OrderServer) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	log.Printf("gRPC ListOrders called for user: %s", req.UserId)

	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	orders, err := s.orderService.GetUserOrders(req.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}

	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= len(orders) {
		return &pb.ListOrdersResponse{
			Orders:     []*pb.Order{},
			TotalCount: int32(len(orders)),
			Page:       int32(page),
			PageSize:   int32(pageSize),
		}, nil
	}

	if end > len(orders) {
		end = len(orders)
	}

	paginatedOrders := orders[start:end]
	var pbOrders []*pb.Order
	for i := range paginatedOrders {
		o := paginatedOrders[i]
		pbOrders = append(pbOrders, convertOrderToProto(&o))
	}

	return &pb.ListOrdersResponse{
		Orders:     pbOrders,
		TotalCount: int32(len(orders)),
		Page:       int32(page),
		PageSize:   int32(pageSize),
	}, nil
}

func (s *OrderServer) ConfirmOrder(ctx context.Context, req *pb.ConfirmOrderRequest) (*pb.OrderStatusResponse, error) {
	log.Printf("gRPC ConfirmOrder called for order: %s", req.Id)

	if req.Id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	if err := s.orderService.ConfirmOrderForUser(req.Id, req.UserId); err != nil {
		return &pb.OrderStatusResponse{
			Success: false,
			Message: err.Error(),
			OrderId: req.Id,
			Status:  "failed",
		}, nil
	}

	return &pb.OrderStatusResponse{
		Success: true,
		Message: "Order confirmed successfully",
		OrderId: req.Id,
		Status:  "confirmed",
	}, nil
}

func (s *OrderServer) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.OrderStatusResponse, error) {
	log.Printf("gRPC CancelOrder called for order: %s, reason: %s", req.Id, req.Reason)

	if req.Id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	if err := s.orderService.CancelOrderForUser(req.Id, req.UserId); err != nil {
		return &pb.OrderStatusResponse{
			Success: false,
			Message: err.Error(),
			OrderId: req.Id,
			Status:  "failed",
		}, nil
	}

	return &pb.OrderStatusResponse{
		Success: true,
		Message: fmt.Sprintf("Order cancelled: %s", req.Reason),
		OrderId: req.Id,
		Status:  "cancelled",
	}, nil
}

func (s *OrderServer) GetOrdersByStatus(ctx context.Context, req *pb.GetOrdersByStatusRequest) (*pb.ListOrdersResponse, error) {
	log.Printf("gRPC GetOrdersByStatus called for user: %s, status: %s", req.UserId, req.Status)

	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	orders, err := s.orderService.GetOrdersByStatus(req.UserId, req.Status)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by status: %w", err)
	}

	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= len(orders) {
		return &pb.ListOrdersResponse{
			Orders:     []*pb.Order{},
			TotalCount: int32(len(orders)),
			Page:       int32(page),
			PageSize:   int32(pageSize),
		}, nil
	}

	if end > len(orders) {
		end = len(orders)
	}

	paginatedOrders := orders[start:end]
	var pbOrders []*pb.Order
	for i := range paginatedOrders {
		o := paginatedOrders[i]
		pbOrders = append(pbOrders, convertOrderToProto(&o))
	}

	return &pb.ListOrdersResponse{
		Orders:     pbOrders,
		TotalCount: int32(len(orders)),
		Page:       int32(page),
		PageSize:   int32(pageSize),
	}, nil
}

func (s *OrderServer) GetOrderStatistics(ctx context.Context, req *pb.GetOrderStatisticsRequest) (*pb.OrderStatisticsResponse, error) {
	log.Printf("gRPC GetOrderStatistics called for user: %s", req.UserId)

	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	stats, err := s.orderService.GetStatistics(req.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	return &pb.OrderStatisticsResponse{
		TotalOrders:     int32(stats.TotalOrders),
		PendingOrders:   int32(stats.PendingOrders),
		ConfirmedOrders: int32(stats.ConfirmedOrders),
		CancelledOrders: int32(stats.CancelledOrders),
		TotalRevenue:    stats.TotalRevenue,
		Currency:        stats.Currency,
	}, nil
}

func (s *OrderServer) SearchOrders(ctx context.Context, req *pb.SearchOrdersRequest) (*pb.ListOrdersResponse, error) {
	log.Printf("gRPC SearchOrders called for user: %s, query: %s", req.UserId, req.Query)

	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	orders, err := s.orderService.SearchOrders(req.UserId, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to search orders: %w", err)
	}

	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= len(orders) {
		return &pb.ListOrdersResponse{
			Orders:     []*pb.Order{},
			TotalCount: int32(len(orders)),
			Page:       int32(page),
			PageSize:   int32(pageSize),
		}, nil
	}

	if end > len(orders) {
		end = len(orders)
	}

	paginatedOrders := orders[start:end]
	var pbOrders []*pb.Order
	for i := range paginatedOrders {
		o := paginatedOrders[i]
		pbOrders = append(pbOrders, convertOrderToProto(&o))
	}

	return &pb.ListOrdersResponse{
		Orders:     pbOrders,
		TotalCount: int32(len(orders)),
		Page:       int32(page),
		PageSize:   int32(pageSize),
	}, nil
}

func convertOrderToProto(order *models.Order) *pb.Order {
	var pbItems []*pb.OrderItem
	for _, item := range order.Items {
		pbItems = append(pbItems, &pb.OrderItem{
			Id:        item.ID,
			OrderId:   item.OrderID,
			ProductId: item.ProductID,
			Quantity:  int32(item.Quantity),
			UnitPrice: item.UnitPrice,
			Currency:  item.Currency,
			CreatedAt: timestamppb.New(item.CreatedAt),
		})
	}

	return &pb.Order{
		Id:          order.ID,
		UserId:      order.UserID,
		Items:       pbItems,
		TotalAmount: order.TotalAmount,
		Currency:    order.Currency,
		Status:      order.Status,
		CreatedAt:   timestamppb.New(order.CreatedAt),
		UpdatedAt:   timestamppb.New(order.UpdatedAt),
	}
}
