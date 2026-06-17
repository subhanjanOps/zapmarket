module github.com/zapmarket/zapmarket/services/product-catalog-service

go 1.25.0

require (
	github.com/go-chi/chi/v5 v5.3.0
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.12.3
	github.com/redis/go-redis/v9 v9.7.3
	github.com/swaggo/http-swagger/v2 v2.0.2
	github.com/swaggo/swag v1.16.6
	github.com/zapmarket/zapmarket/pkg/config v0.0.0
	github.com/zapmarket/zapmarket/pkg/database v0.0.0
	github.com/zapmarket/zapmarket/pkg/errors v0.0.0
	github.com/zapmarket/zapmarket/pkg/grpcx v0.0.0
	github.com/zapmarket/zapmarket/pkg/httpx v0.0.0
	github.com/zapmarket/zapmarket/pkg/logger v0.0.0
	github.com/zapmarket/zapmarket/pkg/migrate v0.0.0
	github.com/zapmarket/zapmarket/pkg/proto v0.0.0
	github.com/zapmarket/zapmarket/pkg/storage v0.0.0
	github.com/zapmarket/zapmarket/pkg/swaggerx v0.0.0
	google.golang.org/grpc v1.81.1
)

require (
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-openapi/jsonpointer v0.23.1 // indirect
	github.com/go-openapi/jsonreference v0.21.6 // indirect
	github.com/go-openapi/spec v0.22.5 // indirect
	github.com/go-openapi/swag/conv v0.26.0 // indirect
	github.com/go-openapi/swag/jsonname v0.26.0 // indirect
	github.com/go-openapi/swag/jsonutils v0.26.0 // indirect
	github.com/go-openapi/swag/loading v0.26.0 // indirect
	github.com/go-openapi/swag/stringutils v0.26.0 // indirect
	github.com/go-openapi/swag/typeutils v0.26.0 // indirect
	github.com/go-openapi/swag/yamlutils v0.26.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/golang-migrate/migrate/v4 v4.18.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.11 // indirect
	github.com/minio/crc64nvme v1.0.2 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/minio-go/v7 v7.0.95 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/swaggo/files/v2 v2.0.0 // indirect
	github.com/tinylib/msgp v1.3.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.52.0 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/zapmarket/zapmarket/pkg/config => ../../pkg/config
	github.com/zapmarket/zapmarket/pkg/database => ../../pkg/database
	github.com/zapmarket/zapmarket/pkg/errors => ../../pkg/errors
	github.com/zapmarket/zapmarket/pkg/grpcx => ../../pkg/grpcx
	github.com/zapmarket/zapmarket/pkg/httpx => ../../pkg/httpx
	github.com/zapmarket/zapmarket/pkg/logger => ../../pkg/logger
	github.com/zapmarket/zapmarket/pkg/migrate => ../../pkg/migrate
	github.com/zapmarket/zapmarket/pkg/proto => ../../pkg/proto
	github.com/zapmarket/zapmarket/pkg/storage => ../../pkg/storage
	github.com/zapmarket/zapmarket/pkg/swaggerx => ../../pkg/swaggerx
)
