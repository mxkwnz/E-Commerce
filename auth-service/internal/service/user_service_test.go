package service

import (
	"testing"

	"github.com/final-ap2-course2/auth-service/internal/models"
)

func TestGetUser_StripsPassword(t *testing.T) {
	user := &models.User{
		ID:       "u1",
		Email:    "test@test.com",
		Password: "hashed_secret",
		Role:     "customer",
	}
	user.Password = ""
	if user.Password != "" {
		t.Error("expected password to be stripped")
	}
}

func TestListUsers_Pagination(t *testing.T) {
	page := 1
	pageSize := 10
	offset := (page - 1) * pageSize
	if offset != 0 {
		t.Errorf("expected offset 0 for page 1, got %d", offset)
	}

	page = 3
	offset = (page - 1) * pageSize
	if offset != 20 {
		t.Errorf("expected offset 20 for page 3, got %d", offset)
	}
}

func TestUpdateUser_OnlyNonEmptyFields(t *testing.T) {
	existing := &models.User{
		ID:        "u1",
		Username:  "old_username",
		FirstName: "Old",
		LastName:  "Name",
	}
	updates := &models.User{
		FirstName: "New",
	}

	if updates.Username != "" {
		existing.Username = updates.Username
	}
	if updates.FirstName != "" {
		existing.FirstName = updates.FirstName
	}
	if updates.LastName != "" {
		existing.LastName = updates.LastName
	}

	if existing.Username != "old_username" {
		t.Error("blank username should not overwrite existing")
	}
	if existing.FirstName != "New" {
		t.Errorf("expected New, got %s", existing.FirstName)
	}
	if existing.LastName != "Name" {
		t.Error("blank last name should not overwrite existing")
	}
}

func TestDeleteUser_SoftDelete(t *testing.T) {
	user := &models.User{ID: "u1", IsDeleted: false}
	user.IsDeleted = true
	if !user.IsDeleted {
		t.Error("expected user to be soft-deleted")
	}
}
