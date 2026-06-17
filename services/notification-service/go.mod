module github.com/zapmarket/zapmarket/services/notification-service

go 1.25.0

require (
	github.com/joho/godotenv v1.5.1
	github.com/redis/go-redis/v9 v9.7.3
	github.com/zapmarket/zapmarket/pkg/config v0.0.0
	github.com/zapmarket/zapmarket/pkg/kafka v0.0.0
	github.com/zapmarket/zapmarket/pkg/logger v0.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/segmentio/kafka-go v0.4.47 // indirect
	golang.org/x/text v0.37.0 // indirect
)

replace (
	github.com/zapmarket/zapmarket/pkg/config => ../../pkg/config
	github.com/zapmarket/zapmarket/pkg/kafka => ../../pkg/kafka
	github.com/zapmarket/zapmarket/pkg/logger => ../../pkg/logger
)
