package service

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/final-ap2-course2/auth-service/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type mockAuthUserRepo struct {
	users []models.User
}

func (m *mockAuthUserRepo) EmailExists(email string) (bool, error) {
	for _, u := range m.users {
		if strings.EqualFold(u.Email, email) && !u.IsDeleted {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockAuthUserRepo) UsernameExists(username string) (bool, error) {
	for _, u := range m.users {
		if u.Username == username && u.Username != "" && !u.IsDeleted {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockAuthUserRepo) Create(user *models.User) error {
	for _, u := range m.users {
		if strings.EqualFold(u.Email, user.Email) && !u.IsDeleted {
			return fmt.Errorf("duplicate")
		}
	}
	m.users = append(m.users, *user)
	return nil
}

func (m *mockAuthUserRepo) GetByEmail(email string) (*models.User, error) {
	for i := range m.users {
		if strings.EqualFold(m.users[i].Email, email) && !m.users[i].IsDeleted {
			cp := m.users[i]
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (m *mockAuthUserRepo) GetByID(id string) (*models.User, error) {
	for i := range m.users {
		if m.users[i].ID == id && !m.users[i].IsDeleted {
			cp := m.users[i]
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (m *mockAuthUserRepo) UpdatePassword(userID, passwordHash string) error {
	for i := range m.users {
		if m.users[i].ID == userID {
			m.users[i].Password = passwordHash
			return nil
		}
	}
	return fmt.Errorf("user not found")
}

type mockAuthSessionRepo struct {
	sessions []models.Session
}

func (m *mockAuthSessionRepo) Create(session *models.Session) error {
	m.sessions = append(m.sessions, *session)
	return nil
}

func (m *mockAuthSessionRepo) GetByToken(token string) (*models.Session, error) {
	now := time.Now()
	for i := range m.sessions {
		if m.sessions[i].Token == token && m.sessions[i].ExpiresAt.After(now) {
			cp := m.sessions[i]
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("session not found or expired")
}

func (m *mockAuthSessionRepo) DeleteByToken(token string) error {
	var kept []models.Session
	for _, s := range m.sessions {
		if s.Token != token {
			kept = append(kept, s)
		}
	}
	m.sessions = kept
	return nil
}

func (m *mockAuthSessionRepo) DeleteByUserID(userID string) error {
	var kept []models.Session
	for _, s := range m.sessions {
		if s.UserID != userID {
			kept = append(kept, s)
		}
	}
	m.sessions = kept
	return nil
}

type mockAuthResetRepo struct {
	tokens []models.PasswordResetToken
}

func (m *mockAuthResetRepo) Create(resetToken *models.PasswordResetToken) error {
	m.tokens = append(m.tokens, *resetToken)
	return nil
}

func (m *mockAuthResetRepo) GetByToken(token string) (*models.PasswordResetToken, error) {
	now := time.Now()
	for i := range m.tokens {
		if m.tokens[i].Token == token && m.tokens[i].ExpiresAt.After(now) && !m.tokens[i].Used {
			cp := m.tokens[i]
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("reset token not found or expired")
}

func (m *mockAuthResetRepo) MarkAsUsed(token string) error {
	for i := range m.tokens {
		if m.tokens[i].Token == token {
			m.tokens[i].Used = true
			return nil
		}
	}
	return fmt.Errorf("not found")
}

func TestRegister_InvalidEmail(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	svc := NewAuthService(&mockAuthUserRepo{}, &mockAuthSessionRepo{}, &mockAuthResetRepo{})
	_, err := svc.Register(&models.RegisterRequest{
		Email:    "bad",
		Password: "secret1",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegister_Success(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	users := &mockAuthUserRepo{}
	sessions := &mockAuthSessionRepo{}
	svc := NewAuthService(users, sessions, &mockAuthResetRepo{})
	resp, err := svc.Register(&models.RegisterRequest{
		Email:    "u@example.com",
		Password: "secret1",
		Username: "u1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.UserID == "" || resp.AccessToken == "" {
		t.Fatalf("missing fields %+v", resp)
	}
	if len(users.users) != 1 {
		t.Fatalf("users %d", len(users.users))
	}
	if len(sessions.sessions) != 1 {
		t.Fatalf("sessions %d", len(sessions.sessions))
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	hash, _ := bcrypt.GenerateFromPassword([]byte("right"), bcrypt.DefaultCost)
	users := &mockAuthUserRepo{users: []models.User{{
		ID: "id1", Email: "e@e.com", Password: string(hash),
	}}}
	svc := NewAuthService(users, &mockAuthSessionRepo{}, &mockAuthResetRepo{})
	_, err := svc.Login(&models.LoginRequest{Email: "e@e.com", Password: "wrong"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLogin_Success(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	hash, _ := bcrypt.GenerateFromPassword([]byte("right"), bcrypt.DefaultCost)
	users := &mockAuthUserRepo{users: []models.User{{
		ID: "id1", Email: "e@e.com", Password: string(hash),
	}}}
	sessions := &mockAuthSessionRepo{}
	svc := NewAuthService(users, sessions, &mockAuthResetRepo{})
	resp, err := svc.Login(&models.LoginRequest{Email: "e@e.com", Password: "right"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.AccessToken == "" {
		t.Fatal("no token")
	}
	if len(sessions.sessions) != 1 {
		t.Fatalf("sessions %d", len(sessions.sessions))
	}
}

func TestForgotPassword_UnknownEmail(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	svc := NewAuthService(&mockAuthUserRepo{}, &mockAuthSessionRepo{}, &mockAuthResetRepo{})
	tok, err := svc.ForgotPassword("nobody@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if tok != "" {
		t.Fatalf("token %q", tok)
	}
}

func TestChangePassword_WrongOld(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	hash, _ := bcrypt.GenerateFromPassword([]byte("oldpass"), bcrypt.DefaultCost)
	users := &mockAuthUserRepo{users: []models.User{{
		ID: "id1", Email: "e@e.com", Password: string(hash),
	}}}
	svc := NewAuthService(users, &mockAuthSessionRepo{}, &mockAuthResetRepo{})
	err := svc.ChangePassword("id1", "nope", "newpass1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResetPassword_Success(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	hash, _ := bcrypt.GenerateFromPassword([]byte("old"), bcrypt.DefaultCost)
	users := &mockAuthUserRepo{users: []models.User{{
		ID: "id1", Email: "e@e.com", Password: string(hash),
	}}}
	resets := &mockAuthResetRepo{tokens: []models.PasswordResetToken{{
		ID: "rt1", UserID: "id1", Token: "tok1",
		ExpiresAt: time.Now().Add(time.Hour),
	}}}
	sessions := &mockAuthSessionRepo{sessions: []models.Session{{
		UserID: "id1", Token: "sess1",
	}}}
	svc := NewAuthService(users, sessions, resets)
	if err := svc.ResetPassword("tok1", "newpass1"); err != nil {
		t.Fatal(err)
	}
	u, _ := users.GetByID("id1")
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte("newpass1")); err != nil {
		t.Fatal("password not updated")
	}
	if len(sessions.sessions) != 0 {
		t.Fatalf("expected sessions cleared, got %d", len(sessions.sessions))
	}
}
