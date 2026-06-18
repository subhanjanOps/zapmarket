package middleware

import (
	"net"
	"net/http"
	"strings"

	goredis "github.com/redis/go-redis/v9"
)

const BlocklistKey     = "gw:blocklist"
const BlocklistMetaKey = "gw:blocklist:meta"

// ClientIP returns the originating IP, respecting X-Forwarded-For.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
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
			if err == nil && blocked {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"IP_BLOCKED","message":"your IP address has been blocked"}}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
