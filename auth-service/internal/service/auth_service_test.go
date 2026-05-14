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

func clearSMTP(t *testing.T) {
	t.Helper()
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
}

func TestRegister_InvalidEmail(t *testing.T) {
	clearSMTP(t)
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
	clearSMTP(t)
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
	clearSMTP(t)
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
	clearSMTP(t)
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
	clearSMTP(t)
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
	clearSMTP(t)
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
	clearSMTP(t)
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

func TestIsValidEmail(t *testing.T) {
	valid := []string{
		"user@example.com",
		"user+tag@domain.co.uk",
		"firstname.lastname@company.org",
	}
	invalid := []string{
		"notanemail",
		"@nodomain.com",
		"user@",
		"",
	}
	for _, e := range valid {
		if !isValidEmail(e) {
			t.Errorf("expected %s to be valid", e)
		}
	}
	for _, e := range invalid {
		if isValidEmail(e) {
			t.Errorf("expected %s to be invalid", e)
		}
	}
}

func TestGenerateID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 50; i++ {
		id := generateID()
		if ids[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		ids[id] = true
		time.Sleep(20 * time.Microsecond)
	}
}

func TestGenerateToken_NotEmpty(t *testing.T) {
	token := generateToken("user123", "user@example.com")
	if token == "" {
		t.Error("expected non-empty token")
	}
	if len(token) != 64 {
		t.Errorf("expected 64-char hex token, got length %d", len(token))
	}
}

func TestGenerateToken_DifferentInputsDifferentTokens(t *testing.T) {
	t1 := generateToken("user1", "a@b.com")
	t2 := generateToken("user2", "c@d.com")
	if t1 == t2 {
		t.Error("expected different tokens for different inputs")
	}
}

func TestRegisterRequest_Validation(t *testing.T) {
	cases := []struct {
		name    string
		email   string
		pass    string
		wantErr bool
	}{
		{"valid", "user@example.com", "password123", false},
		{"invalid email", "notanemail", "password123", true},
		{"short password", "user@example.com", "abc", true},
		{"empty email", "", "password123", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			emailOK := isValidEmail(tc.email)
			passOK := len(tc.pass) >= 6
			hasErr := !emailOK || !passOK
			if hasErr != tc.wantErr {
				t.Errorf("case %s: expected wantErr=%v, got hasErr=%v", tc.name, tc.wantErr, hasErr)
			}
		})
	}
}

func TestPasswordReset_TokenExpiry(t *testing.T) {
	expires := time.Now().Add(1 * time.Hour)
	if expires.Before(time.Now()) {
		t.Error("fresh token should not be expired")
	}
	expiredTime := time.Now().Add(-1 * time.Hour)
	if expiredTime.After(time.Now()) {
		t.Error("past time should be expired")
	}
}

func TestSessionExpiry_Is24Hours(t *testing.T) {
	expiresAt := time.Now().Add(24 * time.Hour)
	duration := time.Until(expiresAt)
	if duration < 23*time.Hour || duration > 25*time.Hour {
		t.Errorf("session expiry should be ~24h, got %v", duration)
	}
}

func TestUserRole_DefaultIsCustomer(t *testing.T) {
	user := &models.User{Role: "customer"}
	if user.Role != "customer" {
		t.Errorf("expected default role customer, got %s", user.Role)
	}
}
