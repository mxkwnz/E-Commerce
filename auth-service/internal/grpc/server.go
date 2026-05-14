package grpc

import (
	"context"
	"fmt"
	"log"

	"github.com/final-ap2-course2/auth-service/internal/models"
	"github.com/final-ap2-course2/auth-service/internal/repository"
	"github.com/final-ap2-course2/auth-service/internal/service"
	pb "github.com/final-ap2-course2/auth-service/proto"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthServer struct {
	pb.UnimplementedAuthServiceServer
	authService *service.AuthService
	userService *service.UserService
}

func NewAuthServer() *AuthServer {
	userRepo := repository.NewUserRepository()
	sessionRepo := repository.NewSessionRepository()
	resetTokenRepo := repository.NewResetTokenRepository()

	return &AuthServer{
		authService: service.NewAuthService(userRepo, sessionRepo, resetTokenRepo),
		userService: service.NewUserService(userRepo),
	}
}

func (s *AuthServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.AuthResponse, error) {
	log.Printf("gRPC Register called: %s", req.Email)

	if req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	registerReq := &models.RegisterRequest{
		Email:       req.Email,
		Password:    req.Password,
		Username:    req.Username,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		PhoneNumber: req.PhoneNumber,
	}

	resp, err := s.authService.Register(registerReq)
	if err != nil {
		return nil, fmt.Errorf("registration failed: %w", err)
	}

	user, _ := s.userService.GetUser(resp.UserID)

	return &pb.AuthResponse{
		UserId:      resp.UserID,
		AccessToken: resp.AccessToken,
		User:        convertUserToProto(user),
	}, nil
}

func (s *AuthServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	log.Printf("gRPC Login called: %s", req.Email)

	if req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	loginReq := &models.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	resp, err := s.authService.Login(loginReq)
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	user, _ := s.userService.GetUser(resp.UserID)

	return &pb.AuthResponse{
		UserId:      resp.UserID,
		AccessToken: resp.AccessToken,
		User:        convertUserToProto(user),
	}, nil
}

func (s *AuthServer) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	log.Printf("gRPC ValidateToken called")

	if req.Token == "" {
		return nil, fmt.Errorf("token is required")
	}

	user, err := s.authService.ValidateToken(req.Token)
	if err != nil {
		return &pb.ValidateTokenResponse{
			Valid: false,
		}, nil
	}

	return &pb.ValidateTokenResponse{
		Valid:  true,
		UserId: user.ID,
		User:   convertUserToProto(user),
	}, nil
}

func (s *AuthServer) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	log.Printf("gRPC Logout called")

	if req.Token == "" {
		return nil, fmt.Errorf("token is required")
	}

	if err := s.authService.Logout(req.Token); err != nil {
		return &pb.LogoutResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.LogoutResponse{
		Success: true,
		Message: "Successfully logged out",
	}, nil
}

func (s *AuthServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	log.Printf("gRPC GetUser called: %s", req.Id)

	if req.Id == "" {
		return nil, fmt.Errorf("user id is required")
	}

	user, err := s.userService.GetUser(req.Id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &pb.UserResponse{
		User: convertUserToProto(user),
	}, nil
}

func (s *AuthServer) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	log.Printf("gRPC ListUsers called: page=%d, size=%d, role=%s", req.Page, req.PageSize, req.Role)

	page := int(req.Page)
	pageSize := int(req.PageSize)

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	users, totalCount, err := s.userService.ListUsers(page, pageSize, req.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	var pbUsers []*pb.User
	for _, user := range users {
		pbUsers = append(pbUsers, convertUserToProto(&user))
	}

	return &pb.ListUsersResponse{
		Users:      pbUsers,
		TotalCount: int32(totalCount),
		Page:       int32(page),
		PageSize:   int32(pageSize),
	}, nil
}

func (s *AuthServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UserResponse, error) {
	log.Printf("gRPC UpdateUser called: %s", req.Id)

	if req.Id == "" {
		return nil, fmt.Errorf("user id is required")
	}

	updates := &models.User{
		Username:    req.Username,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		PhoneNumber: req.PhoneNumber,
	}

	user, err := s.userService.UpdateUser(req.Id, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &pb.UserResponse{
		User: convertUserToProto(user),
	}, nil
}

func (s *AuthServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	log.Printf("gRPC DeleteUser called: %s", req.Id)

	if req.Id == "" {
		return nil, fmt.Errorf("user id is required")
	}

	user, err := s.userService.GetUser(req.Id)
	if err != nil {
		return &pb.DeleteUserResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	if err := s.userService.DeleteUser(req.Id); err != nil {
		return &pb.DeleteUserResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	s.authService.NotifyUserDeleted(user.ID, user.Email, user.Username, user.Role)

	return &pb.DeleteUserResponse{
		Success: true,
		Message: "User deleted successfully",
	}, nil
}

func (s *AuthServer) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	log.Printf("gRPC ChangePassword called for user: %s", req.UserId)

	if req.UserId == "" || req.OldPassword == "" || req.NewPassword == "" {
		return nil, fmt.Errorf("user id, old password, and new password are required")
	}

	if err := s.authService.ChangePassword(req.UserId, req.OldPassword, req.NewPassword); err != nil {
		return &pb.ChangePasswordResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.ChangePasswordResponse{
		Success: true,
		Message: "Password changed successfully",
	}, nil
}

func (s *AuthServer) ForgotPassword(ctx context.Context, req *pb.ForgotPasswordRequest) (*pb.ForgotPasswordResponse, error) {
	log.Printf("gRPC ForgotPassword called: %s", req.Email)

	if req.Email == "" {
		return nil, fmt.Errorf("email is required")
	}

	resetToken, err := s.authService.ForgotPassword(req.Email)
	if err != nil {
		log.Printf("ForgotPassword error: %v", err)
	}

	return &pb.ForgotPasswordResponse{
		Success:    true,
		Message:    "If the email exists, a reset link has been sent",
		ResetToken: resetToken,
	}, nil
}

func (s *AuthServer) ResetPassword(ctx context.Context, req *pb.ResetPasswordRequest) (*pb.ResetPasswordResponse, error) {
	log.Printf("gRPC ResetPassword called")

	if req.Token == "" || req.NewPassword == "" {
		return nil, fmt.Errorf("token and new password are required")
	}

	if err := s.authService.ResetPassword(req.Token, req.NewPassword); err != nil {
		return &pb.ResetPasswordResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.ResetPasswordResponse{
		Success: true,
		Message: "Password reset successfully",
	}, nil
}

func (s *AuthServer) VerifyResetToken(ctx context.Context, req *pb.VerifyResetTokenRequest) (*pb.VerifyResetTokenResponse, error) {
	log.Printf("gRPC VerifyResetToken called")

	if req.Token == "" {
		return nil, fmt.Errorf("token is required")
	}

	userID, err := s.authService.VerifyResetToken(req.Token)
	if err != nil {
		return &pb.VerifyResetTokenResponse{
			Valid:   false,
			Message: "Invalid or expired token",
		}, nil
	}

	return &pb.VerifyResetTokenResponse{
		Valid:   true,
		UserId:  userID,
		Message: "Token is valid",
	}, nil
}

func (s *AuthServer) SetNATS(nc *nats.Conn) {
	if nc != nil {
		s.authService.SetPublisher(nc)
	}
}

func convertUserToProto(user *models.User) *pb.User {
	if user == nil {
		return nil
	}

	return &pb.User{
		Id:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		PhoneNumber: user.PhoneNumber,
		Balance:     user.Balance,
		Role:        user.Role,
		CreatedAt:   timestamppb.New(user.CreatedAt),
		UpdatedAt:   timestamppb.New(user.UpdatedAt),
	}
}
