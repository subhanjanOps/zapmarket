package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	registryKeyPrefix = "svc:registry:"
	manifestKeyPrefix = "svc:manifest:"
	heartbeatTTL      = 30 * time.Second
	heartbeatInterval = 10 * time.Second
)

// Instance is one live endpoint of a service.
type Instance struct {
	Addr       string    `json:"addr"`
	InstanceID string    `json:"instance_id"`
	StartedAt  time.Time `json:"started_at"`
}

// Manifest is what a service declares about itself on startup.
type Manifest struct {
	Service string          `json:"service"`
	Version string          `json:"version"`
	Routes  []ManifestRoute `json:"routes"`
}

type ManifestRoute struct {
	PathPrefix  string `json:"path_prefix"`
	AuthMode    string `json:"auth_mode"`
	StripPrefix bool   `json:"strip_prefix"`
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
	keys, err := r.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("registry scan %s: %w", name, err)
	}
	if len(keys) == 0 {
		return nil, nil
	}

	var instances []Instance
	for _, k := range keys {
		raw, err := r.rdb.Get(ctx, k).Bytes()
		if err != nil {
			continue // key expired between KEYS and GET
		}
		var inst Instance
		if err := json.Unmarshal(raw, &inst); err != nil {
			r.logger.Warn("bad registry value", "key", k, "error", err)
			continue
		}
		instances = append(instances, inst)
	}
	return instances, nil
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

// PublishManifest writes the service manifest to Redis.
// The gateway's auto-bind watcher reads these keys to auto-register routes.
func PublishManifest(ctx context.Context, rdb *goredis.Client, m Manifest) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	key := manifestKeyPrefix + m.Service
	return rdb.Set(ctx, key, data, 60*time.Second).Err()
}
