package middleware

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
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
local limit  = tonumber(ARGV[4])

-- Evict entries older than the window first.
redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)

-- Count requests already in the window.
local count = redis.call('ZCARD', key)

-- Reject over-limit requests WITHOUT recording them, so a rejected request
-- does not occupy a slot or refresh the window and keep an abuser locked out.
if count >= limit then
	return count + 1
end

-- Record this request (score = timestamp ms, member = timestamp ms as string).
-- Using the timestamp as both score and member means duplicate ms timestamps
-- overwrite each other, which is fine: a 1ms collision costs at most one
-- missed count, and millisecond-resolution is more than sufficient.
redis.call('ZADD', key, now, tostring(now))

-- Keep the key alive for one full window after the last request.
redis.call('EXPIRE', key, ttl)

-- Return the count of entries now in the window.
return count + 1
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
			nowMS, windowMS, ttlSecs, limit,
		).Int64()

		remaining := max(int64(0), int64(limit)-count)
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		if err != nil {
			// Redis error — fail open to avoid blocking all traffic, but log a warning.
			slog.Warn("rate limiter Redis error; failing open",
				"error", err,
				"key", key,
				"path", r.URL.Path,
			)
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
	// This gateway IS the edge — trust only r.RemoteAddr, never client-supplied
	// X-Forwarded-For which can be trivially spoofed to bypass rate limiting.
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	return fmt.Sprintf("ratelimit:ip:%s", ip)
}

