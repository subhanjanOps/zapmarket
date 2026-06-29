module github.com/zapmarket/zapmarket/services/logistics-service

go 1.25.0

require github.com/google/uuid v1.6.0

replace (
	github.com/zapmarket/zapmarket/pkg/config => ../../pkg/config
	github.com/zapmarket/zapmarket/pkg/database => ../../pkg/database
	github.com/zapmarket/zapmarket/pkg/kafka => ../../pkg/kafka
	github.com/zapmarket/zapmarket/pkg/logger => ../../pkg/logger
	github.com/zapmarket/zapmarket/pkg/migrate => ../../pkg/migrate
)
