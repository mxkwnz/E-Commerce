package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/final-ap2-course2/auth-service/internal/database"
	"github.com/final-ap2-course2/auth-service/internal/models"
)

type ResetTokenRepository struct{}

func NewResetTokenRepository() *ResetTokenRepository {
	return &ResetTokenRepository{}
}

func (r *ResetTokenRepository) Create(resetToken *models.PasswordResetToken) error {
	query := `INSERT INTO password_reset_tokens (id, user_id, token, expires_at)
			  VALUES ($1, $2, $3, $4)`
	_, err := database.DB.Exec(query, resetToken.ID, resetToken.UserID, resetToken.Token, resetToken.ExpiresAt)
	return err
}

func (r *ResetTokenRepository) GetByToken(token string) (*models.PasswordResetToken, error) {
	query := `SELECT id, user_id, token, expires_at, used, created_at
			  FROM password_reset_tokens 
			  WHERE token = $1 AND expires_at > $2 AND used = false`
	var resetToken models.PasswordResetToken
	err := database.DB.QueryRow(query, token, time.Now()).Scan(&resetToken.ID, &resetToken.UserID,
		&resetToken.Token, &resetToken.ExpiresAt, &resetToken.Used, &resetToken.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("reset token not found or expired")
	}
	return &resetToken, err
}

func (r *ResetTokenRepository) MarkAsUsed(token string) error {
	query := `UPDATE password_reset_tokens SET used = true WHERE token = $1`
	_, err := database.DB.Exec(query, token)
	return err
}

func (r *ResetTokenRepository) DeleteExpired() error {
	query := `DELETE FROM password_reset_tokens WHERE expires_at < $1 OR used = true`
	_, err := database.DB.Exec(query, time.Now())
	return err
}
