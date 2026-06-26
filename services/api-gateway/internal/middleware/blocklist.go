package middleware

import (
	"net"
	"net/http"

	goredis "github.com/redis/go-redis/v9"
)

const BlocklistKey     = "gw:blocklist"
const BlocklistMetaKey = "gw:blocklist:meta"

// ClientIP returns the true originating IP from r.RemoteAddr.
// This gateway is the edge — we never trust client-supplied X-Forwarded-For
// because it can be spoofed to bypass the blocklist.
func ClientIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// Blocklist returns a middleware that rejects requests from IPs in the Redis blocklist set.
func Blocklist(rdb *goredis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := ClientIP(r)
			blocked, err := rdb.SIsMember(r.Context(), BlocklistKey, ip).Result()
			if err != nil {
				// Redis unavailable — fail closed: deny rather than allow unknown blocklist state.
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"SERVICE_UNAVAILABLE","message":"service temporarily unavailable"}}`))
				return
			}
			if blocked {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"IP_BLOCKED","message":"your IP address has been blocked"}}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
