package http

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const (
	sensitiveRateLimit  = 10
	sensitiveRateWindow = time.Minute
)

// slidingWindowScript is the same Lua sliding-window used by the api-gateway.
// Returns the request count in the current window; rejected requests are not
// recorded so an abuser cannot hold themselves permanently locked out.
var slidingWindowScript = goredis.NewScript(`
local key    = KEYS[1]
local now    = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local ttl    = tonumber(ARGV[3])
local limit  = tonumber(ARGV[4])

redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)
local count = redis.call('ZCARD', key)

if count >= limit then
	return count + 1
end

redis.call('ZADD', key, now, ARGV[5])
redis.call('EXPIRE', key, ttl)
return count + 1
`)

// IPRateLimit limits requests from a single IP to sensitiveRateLimit per
// sensitiveRateWindow using a Redis sliding-window counter. Fails open on
// Redis errors so the service stays functional.
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

			key := fmt.Sprintf("rl:auth:%s:%s", endpoint, ip)
			nowMS := time.Now().UnixMilli()
			windowMS := sensitiveRateWindow.Milliseconds()
			ttlSecs := int(sensitiveRateWindow.Seconds()) + 1
			member := uuid.New().String()

			ctx := r.Context()
			count, scriptErr := slidingWindowScript.Run(ctx, rdb,
				[]string{key},
				nowMS, windowMS, ttlSecs, sensitiveRateLimit, member,
			).Int64()

			if scriptErr != nil {
				next(w, r)
				return
			}

			if count > sensitiveRateLimit {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(sensitiveRateWindow.Seconds())))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"success": false,
					"error":   map[string]string{"code": "RATE_LIMITED", "message": "too many requests, please slow down"},
				})
				return
			}

			next(w, r)
		}
	}
}
