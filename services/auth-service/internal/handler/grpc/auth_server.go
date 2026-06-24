package grpc

import (
	"context"
	"time"

	"log/slog"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/service"
	"github.com/zapmarket/zapmarket/pkg/crypto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authpb "github.com/zapmarket/zapmarket/pkg/proto/auth"
)

// AuthServer implements the gRPC AuthService server
// This is a placeholder structure until proto stubs are generated
// After running: protoc --go_out=. --go-grpc_out=. proto/auth.proto
// Uncomment the UnimplementedAuthServiceServer below and update method signatures
type AuthServer struct {
	authSvc                               *service.AuthService
	cfg                                   *config.Config
	authpb.UnimplementedAuthServiceServer // Uncomment after proto generation
}

// NewAuthServer creates a new gRPC auth server
func NewAuthServer(authSvc *service.AuthService, cfg *config.Config) *AuthServer {
	return &AuthServer{
		authSvc: authSvc,
		cfg:     cfg,
	}
}

// ValidateToken validates a JWT token and returns user info
func (s *AuthServer) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	start := time.Now()
	slog.Info("gRPC request",
		"method", "ValidateToken",
	)

	user, err := s.authSvc.ValidateAccessToken(ctx, req.Token)
	duration := time.Since(start)

	if err != nil {
		slog.Info("gRPC response",
			"method", "ValidateToken",
			"status", "error",
			"error", err.Error(),
			"duration_ms", duration.Milliseconds(),
		)
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	slog.Info("gRPC response",
		"method", "ValidateToken",
		"status", "success",
		"duration_ms", duration.Milliseconds(),
	)

	return &authpb.ValidateTokenResponse{
		Valid: true,
		User:  s.domainUserToProto(user),
	}, nil
}

// GetUser retrieves user details by ID
func (s *AuthServer) GetUser(ctx context.Context, req *authpb.GetUserRequest) (*authpb.GetUserResponse, error) {
	start := time.Now()
	slog.Info("gRPC request",
		"method", "GetUser",
	)

	userID, err := s.parseUUID(req.UserId)
	if err != nil {
		duration := time.Since(start)
		slog.Info("gRPC response",
			"method", "GetUser",
			"status", "error",
			"error", err.Error(),
			"duration_ms", duration.Milliseconds(),
		)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	user, err := s.authSvc.GetUserByID(ctx, userID)
	if err != nil {
		duration := time.Since(start)
		slog.Info("gRPC response",
			"method", "GetUser",
			"status", "error",
			"error", err.Error(),
			"duration_ms", duration.Milliseconds(),
		)
		return nil, status.Error(codes.Internal, err.Error())
	}
	duration := time.Since(start)
	slog.Info("gRPC response",
		"method", "GetUser",
		"status", "success",
		"duration_ms", duration.Milliseconds(),
	)

	return &authpb.GetUserResponse{
		User: s.domainUserToProto(user),
	}, nil
}

// LoginPassword authenticates a user with email and password
func (s *AuthServer) LoginPassword(ctx context.Context, req *authpb.LoginPasswordRequest) (*authpb.LoginPasswordResponse, error) {
	start := time.Now()
	slog.Info("gRPC request",
		"method", "LoginPassword",
	)

	user, refreshToken, err := s.authSvc.LoginPassword(ctx, req.Email, req.Password)
	if err != nil {
		duration := time.Since(start)
		slog.Info("gRPC response",
			"method", "LoginPassword",
			"status", "error",
			"error", err.Error(),
			"duration_ms", duration.Milliseconds(),
		)
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		duration := time.Since(start)
		slog.Info("gRPC response",
			"method", "LoginPassword",
			"status", "error",
			"error", err.Error(),
			"duration_ms", duration.Milliseconds(),
		)
		return nil, status.Error(codes.Internal, err.Error())
	}
	duration := time.Since(start)
	slog.Info("gRPC response",
		"method", "LoginPassword",
		"status", "success",
		"duration_ms", duration.Milliseconds(),
	)

	return &authpb.LoginPasswordResponse{
		User:         s.domainUserToProto(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
	}, nil
}

// RegisterUser registers a new user
func (s *AuthServer) RegisterUser(ctx context.Context, req *authpb.RegisterUserRequest) (*authpb.RegisterUserResponse, error) {
	start := time.Now()
	slog.Info("gRPC request",
		"method", "RegisterUser",
	)

	// gRPC registration always creates buyers; sellers must register via HTTP.
	// Add a role field to the proto to support seller registration over gRPC.
	user, refreshToken, err := s.authSvc.RegisterUserPassword(ctx, req.Email, req.Password, req.FullName, string(domain.RoleBuyer))
	if err != nil {
		duration := time.Since(start)
		slog.Info("gRPC response",
			"method", "RegisterUser",
			"status", "error",
			"error", err.Error(),
			"duration_ms", duration.Milliseconds(),
		)
		return nil, status.Error(codes.Internal, err.Error())
	}
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		duration := time.Since(start)
		slog.Info("gRPC response",
			"method", "RegisterUser",
			"status", "error",
			"error", err.Error(),
			"duration_ms", duration.Milliseconds(),
		)
		return nil, status.Error(codes.Internal, err.Error())
	}
	duration := time.Since(start)
	slog.Info("gRPC response",
		"method", "RegisterUser",
		"status", "success",
		"duration_ms", duration.Milliseconds(),
	)

	return &authpb.RegisterUserResponse{
		User:         s.domainUserToProto(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
	}, nil
}

// RefreshAccessToken issues a new access token using refresh token
func (s *AuthServer) RefreshAccessToken(ctx context.Context, req *authpb.RefreshTokenRequest) (*authpb.RefreshTokenResponse, error) {
	start := time.Now()
	slog.Info("gRPC request",
		"method", "RefreshAccessToken",
	)

	accessToken, err := s.authSvc.RefreshAccessToken(ctx, req.RefreshToken)
	if err != nil {
		duration := time.Since(start)
		slog.Info("gRPC response",
			"method", "RefreshAccessToken",
			"status", "error",
			"error", err.Error(),
			"duration_ms", duration.Milliseconds(),
		)
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	duration := time.Since(start)
	slog.Info("gRPC response",
		"method", "RefreshAccessToken",
		"status", "success",
		"duration_ms", duration.Milliseconds(),
	)

	return &authpb.RefreshTokenResponse{
		AccessToken: accessToken,
	}, nil
}

// Helper functions

func (s *AuthServer) domainUserToProto(u *domain.User) *authpb.User {
	if u == nil {
		return nil
	}
	phone := ""
	if u.Phone != nil {
		phone = *u.Phone
	}
	return &authpb.User{
		Id:         u.ID.String(),
		Email:      u.Email,
		Phone:      phone,
		FullName:   u.FullName,
		Role:       u.Role,
		IsVerified: u.IsVerified,
		CreatedAt:  u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  u.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *AuthServer) parseUUID(id string) (uuid.UUID, error) {
	return uuid.Parse(id)
}

func (s *AuthServer) generateAccessToken(u *domain.User) (string, error) {
	return crypto.GenerateAccessToken(u.ID, u.Email, u.Role, s.cfg.JWTSecretKey, s.cfg.JWTAccessExpiryHours)
}
