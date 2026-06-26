package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus counters/histograms for the currency service.
type Metrics struct {
	registry *prometheus.Registry

	IngestTotal   *prometheus.CounterVec
	FetchDuration prometheus.Histogram
	CacheHits     prometheus.Counter
	CacheMisses   prometheus.Counter
}

func New() *Metrics {
	reg := prometheus.NewRegistry()

	m := &Metrics{
		registry: reg,

		IngestTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "currency",
			Name:      "ingest_total",
			Help:      "Total number of rate ingestion attempts, labeled by result (success|failure).",
		}, []string{"result"}),

		FetchDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "currency",
			Name:      "provider_fetch_duration_seconds",
			Help:      "Duration of external FX provider fetch requests.",
			Buckets:   prometheus.DefBuckets,
		}),

		CacheHits: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "currency",
			Name:      "cache_hits_total",
			Help:      "Total number of Redis cache hits for rates.",
		}),

		CacheMisses: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "currency",
			Name:      "cache_misses_total",
			Help:      "Total number of Redis cache misses for rates.",
		}),
	}

	reg.MustRegister(m.IngestTotal, m.FetchDuration, m.CacheHits, m.CacheMisses)
	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// ObserveFetch records the duration of an external provider fetch.
func (m *Metrics) ObserveFetch(start time.Time) {
	m.FetchDuration.Observe(time.Since(start).Seconds())
}

// RecordFetchDuration implements ports.MetricsRecorder.
func (m *Metrics) RecordFetchDuration(start time.Time) { m.ObserveFetch(start) }

// RecordCacheHit implements ports.MetricsRecorder.
func (m *Metrics) RecordCacheHit() { m.CacheHits.Inc() }

// RecordCacheMiss implements ports.MetricsRecorder.
func (m *Metrics) RecordCacheMiss() { m.CacheMisses.Inc() }

// RecordIngest increments the ingestion counter with the given result label.
func (m *Metrics) RecordIngest(success bool) {
	result := "success"
	if !success {
		result = "failure"
	}
	m.IngestTotal.WithLabelValues(result).Inc()
}
