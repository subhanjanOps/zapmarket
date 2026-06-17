package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	rateWindowSecs  = 60
	ratePerIP       = 200 // requests per window per IP
	ratePerUser     = 500 // requests per window per authenticated user
)

// RateLimiter is chi middleware backed by Redis sliding-window counters.
type RateLimiter struct {
	rdb *goredis.Client
}

func NewRateLimiter(rdb *goredis.Client) *RateLimiter {
	return &RateLimiter{rdb: rdb}
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Prefer authenticated user ID; fall back to IP.
		key, limit := ipKey(r), ratePerIP
		if u := UserFromContext(ctx); u != nil {
			key, limit = fmt.Sprintf("ratelimit:user:%s", u.ID), ratePerUser
		}

		count, err := rl.rdb.Incr(ctx, key).Result()
		if err == nil && count == 1 {
			_ = rl.rdb.Expire(ctx, key, rateWindowSecs*time.Second).Err()
		}

		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, int64(limit)-count)))

		if err == nil && count > int64(limit) {
			jsonError(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests, please slow down")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func ipKey(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}
	// Take only the first IP from a possible comma-separated list.
	if idx := strings.Index(ip, ","); idx != -1 {
		ip = strings.TrimSpace(ip[:idx])
	}
	// Strip port from RemoteAddr if present.
	if idx := strings.LastIndex(ip, ":"); idx != -1 && strings.Contains(ip, ":") && !strings.Contains(ip, "[") {
		ip = ip[:idx]
	}
	return fmt.Sprintf("ratelimit:ip:%s", ip)
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
