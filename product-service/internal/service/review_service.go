package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/final-ap2-course2/product-service/internal/models"
	"github.com/final-ap2-course2/product-service/internal/repository"
)

type ReviewRepository interface {
	Create(review *models.Review) error
	Update(review *models.Review) error
	GetByProductID(productID string) ([]models.Review, error)
	GetByUserID(userID string) ([]models.Review, error)
	GetByID(id string) (*models.Review, error)
	List(productID, userID string, minRating int) ([]models.Review, error)
	Delete(id string) error
}

type ReviewService struct {
	reviewRepo ReviewRepository
	prodRepo   *repository.ProductRepository
}

func NewReviewService(reviewRepo ReviewRepository, prodRepo *repository.ProductRepository) *ReviewService {
	return &ReviewService{reviewRepo: reviewRepo, prodRepo: prodRepo}
}

func generateReviewID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

func (s *ReviewService) CreateReview(review *models.Review) error {
	if review.Rating < 1 || review.Rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}
	// Check if product exists
	_, err := s.prodRepo.GetByID(review.ProductID)
	if err != nil {
		return fmt.Errorf("product not found")
	}

	// Check if user already reviewed this product
	existing, err := s.reviewRepo.List(review.ProductID, review.UserID, 0)
	if err == nil && len(existing) > 0 {
		return fmt.Errorf("you have already reviewed this shoe")
	}

	review.ID = generateReviewID()
	return s.reviewRepo.Create(review)
}

func (s *ReviewService) UpdateReview(reviewID, userID string, rating int, comment string) (*models.Review, error) {
	if rating < 1 || rating > 5 {
		return nil, fmt.Errorf("rating must be between 1 and 5")
	}

	review, err := s.reviewRepo.GetByID(reviewID)
	if err != nil {
		return nil, err
	}

	if review.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	review.Rating = rating
	review.Comment = comment
	if err := s.reviewRepo.Update(review); err != nil {
		return nil, err
	}
	return review, nil
}

func (s *ReviewService) GetProductReviews(productID string) ([]models.Review, error) {
	return s.reviewRepo.GetByProductID(productID)
}

func (s *ReviewService) GetUserReviews(userID string) ([]models.Review, error) {
	return s.reviewRepo.GetByUserID(userID)
}

func (s *ReviewService) ListReviews(productID, userID string, minRating int) ([]models.Review, error) {
	return s.reviewRepo.List(productID, userID, minRating)
}

func (s *ReviewService) GetReview(id string) (*models.Review, error) {
	return s.reviewRepo.GetByID(id)
}

func (s *ReviewService) DeleteReview(reviewID, userID, userRole string) error {
	review, err := s.reviewRepo.GetByID(reviewID)
	if err != nil {
		return err
	}
	if review.UserID != userID && userRole != "admin" {
		return fmt.Errorf("unauthorized")
	}
	return s.reviewRepo.Delete(reviewID)
}
