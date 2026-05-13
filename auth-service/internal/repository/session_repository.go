package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/final-ap2-course2/auth-service/internal/database"
	"github.com/final-ap2-course2/auth-service/internal/models"
)

type SessionRepository struct{}

func NewSessionRepository() *SessionRepository {
	return &SessionRepository{}
}

func (r *SessionRepository) Create(session *models.Session) error {
	query := `INSERT INTO sessions (id, user_id, token, expires_at)
			  VALUES ($1, $2, $3, $4)`
	_, err := database.DB.Exec(query, session.ID, session.UserID, session.Token, session.ExpiresAt)
	return err
}

func (r *SessionRepository) GetByToken(token string) (*models.Session, error) {
	query := `SELECT id, user_id, token, expires_at, created_at
			  FROM sessions WHERE token = $1 AND expires_at > $2`
	var session models.Session
	err := database.DB.QueryRow(query, token, time.Now()).Scan(&session.ID, &session.UserID,
		&session.Token, &session.ExpiresAt, &session.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found or expired")
	}
	return &session, err
}

func (r *SessionRepository) DeleteByToken(token string) error {
	query := `DELETE FROM sessions WHERE token = $1`
	_, err := database.DB.Exec(query, token)
	return err
}

func (r *SessionRepository) DeleteByUserID(userID string) error {
	query := `DELETE FROM sessions WHERE user_id = $1`
	_, err := database.DB.Exec(query, userID)
	return err
}

func (r *SessionRepository) DeleteExpired() error {
	query := `DELETE FROM sessions WHERE expires_at < $1`
	_, err := database.DB.Exec(query, time.Now())
	return err
}
