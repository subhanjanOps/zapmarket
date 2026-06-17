package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Client wraps *goredis.Client with a health check and standard pool config.
type Client struct {
	*goredis.Client
}

// New creates a connected Redis client. Returns an error if the server is
// unreachable (Ping fails within the default timeout).
func New(addr string) (*Client, error) {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:         addr,
		PoolSize:     10,
		MinIdleConns: 2,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}

	return &Client{rdb}, nil
}
