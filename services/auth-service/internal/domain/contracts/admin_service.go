package contracts

import (
	"context"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
)

// AdminService defines admin operations on users and sellers.
// AdminHandler depends only on this interface, never on a concrete type.
type AdminService interface {
	ListUsers(ctx context.Context, params UserListParams) ([]*domain.User, int64, error)
	GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error)
	PromoteUser(ctx context.Context, id uuid.UUID, newRole string) (*domain.User, error)
	DeactivateUser(ctx context.Context, id uuid.UUID) error
	ListSellers(ctx context.Context, status string, limit, offset int) ([]*domain.User, int64, error)
	UpdateSellerStatus(ctx context.Context, id uuid.UUID, status string) error
}
