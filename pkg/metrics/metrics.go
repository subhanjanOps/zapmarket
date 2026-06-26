// Package metrics provides a shared Prometheus registry and base instrumentation
// for ZapMarket backend services.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Base holds the minimum set of Prometheus instruments required by all services.
type Base struct {
	registry      *prometheus.Registry
	RequestsTotal *prometheus.CounterVec
	ErrorsTotal   *prometheus.CounterVec
	DomainOps     *prometheus.CounterVec
}

// New creates a Base metrics registry for the given service namespace.
// namespace should be the service name with underscores (e.g. "auth", "inventory").
// DomainOps is a general-purpose domain counter; use the "op" and "result" labels.
func New(namespace string) *Base {
	reg := prometheus.NewRegistry()

	m := &Base{
		registry: reg,

		RequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total HTTP requests by method and status class.",
		}, []string{"method", "status"}),

		ErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "errors_total",
			Help:      "Total errors by kind.",
		}, []string{"kind"}),

		DomainOps: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "domain_ops_total",
			Help:      "Total domain-level operations by op and result.",
		}, []string{"op", "result"}),
	}

	reg.MustRegister(m.RequestsTotal, m.ErrorsTotal, m.DomainOps)
	return m
}

// Handler returns an http.Handler that serves the Prometheus metrics page.
func (m *Base) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
