package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	// sensitiveRateLimit is the max requests per IP per window on auth endpoints.
	sensitiveRateLimit = 10
	// sensitiveRateWindow is the sliding window duration.
	sensitiveRateWindow = time.Minute
)

// IPRateLimit returns a middleware that limits requests from a single IP to
// sensitiveRateLimit per sensitiveRateWindow using Redis INCR/EXPIRE.
// When Redis is unavailable the middleware passes through (fail-open) so the
// service remains functional.
func IPRateLimit(rdb *goredis.Client, endpoint string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if rdb == nil {
				next(w, r)
				return
			}

			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			windowSec := int(sensitiveRateWindow.Seconds())
			key := fmt.Sprintf("rl:ip:%s:%s:%d", endpoint, ip, time.Now().Unix()/int64(windowSec))

			ctx, cancel := context.WithTimeout(r.Context(), 200*time.Millisecond)
			defer cancel()

			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				// Redis error — fail open.
				next(w, r)
				return
			}
			if count == 1 {
				rdb.Expire(ctx, key, sensitiveRateWindow) //nolint:errcheck
			}
			if count > sensitiveRateLimit {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", windowSec))
				http.Error(w, `{"error":"rate limit exceeded","code":429}`, http.StatusTooManyRequests)
				return
			}

			next(w, r)
		}
	}
}
