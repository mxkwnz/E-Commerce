package grpc

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/final-ap2-course2/product-service/internal/models"
	"github.com/final-ap2-course2/product-service/internal/repository"
	"github.com/final-ap2-course2/product-service/internal/service"
	pb "github.com/final-ap2-course2/product-service/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProductServer struct {
	pb.UnimplementedProductServiceServer
	productService *service.ProductService
}

func NewProductServer() *ProductServer {
	productRepo := repository.NewProductRepository()
	invRepo := repository.NewInventoryRepository()
	return &ProductServer{
		productService: service.NewProductService(productRepo, invRepo),
	}
}

func (s *ProductServer) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.ProductResponse, error) {
	log.Printf("gRPC CreateProduct called: %s", req.Name)
	if req.Name == "" {
		return nil, fmt.Errorf("product name is required")
	}
	if req.Price < 0 {
		return nil, fmt.Errorf("price cannot be negative")
	}

	product := &models.Product{
		Name:        req.Name,
		PhotoURL:    req.PhotoUrl,
		Description: req.Description,
		Brand:       req.Brand,
		Price:       req.Price,
		Currency:    req.Currency,
		Category:    req.Category,
		Stock:       int(req.InitialStock),
	}
	if product.Currency == "" {
		product.Currency = "USD"
	}

	if err := s.productService.CreateProduct(product); err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	return &pb.ProductResponse{
		Product: &pb.Product{
			Id:          product.ID,
			Name:        product.Name,
			PhotoUrl:    product.PhotoURL,
			Description: product.Description,
			Brand:       product.Brand,
			Price:       product.Price,
			Currency:    product.Currency,
			Category:    product.Category,
			Stock:       int32(product.Stock),
			CreatedAt:   timestamppb.New(product.CreatedAt),
			UpdatedAt:   timestamppb.New(product.UpdatedAt),
		},
	}, nil
}

func (s *ProductServer) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.ProductResponse, error) {
	log.Printf("gRPC GetProduct called: %s", req.Id)
	if req.Id == "" {
		return nil, fmt.Errorf("product id is required")
	}
	product, err := s.productService.GetProduct(req.Id)
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}
	return &pb.ProductResponse{Product: convertProductToProto(product)}, nil
}

func (s *ProductServer) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	log.Printf("gRPC ListProducts called: page=%d, size=%d", req.Page, req.PageSize)
	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	products, totalCount, err := s.productService.ListProducts(page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	var pbProducts []*pb.Product
	for _, product := range products {
		pbProducts = append(pbProducts, convertProductToProto(&product))
	}
	return &pb.ListProductsResponse{Products: pbProducts, TotalCount: int32(totalCount), Page: int32(page), PageSize: int32(pageSize)}, nil
}

func (s *ProductServer) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.ProductResponse, error) {
	log.Printf("gRPC UpdateProduct called: %s", req.Id)
	if req.Id == "" {
		return nil, fmt.Errorf("product id is required")
	}

	updates := &models.Product{
		Name:        req.Name,
		PhotoURL:    req.PhotoUrl,
		Description: req.Description,
		Brand:       req.Brand,
		Price:       req.Price,
		Currency:    req.Currency,
		Category:    req.Category,
	}

	if err := s.productService.UpdateProduct(req.Id, updates); err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}
	product, err := s.productService.GetProduct(req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.ProductResponse{Product: convertProductToProto(product)}, nil
}

func (s *ProductServer) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*pb.DeleteProductResponse, error) {
	log.Printf("gRPC DeleteProduct called: %s", req.Id)
	if req.Id == "" {
		return nil, fmt.Errorf("product id is required")
	}
	if err := s.productService.DeleteProduct(req.Id); err != nil {
		return &pb.DeleteProductResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.DeleteProductResponse{Success: true, Message: "Product deleted successfully"}, nil
}

func (s *ProductServer) GetInventory(ctx context.Context, req *pb.GetInventoryRequest) (*pb.InventoryResponse, error) {
	log.Printf("gRPC GetInventory called: %s", req.ProductId)
	if req.ProductId == "" {
		return nil, fmt.Errorf("product id is required")
	}
	inventory, err := s.productService.GetInventory(req.ProductId)
	if err != nil {
		return nil, fmt.Errorf("inventory not found: %w", err)
	}
	return &pb.InventoryResponse{Inventory: &pb.Inventory{
		Id:        inventory.ID,
		ProductId: inventory.ProductID,
		Quantity:  int32(inventory.Quantity),
		Reserved:  int32(inventory.Reserved),
		Available: int32(inventory.Available),
		CreatedAt: timestamppb.New(inventory.CreatedAt),
		UpdatedAt: timestamppb.New(inventory.UpdatedAt),
	}}, nil
}

func (s *ProductServer) UpdateInventory(ctx context.Context, req *pb.UpdateInventoryRequest) (*pb.InventoryResponse, error) {
	log.Printf("gRPC UpdateInventory called: product=%s, quantity=%d", req.ProductId, req.Quantity)
	if req.ProductId == "" {
		return nil, fmt.Errorf("product id is required")
	}
	if req.Quantity < 0 {
		return nil, fmt.Errorf("quantity cannot be negative")
	}
	if err := s.productService.UpdateInventory(req.ProductId, int(req.Quantity)); err != nil {
		return nil, fmt.Errorf("failed to update inventory: %w", err)
	}
	inventory, err := s.productService.GetInventory(req.ProductId)
	if err != nil {
		return nil, err
	}
	return &pb.InventoryResponse{Inventory: &pb.Inventory{
		Id:        inventory.ID,
		ProductId: inventory.ProductID,
		Quantity:  int32(inventory.Quantity),
		Reserved:  int32(inventory.Reserved),
		Available: int32(inventory.Available),
		CreatedAt: timestamppb.New(inventory.CreatedAt),
		UpdatedAt: timestamppb.New(inventory.UpdatedAt),
	}}, nil
}

func (s *ProductServer) ReserveStock(ctx context.Context, req *pb.ReserveStockRequest) (*pb.ReserveStockResponse, error) {
	log.Printf("gRPC ReserveStock called: product=%s, amount=%d", req.ProductId, req.Amount)
	if req.ProductId == "" {
		return nil, fmt.Errorf("product id is required")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if err := s.productService.ReserveStock(req.ProductId, int(req.Amount)); err != nil {
		return &pb.ReserveStockResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.ReserveStockResponse{Success: true, Message: "Stock reserved successfully", ReservedAmount: req.Amount}, nil
}

func (s *ProductServer) ReleaseStock(ctx context.Context, req *pb.ReleaseStockRequest) (*pb.ReleaseStockResponse, error) {
	log.Printf("gRPC ReleaseStock called: product=%s, amount=%d", req.ProductId, req.Amount)
	if req.ProductId == "" {
		return nil, fmt.Errorf("product id is required")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if err := s.productService.ReleaseStock(req.ProductId, int(req.Amount)); err != nil {
		return &pb.ReleaseStockResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.ReleaseStockResponse{Success: true, Message: "Stock released successfully", ReleasedAmount: req.Amount}, nil
}

func (s *ProductServer) SearchProducts(ctx context.Context, req *pb.SearchProductsRequest) (*pb.ListProductsResponse, error) {
	log.Printf("gRPC SearchProducts called: query=%s", req.Query)
	if req.Query == "" {
		return nil, fmt.Errorf("search query is required")
	}
	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	products, totalCount, err := s.productService.SearchProducts(req.Query, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	var pbProducts []*pb.Product
	for _, product := range products {
		pbProducts = append(pbProducts, convertProductToProto(&product))
	}
	return &pb.ListProductsResponse{Products: pbProducts, TotalCount: int32(totalCount), Page: int32(page), PageSize: int32(pageSize)}, nil
}

func (s *ProductServer) GetProductsByBrand(ctx context.Context, req *pb.GetProductsByBrandRequest) (*pb.ListProductsResponse, error) {
	log.Printf("gRPC GetProductsByBrand called: brand=%s", req.Brand)
	if req.Brand == "" {
		return nil, fmt.Errorf("brand is required")
	}
	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	products, totalCount, err := s.productService.GetProductsByBrand(req.Brand, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get products by brand: %w", err)
	}
	var pbProducts []*pb.Product
	for _, product := range products {
		pbProducts = append(pbProducts, convertProductToProto(&product))
	}
	return &pb.ListProductsResponse{Products: pbProducts, TotalCount: int32(totalCount), Page: int32(page), PageSize: int32(pageSize)}, nil
}

func (s *ProductServer) GetProductsByCategory(ctx context.Context, req *pb.GetProductsByCategoryRequest) (*pb.ListProductsResponse, error) {
	log.Printf("gRPC GetProductsByCategory called: category=%s", req.Category)
	if req.Category == "" {
		return nil, fmt.Errorf("category is required")
	}
	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	products, totalCount, err := s.productService.GetProductsByCategory(req.Category, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get products by category: %w", err)
	}
	var pbProducts []*pb.Product
	for _, product := range products {
		pbProducts = append(pbProducts, convertProductToProto(&product))
	}
	return &pb.ListProductsResponse{Products: pbProducts, TotalCount: int32(totalCount), Page: int32(page), PageSize: int32(pageSize)}, nil
}

func (s *ProductServer) GetProductStatistics(ctx context.Context, req *pb.GetProductStatisticsRequest) (*pb.ProductStatisticsResponse, error) {
	log.Printf("gRPC GetProductStatistics called")
	stats, err := s.productService.GetStatistics()
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}
	return &pb.ProductStatisticsResponse{
		TotalProducts:      int32(stats.TotalProducts),
		TotalBrands:        int32(stats.TotalBrands),
		TotalCategories:    int32(stats.TotalCategories),
		LowStockProducts:   int32(stats.LowStockProducts),
		OutOfStockProducts: int32(stats.OutOfStockProducts),
		AveragePrice:       stats.AveragePrice,
		Currency:           "USD",
	}, nil
}

func convertProductToProto(product *models.Product) *pb.Product {
	return &pb.Product{
		Id:          product.ID,
		Name:        product.Name,
		PhotoUrl:    product.PhotoURL,
		Description: product.Description,
		Brand:       product.Brand,
		Price:       product.Price,
		Currency:    product.Currency,
		Category:    product.Category,
		Stock:       int32(product.Stock),
		CreatedAt:   timestamppb.New(product.CreatedAt),
		UpdatedAt:   timestamppb.New(product.UpdatedAt),
	}
}

func generateID() string {
	return strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
}
