package repository

import (
	"fmt"

	"github.com/final-ap2-course2/product-service/internal/database"
)

type FavoriteRepository struct{}

func NewFavoriteRepository() *FavoriteRepository {
	return &FavoriteRepository{}
}

func (r *FavoriteRepository) Add(userID, productID string) error {
	_, err := database.DB.Exec(
		`INSERT INTO user_favorites (user_id, product_id) VALUES ($1, $2)
		 ON CONFLICT (user_id, product_id) DO NOTHING`,
		userID, productID,
	)
	return err
}

func (r *FavoriteRepository) Remove(userID, productID string) error {
	res, err := database.DB.Exec(
		`DELETE FROM user_favorites WHERE user_id = $1 AND product_id = $2`,
		userID, productID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("favorite not found")
	}
	return nil
}

func (r *FavoriteRepository) ListProductIDs(userID string) ([]string, error) {
	rows, err := database.DB.Query(
		`SELECT product_id FROM user_favorites WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			return nil, err
		}
		ids = append(ids, pid)
	}
	return ids, nil
}
