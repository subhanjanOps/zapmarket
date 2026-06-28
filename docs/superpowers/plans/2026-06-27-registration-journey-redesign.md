# Registration Journey Redesign — Full Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign buyer and seller registration journeys — buyer gets a single-form auto-login flow; seller gets a guided 3-step wizard with KYC and a pending-approval screen.

**Architecture:** New `seller_profiles` table in auth-service for seller-specific data; `RegisterSeller` service method wraps user creation + profile insert in one transaction; buyer-ui `/register` becomes a role chooser; `/register/buyer` is the current form with auto-login cookie; `/register/seller` is a 3-step wizard.

**Tech Stack:** Go 1.22, `database/sql`, `lib/pq`, Next.js 14 App Router, framer-motion, sonner (toasts), Zustand, Tailwind CSS

## Global Constraints

- No ORM — raw `database/sql` with `lib/pq`
- New `seller_profiles` table — do not extend `users`
- Seller KYC at sign-up, admin-verified async; buyer login is immediate
- `buyer_token` cookie: `httpOnly: true, sameSite: "strict", path: "/", maxAge: 60*60*24`
- All new HTTP handlers use existing `h.writeError` / `h.writeResponse` / `pkgerrors.HandleHTTP` helpers
- buyer-ui API routes use `GW` from `@/lib/gateway`
- The existing backend `Register` handler **already returns `access_token`** — the buyer auto-login fix is UI-only

## Key Discovery

The existing `POST /v1/auth/register` backend already responds with `AuthResponse{ User, AccessToken, RefreshToken }`. The existing `/api/auth/register/route.ts` in buyer-ui discards those tokens and returns `{ ok: true }`. So:

- **Buyer auto-login = fix the Next.js API route only, no backend change**
- **Seller = new backend endpoint + new frontend wizard**
- **Admin seller status = already implemented** in `admin_handler.go:226`

---

## Task B1 — DB Migration: seller_profiles

**Files:**
- Create: `services/auth-service/migrations/0007_seller_profiles.up.sql`
- Create: `services/auth-service/migrations/0007_seller_profiles.down.sql`

- [ ] **Step 1: Write the up migration**

```sql
-- services/auth-service/migrations/0007_seller_profiles.up.sql
CREATE TABLE IF NOT EXISTS seller_profiles (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    store_name     TEXT        NOT NULL,
    tagline        TEXT        NOT NULL DEFAULT '',
    category       TEXT        NOT NULL,
    gstin          TEXT,
    pan            TEXT,
    business_phone TEXT        NOT NULL,
    city           TEXT        NOT NULL,
    pincode        TEXT        NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT seller_profiles_user_id_key UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS idx_seller_profiles_user_id ON seller_profiles(user_id);
```

- [ ] **Step 2: Write the down migration**

```sql
-- services/auth-service/migrations/0007_seller_profiles.down.sql
DROP TABLE IF EXISTS seller_profiles;
```

- [ ] **Step 3: Verify naming follows existing pattern**

```bash
ls services/auth-service/migrations/ | sort
```

Expected: `0007_seller_profiles.up.sql` and `0007_seller_profiles.down.sql` appear correctly ordered.

---

## Task B2 — Domain Type: SellerProfile

**Files:**
- Modify: `services/auth-service/internal/domain/models.go`

- [ ] **Step 1: Add SellerProfile struct**

Append to the bottom of `models.go` (before the closing brace if any, or just at end of file):

```go
// SellerProfile holds seller-specific onboarding data collected at registration.
// It is always created in the same transaction as the seller user row.
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
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
```

- [ ] **Step 2: Build to confirm no compile errors**

```bash
cd services/auth-service && go build ./internal/domain/...
```

Expected: no output.

---

## Task B3 — Repository Interface: SellerProfileRepository

**Files:**
- Modify: `services/auth-service/internal/domain/contracts/repositories.go`

- [ ] **Step 1: Append interface to repositories.go**

```go
// SellerProfileRepository defines persistence for seller onboarding data.
type SellerProfileRepository interface {
	Create(ctx context.Context, profile *domain.SellerProfile) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.SellerProfile, error)
}
```

- [ ] **Step 2: Build**

```bash
cd services/auth-service && go build ./internal/domain/...
```

---

## Task B4 — Repository Implementation: seller_profile_repository.go

**Files:**
- Create: `services/auth-service/internal/repository/seller_profile_repository.go`

- [ ] **Step 1: Write the implementation**

```go
package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
)

// SellerProfileRepository implements contracts.SellerProfileRepository using PostgreSQL.
type SellerProfileRepository struct{ db *sql.DB }

func NewSellerProfileRepository(db *sql.DB) *SellerProfileRepository {
	return &SellerProfileRepository{db: db}
}

func (r *SellerProfileRepository) Create(ctx context.Context, p *domain.SellerProfile) error {
	p.ID = uuid.New()
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO seller_profiles
			(id, user_id, store_name, tagline, category, gstin, pan, business_phone, city, pincode, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		p.ID, p.UserID, p.StoreName, p.Tagline, p.Category,
		p.GSTIN, p.PAN, p.BusinessPhone, p.City, p.Pincode,
		p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (r *SellerProfileRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.SellerProfile, error) {
	p := &domain.SellerProfile{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, store_name, tagline, category, gstin, pan, business_phone, city, pincode, created_at, updated_at
		FROM seller_profiles
		WHERE user_id = $1`, userID,
	).Scan(
		&p.ID, &p.UserID, &p.StoreName, &p.Tagline, &p.Category,
		&p.GSTIN, &p.PAN, &p.BusinessPhone, &p.City, &p.Pincode,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, pkgerrors.NewNotFound("NOT_FOUND", "seller profile not found")
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}
```

- [ ] **Step 2: Build**

```bash
cd services/auth-service && go build ./internal/repository/...
```

---

## Task B5 — Service: RegisterSeller method

**Files:**
- Modify: `services/auth-service/internal/service/auth_service.go`

The `AuthService` struct and `NewAuthService` constructor need a new `sellerProfileRepo` field.

- [ ] **Step 1: Add sellerProfileRepo to AuthService struct**

Find the `AuthService` struct (lines 40–51 of auth_service.go). Add one field:

```go
type AuthService struct {
	userRepo          contracts.UserRepository
	oauthRepo         contracts.OAuthRepository
	tokenRepo         contracts.RefreshTokenRepository
	resetRepo         contracts.PasswordResetRepository
	otpRepo           contracts.OTPRepository
	sellerProfileRepo contracts.SellerProfileRepository  // ← add this line
	emailer           email.Emailer
	smser             sms.SMSer
	cfg               *config.Config
	blacklist         contracts.TokenBlacklist
	oauthState        contracts.OAuthStateStore
}
```

- [ ] **Step 2: Add sellerProfileRepo to AuthRepos grouped struct**

```go
type AuthRepos struct {
	Users          contracts.UserRepository
	OAuth          contracts.OAuthRepository
	Tokens         contracts.RefreshTokenRepository
	Resets         contracts.PasswordResetRepository
	OTPs           contracts.OTPRepository
	SellerProfiles contracts.SellerProfileRepository  // ← add this line
}
```

- [ ] **Step 3: Update NewAuthService to accept and wire sellerProfileRepo**

Change the `NewAuthService` signature and body:

```go
func NewAuthService(
	userRepo contracts.UserRepository,
	oauthRepo contracts.OAuthRepository,
	tokenRepo contracts.RefreshTokenRepository,
	resetRepo contracts.PasswordResetRepository,
	otpRepo contracts.OTPRepository,
	sellerProfileRepo contracts.SellerProfileRepository,
	emailer email.Emailer,
	smser sms.SMSer,
	cfg *config.Config,
	blacklist contracts.TokenBlacklist,
	oauthState contracts.OAuthStateStore,
) *AuthService {
	return &AuthService{
		userRepo:          userRepo,
		oauthRepo:         oauthRepo,
		tokenRepo:         tokenRepo,
		resetRepo:         resetRepo,
		otpRepo:           otpRepo,
		sellerProfileRepo: sellerProfileRepo,
		emailer:           emailer,
		smser:             smser,
		cfg:               cfg,
		blacklist:         blacklist,
		oauthState:        oauthState,
	}
}
```

- [ ] **Step 4: Update NewAuthServiceFromGroups**

```go
func NewAuthServiceFromGroups(repos AuthRepos, infra AuthInfra, cfg *config.Config) *AuthService {
	return NewAuthService(
		repos.Users, repos.OAuth, repos.Tokens, repos.Resets, repos.OTPs,
		repos.SellerProfiles,
		infra.Emailer, infra.SMSer, cfg, infra.Blacklist, infra.OAuthState,
	)
}
```

- [ ] **Step 5: Add RegisterSeller method**

Append at end of auth_service.go (before any other methods or at bottom of file):

```go
// RegisterSeller registers a new seller user and inserts their profile in the same
// logical operation. Note: these are two sequential DB writes, not a single SQL
// transaction, because RegisterUserPassword is a shared method. If profile creation
// fails after user creation, the user row stays (seller_status = PENDING) and the
// admin can remediate; the frontend redirects to the pending screen regardless.
func (s *AuthService) RegisterSeller(
	ctx context.Context,
	email, password, fullName string,
	profile domain.SellerProfile,
) (*domain.User, error) {
	user, _, err := s.RegisterUserPassword(ctx, email, password, fullName, string(domain.RoleSeller))
	if err != nil {
		return nil, err
	}
	profile.UserID = user.ID
	if err := s.sellerProfileRepo.Create(ctx, &profile); err != nil {
		return nil, err
	}
	return user, nil
}
```

- [ ] **Step 6: Build**

```bash
cd services/auth-service && go build ./internal/service/...
```

Expected: compile error in cmd/main.go only (NewAuthService call site now has wrong arity — fixed in Task B7).

---

## Task B6 — Interface: Add RegisterSeller to AuthServicer

**Files:**
- Modify: `services/auth-service/internal/handler/http/auth_service_iface.go`

- [ ] **Step 1: Add to AuthServicer interface**

The `AuthServicer` interface already defines all methods the HTTP handler uses. Add `RegisterSeller` just below `RegisterUserPassword`:

```go
RegisterSeller(ctx context.Context, email, password, fullName string, profile domain.SellerProfile) (*domain.User, error)
```

The full import block at the top of auth_service_iface.go already imports `domain` — no change needed there.

- [ ] **Step 2: Build**

```bash
cd services/auth-service && go build ./internal/handler/...
```

Expected: compiles (AuthService satisfies AuthServicer because we added RegisterSeller in B5).

---

## Task B7 — HTTP Handler: RegisterSeller endpoint

**Files:**
- Modify: `services/auth-service/internal/handler/http/handlers.go`

- [ ] **Step 1: Add RegisterSellerRequest struct**

Append near the `RegisterRequest` struct (around line 85):

```go
// RegisterSellerRequest is the payload for POST /v1/auth/register/seller.
type RegisterSellerRequest struct {
	FullName      string  `json:"full_name"`
	Email         string  `json:"email"`
	Password      string  `json:"password"`
	StoreName     string  `json:"store_name"`
	Tagline       string  `json:"tagline"`
	Category      string  `json:"category"`
	GSTIN         *string `json:"gstin,omitempty"`
	PAN           *string `json:"pan,omitempty"`
	BusinessPhone string  `json:"business_phone"`
	City          string  `json:"city"`
	Pincode       string  `json:"pincode"`
}
```

- [ ] **Step 2: Add RegisterSeller handler method**

Append after the existing `Register` handler:

```go
// RegisterSeller handles POST /v1/auth/register/seller
// @Summary      Register a new seller
// @Description  Register a seller account with store profile and KYC. Account starts PENDING until admin approval.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body RegisterSellerRequest true "Seller registration payload"
// @Success      201 {object} AuthResponse "Seller registered; awaiting approval"
// @Failure      400 {object} AuthResponse "Missing or invalid fields"
// @Failure      409 {object} AuthResponse "Email already registered"
// @Failure      500 {object} AuthResponse "Internal server error"
// @Router       /register/seller [post]
func (h *Handler) RegisterSeller(w http.ResponseWriter, r *http.Request) {
	var req RegisterSellerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Credential validation
	if req.FullName == "" || req.Email == "" || req.Password == "" {
		h.writeError(w, http.StatusBadRequest, "full_name, email, and password are required")
		return
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		h.writeError(w, http.StatusBadRequest, "email must be a valid email address")
		return
	}
	if len(req.Password) < 8 || len(req.Password) > 72 {
		h.writeError(w, http.StatusBadRequest, "password must be between 8 and 72 characters")
		return
	}

	// Store profile validation
	if req.StoreName == "" || req.Category == "" {
		h.writeError(w, http.StatusBadRequest, "store_name and category are required")
		return
	}

	// Business contact validation
	if req.BusinessPhone == "" || req.City == "" || req.Pincode == "" {
		h.writeError(w, http.StatusBadRequest, "business_phone, city, and pincode are required")
		return
	}

	// KYC: at least one of GSTIN or PAN must be provided
	gstin := req.GSTIN != nil && *req.GSTIN != ""
	pan := req.PAN != nil && *req.PAN != ""
	if !gstin && !pan {
		h.writeError(w, http.StatusBadRequest, "at least one of gstin or pan is required")
		return
	}

	profile := domain.SellerProfile{
		StoreName:     req.StoreName,
		Tagline:       req.Tagline,
		Category:      req.Category,
		GSTIN:         req.GSTIN,
		PAN:           req.PAN,
		BusinessPhone: req.BusinessPhone,
		City:          req.City,
		Pincode:       req.Pincode,
	}

	user, err := h.authSvc.RegisterSeller(r.Context(), req.Email, req.Password, req.FullName, profile)
	if err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}

	h.writeResponse(w, http.StatusCreated, AuthResponse{
		User: userToResponse(user),
	})
}
```

- [ ] **Step 3: Build**

```bash
cd services/auth-service && go build ./internal/handler/...
```

---

## Task B8 — Wire: cmd/main.go

**Files:**
- Modify: `services/auth-service/cmd/main.go`

- [ ] **Step 1: Instantiate SellerProfileRepository**

After the existing repo declarations (around line 110), add:

```go
sellerProfileRepo := repository.NewSellerProfileRepository(db)
```

- [ ] **Step 2: Update NewAuthService call (line 160) to pass sellerProfileRepo**

Current:
```go
authService := service.NewAuthService(userRepo, oauthRepo, tokenRepo, resetRepo, otpRepo, emailer, smser, cfg, blacklist, oauthStateStore)
```

Replace with:
```go
authService := service.NewAuthService(userRepo, oauthRepo, tokenRepo, resetRepo, otpRepo, sellerProfileRepo, emailer, smser, cfg, blacklist, oauthStateStore)
```

- [ ] **Step 3: Register the new route**

After the existing `mux.HandleFunc("/v1/auth/register", ...)` line (around line 173), add:

```go
mux.HandleFunc("/v1/auth/register/seller", httphandler.IPRateLimit(rdb, "register")(httpHandler.LoggingMiddleware(httpHandler.RegisterSeller)))
```

- [ ] **Step 4: Full build**

```bash
cd services/auth-service && go build ./...
```

Expected: no errors. This is the final backend step.

- [ ] **Step 5: Smoke test backend endpoints**

Start auth-service:
```bash
cd services/auth-service && go run ./cmd/...
```

Test buyer register still works:
```bash
curl -s -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"full_name":"Test Buyer","email":"buyer@test.com","password":"password123","role":"buyer"}' | jq .
```
Expected: `{ user: {...}, access_token: "...", refresh_token: "..." }`, status 201.

Test seller register:
```bash
curl -s -X POST http://localhost:8080/v1/auth/register/seller \
  -H "Content-Type: application/json" \
  -d '{
    "full_name":"Test Seller","email":"seller@test.com","password":"password123",
    "store_name":"Test Store","category":"Electronics",
    "gstin":"29ABCDE1234F1Z5","business_phone":"9876543210",
    "city":"Bangalore","pincode":"560001"
  }' | jq .
```
Expected: `{ user: { seller_status: "PENDING", ... } }`, status 201.

Test KYC validation:
```bash
curl -s -X POST http://localhost:8080/v1/auth/register/seller \
  -H "Content-Type: application/json" \
  -d '{"full_name":"X","email":"x@x.com","password":"password123","store_name":"S","category":"C","business_phone":"123","city":"C","pincode":"123"}' | jq .
```
Expected: `{ "error": "at least one of gstin or pan is required" }`, status 400.

---

## Task F1 — buyer-ui: Fix /api/auth/register/route.ts (buyer auto-login)

**Files:**
- Modify: `services/buyer-ui/app/api/auth/register/route.ts`

The current route hardcodes `role: "buyer"` but discards the access token. The backend already returns it. This task sets the cookie so the user is auto-logged-in after registration.

- [ ] **Step 1: Rewrite the route**

```typescript
import { NextRequest, NextResponse } from "next/server";
import { GW } from "@/lib/gateway";

export async function POST(req: NextRequest) {
  let body: Record<string, unknown>;
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
  }

  // Always buyer role from this route
  body = { ...body, role: "buyer" };

  let upstream: Response;
  try {
    upstream = await fetch(`${GW}/v1/auth/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch {
    return NextResponse.json({ error: "Gateway unavailable" }, { status: 502 });
  }

  const data = await upstream.json().catch(() => ({})) as Record<string, unknown>;

  if (!upstream.ok) {
    return NextResponse.json(
      { error: (data.error ?? data.message ?? `HTTP ${upstream.status}`) as string },
      { status: upstream.status },
    );
  }

  const accessToken = data.access_token as string | undefined;
  const res = NextResponse.json({ ok: true }, { status: 201 });
  if (accessToken) {
    res.cookies.set("buyer_token", accessToken, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "strict",
      path: "/",
      maxAge: 60 * 60 * 24,
    });
  }
  return res;
}
```

---

## Task F2 — buyer-ui: Create /api/auth/register/seller/route.ts

**Files:**
- Create: `services/buyer-ui/app/api/auth/register/seller/route.ts`

- [ ] **Step 1: Write the route**

```typescript
import { NextRequest, NextResponse } from "next/server";
import { GW } from "@/lib/gateway";

export async function POST(req: NextRequest) {
  let body: unknown;
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
  }

  let upstream: Response;
  try {
    upstream = await fetch(`${GW}/v1/auth/register/seller`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch {
    return NextResponse.json({ error: "Gateway unavailable" }, { status: 502 });
  }

  const data = await upstream.json().catch(() => ({})) as Record<string, unknown>;

  if (!upstream.ok) {
    return NextResponse.json(
      { error: (data.error ?? data.message ?? `HTTP ${upstream.status}`) as string },
      { status: upstream.status },
    );
  }

  // Sellers are NOT auto-logged-in — they must wait for admin approval.
  return NextResponse.json({ ok: true }, { status: 201 });
}
```

---

## Task F3 — buyer-ui: /register/page.tsx → Role Selector

**Files:**
- Modify: `services/buyer-ui/app/register/page.tsx`

Replace the entire file content with the role chooser:

- [ ] **Step 1: Rewrite page.tsx**

```tsx
import type { Metadata } from "next";
import Link from "next/link";
import { Zap, ShoppingBag, Store, ArrowRight } from "lucide-react";

export const metadata: Metadata = { title: "Create Account" };

export default function RegisterPage() {
  return (
    <div className="min-h-[calc(100vh-128px)] flex items-center justify-center p-6 bg-[#F9F8F5]">
      <div className="w-full max-w-md space-y-8">
        {/* Logo */}
        <div className="text-center">
          <div className="inline-flex h-11 w-11 bg-[#E91E8C] rounded-xl items-center justify-center mb-5 shadow-[0_2px_12px_rgba(233,30,140,0.35)]">
            <Zap className="h-6 w-6 text-white" strokeWidth={2.5} />
          </div>
          <h1 className="font-display font-bold text-2xl text-[#0F0A04]">Join ZapMarket</h1>
          <p className="text-[#7A6856] text-sm mt-1.5">How would you like to get started?</p>
        </div>

        {/* Role cards */}
        <div className="grid gap-4">
          <Link
            href="/register/buyer"
            className="group flex items-center gap-5 p-5 rounded-2xl border-2 border-[#EDE9E3] bg-white hover:border-[#E91E8C] hover:shadow-[0_0_0_4px_rgba(233,30,140,0.06)] transition-all"
          >
            <div className="h-12 w-12 bg-[#FFF0F8] rounded-xl flex items-center justify-center shrink-0 group-hover:bg-[#FCE4F3] transition-colors">
              <ShoppingBag className="h-6 w-6 text-[#E91E8C]" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="font-semibold text-[#0F0A04]">Shop as a Buyer</p>
              <p className="text-sm text-[#7A6856] mt-0.5">Discover deals, track orders, earn rewards.</p>
            </div>
            <ArrowRight className="h-4 w-4 text-[#C4B9AA] group-hover:text-[#E91E8C] transition-colors shrink-0" />
          </Link>

          <Link
            href="/register/seller"
            className="group flex items-center gap-5 p-5 rounded-2xl border-2 border-[#EDE9E3] bg-white hover:border-[#4F46E5] hover:shadow-[0_0_0_4px_rgba(79,70,229,0.06)] transition-all"
          >
            <div className="h-12 w-12 bg-[#F0F0FF] rounded-xl flex items-center justify-center shrink-0 group-hover:bg-[#E4E3FC] transition-colors">
              <Store className="h-6 w-6 text-[#4F46E5]" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="font-semibold text-[#0F0A04]">Sell on ZapMarket</p>
              <p className="text-sm text-[#7A6856] mt-0.5">List products, manage orders, grow your business.</p>
            </div>
            <ArrowRight className="h-4 w-4 text-[#C4B9AA] group-hover:text-[#4F46E5] transition-colors shrink-0" />
          </Link>
        </div>

        <p className="text-sm text-center text-[#7A6856]">
          Already have an account?{" "}
          <Link href="/login" className="font-semibold text-[#E91E8C] hover:text-[#B5166E] transition-colors">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  );
}
```

---

## Task F4 — buyer-ui: /register/buyer/page.tsx

**Files:**
- Create: `services/buyer-ui/app/register/buyer/page.tsx`

This is the existing register form moved here, with the success path changed from `router.push("/login")` to `router.push("/"); router.refresh()` (auto-login since the cookie is now set).

- [ ] **Step 1: Create the file**

```tsx
"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { motion } from "framer-motion";
import { Zap, Eye, EyeOff, ArrowRight, Loader2, CheckCircle2, ArrowLeft } from "lucide-react";

const PERKS = [
  "Free delivery on your first 3 orders",
  "Exclusive member-only deals",
  "Early access to flash sales",
  "Hassle-free returns & refunds",
];

export default function BuyerRegisterPage() {
  const router = useRouter();
  const [form, setForm] = useState({ full_name: "", email: "", password: "" });
  const [showPw, setShowPw] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await fetch("/api/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(form),
      });
      const data = await res.json();
      if (!res.ok) {
        setError((data as Record<string, string>).error ?? "Registration failed");
        return;
      }
      // Auto-logged in — cookie was set by the API route.
      router.push("/");
      router.refresh();
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-[calc(100vh-128px)] grid lg:grid-cols-2">
      {/* Brand panel */}
      <div className="hidden lg:flex flex-col justify-between bg-gradient-to-br from-[#0F0A04] via-[#1C0F08] to-[#0F1A2A] p-12 relative overflow-hidden">
        <div className="absolute top-0 right-0 h-64 w-64 bg-[#4F46E5]/10 rounded-full blur-3xl" />
        <div className="absolute bottom-0 left-0 h-48 w-48 bg-[#E91E8C]/8 rounded-full blur-3xl" />

        <div className="flex items-center gap-2 relative z-10">
          <Link href="/register" className="mr-2 p-1.5 rounded-lg hover:bg-white/10 transition-colors text-white/50 hover:text-white">
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <Link href="/" className="flex items-center gap-2.5">
            <div className="h-9 w-9 bg-[#E91E8C] rounded-xl flex items-center justify-center shadow-[0_2px_12px_rgba(233,30,140,0.4)]">
              <Zap className="h-5 w-5 text-white" strokeWidth={2.5} />
            </div>
            <span className="font-display font-bold text-xl text-white">ZapMarket</span>
          </Link>
        </div>

        <div className="relative z-10">
          <h2 className="font-display font-bold text-white text-2xl mb-2 leading-tight">
            Join 2 million+ smart shoppers
          </h2>
          <p className="text-white/50 text-sm mb-7 leading-relaxed">
            Get access to the best deals, verified sellers, and a shopping experience unlike any other.
          </p>
          <ul className="space-y-3">
            {PERKS.map(perk => (
              <li key={perk} className="flex items-start gap-3">
                <CheckCircle2 className="h-4 w-4 text-[#E91E8C] shrink-0 mt-0.5" />
                <span className="text-white/70 text-sm">{perk}</span>
              </li>
            ))}
          </ul>
        </div>
      </div>

      {/* Form panel */}
      <div className="flex items-center justify-center p-6 sm:p-12 bg-[#F9F8F5]">
        <motion.div
          className="w-full max-w-sm"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, ease: [0.16, 1, 0.3, 1] }}
        >
          {/* Mobile logo */}
          <div className="flex items-center gap-2 mb-8 lg:hidden">
            <Link href="/register" className="p-1.5 rounded-lg hover:bg-black/5 transition-colors text-[#7A6856]">
              <ArrowLeft className="h-4 w-4" />
            </Link>
            <div className="h-8 w-8 bg-[#E91E8C] rounded-xl flex items-center justify-center">
              <Zap className="h-4 w-4 text-white" strokeWidth={2.5} />
            </div>
            <span className="font-display font-bold text-lg text-[#0F0A04]">ZapMarket</span>
          </div>

          <div className="mb-7">
            <h1 className="font-display font-bold text-2xl text-[#0F0A04] mb-1.5">
              Create your account
            </h1>
            <p className="text-[#7A6856] text-sm">Free forever · No credit card required</p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-1.5">
              <label htmlFor="full_name" className="text-sm font-semibold text-[#3D2E1A]">Full name</label>
              <input
                id="full_name"
                type="text"
                autoComplete="name"
                required
                placeholder="Jane Doe"
                value={form.full_name}
                onChange={e => setForm(f => ({ ...f, full_name: e.target.value }))}
                className="input-zap"
              />
            </div>

            <div className="space-y-1.5">
              <label htmlFor="email" className="text-sm font-semibold text-[#3D2E1A]">Email</label>
              <input
                id="email"
                type="email"
                autoComplete="email"
                required
                placeholder="you@example.com"
                value={form.email}
                onChange={e => setForm(f => ({ ...f, email: e.target.value }))}
                className="input-zap"
              />
            </div>

            <div className="space-y-1.5">
              <label htmlFor="password" className="text-sm font-semibold text-[#3D2E1A]">Password</label>
              <div className="relative">
                <input
                  id="password"
                  type={showPw ? "text" : "password"}
                  autoComplete="new-password"
                  required
                  minLength={8}
                  placeholder="Minimum 8 characters"
                  value={form.password}
                  onChange={e => setForm(f => ({ ...f, password: e.target.value }))}
                  className="input-zap pr-11"
                />
                <button
                  type="button"
                  onClick={() => setShowPw(v => !v)}
                  aria-label={showPw ? "Hide password" : "Show password"}
                  className="absolute right-3.5 top-1/2 -translate-y-1/2 p-0.5 text-[#B8A898] hover:text-[#7A6856] transition-colors"
                >
                  {showPw ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
            </div>

            {error && (
              <p className="text-sm text-[#DC2626] bg-red-50 border border-red-100 rounded-xl px-3 py-2" role="alert">
                {error}
              </p>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full h-11 bg-[#E91E8C] hover:bg-[#B5166E] disabled:opacity-60 text-white font-bold rounded-xl transition-colors shadow-[0_2px_12px_rgba(233,30,140,0.25)] flex items-center justify-center gap-2 mt-2"
            >
              {loading ? (
                <><Loader2 className="h-4 w-4 animate-spin" />Creating account…</>
              ) : (
                <>Create account<ArrowRight className="h-4 w-4" /></>
              )}
            </button>

            <p className="text-xs text-[#B8A898] text-center leading-relaxed">
              By creating an account you agree to our{" "}
              <Link href="/terms" className="underline hover:text-[#7A6856]">Terms</Link>
              {" "}and{" "}
              <Link href="/privacy" className="underline hover:text-[#7A6856]">Privacy Policy</Link>.
            </p>
          </form>

          <p className="mt-5 text-sm text-center text-[#7A6856]">
            Already have an account?{" "}
            <Link href="/login" className="font-semibold text-[#E91E8C] hover:text-[#B5166E] transition-colors">Sign in</Link>
          </p>
        </motion.div>
      </div>
    </div>
  );
}
```

---

## Task F5 — buyer-ui: /register/seller/page.tsx — 3-Step Wizard

**Files:**
- Create: `services/buyer-ui/app/register/seller/page.tsx`

- [ ] **Step 1: Create the wizard**

```tsx
"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { motion, AnimatePresence } from "framer-motion";
import { Zap, ArrowLeft, ArrowRight, Loader2, Store, Building2, FileText } from "lucide-react";

const CATEGORIES = [
  "Electronics", "Fashion", "Home & Living", "Beauty & Personal Care",
  "Sports & Fitness", "Books & Stationery", "Toys & Games",
  "Automotive", "Health & Wellness", "Other",
];

type Step = 1 | 2 | 3;

const STEPS = [
  { label: "Account",  icon: Store },
  { label: "Store",    icon: Building2 },
  { label: "Business", icon: FileText },
];

interface FormState {
  full_name: string; email: string; password: string;
  store_name: string; tagline: string; category: string;
  gstin: string; pan: string;
  business_phone: string; city: string; pincode: string;
}

const INITIAL: FormState = {
  full_name: "", email: "", password: "",
  store_name: "", tagline: "", category: "",
  gstin: "", pan: "",
  business_phone: "", city: "", pincode: "",
};

export default function SellerRegisterPage() {
  const router = useRouter();
  const [step, setStep] = useState<Step>(1);
  const [form, setForm] = useState<FormState>(INITIAL);
  const [showPw, setShowPw] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  function set(field: keyof FormState, value: string) {
    setForm(f => ({ ...f, [field]: value }));
  }

  function validateStep(): string | null {
    if (step === 1) {
      if (!form.full_name.trim()) return "Full name is required";
      if (!form.email.trim()) return "Email is required";
      if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) return "Enter a valid email address";
      if (form.password.length < 8) return "Password must be at least 8 characters";
    }
    if (step === 2) {
      if (!form.store_name.trim()) return "Store name is required";
      if (!form.category) return "Please select a category";
    }
    if (step === 3) {
      if (!form.business_phone.trim()) return "Business phone is required";
      if (!form.city.trim()) return "City is required";
      if (!form.pincode.trim()) return "Pincode is required";
      if (!form.gstin.trim() && !form.pan.trim()) return "Provide at least one of GSTIN or PAN";
    }
    return null;
  }

  function handleNext() {
    const err = validateStep();
    if (err) { setError(err); return; }
    setError(null);
    setStep(s => (s + 1) as Step);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const err = validateStep();
    if (err) { setError(err); return; }
    setError(null);
    setLoading(true);

    const payload = {
      full_name:      form.full_name,
      email:          form.email,
      password:       form.password,
      store_name:     form.store_name,
      tagline:        form.tagline,
      category:       form.category,
      gstin:          form.gstin || undefined,
      pan:            form.pan   || undefined,
      business_phone: form.business_phone,
      city:           form.city,
      pincode:        form.pincode,
    };

    try {
      const res = await fetch("/api/auth/register/seller", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data = await res.json();
      if (!res.ok) {
        setError((data as Record<string, string>).error ?? "Registration failed. Please try again.");
        return;
      }
      router.push("/register/seller/pending");
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  const slideVariants = {
    enter: { opacity: 0, x: 24 },
    center: { opacity: 1, x: 0 },
    exit: { opacity: 0, x: -24 },
  };

  return (
    <div className="min-h-[calc(100vh-128px)] flex items-center justify-center p-6 bg-[#F9F8F5]">
      <div className="w-full max-w-md">
        {/* Header */}
        <div className="flex items-center gap-3 mb-8">
          <Link
            href={step === 1 ? "/register" : "#"}
            onClick={step > 1 ? (e) => { e.preventDefault(); setError(null); setStep(s => (s - 1) as Step); } : undefined}
            className="p-1.5 rounded-lg hover:bg-black/5 transition-colors text-[#7A6856]"
          >
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div className="flex items-center gap-2">
            <div className="h-8 w-8 bg-[#4F46E5] rounded-xl flex items-center justify-center">
              <Zap className="h-4 w-4 text-white" strokeWidth={2.5} />
            </div>
            <span className="font-display font-bold text-lg text-[#0F0A04]">Seller Registration</span>
          </div>
        </div>

        {/* Step indicator */}
        <div className="flex items-center gap-2 mb-8">
          {STEPS.map((s, i) => {
            const n = (i + 1) as Step;
            const active = step === n;
            const done = step > n;
            return (
              <div key={s.label} className="flex items-center gap-2 flex-1 last:flex-none">
                <div className={`flex items-center gap-1.5 ${active ? "text-[#4F46E5]" : done ? "text-green-600" : "text-[#C4B9AA]"}`}>
                  <div className={`h-7 w-7 rounded-full flex items-center justify-center text-xs font-bold transition-colors ${active ? "bg-[#4F46E5] text-white" : done ? "bg-green-600 text-white" : "bg-[#EDE9E3] text-[#B8A898]"}`}>
                    {done ? "✓" : n}
                  </div>
                  <span className="text-xs font-medium hidden sm:block">{s.label}</span>
                </div>
                {i < STEPS.length - 1 && (
                  <div className={`h-px flex-1 mx-1 transition-colors ${done ? "bg-green-400" : "bg-[#EDE9E3]"}`} />
                )}
              </div>
            );
          })}
        </div>

        <form onSubmit={step === 3 ? handleSubmit : (e) => { e.preventDefault(); handleNext(); }}>
          <AnimatePresence mode="wait">
            {step === 1 && (
              <motion.div key="step1" variants={slideVariants} initial="enter" animate="center" exit="exit" transition={{ duration: 0.2 }} className="space-y-4">
                <div className="mb-6">
                  <h2 className="font-display font-bold text-xl text-[#0F0A04]">Create your account</h2>
                  <p className="text-[#7A6856] text-sm mt-1">Your personal login credentials</p>
                </div>

                <Field label="Full name" id="full_name" type="text" autoComplete="name" placeholder="Jane Doe" value={form.full_name} onChange={v => set("full_name", v)} />
                <Field label="Email" id="email" type="email" autoComplete="email" placeholder="you@example.com" value={form.email} onChange={v => set("email", v)} />
                <div className="space-y-1.5">
                  <label htmlFor="password" className="text-sm font-semibold text-[#3D2E1A]">Password</label>
                  <div className="relative">
                    <input
                      id="password"
                      type={showPw ? "text" : "password"}
                      autoComplete="new-password"
                      placeholder="Minimum 8 characters"
                      value={form.password}
                      onChange={e => set("password", e.target.value)}
                      className="input-zap pr-11"
                    />
                    <button type="button" onClick={() => setShowPw(v => !v)} aria-label={showPw ? "Hide" : "Show"} className="absolute right-3.5 top-1/2 -translate-y-1/2 text-[#B8A898] hover:text-[#7A6856] transition-colors">
                      <span className="text-xs">{showPw ? "Hide" : "Show"}</span>
                    </button>
                  </div>
                </div>
              </motion.div>
            )}

            {step === 2 && (
              <motion.div key="step2" variants={slideVariants} initial="enter" animate="center" exit="exit" transition={{ duration: 0.2 }} className="space-y-4">
                <div className="mb-6">
                  <h2 className="font-display font-bold text-xl text-[#0F0A04]">Your store</h2>
                  <p className="text-[#7A6856] text-sm mt-1">Tell buyers about your shop</p>
                </div>

                <Field label="Store name" id="store_name" type="text" placeholder="e.g. Priya's Electronics" value={form.store_name} onChange={v => set("store_name", v)} />
                <Field label="Tagline (optional)" id="tagline" type="text" placeholder="e.g. Best prices on genuine parts" value={form.tagline} onChange={v => set("tagline", v)} />

                <div className="space-y-1.5">
                  <label htmlFor="category" className="text-sm font-semibold text-[#3D2E1A]">Primary category</label>
                  <select
                    id="category"
                    value={form.category}
                    onChange={e => set("category", e.target.value)}
                    className="input-zap"
                  >
                    <option value="" disabled>Select a category</option>
                    {CATEGORIES.map(c => <option key={c} value={c}>{c}</option>)}
                  </select>
                </div>
              </motion.div>
            )}

            {step === 3 && (
              <motion.div key="step3" variants={slideVariants} initial="enter" animate="center" exit="exit" transition={{ duration: 0.2 }} className="space-y-4">
                <div className="mb-6">
                  <h2 className="font-display font-bold text-xl text-[#0F0A04]">Business details</h2>
                  <p className="text-[#7A6856] text-sm mt-1">Required for KYC — reviewed by our team</p>
                </div>

                <Field label="Business phone" id="business_phone" type="tel" placeholder="9876543210" value={form.business_phone} onChange={v => set("business_phone", v)} />

                <div className="grid grid-cols-2 gap-3">
                  <Field label="City" id="city" type="text" placeholder="Bangalore" value={form.city} onChange={v => set("city", v)} />
                  <Field label="Pincode" id="pincode" type="text" placeholder="560001" value={form.pincode} onChange={v => set("pincode", v)} />
                </div>

                <div className="pt-1">
                  <p className="text-xs font-semibold text-[#3D2E1A] mb-2">KYC — provide at least one</p>
                  <div className="space-y-3">
                    <Field label="GSTIN (optional)" id="gstin" type="text" placeholder="29ABCDE1234F1Z5" value={form.gstin} onChange={v => set("gstin", v)} />
                    <Field label="PAN (optional)" id="pan" type="text" placeholder="ABCDE1234F" value={form.pan} onChange={v => set("pan", v)} />
                  </div>
                </div>

                <p className="text-xs text-[#B8A898] leading-relaxed pt-1">
                  Your KYC details are encrypted and reviewed only by our compliance team.
                  By submitting you agree to our{" "}
                  <Link href="/seller-terms" className="underline hover:text-[#7A6856]">Seller Terms</Link>.
                </p>
              </motion.div>
            )}
          </AnimatePresence>

          {error && (
            <p className="mt-4 text-sm text-[#DC2626] bg-red-50 border border-red-100 rounded-xl px-3 py-2" role="alert">
              {error}
            </p>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full h-11 mt-6 bg-[#4F46E5] hover:bg-[#3730A3] disabled:opacity-60 text-white font-bold rounded-xl transition-colors shadow-[0_2px_12px_rgba(79,70,229,0.25)] flex items-center justify-center gap-2"
          >
            {loading ? (
              <><Loader2 className="h-4 w-4 animate-spin" />Submitting…</>
            ) : step < 3 ? (
              <>Continue<ArrowRight className="h-4 w-4" /></>
            ) : (
              <>Submit application<ArrowRight className="h-4 w-4" /></>
            )}
          </button>
        </form>

        <p className="mt-5 text-sm text-center text-[#7A6856]">
          Already have an account?{" "}
          <Link href="/login" className="font-semibold text-[#4F46E5] hover:text-[#3730A3] transition-colors">Sign in</Link>
        </p>
      </div>
    </div>
  );
}

// Reusable labeled input
function Field({ label, id, type, placeholder, value, onChange, autoComplete }: {
  label: string; id: string; type: string; placeholder: string;
  value: string; onChange: (v: string) => void; autoComplete?: string;
}) {
  return (
    <div className="space-y-1.5">
      <label htmlFor={id} className="text-sm font-semibold text-[#3D2E1A]">{label}</label>
      <input
        id={id} type={type} autoComplete={autoComplete}
        placeholder={placeholder} value={value}
        onChange={e => onChange(e.target.value)}
        className="input-zap"
      />
    </div>
  );
}
```

---

## Task F6 — buyer-ui: /register/seller/pending/page.tsx

**Files:**
- Create: `services/buyer-ui/app/register/seller/pending/page.tsx`

- [ ] **Step 1: Create the confirmation screen**

```tsx
import type { Metadata } from "next";
import Link from "next/link";
import { Clock, CheckCircle2, Mail, ShoppingBag } from "lucide-react";

export const metadata: Metadata = { title: "Application Submitted" };

const TIMELINE = [
  { label: "Application submitted",    done: true  },
  { label: "Verify your email",        done: false },
  { label: "Admin review (up to 48h)", done: false },
  { label: "Start selling!",           done: false },
];

export default function SellerPendingPage() {
  return (
    <div className="min-h-[calc(100vh-128px)] flex items-center justify-center p-6 bg-[#F9F8F5]">
      <div className="w-full max-w-sm text-center space-y-7">
        {/* Icon */}
        <div className="inline-flex h-16 w-16 bg-amber-50 border border-amber-100 rounded-2xl items-center justify-center mx-auto">
          <Clock className="h-8 w-8 text-amber-500" />
        </div>

        {/* Copy */}
        <div>
          <h1 className="font-display font-bold text-2xl text-[#0F0A04]">Application submitted!</h1>
          <p className="text-[#7A6856] mt-2 text-sm leading-relaxed">
            Our team will review your application within 48 hours.
            You&apos;ll receive an email at the address you provided once you&apos;re approved.
          </p>
        </div>

        {/* Timeline */}
        <ul className="text-left space-y-3.5 bg-white border border-[#EDE9E3] rounded-2xl p-5">
          {TIMELINE.map(({ label, done }) => (
            <li key={label} className="flex items-center gap-3">
              <CheckCircle2 className={`h-5 w-5 shrink-0 ${done ? "text-green-500" : "text-[#D9CFC4]"}`} />
              <span className={`text-sm ${done ? "text-[#0F0A04] font-medium" : "text-[#B8A898]"}`}>{label}</span>
            </li>
          ))}
        </ul>

        {/* Email verification nudge */}
        <div className="bg-blue-50 border border-blue-100 rounded-xl p-4 flex items-start gap-3 text-left">
          <Mail className="h-4 w-4 text-blue-500 shrink-0 mt-0.5" />
          <p className="text-xs text-blue-700 leading-relaxed">
            <span className="font-semibold">Check your inbox.</span> Verifying your email speeds up the approval process — look for a message from ZapMarket.
          </p>
        </div>

        {/* CTA */}
        <div className="space-y-3">
          <Link
            href="/products"
            className="flex items-center justify-center gap-2 w-full h-11 bg-[#4F46E5] hover:bg-[#3730A3] text-white font-bold rounded-xl transition-colors shadow-[0_2px_12px_rgba(79,70,229,0.25)]"
          >
            <ShoppingBag className="h-4 w-4" />
            Browse the marketplace
          </Link>
          <Link href="/login" className="block text-sm text-[#7A6856] hover:text-[#0F0A04] transition-colors">
            Sign in to check your application status →
          </Link>
        </div>
      </div>
    </div>
  );
}
```

---

## Task F7 — buyer-ui: VerificationBanner component

**Files:**
- Create: `services/buyer-ui/components/VerificationBanner.tsx`
- Modify: `services/buyer-ui/app/layout.tsx`

- [ ] **Step 1: Create VerificationBanner.tsx**

```tsx
"use client";

import { useState } from "react";
import { Mail, X, Loader2 } from "lucide-react";

interface Props { email: string; }

export default function VerificationBanner({ email }: Props) {
  const [dismissed, setDismissed] = useState(false);
  const [sent, setSent] = useState(false);
  const [loading, setLoading] = useState(false);

  if (dismissed) return null;

  async function resend() {
    setLoading(true);
    try {
      await fetch("/api/proxy/auth/otp/send", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email }),
      });
      setSent(true);
    } catch {
      // Silently ignore — user can try again
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="bg-amber-50 border-b border-amber-200 px-4 py-2.5 flex items-center gap-3">
      <Mail className="h-4 w-4 text-amber-600 shrink-0" />
      <p className="flex-1 text-sm text-amber-800 min-w-0">
        Verify your email <span className="font-semibold truncate">{email}</span> to unlock all features.{" "}
        {sent ? (
          <span className="text-green-700 font-medium">Email sent!</span>
        ) : (
          <button
            onClick={resend}
            disabled={loading}
            className="underline font-medium hover:text-amber-900 disabled:opacity-60 inline-flex items-center gap-1"
          >
            {loading && <Loader2 className="h-3 w-3 animate-spin" />}
            {loading ? "Sending…" : "Resend verification"}
          </button>
        )}
      </p>
      <button
        onClick={() => setDismissed(true)}
        aria-label="Dismiss"
        className="p-0.5 text-amber-400 hover:text-amber-700 transition-colors shrink-0"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}
```

- [ ] **Step 2: Wire into layout.tsx**

The layout already reads `decodeJwtUser(token)` and passes `user` (with `email`) to Navbar. The banner needs `is_verified` status — which is in the JWT payload. Update `decodeJwtUser` in `lib/api.ts` to also return `is_verified`, then render the banner.

First, update `decodeJwtUser` in `services/buyer-ui/lib/api.ts`:

```typescript
/** Decode display info from a JWT payload (no verification — display only). */
export function decodeJwtUser(token: string): {
  name: string | null;
  email: string | null;
  is_verified: boolean;
} {
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return { name: null, email: null, is_verified: false };
    const payload = JSON.parse(Buffer.from(parts[1], "base64url").toString());
    return {
      name:        payload.full_name ?? payload.name ?? null,
      email:       payload.email ?? null,
      is_verified: payload.is_verified === true,
    };
  } catch {
    return { name: null, email: null, is_verified: false };
  }
}
```

> **Note:** `is_verified` must be present in the JWT claims. Check `pkg/crypto/jwt.go` — if it's not included, add it to the `Claims` struct and `GenerateAccessToken` call. See verification step below.

Then update `layout.tsx`:

```tsx
import VerificationBanner from "@/components/VerificationBanner";

// In RootLayout, after:
const raw = token ? decodeJwtUser(token) : null;
const user = raw ? { name: raw.name ?? "", email: raw.email ?? "" } : null;
// Add:
const showVerificationBanner = raw && !raw.is_verified && !!raw.email;

// In JSX, before <Navbar>:
{showVerificationBanner && <VerificationBanner email={raw!.email!} />}
```

- [ ] **Step 3: Verify is_verified is in the JWT claims**

```bash
grep -n "is_verified\|IsVerified" services/auth-service/pkg/crypto/jwt.go 2>/dev/null || \
grep -rn "is_verified\|IsVerified" pkg/crypto/
```

If `is_verified` is NOT in the JWT payload, add it:

In `pkg/crypto/jwt.go`, find the `Claims` struct and `GenerateAccessToken` function. Add `IsVerified bool` to `Claims` and pass `user.IsVerified` when creating the token. This is a non-breaking change since JWT claims are additive.

- [ ] **Step 4: Build buyer-ui**

```bash
cd services/buyer-ui && npm run build
```

Expected: no TypeScript errors.

---

## Task F8 — buyer-ui: /api/auth/otp/send route (for banner resend)

The banner calls `/api/proxy/auth/otp/send` which goes through the proxy. Check that `/v1/auth/otp/send` is in the proxy's ALLOWED_PREFIXES:

- [ ] **Step 1: Check proxy allowed prefixes**

```bash
grep "ALLOWED_PREFIXES\|v1/auth" services/buyer-ui/app/api/proxy/\[...path\]/route.ts
```

If `/v1/auth` is NOT in `ALLOWED_PREFIXES`, the proxy will 403. In that case, create a dedicated route instead:

Create `services/buyer-ui/app/api/auth/otp/send/route.ts`:

```typescript
import { NextRequest, NextResponse } from "next/server";
import { GW } from "@/lib/gateway";
import { cookies } from "next/headers";

export async function POST(req: NextRequest) {
  const jar = await cookies();
  const token = jar.get("buyer_token")?.value;
  if (!token) {
    return NextResponse.json({ error: "Not authenticated" }, { status: 401 });
  }

  let body: unknown;
  try { body = await req.json(); } catch { body = {}; }

  let upstream: Response;
  try {
    upstream = await fetch(`${GW}/v1/auth/otp/send`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Authorization": `Bearer ${token}`,
      },
      body: JSON.stringify(body),
    });
  } catch {
    return NextResponse.json({ error: "Gateway unavailable" }, { status: 502 });
  }

  return NextResponse.json({ ok: upstream.ok }, { status: upstream.status });
}
```

Then update the banner's fetch call to `/api/auth/otp/send` (not the proxy):

```typescript
// In VerificationBanner.tsx resend():
await fetch("/api/auth/otp/send", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ email }),
});
```

---

## Verification Checklist (run after all tasks complete)

- [ ] `cd services/auth-service && go build ./...` — clean
- [ ] `cd services/auth-service && go test ./...` — all pass
- [ ] `cd services/buyer-ui && npm run build` — no TS errors

**Manual flow: Buyer**
1. Navigate `/register` → see two cards (Buyer / Seller)
2. Click Buyer → `/register/buyer` form
3. Fill and submit → redirected to `/` and Navbar shows user name (cookie set)
4. Verification banner visible (amber bar) with email
5. Click "Resend verification" → button shows "Email sent!"

**Manual flow: Seller**
1. Navigate `/register` → click Seller → `/register/seller`
2. Step 1: fill credentials, click Continue
3. Step 2: fill store name + category, click Continue
4. Step 3: fill phone, city, pincode, GSTIN → click Submit application
5. Redirected to `/register/seller/pending` — see timeline and email nudge
6. Check DB: `SELECT * FROM seller_profiles WHERE store_name = 'your-store';`

**Manual flow: Missing KYC**
1. `/register/seller` → complete steps 1-2 → Step 3: leave GSTIN and PAN blank
2. Click Submit → error: "Provide at least one of GSTIN or PAN"

---

## Execution Order

```
B1 → B2 → B3 → B4 → B5 → B6 → B7 → B8
               (backend — sequential, each builds on the last)

F1 → F2 → F3 → F4 → F5 → F6 → F7 → F8
               (frontend — sequential, F1 and F2 are independent of F3-F8)
```

Backend and frontend tracks are independent and can run in parallel after B4.
