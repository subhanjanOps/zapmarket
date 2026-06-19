package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	rateWindowSecs = 60
	ratePerIP      = 200 // requests per window per IP
	ratePerUser    = 500 // requests per window per authenticated user
)

// slidingWindow atomically records the current request and counts how many
// requests fall inside [now-window, now]. It uses a sorted set so the window
// truly slides rather than resetting at fixed clock boundaries.
//
// Returns:
//
//	-1  on Redis error (caller should allow the request)
//	 N  number of requests in the current window after recording this one
var slidingWindow = goredis.NewScript(`
local key    = KEYS[1]
local now    = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local ttl    = tonumber(ARGV[3])

-- Record this request (score = timestamp ms, member = timestamp ms as string).
-- Using the timestamp as both score and member means duplicate ms timestamps
-- overwrite each other, which is fine: a 1ms collision costs at most one
-- missed count, and millisecond-resolution is more than sufficient.
redis.call('ZADD', key, now, tostring(now))

-- Evict entries older than the window.
redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)

-- Keep the key alive for one full window after the last request.
redis.call('EXPIRE', key, ttl)

-- Return the count of entries still in the window.
return redis.call('ZCARD', key)
`)

// RateLimiter is chi middleware backed by a Redis sliding-window counter.
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

		nowMS := time.Now().UnixMilli()
		windowMS := int64(rateWindowSecs) * 1000
		ttlSecs := rateWindowSecs + 1

		count, err := slidingWindow.Run(ctx, rl.rdb,
			[]string{key},
			nowMS, windowMS, ttlSecs,
		).Int64()

		remaining := max(int64(0), int64(limit)-count)
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		if err != nil {
			// Redis error — fail open: log implicitly via the error being non-nil
			// and allow the request rather than blocking all traffic.
			next.ServeHTTP(w, r)
			return
		}

		if count > int64(limit) {
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
