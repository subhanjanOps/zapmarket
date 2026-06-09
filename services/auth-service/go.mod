module github.com/zapmarket/zapmarket/services/auth-service

go 1.25.0

require (
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.12.3
	github.com/swaggo/swag v1.16.6
	github.com/zapmarket/zapmarket/pkg/config v0.0.0
	golang.org/x/crypto v0.52.0
	golang.org/x/oauth2 v0.36.0
	google.golang.org/grpc v1.81.1
	google.golang.org/protobuf v1.36.11
)

require (
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/go-openapi/jsonpointer v0.23.1 // indirect
	github.com/go-openapi/jsonreference v0.21.6 // indirect
	github.com/go-openapi/spec v0.22.5 // indirect
	github.com/go-openapi/swag v0.26.0 // indirect
	github.com/go-swagger/go-swagger v0.34.1 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/swaggo/files v0.0.0-20220610200504-28940afbdbfe // indirect
	github.com/swaggo/http-swagger v1.3.4 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace github.com/zapmarket/zapmarket/pkg/config => ../../pkg/config
