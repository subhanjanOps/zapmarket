module github.com/zapmarket/zapmarket/services/promotions-service

go 1.25.0

replace (
	github.com/zapmarket/zapmarket/pkg/config => ../../pkg/config
	github.com/zapmarket/zapmarket/pkg/database => ../../pkg/database
	github.com/zapmarket/zapmarket/pkg/logger => ../../pkg/logger
	github.com/zapmarket/zapmarket/pkg/migrate => ../../pkg/migrate
)
