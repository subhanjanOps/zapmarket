package service_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	pkgcfg "github.com/zapmarket/zapmarket/pkg/config"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain/contracts"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/service"
)

// ── test doubles ────────────────────────────────────────────────────────────

type fakeUserRepo struct {
	users  map[string]*domain.User
	byID   map[uuid.UUID]*domain.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		users: make(map[string]*domain.User),
		byID:  make(map[uuid.UUID]*domain.User),
	}
}

func (r *fakeUserRepo) CreateUser(_ context.Context, u *domain.User) error {
	if _, ok := r.users[u.Email]; ok {
		return pkgerrors.NewConflict("USER_ALREADY_EXISTS", "duplicate")
	}
	r.users[u.Email] = u
	r.byID[u.ID] = u
	return nil
}
func (r *fakeUserRepo) GetUserByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := r.users[email]
	if !ok {
		return nil, pkgerrors.NewNotFound("USER_NOT_FOUND", "not found")
	}
	return u, nil
}
func (r *fakeUserRepo) GetUserByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := r.byID[id]
	if !ok {
		return nil, pkgerrors.NewNotFound("USER_NOT_FOUND", "not found")
	}
	return u, nil
}
func (r *fakeUserRepo) UpdateUser(_ context.Context, u *domain.User) error {
	r.users[u.Email] = u
	r.byID[u.ID] = u
	return nil
}
func (r *fakeUserRepo) VerifyUser(_ context.Context, _ uuid.UUID) error { return nil }
func (r *fakeUserRepo) DeleteUser(_ context.Context, _ uuid.UUID) error { return nil }
func (r *fakeUserRepo) ListUsers(_ context.Context, p contracts.UserListParams) ([]*domain.User, int64, error) {
	if p.Role == "" {
		var all []*domain.User
		for _, u := range r.users {
			all = append(all, u)
		}
		return all, int64(len(all)), nil
	}
	var out []*domain.User
	for _, u := range r.users {
		if u.Role == p.Role {
			out = append(out, u)
		}
	}
	return out, int64(len(out)), nil
}
func (r *fakeUserRepo) UpdateSellerStatus(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (r *fakeUserRepo) GetUserByPhone(_ context.Context, _ string) (*domain.User, error) {
	return nil, pkgerrors.NewNotFound("USER_NOT_FOUND", "user not found")
}
func (r *fakeUserRepo) ListSellers(_ context.Context, _ string, _, _ int) ([]*domain.User, int64, error) {
	return nil, 0, nil
}

type fakeOAuthRepo struct{}

func (r *fakeOAuthRepo) CreateOAuthAccount(_ context.Context, _ *domain.OAuthAccount) error { return nil }
func (r *fakeOAuthRepo) GetOAuthAccountByProviderUID(_ context.Context, _ domain.OAuthProvider, _ string) (*domain.OAuthAccount, *domain.User, error) {
	return nil, nil, pkgerrors.NewNotFound("NOT_FOUND", "not found")
}
func (r *fakeOAuthRepo) UpdateOAuthAccount(_ context.Context, _ *domain.OAuthAccount) error { return nil }
func (r *fakeOAuthRepo) DeleteOAuthAccount(_ context.Context, _ uuid.UUID) error             { return nil }

type fakeTokenRepo struct {
	tokens map[string]*domain.RefreshToken
}

func newFakeTokenRepo() *fakeTokenRepo {
	return &fakeTokenRepo{tokens: make(map[string]*domain.RefreshToken)}
}

func (r *fakeTokenRepo) CreateRefreshToken(_ context.Context, userID uuid.UUID, token string, expiresAt time.Time) (*domain.RefreshToken, error) {
	rt := &domain.RefreshToken{ID: uuid.New(), UserID: userID, Token: token, ExpiresAt: expiresAt}
	r.tokens[token] = rt
	return rt, nil
}
func (r *fakeTokenRepo) GetRefreshTokenByHash(_ context.Context, _ string) (*domain.RefreshToken, error) {
	return nil, pkgerrors.NewNotFound("NOT_FOUND", "not found")
}
func (r *fakeTokenRepo) GetRefreshTokenByTokenString(_ context.Context, token string) (*domain.RefreshToken, error) {
	rt, ok := r.tokens[token]
	if !ok {
		return nil, pkgerrors.NewNotFound("NOT_FOUND", "not found")
	}
	return rt, nil
}
func (r *fakeTokenRepo) RotateRefreshToken(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}
func (r *fakeTokenRepo) RevokeRefreshToken(_ context.Context, _ uuid.UUID) error  { return nil }
func (r *fakeTokenRepo) InvalidateUserTokens(_ context.Context, _ uuid.UUID) error { return nil }

type fakeResetRepo struct {
	tokens map[string]*domain.PasswordResetToken
}

func newFakeResetRepo() *fakeResetRepo {
	return &fakeResetRepo{tokens: make(map[string]*domain.PasswordResetToken)}
}

func (r *fakeResetRepo) CreatePasswordResetToken(_ context.Context, userID uuid.UUID, hash string, expiresAt time.Time) (*domain.PasswordResetToken, error) {
	t := &domain.PasswordResetToken{ID: uuid.New(), UserID: userID, TokenHash: hash, ExpiresAt: expiresAt}
	r.tokens[hash] = t
	return t, nil
}
func (r *fakeResetRepo) GetPasswordResetTokenByHash(_ context.Context, hash string) (*domain.PasswordResetToken, error) {
	t, ok := r.tokens[hash]
	if !ok {
		return nil, pkgerrors.NewNotFound("NOT_FOUND", "not found")
	}
	return t, nil
}
func (r *fakeResetRepo) MarkPasswordResetTokenUsed(_ context.Context, id uuid.UUID) error {
	for _, t := range r.tokens {
		if t.ID == id {
			now := time.Now()
			t.UsedAt = &now
		}
	}
	return nil
}

type fakeEmailer struct{ sent []string }

func (e *fakeEmailer) SendPasswordResetEmail(_ context.Context, to, _ string) error {
	e.sent = append(e.sent, to)
	return nil
}

func (e *fakeEmailer) SendOTPEmail(_ context.Context, to, _ string) error {
	e.sent = append(e.sent, to)
	return nil
}

type fakeOTPRepo struct{}

func (r *fakeOTPRepo) CreateOTP(_ context.Context, _ *domain.OTPVerification) error { return nil }
func (r *fakeOTPRepo) GetLatestUnusedOTP(_ context.Context, _ uuid.UUID, _ domain.OTPPurpose) (*domain.OTPVerification, error) {
	return nil, pkgerrors.NewNotFound("OTP_NOT_FOUND", "not found")
}
func (r *fakeOTPRepo) MarkOTPUsed(_ context.Context, _ uuid.UUID) error { return nil }

type fakeSMSer struct{}

func (s *fakeSMSer) SendOTP(_ context.Context, _, _ string) error { return nil }

type fakeBlacklist struct{ revoked map[string]bool }

func newFakeBlacklist() *fakeBlacklist { return &fakeBlacklist{revoked: make(map[string]bool)} }
func (b *fakeBlacklist) Add(_ context.Context, hash string, _ time.Duration) error {
	b.revoked[hash] = true
	return nil
}
func (b *fakeBlacklist) IsRevoked(_ context.Context, hash string) (bool, error) {
	return b.revoked[hash], nil
}

type fakeOAuthState struct{}

func (s *fakeOAuthState) Store(_ context.Context, _ string, _ time.Duration) error { return nil }
func (s *fakeOAuthState) ConsumeAndValidate(_ context.Context, _ string) (bool, error) {
	return true, nil
}

func tokenHash(tok string) string {
	h := sha256.Sum256([]byte(tok))
	return fmt.Sprintf("%x", h)
}

// ── fixture ──────────────────────────────────────────────────────────────────

func testConfig() *pkgcfg.Config {
	return &pkgcfg.Config{
		JWTSecretKey:         "test-secret-32-bytes-long-enough!",
		JWTRefreshSecretKey:  "refresh-secret-32-bytes-long-enou",
		JWTAccessExpiryHours: 1,
		JWTRefreshExpiryDays: 7,
		PasswordResetBaseURL: "http://localhost:3000/reset",
	}
}

type testFixture struct {
	svc       *service.AuthService
	userRepo  *fakeUserRepo
	tokenRepo *fakeTokenRepo
	resetRepo *fakeResetRepo
	emailer   *fakeEmailer
	bl        *fakeBlacklist
}

func newFixture() *testFixture {
	userRepo := newFakeUserRepo()
	tokenRepo := newFakeTokenRepo()
	resetRepo := newFakeResetRepo()
	emailer := &fakeEmailer{}
	bl := newFakeBlacklist()
	svc := service.NewAuthService(
		userRepo, &fakeOAuthRepo{}, tokenRepo, resetRepo,
		&fakeOTPRepo{}, nil, emailer, &fakeSMSer{}, testConfig(), bl, &fakeOAuthState{},
	)
	return &testFixture{svc, userRepo, tokenRepo, resetRepo, emailer, bl}
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	f := newFixture()
	user, rt, err := f.svc.RegisterUserPassword(context.Background(), "a@b.com", "password123", "Alice", "buyer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Email != "a@b.com" {
		t.Errorf("want email a@b.com, got %s", user.Email)
	}
	if rt == nil || rt.Token == "" {
		t.Error("expected refresh token")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	f := newFixture()
	_, _, _ = f.svc.RegisterUserPassword(context.Background(), "dup@b.com", "pass1234", "Alice", "buyer")
	_, _, err := f.svc.RegisterUserPassword(context.Background(), "dup@b.com", "pass1234", "Alice2", "buyer")
	if err == nil {
		t.Fatal("expected conflict error on duplicate email")
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != pkgerrors.Conflict {
		t.Errorf("expected Conflict, got %T: %v", err, err)
	}
}

func TestLogin_Success(t *testing.T) {
	f := newFixture()
	_, _, _ = f.svc.RegisterUserPassword(context.Background(), "login@b.com", "mypassword", "Bob", "buyer")
	user, rt, err := f.svc.LoginPassword(context.Background(), "login@b.com", "mypassword")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if user.Email != "login@b.com" {
		t.Errorf("unexpected email: %s", user.Email)
	}
	if rt == nil || rt.Token == "" {
		t.Error("expected refresh token on login")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	f := newFixture()
	_, _, _ = f.svc.RegisterUserPassword(context.Background(), "wp@b.com", "correct", "Bob", "buyer")
	_, _, err := f.svc.LoginPassword(context.Background(), "wp@b.com", "wrong")
	if err == nil {
		t.Fatal("expected unauthorized")
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != pkgerrors.Unauthorized {
		t.Errorf("expected Unauthorized, got %v", err)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	f := newFixture()
	_, _, err := f.svc.LoginPassword(context.Background(), "nobody@b.com", "pass")
	if err == nil {
		t.Fatal("expected unauthorized for unknown email")
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != pkgerrors.Unauthorized {
		t.Errorf("expected Unauthorized, got %v", err)
	}
}

func TestValidateAccessToken_RevokedToken(t *testing.T) {
	f := newFixture()
	fakeToken := "not.a.real.jwt"
	hash := tokenHash(fakeToken)
	_ = f.bl.Add(context.Background(), hash, time.Minute)

	_, err := f.svc.ValidateAccessToken(context.Background(), fakeToken)
	if err == nil {
		t.Fatal("expected error for revoked token")
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != pkgerrors.Unauthorized {
		t.Errorf("expected Unauthorized, got %v", err)
	}
}

func TestPasswordReset_DoesNotLeakEmail(t *testing.T) {
	f := newFixture()
	err := f.svc.RequestPasswordReset(context.Background(), "noone@b.com")
	if err != nil {
		t.Fatalf("expected nil for unknown email, got %v", err)
	}
	if len(f.emailer.sent) > 0 {
		t.Error("no email should be sent for unknown address")
	}
}

func TestPasswordReset_EmailSentForKnownUser(t *testing.T) {
	f := newFixture()
	_, _, _ = f.svc.RegisterUserPassword(context.Background(), "reset@b.com", "pass1234", "D", "buyer")
	// Drain any OTP emails from the async goroutine before asserting.
	time.Sleep(20 * time.Millisecond)
	f.emailer.sent = nil

	err := f.svc.RequestPasswordReset(context.Background(), "reset@b.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.emailer.sent) != 1 || f.emailer.sent[0] != "reset@b.com" {
		t.Errorf("expected one email to reset@b.com, got %v", f.emailer.sent)
	}
}

func TestBootstrapAdmin_OnlyOnce(t *testing.T) {
	f := newFixture()
	_, _, err := f.svc.BootstrapAdmin(context.Background(), "admin@b.com", "adminpass1", "Admin")
	if err != nil {
		t.Fatalf("first bootstrap should succeed: %v", err)
	}
	_, _, err = f.svc.BootstrapAdmin(context.Background(), "admin2@b.com", "adminpass1", "Admin2")
	if err == nil {
		t.Fatal("second bootstrap should fail")
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != pkgerrors.Conflict {
		t.Errorf("expected Conflict, got %v", err)
	}
}
