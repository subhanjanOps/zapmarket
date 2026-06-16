// Package http holds the inventory-service's HTTP surface, which is
// deliberately minimal — per design.md, Inventory has no public REST API,
// only gRPC. This package exists solely for an ops health-check endpoint.
package http

import "net/http"

// Health responds 200 if the process is up. It does not check DB
// connectivity — main.go already fails fast at startup if the DB is
// unreachable, so a successfully-running process implies a working DB
// connection at boot; deeper liveness/readiness checks are a Stage 10
// (observability) concern.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
