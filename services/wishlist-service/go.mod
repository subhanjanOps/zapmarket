module github.com/zapmarket/zapmarket/services/wishlist-service

go 1.25.0

require (
	github.com/go-chi/chi/v5 v5.3.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.12.3
	github.com/zapmarket/zapmarket/pkg/config v0.0.0
	github.com/zapmarket/zapmarket/pkg/crypto v0.0.0
	github.com/zapmarket/zapmarket/pkg/database v0.0.0
	github.com/zapmarket/zapmarket/pkg/logger v0.0.0
	github.com/zapmarket/zapmarket/pkg/migrate v0.0.0
)

require (
	github.com/golang-migrate/migrate/v4 v4.18.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/crypto v0.52.0 // indirect
)

replace (
	github.com/zapmarket/zapmarket/pkg/config => ../../pkg/config
	github.com/zapmarket/zapmarket/pkg/crypto => ../../pkg/crypto
	github.com/zapmarket/zapmarket/pkg/database => ../../pkg/database
	github.com/zapmarket/zapmarket/pkg/logger => ../../pkg/logger
	github.com/zapmarket/zapmarket/pkg/migrate => ../../pkg/migrate
)
