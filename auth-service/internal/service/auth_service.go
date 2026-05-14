package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/final-ap2-course2/auth-service/internal/models"
	"github.com/final-ap2-course2/auth-service/internal/usecase"
	"golang.org/x/crypto/bcrypt"
)

type authUserRepository interface {
	EmailExists(email string) (bool, error)
	UsernameExists(username string) (bool, error)
	Create(user *models.User) error
	GetByEmail(email string) (*models.User, error)
	GetByID(id string) (*models.User, error)
	UpdatePassword(userID, passwordHash string) error
}

type authSessionRepository interface {
	Create(session *models.Session) error
	GetByToken(token string) (*models.Session, error)
	DeleteByToken(token string) error
	DeleteByUserID(userID string) error
}

type authResetTokenRepository interface {
	Create(resetToken *models.PasswordResetToken) error
	GetByToken(token string) (*models.PasswordResetToken, error)
	MarkAsUsed(token string) error
}

type AuthService struct {
	userRepo       authUserRepository
	sessionRepo    authSessionRepository
	resetTokenRepo authResetTokenRepository
}

func NewAuthService(userRepo authUserRepository, sessionRepo authSessionRepository, resetTokenRepo authResetTokenRepository) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		resetTokenRepo: resetTokenRepo,
	}
}

func (s *AuthService) Register(req *models.RegisterRequest) (*models.AuthResponse, error) {
	if !isValidEmail(req.Email) {
		return nil, fmt.Errorf("invalid email format")
	}

	if len(req.Password) < 6 {
		return nil, fmt.Errorf("password must be at least 6 characters")
	}

	exists, err := s.userRepo.EmailExists(req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("email already registered")
	}

	if req.Username != "" {
		exists, err := s.userRepo.UsernameExists(req.Username)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, fmt.Errorf("username already taken")
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password")
	}

	user := &models.User{
		ID:          generateID(),
		Username:    req.Username,
		Email:       req.Email,
		Password:    string(hashedPassword),
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		PhoneNumber: req.PhoneNumber,
		Role:        "customer",
		Balance:     0.00,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	token := generateToken(user.ID, user.Email)
	session := &models.Session{
		ID:        generateID(),
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := s.sessionRepo.Create(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	_ = usecase.SendWelcomeEmail(user.Email, user.Username)

	return &models.AuthResponse{
		UserID:      user.ID,
		AccessToken: token,
	}, nil
}

func (s *AuthService) Login(req *models.LoginRequest) (*models.AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	_ = s.sessionRepo.DeleteByUserID(user.ID)

	token := generateToken(user.ID, user.Email)
	session := &models.Session{
		ID:        generateID(),
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := s.sessionRepo.Create(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &models.AuthResponse{
		UserID:      user.ID,
		AccessToken: token,
	}, nil
}

func (s *AuthService) ValidateToken(token string) (*models.User, error) {
	session, err := s.sessionRepo.GetByToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired token")
	}

	user, err := s.userRepo.GetByID(session.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	user.Password = ""

	return user, nil
}

func (s *AuthService) Logout(token string) error {
	return s.sessionRepo.DeleteByToken(token)
}

func (s *AuthService) ForgotPassword(email string) (string, error) {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return "", nil
	}

	resetToken := generateToken(user.ID, email)
	token := &models.PasswordResetToken{
		ID:        generateID(),
		UserID:    user.ID,
		Token:     resetToken,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := s.resetTokenRepo.Create(token); err != nil {
		return "", fmt.Errorf("failed to create reset token: %w", err)
	}

	if err := usecase.SendResetEmail(email, resetToken); err != nil {
		return "", fmt.Errorf("failed to send reset email: %w", err)
	}

	return resetToken, nil
}

func (s *AuthService) VerifyResetToken(token string) (string, error) {
	resetToken, err := s.resetTokenRepo.GetByToken(token)
	if err != nil {
		return "", fmt.Errorf("invalid or expired reset token")
	}

	return resetToken.UserID, nil
}

func (s *AuthService) ResetPassword(token, newPassword string) error {
	resetToken, err := s.resetTokenRepo.GetByToken(token)
	if err != nil {
		return fmt.Errorf("invalid or expired reset token")
	}

	if len(newPassword) < 6 {
		return fmt.Errorf("password must be at least 6 characters")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password")
	}

	if err := s.userRepo.UpdatePassword(resetToken.UserID, string(hashedPassword)); err != nil {
		return err
	}

	if err := s.resetTokenRepo.MarkAsUsed(token); err != nil {
		return err
	}

	_ = s.sessionRepo.DeleteByUserID(resetToken.UserID)

	if u, err := s.userRepo.GetByID(resetToken.UserID); err == nil {
		_ = usecase.SendPasswordChangedEmail(u.Email)
	}

	return nil
}

func (s *AuthService) ChangePassword(userID, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return fmt.Errorf("incorrect old password")
	}

	if len(newPassword) < 6 {
		return fmt.Errorf("password must be at least 6 characters")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password")
	}

	if err := s.userRepo.UpdatePassword(userID, string(hashedPassword)); err != nil {
		return err
	}

	_ = usecase.SendPasswordChangedEmail(user.Email)

	_ = s.sessionRepo.DeleteByUserID(userID)

	return nil
}

func isValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

func generateID() string {
	return strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
}

func generateToken(userID, email string) string {
	data := fmt.Sprintf("%s:%s:%d", userID, email, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
