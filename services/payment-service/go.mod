module github.com/zapmarket/zapmarket/services/payment-service

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.12.3
	github.com/redis/go-redis/v9 v9.20.1
	github.com/stripe/stripe-go/v82 v82.5.1
	github.com/zapmarket/zapmarket/pkg/config v0.0.0
	github.com/zapmarket/zapmarket/pkg/database v0.0.0
	github.com/zapmarket/zapmarket/pkg/errors v0.0.0
	github.com/zapmarket/zapmarket/pkg/grpcx v0.0.0
	github.com/zapmarket/zapmarket/pkg/httpx v0.0.0-00010101000000-000000000000
	github.com/zapmarket/zapmarket/pkg/kafka v0.0.0
	github.com/zapmarket/zapmarket/pkg/logger v0.0.0
	github.com/zapmarket/zapmarket/pkg/metrics v0.0.0
	github.com/zapmarket/zapmarket/pkg/migrate v0.0.0
	github.com/zapmarket/zapmarket/pkg/proto v0.0.0
	github.com/zapmarket/zapmarket/pkg/relay v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.81.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/golang-migrate/migrate/v4 v4.18.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.11 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/prometheus/client_golang v1.22.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/segmentio/kafka-go v0.4.47 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/zapmarket/zapmarket/pkg/config => ../../pkg/config
	github.com/zapmarket/zapmarket/pkg/database => ../../pkg/database
	github.com/zapmarket/zapmarket/pkg/errors => ../../pkg/errors
	github.com/zapmarket/zapmarket/pkg/grpcx => ../../pkg/grpcx
	github.com/zapmarket/zapmarket/pkg/httpx => ../../pkg/httpx
	github.com/zapmarket/zapmarket/pkg/kafka => ../../pkg/kafka
	github.com/zapmarket/zapmarket/pkg/logger => ../../pkg/logger
	github.com/zapmarket/zapmarket/pkg/metrics => ../../pkg/metrics
	github.com/zapmarket/zapmarket/pkg/migrate => ../../pkg/migrate
	github.com/zapmarket/zapmarket/pkg/proto => ../../pkg/proto
	github.com/zapmarket/zapmarket/pkg/registry => ../../pkg/registry
	github.com/zapmarket/zapmarket/pkg/relay => ../../pkg/relay
)
