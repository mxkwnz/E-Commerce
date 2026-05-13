package service

import (
	"fmt"

	"github.com/final-ap2-course2/auth-service/internal/models"
	"github.com/final-ap2-course2/auth-service/internal/repository"
)

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
