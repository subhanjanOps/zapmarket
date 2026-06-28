package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a user's role in the system.
type Role string

const (
	RoleBuyer  Role = "buyer"
	RoleSeller Role = "seller"
	RoleAdmin  Role = "admin"
)

// User represents a registered user in the system
type User struct {
	ID               uuid.UUID
	Email            string
	Phone            *string
	PasswordHash     *string // nil for OAuth-only accounts
	FullName         string
	Role             string
	IsVerified       bool
	PhoneVerified    bool
	SellerStatus     *string    // nil for non-sellers; "PENDING" | "APPROVED" | "SUSPENDED"
	DOB              *time.Time // date of birth
	Gender           *string
	PfpURL           *string
	TermsAcceptedAt  *time.Time
	RegistrationStep int // 0–4; login blocked until 4
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
}

// OAuthProvider represents OAuth provider types
type OAuthProvider string

const (
	GoogleProvider   OAuthProvider = "google"
	FacebookProvider OAuthProvider = "facebook"
)

// OAuthAccount represents a linked OAuth provider account
type OAuthAccount struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Provider    OAuthProvider
	ProviderUID string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// RefreshToken represents a stored refresh token
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string // raw token, never persisted; only populated when freshly issued
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// Address represents a user's shipping or billing address
type Address struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Label     *string // "home", "office", etc.
	Line1     string
	Line2     *string
	City      string
	State     string
	Country   string
	Pincode   string
	IsDefault bool
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// NotificationPreference represents a user's notification settings
type NotificationPreference struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Channel   string // "email", "sms", "push"
	EventType string // "order.created", etc.
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// PasswordResetToken represents a one-time password reset request
type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// OTPPurpose is the reason an OTP was issued.
type OTPPurpose string

const (
	OTPPurposeEmailVerify         OTPPurpose = "email_verify"
	OTPPurposePhoneVerify         OTPPurpose = "phone_verify"
	OTPPurposePhonePasswordReset  OTPPurpose = "phone_password_reset"
)

// OTPVerification represents a one-time password issued to a user.
type OTPVerification struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	CodeHash  string
	Purpose   OTPPurpose
	Recipient string // email address or E.164 phone number
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// Claims represents JWT claims
type Claims struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Type      string    `json:"type"` // "access" or "refresh"
	ExpiresAt int64     `json:"exp"`
	IssuedAt  int64     `json:"iat"`
}

type SellerProfile struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	StoreName     string     `json:"store_name"`
	Tagline       string     `json:"tagline"`
	Category      string     `json:"category"`
	GSTIN         *string    `json:"gstin,omitempty"`
	PAN           *string    `json:"pan,omitempty"`
	BusinessPhone string     `json:"business_phone"`
	City          string     `json:"city"`
	Pincode       string     `json:"pincode"`
	BusinessType  string     `json:"business_type"` // "individual" | "registered_business"
	TaxID         *string    `json:"tax_id,omitempty"`
	BizLine1      *string    `json:"biz_line1,omitempty"`
	BizCity       *string    `json:"biz_city,omitempty"`
	BizState      *string    `json:"biz_state,omitempty"`
	BizCountry    *string    `json:"biz_country,omitempty"`
	BizPincode    *string    `json:"biz_pincode,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

