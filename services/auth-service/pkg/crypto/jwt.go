package crypto

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
)

// GenerateAccessToken creates a new JWT access token
func GenerateAccessToken(userID uuid.UUID, email, role string, secretKey string, expiryHours int) (string, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(expiryHours) * time.Hour)

	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"email":   email,
		"role":    role,
		"type":    "access",
		"exp":     expiresAt.Unix(),
		"iat":     now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// GenerateRefreshToken creates a new JWT refresh token
func GenerateRefreshToken(userID uuid.UUID, secretKey string, expiryDays int) (string, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(expiryDays) * 24 * time.Hour)

	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"type":    "refresh",
		"exp":     expiresAt.Unix(),
		"iat":     now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, nil
}

// ValidateAccessToken verifies an access token and returns claims
func ValidateAccessToken(tokenString, secretKey string) (*domain.Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, domain.NewDomainError(domain.ErrInvalidToken, "token is invalid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, domain.NewDomainError(domain.ErrInvalidToken, "invalid token claims")
	}

	// Verify token type
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "access" {
		return nil, domain.NewDomainError(domain.ErrInvalidToken, "not an access token")
	}

	// Extract fields
	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return nil, domain.NewDomainError(domain.ErrInvalidToken, "missing user_id in token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrInvalidToken, "invalid user_id format")
	}

	email, ok := claims["email"].(string)
	if !ok {
		return nil, domain.NewDomainError(domain.ErrInvalidToken, "missing email in token")
	}

	role, ok := claims["role"].(string)
	if !ok {
		role = "customer"
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, domain.NewDomainError(domain.ErrInvalidToken, "missing exp in token")
	}

	iat, ok := claims["iat"].(float64)
	if !ok {
		return nil, domain.NewDomainError(domain.ErrInvalidToken, "missing iat in token")
	}

	return &domain.Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		Type:      "access",
		ExpiresAt: int64(exp),
		IssuedAt:  int64(iat),
	}, nil
}

// ValidateRefreshToken verifies a refresh token and returns claims
func ValidateRefreshToken(tokenString, secretKey string) (*domain.Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, domain.NewDomainError(domain.ErrInvalidToken, "token is invalid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, domain.NewDomainError(domain.ErrInvalidToken, "invalid token claims")
	}

	// Verify token type
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		return nil, domain.NewDomainError(domain.ErrInvalidToken, "not a refresh token")
	}

	// Extract fields
	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return nil, domain.NewDomainError(domain.ErrInvalidToken, "missing user_id in token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrInvalidToken, "invalid user_id format")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, domain.NewDomainError(domain.ErrInvalidToken, "missing exp in token")
	}

	iat, ok := claims["iat"].(float64)
	if !ok {
		return nil, domain.NewDomainError(domain.ErrInvalidToken, "missing iat in token")
	}

	return &domain.Claims{
		UserID:    userID,
		Type:      "refresh",
		ExpiresAt: int64(exp),
		IssuedAt:  int64(iat),
	}, nil
}
