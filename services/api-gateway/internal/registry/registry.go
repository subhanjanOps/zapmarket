package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	registryKeyPrefix = "svc:registry:"
	heartbeatTTL      = 30 * time.Second
	heartbeatInterval = 10 * time.Second
)

// Instance is one live endpoint of a service.
type Instance struct {
	Addr       string    `json:"addr"`
	InstanceID string    `json:"instance_id"`
	StartedAt  time.Time `json:"started_at"`
}

// ServiceRegistry is the interface the gateway uses to resolve upstream addresses.
type ServiceRegistry interface {
	// Instances returns all healthy live instances of the named service.
	Instances(ctx context.Context, name string) ([]Instance, error)
}

// RedisRegistry discovers services via Redis heartbeat keys.
// Key pattern: svc:registry:{service-name}:{instance-id}  → JSON Instance  (TTL 30s)
type RedisRegistry struct {
	rdb    *goredis.Client
	logger *slog.Logger

	// round-robin cursor per service name
	mu      sync.Mutex
	cursors map[string]int
}

func NewRedisRegistry(rdb *goredis.Client, logger *slog.Logger) *RedisRegistry {
	return &RedisRegistry{rdb: rdb, logger: logger, cursors: make(map[string]int)}
}

func (r *RedisRegistry) Instances(ctx context.Context, name string) ([]Instance, error) {
	pattern := registryKeyPrefix + name + ":*"
	var instances []Instance
	var cursor uint64
	for {
		keys, next, err := r.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("registry scan %s: %w", name, err)
		}
		for _, k := range keys {
			raw, err := r.rdb.Get(ctx, k).Bytes()
			if err != nil {
				continue // key expired between SCAN and GET
			}
			var inst Instance
			if err := json.Unmarshal(raw, &inst); err != nil {
				r.logger.Warn("bad registry value", "key", k, "error", err)
				continue
			}
			instances = append(instances, inst)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return instances, nil
}

// AllInstances returns every live instance grouped by service name.
func (r *RedisRegistry) AllInstances(ctx context.Context) (map[string][]Instance, error) {
	result := make(map[string][]Instance)
	var cursor uint64
	for {
		keys, next, err := r.rdb.Scan(ctx, cursor, registryKeyPrefix+"*", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("registry scan: %w", err)
		}
		for _, k := range keys {
			// key: svc:registry:{service}:{instance}
			parts := strings.SplitN(strings.TrimPrefix(k, registryKeyPrefix), ":", 2)
			if len(parts) != 2 {
				continue
			}
			svc := parts[0]
			raw, err := r.rdb.Get(ctx, k).Bytes()
			if err != nil {
				continue
			}
			var inst Instance
			if err := json.Unmarshal(raw, &inst); err != nil {
				continue
			}
			result[svc] = append(result[svc], inst)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return result, nil
}

// Pick returns the next instance for the named service using round-robin.
// Returns ("", false) if no instances are available.
func (r *RedisRegistry) Pick(ctx context.Context, name string) (string, bool) {
	instances, err := r.Instances(ctx, name)
	if err != nil || len(instances) == 0 {
		return "", false
	}

	r.mu.Lock()
	idx := r.cursors[name] % len(instances)
	r.cursors[name] = idx + 1
	r.mu.Unlock()

	return instances[idx].Addr, true
}

// StaticRegistry resolves service names from a fixed map (env-var fallback).
type StaticRegistry struct {
	addrs map[string]string
}

func NewStaticRegistry(addrs map[string]string) *StaticRegistry {
	return &StaticRegistry{addrs: addrs}
}

func (s *StaticRegistry) Instances(_ context.Context, name string) ([]Instance, error) {
	addr, ok := s.addrs[name]
	if !ok {
		return nil, nil
	}
	return []Instance{{Addr: addr, InstanceID: "static"}}, nil
}

// Heartbeat registers this process in Redis and keeps it alive until ctx is cancelled.
// Each service calls this from its main.go.
func Heartbeat(ctx context.Context, rdb *goredis.Client, serviceName, instanceID, addr string, logger *slog.Logger) {
	inst := Instance{Addr: addr, InstanceID: instanceID, StartedAt: time.Now()}
	data, _ := json.Marshal(inst)
	key := registryKeyPrefix + serviceName + ":" + instanceID

	register := func() {
		if err := rdb.Set(ctx, key, data, heartbeatTTL).Err(); err != nil {
			logger.Warn("service registration heartbeat failed", "error", err)
		}
	}
	register()

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			// Best-effort deregister.
			_ = rdb.Del(context.Background(), key).Err()
			return
		case <-ticker.C:
			register()
		}
	}
}

