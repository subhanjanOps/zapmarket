package ports

import "time"

// MetricsRecorder abstracts observability instrumentation so use cases don't
// import infrastructure packages.
type MetricsRecorder interface {
	RecordCacheHit()
	RecordCacheMiss()
	RecordFetchDuration(start time.Time)
}
