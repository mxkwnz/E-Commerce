package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/final-ap2-course2/auth-service/internal/database"
	"github.com/final-ap2-course2/auth-service/internal/models"
)

type ChangePasswordCodeRepository struct{}

func NewChangePasswordCodeRepository() *ChangePasswordCodeRepository {
	return &ChangePasswordCodeRepository{}
}

func (r *ChangePasswordCodeRepository) DeleteUnusedForUser(userID string) error {
	_, err := database.DB.Exec(
		`DELETE FROM password_change_codes WHERE user_id = $1 AND used = false`,
		userID,
	)
	return err
}

func (r *ChangePasswordCodeRepository) Create(row *models.PasswordChangeCode) error {
	_, err := database.DB.Exec(
		`INSERT INTO password_change_codes (id, user_id, code, expires_at) VALUES ($1, $2, $3, $4)`,
		row.ID, row.UserID, row.Code, row.ExpiresAt,
	)
	return err
}

func (r *ChangePasswordCodeRepository) FindValid(userID, plainCode string) (*models.PasswordChangeCode, error) {
	q := `SELECT id, user_id, code, expires_at, used, created_at
		  FROM password_change_codes
		  WHERE user_id = $1 AND code = $2 AND used = false AND expires_at > $3`
	var row models.PasswordChangeCode
	err := database.DB.QueryRow(q, userID, plainCode, time.Now()).Scan(
		&row.ID, &row.UserID, &row.Code, &row.ExpiresAt, &row.Used, &row.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid or expired code")
	}
	return &row, err
}

func (r *ChangePasswordCodeRepository) MarkUsed(id string) error {
	_, err := database.DB.Exec(`UPDATE password_change_codes SET used = true WHERE id = $1`, id)
	return err
}
