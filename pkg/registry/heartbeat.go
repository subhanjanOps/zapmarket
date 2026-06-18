package registry

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	KeyPrefix = "svc:registry:"
	TTL       = 30 * time.Second
	Interval  = 10 * time.Second
)

// Instance is one live endpoint of a service stored in Redis.
type Instance struct {
	Addr       string    `json:"addr"`
	InstanceID string    `json:"instance_id"`
	StartedAt  time.Time `json:"started_at"`
}

// Heartbeat registers this process in Redis and refreshes it every 10s until
// ctx is cancelled, at which point it deregisters immediately.
//
// Call it in a goroutine from main:
//
//	go registry.Heartbeat(ctx, rdb, "auth-service", instanceID, "http://host:port", log)
func Heartbeat(ctx context.Context, rdb *goredis.Client, serviceName, instanceID, addr string, logger *slog.Logger) {
	inst := Instance{Addr: addr, InstanceID: instanceID, StartedAt: time.Now()}
	data, _ := json.Marshal(inst)
	key := KeyPrefix + serviceName + ":" + instanceID

	register := func() {
		if err := rdb.Set(ctx, key, data, TTL).Err(); err != nil {
			logger.Warn("heartbeat failed", "service", serviceName, "error", err)
		}
	}
	register()

	ticker := time.NewTicker(Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = rdb.Del(context.Background(), key).Err()
			return
		case <-ticker.C:
			register()
		}
	}
}
