package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/final-ap2-course2/auth-service/internal/models"
	"github.com/final-ap2-course2/auth-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

func newUserID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type UserService struct {
	userRepo *repository.UserRepository
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) GetUser(id string) (*models.User, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	user.Password = ""
	return user, nil
}

func (s *UserService) ListUsers(page, pageSize int, role string) ([]models.User, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	users, totalCount, err := s.userRepo.GetAll(pageSize, offset, role)
	if err != nil {
		return nil, 0, err
	}

	for i := range users {
		users[i].Password = ""
	}

	return users, totalCount, nil
}

func (s *UserService) UpdateUser(id string, updates *models.User) (*models.User, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if updates.Username != "" {
		exists, err := s.userRepo.UsernameExists(updates.Username)
		if err != nil {
			return nil, err
		}
		if exists && updates.Username != user.Username {
			return nil, fmt.Errorf("username already taken")
		}
		user.Username = updates.Username
	}
	if updates.FirstName != "" {
		user.FirstName = updates.FirstName
	}
	if updates.LastName != "" {
		user.LastName = updates.LastName
	}
	if updates.PhoneNumber != "" {
		user.PhoneNumber = updates.PhoneNumber
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	user.Password = ""
	return user, nil
}

func (s *UserService) DeleteUser(id string) error {
	return s.userRepo.Delete(id)
}

func (s *UserService) TopUpBalance(targetUserID string, amount float64) (*models.User, error) {
	if err := s.ApplyTopUpBalance(targetUserID, amount); err != nil {
		return nil, err
	}
	return s.GetUser(targetUserID)
}

func (s *UserService) ApplyTopUpBalance(targetUserID string, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if amount > 1_000_000 {
		return fmt.Errorf("amount exceeds maximum allowed top-up")
	}
	return s.userRepo.AddBalance(targetUserID, amount)
}

func (s *UserService) DeductBalance(userID string, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	return s.userRepo.DeductBalance(userID, amount)
}

type AdminCreateUserRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	Username    string `json:"username"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	PhoneNumber string `json:"phoneNumber"`
	Role        string `json:"role"`
}

func (s *UserService) CreateUserAdmin(req *AdminCreateUserRequest) (*models.User, error) {
	if req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password required")
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
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password")
	}
	role := req.Role
	if role == "" {
		role = "customer"
	}
	user := &models.User{
		ID:          newUserID(),
		Username:    req.Username,
		Email:       req.Email,
		Password:    string(hash),
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		PhoneNumber: req.PhoneNumber,
		Role:        role,
		Balance:     0,
	}
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}
	return s.GetUser(user.ID)
}
