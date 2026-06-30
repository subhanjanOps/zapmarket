package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain/contracts"
)

const (
	mfaSessionTTL  = 5 * time.Minute
	backupCodeCount = 8
)

type MFAService struct {
	userRepo contracts.UserRepository
	cache    sessionCache
}

// SessionCacher is the Redis (or no-op) store for MFA session tokens.
type SessionCacher interface {
	SetMFASession(ctx context.Context, token string, userID uuid.UUID, ttl time.Duration) error
	GetMFASession(ctx context.Context, token string) (uuid.UUID, error)
	DeleteMFASession(ctx context.Context, token string) error
}

type sessionCache = SessionCacher

func NewMFAService(userRepo contracts.UserRepository, cache sessionCache) *MFAService {
	return &MFAService{userRepo: userRepo, cache: cache}
}

// Enroll generates a new TOTP secret and returns a QR code URL + backup codes.
// The secret is stored but totp_enabled remains FALSE until VerifyEnrollment succeeds.
func (s *MFAService) Enroll(ctx context.Context, userID uuid.UUID, email string) (otpauthURL string, backupCodes []string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "ZapMarket",
		AccountName: email,
	})
	if err != nil {
		return "", nil, pkgerrors.NewInternal("MFA_ERROR", "failed to generate totp key", err)
	}

	if err := s.userRepo.SetTOTPSecret(ctx, userID, key.Secret()); err != nil {
		return "", nil, err
	}

	codes, hashes, err := generateBackupCodes()
	if err != nil {
		return "", nil, pkgerrors.NewInternal("MFA_ERROR", "failed to generate backup codes", err)
	}
	if err := s.userRepo.SaveBackupCodes(ctx, userID, hashes); err != nil {
		return "", nil, err
	}

	return key.URL(), codes, nil
}

// VerifyEnrollment validates the TOTP code against the stored secret and enables MFA.
func (s *MFAService) VerifyEnrollment(ctx context.Context, userID uuid.UUID, code string) error {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.TOTPSecret == nil {
		return pkgerrors.NewValidation("MFA_NOT_ENROLLED", "no totp secret found — call enroll first")
	}
	if !totp.Validate(code, *user.TOTPSecret) {
		return pkgerrors.NewValidation("INVALID_TOTP", "invalid totp code")
	}
	return s.userRepo.EnableTOTP(ctx, userID)
}

// Validate checks a TOTP code or backup code for a user.
func (s *MFAService) Validate(ctx context.Context, user *domain.User, code string) error {
	if !user.TOTPEnabled || user.TOTPSecret == nil {
		return pkgerrors.NewValidation("MFA_NOT_ENABLED", "mfa is not enabled for this user")
	}

	if totp.Validate(code, *user.TOTPSecret) {
		return nil
	}

	// Fall back to backup code.
	h := hashCode(code)
	ok, err := s.userRepo.GetUnusedBackupCode(ctx, user.ID, h)
	if err != nil {
		return err
	}
	if ok {
		return s.userRepo.MarkBackupCodeUsed(ctx, user.ID, h)
	}
	return pkgerrors.NewValidation("INVALID_TOTP", "invalid totp or backup code")
}

// Disable removes TOTP for a user after verifying their current code.
func (s *MFAService) Disable(ctx context.Context, userID uuid.UUID, code string) error {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.Validate(ctx, user, code); err != nil {
		return err
	}
	return s.userRepo.DisableTOTP(ctx, userID)
}

// MFASessionToken creates a short-lived token stored in cache for the MFA challenge step.
func (s *MFAService) MFASessionToken(ctx context.Context, userID uuid.UUID) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", pkgerrors.NewInternal("MFA_ERROR", "failed to generate mfa session token", err)
	}
	token := hex.EncodeToString(b)
	if err := s.cache.SetMFASession(ctx, token, userID, mfaSessionTTL); err != nil {
		return "", pkgerrors.NewInternal("MFA_ERROR", "failed to store mfa session", err)
	}
	return token, nil
}

// Challenge validates a TOTP code from an mfa_session_token and returns the userID.
func (s *MFAService) Challenge(ctx context.Context, mfaToken, totpCode string) (uuid.UUID, error) {
	userID, err := s.cache.GetMFASession(ctx, mfaToken)
	if err != nil {
		return uuid.Nil, pkgerrors.NewValidation("INVALID_MFA_TOKEN", "mfa session expired or invalid")
	}
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return uuid.Nil, err
	}
	if err := s.Validate(ctx, user, totpCode); err != nil {
		return uuid.Nil, err
	}
	_ = s.cache.DeleteMFASession(ctx, mfaToken)
	return userID, nil
}

func generateBackupCodes() (codes, hashes []string, err error) {
	for i := 0; i < backupCodeCount; i++ {
		b := make([]byte, 5)
		if _, err := rand.Read(b); err != nil {
			return nil, nil, err
		}
		code := fmt.Sprintf("%x", b)
		codes = append(codes, code)
		hashes = append(hashes, hashCode(code))
	}
	return codes, hashes, nil
}

func hashCode(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}
