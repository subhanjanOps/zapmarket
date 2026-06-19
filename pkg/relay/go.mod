module github.com/zapmarket/zapmarket/pkg/relay

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/zapmarket/zapmarket/pkg/kafka v0.0.0
)

require (
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/segmentio/kafka-go v0.4.47 // indirect
)

replace github.com/zapmarket/zapmarket/pkg/kafka => ../kafka
