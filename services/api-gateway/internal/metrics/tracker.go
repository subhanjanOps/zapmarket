package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// UpstreamSnapshot is a point-in-time view of one upstream's request stats.
type UpstreamSnapshot struct {
	Name      string  `json:"name"`
	Total     int64   `json:"total"`
	Errors    int64   `json:"errors"`   // 5xx responses
	ReqPerMin float64 `json:"req_per_min"`
	ErrorRate float64 `json:"error_rate"` // 0.0–1.0
}

type upstreamStats struct {
	total  atomic.Int64
	errors atomic.Int64

	mu     sync.Mutex
	recent []int64 // unix nanoseconds of each request (last 2000)
}

func (s *upstreamStats) track(status int) {
	s.total.Add(1)
	if status >= 500 {
		s.errors.Add(1)
	}
	now := time.Now().UnixNano()
	s.mu.Lock()
	s.recent = append(s.recent, now)
	if len(s.recent) > 2000 {
		s.recent = s.recent[len(s.recent)-2000:]
	}
	s.mu.Unlock()
}

func (s *upstreamStats) snapshot(name string) UpstreamSnapshot {
	total := s.total.Load()
	errs := s.errors.Load()

	cutoff := time.Now().Add(-60 * time.Second).UnixNano()
	s.mu.Lock()
	i := 0
	for i < len(s.recent) && s.recent[i] < cutoff {
		i++
	}
	if i > 0 {
		s.recent = s.recent[i:]
	}
	rpm := float64(len(s.recent))
	s.mu.Unlock()

	er := 0.0
	if total > 0 {
		er = float64(errs) / float64(total)
	}
	return UpstreamSnapshot{Name: name, Total: total, Errors: errs, ReqPerMin: rpm, ErrorRate: er}
}

// Tracker is a thread-safe, in-memory per-upstream request counter.
// Call Track after every proxied request; call Snapshots to read stats.
type Tracker struct {
	mu    sync.RWMutex
	stats map[string]*upstreamStats
}

func NewTracker() *Tracker {
	return &Tracker{stats: make(map[string]*upstreamStats)}
}

// Track records one request for the named upstream.
func (t *Tracker) Track(upstream string, status int) {
	t.mu.RLock()
	s, ok := t.stats[upstream]
	t.mu.RUnlock()

	if !ok {
		t.mu.Lock()
		if s, ok = t.stats[upstream]; !ok {
			s = &upstreamStats{}
			t.stats[upstream] = s
		}
		t.mu.Unlock()
	}

	s.track(status)
}

// Snapshots returns a current snapshot for every tracked upstream.
func (t *Tracker) Snapshots() []UpstreamSnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	out := make([]UpstreamSnapshot, 0, len(t.stats))
	for name, s := range t.stats {
		out = append(out, s.snapshot(name))
	}
	return out
}
