package service

import (
	"context"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain/contracts"
)

type adminService struct {
	userRepo contracts.UserRepository
}

// NewAdminService creates an AdminService backed by userRepo.
func NewAdminService(userRepo contracts.UserRepository) contracts.AdminService {
	return &adminService{userRepo: userRepo}
}

func (s *adminService) ListUsers(ctx context.Context, params contracts.UserListParams) ([]*domain.User, int64, error) {
	return s.userRepo.ListUsers(ctx, params)
}

func (s *adminService) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetUserByID(ctx, id)
}

func (s *adminService) PromoteUser(ctx context.Context, id uuid.UUID, newRole string) (*domain.User, error) {
	user, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	user.Role = newRole
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", "failed to update user role", err)
	}
	return user, nil
}

func (s *adminService) DeactivateUser(ctx context.Context, id uuid.UUID) error {
	return s.userRepo.DeleteUser(ctx, id)
}

func (s *adminService) ListSellers(ctx context.Context, status string, limit, offset int) ([]*domain.User, int64, error) {
	return s.userRepo.ListSellers(ctx, status, limit, offset)
}

func (s *adminService) UpdateSellerStatus(ctx context.Context, id uuid.UUID, status string) error {
	return s.userRepo.UpdateSellerStatus(ctx, id, status)
}
