package repository

import (
	"database/sql"
	"fmt"

	"github.com/final-ap2-course2/product-service/internal/database"
	"github.com/final-ap2-course2/product-service/internal/models"
)

type ReviewRepository struct{}

func NewReviewRepository() *ReviewRepository {
	return &ReviewRepository{}
}

func (r *ReviewRepository) Create(review *models.Review) error {
	query := `INSERT INTO product_reviews (id, product_id, user_id, rating, comment)
			  VALUES ($1, $2, $3, $4, $5)`
	_, err := database.DB.Exec(query, review.ID, review.ProductID, review.UserID, review.Rating, review.Comment)
	return err
}

func (r *ReviewRepository) Update(review *models.Review) error {
	query := `UPDATE product_reviews SET rating = $1, comment = $2, updated_at = CURRENT_TIMESTAMP
			  WHERE id = $3`
	_, err := database.DB.Exec(query, review.Rating, review.Comment, review.ID)
	return err
}

func (r *ReviewRepository) GetByProductID(productID string) ([]models.Review, error) {
	query := `SELECT id, product_id, user_id, rating, comment, created_at, updated_at
			  FROM product_reviews WHERE product_id = $1 ORDER BY created_at DESC`
	rows, err := database.DB.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []models.Review
	for rows.Next() {
		var rev models.Review
		if err := rows.Scan(&rev.ID, &rev.ProductID, &rev.UserID, &rev.Rating, &rev.Comment, &rev.CreatedAt, &rev.UpdatedAt); err != nil {
			return nil, err
		}
		reviews = append(reviews, rev)
	}
	return reviews, nil
}

func (r *ReviewRepository) GetByUserID(userID string) ([]models.Review, error) {
	query := `SELECT id, product_id, user_id, rating, comment, created_at, updated_at
			  FROM product_reviews WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := database.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []models.Review
	for rows.Next() {
		var rev models.Review
		if err := rows.Scan(&rev.ID, &rev.ProductID, &rev.UserID, &rev.Rating, &rev.Comment, &rev.CreatedAt, &rev.UpdatedAt); err != nil {
			return nil, err
		}
		reviews = append(reviews, rev)
	}
	return reviews, nil
}

func (r *ReviewRepository) GetByID(id string) (*models.Review, error) {
	query := `SELECT id, product_id, user_id, rating, comment, created_at, updated_at
			  FROM product_reviews WHERE id = $1`
	var rev models.Review
	err := database.DB.QueryRow(query, id).Scan(&rev.ID, &rev.ProductID, &rev.UserID, &rev.Rating, &rev.Comment, &rev.CreatedAt, &rev.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("review not found")
	}
	return &rev, err
}

func (r *ReviewRepository) List(productID, userID string, minRating int) ([]models.Review, error) {
	q := `SELECT id, product_id, user_id, rating, comment, created_at, updated_at
		  FROM product_reviews WHERE 1=1`
	args := []interface{}{}
	n := 1
	if productID != "" {
		q += fmt.Sprintf(" AND product_id = $%d", n)
		args = append(args, productID)
		n++
	}
	if userID != "" {
		q += fmt.Sprintf(" AND user_id = $%d", n)
		args = append(args, userID)
		n++
	}
	if minRating > 0 {
		q += fmt.Sprintf(" AND rating >= $%d", n)
		args = append(args, minRating)
	}
	q += ` ORDER BY created_at DESC`

	rows, err := database.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []models.Review
	for rows.Next() {
		var rev models.Review
		if err := rows.Scan(&rev.ID, &rev.ProductID, &rev.UserID, &rev.Rating, &rev.Comment, &rev.CreatedAt, &rev.UpdatedAt); err != nil {
			return nil, err
		}
		reviews = append(reviews, rev)
	}
	return reviews, nil
}

func (r *ReviewRepository) Delete(id string) error {
	query := `DELETE FROM product_reviews WHERE id = $1`
	_, err := database.DB.Exec(query, id)
	return err
}
