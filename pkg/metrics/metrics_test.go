package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/zapmarket/zapmarket/pkg/metrics"
)

func TestMiddleware_IncrementsCounter(t *testing.T) {
	m := metrics.New("testsvc")

	handler := m.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/foo", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	count := testutil.ToFloat64(m.RequestsTotal.WithLabelValues("POST", "201"))
	if count != 1 {
		t.Fatalf("expected counter=1, got %v", count)
	}
}

func TestMiddleware_DefaultsTo200(t *testing.T) {
	m := metrics.New("testsvc2")

	handler := m.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/bar", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	count := testutil.ToFloat64(m.RequestsTotal.WithLabelValues("GET", "200"))
	if count != 1 {
		t.Fatalf("expected counter=1, got %v", count)
	}
}
