package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/pkg/config"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/pkg/httpx"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain/contracts"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/service"
)

// AdminHandler provides admin-only endpoints: user management + seller verification.
type AdminHandler struct {
	userRepo contracts.UserRepository
	authSvc  *service.AuthService
	cfg      *config.Config
}

func NewAdminHandler(userRepo contracts.UserRepository, authSvc *service.AuthService, cfg *config.Config) *AdminHandler {
	return &AdminHandler{userRepo: userRepo, authSvc: authSvc, cfg: cfg}
}

// AdminAuthMiddleware validates the JWT and requires role == "admin".
func (h *AdminHandler) AdminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			httpx.Error(w, http.StatusUnauthorized, "MISSING_TOKEN", "authorization header is required")
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || strings.TrimSpace(parts[1]) == "" {
			httpx.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "bearer token required")
			return
		}
		user, err := h.authSvc.ValidateAccessToken(r.Context(), strings.TrimSpace(parts[1]))
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired token")
			return
		}
		if user.Role != string(domain.RoleAdmin) {
			httpx.Error(w, http.StatusForbidden, "FORBIDDEN", "admin role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// userResponse is the safe user projection returned to clients (no password hash).
type userResponse struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	Role         string    `json:"role"`
	IsVerified   bool      `json:"is_verified"`
	SellerStatus *string   `json:"seller_status,omitempty"`
	CreatedAt    string    `json:"created_at"`
}

func toUserResponse(u *domain.User) userResponse {
	return userResponse{
		ID:           u.ID,
		Email:        u.Email,
		FullName:     u.FullName,
		Role:         u.Role,
		IsVerified:   u.IsVerified,
		SellerStatus: u.SellerStatus,
		CreatedAt:    u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// pathID extracts the UUID segment that comes right after `segment` in the URL path,
// or the last segment if no label is found. Works with Go 1.22 ServeMux {id} params
// via r.PathValue as well as plain path trimming.
func pathID(r *http.Request, param string) (uuid.UUID, error) {
	// Go 1.22+ ServeMux path values
	if v := r.PathValue(param); v != "" {
		return uuid.Parse(v)
	}
	// fallback: split URL, find UUID-shaped segment
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	for _, p := range parts {
		if len(p) == 36 {
			return uuid.Parse(p)
		}
	}
	return uuid.Nil, pkgerrors.NewValidation("INVALID_ID", "id not found in path")
}

func adminDecodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// ── Users ─────────────────────────────────────────────────────────────────

// ListUsers handles GET /v1/admin/users
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = 20
	}

	users, total, err := h.userRepo.ListUsers(r.Context(), contracts.UserListParams{
		Role:   q.Get("role"),
		Search: q.Get("search"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}

	resp := make([]userResponse, len(users))
	for i, u := range users {
		resp[i] = toUserResponse(u)
	}
	page := 1
	if limit > 0 && offset > 0 {
		page = offset/limit + 1
	}
	httpx.Paginated(w, http.StatusOK, resp, total, page, limit)
}

// GetUser handles GET /v1/admin/users/{id}
func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ID", "user id must be a valid UUID")
		return
	}
	user, err := h.userRepo.GetUserByID(r.Context(), id)
	if err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}
	httpx.Success(w, http.StatusOK, toUserResponse(user))
}

// UpdateUserRole handles PUT /v1/admin/users/{id}/role
func (h *AdminHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ID", "user id must be a valid UUID")
		return
	}

	var body struct {
		Role string `json:"role"`
	}
	if err := adminDecodeJSON(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	allowed := map[string]struct{}{"buyer": {}, "seller": {}, "admin": {}}
	if _, ok := allowed[body.Role]; !ok {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ROLE", "role must be buyer, seller, or admin")
		return
	}

	user, err := h.userRepo.GetUserByID(r.Context(), id)
	if err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}
	user.Role = body.Role
	if err := h.userRepo.UpdateUser(r.Context(), user); err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}
	httpx.Success(w, http.StatusOK, toUserResponse(user))
}

// DeactivateUser handles DELETE /v1/admin/users/{id}
func (h *AdminHandler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ID", "user id must be a valid UUID")
		return
	}
	if err := h.userRepo.DeleteUser(r.Context(), id); err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}
	httpx.Success(w, http.StatusOK, map[string]string{"message": "user deactivated"})
}

// ── Sellers ───────────────────────────────────────────────────────────────

// ListSellers handles GET /v1/admin/sellers
func (h *AdminHandler) ListSellers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = 20
	}

	sellers, total, err := h.userRepo.ListSellers(r.Context(), q.Get("status"), limit, offset)
	if err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}

	resp := make([]userResponse, len(sellers))
	for i, u := range sellers {
		resp[i] = toUserResponse(u)
	}
	page := 1
	if limit > 0 && offset > 0 {
		page = offset/limit + 1
	}
	httpx.Paginated(w, http.StatusOK, resp, total, page, limit)
}

// UpdateSellerStatus handles PATCH /v1/admin/sellers/{id}/status
func (h *AdminHandler) UpdateSellerStatus(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ID", "seller id must be a valid UUID")
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := adminDecodeJSON(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	allowed := map[string]struct{}{"PENDING": {}, "APPROVED": {}, "SUSPENDED": {}}
	if _, ok := allowed[body.Status]; !ok {
		httpx.Error(w, http.StatusBadRequest, "INVALID_STATUS", "status must be PENDING, APPROVED, or SUSPENDED")
		return
	}

	if err := h.userRepo.UpdateSellerStatus(r.Context(), id, body.Status); err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}
	httpx.Success(w, http.StatusOK, map[string]string{"status": body.Status})
}
