# Auth Service

The Auth Service handles user authentication and authorization for the zapMarket ecommerce platform. It supports:

- **JWT-based authentication** — Password flow and OAuth2 (Google/Facebook)
- **Token management** — Access tokens and refresh tokens
- **User roles** — Customer, Seller, Admin with RBAC support
- **gRPC interfaces** — For internal service-to-service calls
- **REST API** — For client applications
- **Structured logging** — Request/response logging for observability
- **OpenAPI documentation** — Auto-generated API specs

---

## Architecture

```
User/Client
    ↓
REST API (HTTP)
    ↓
┌──────────────────────────────────────────┐
│  Auth Service                            │
│  ┌────────────────────────────────────┐  │
│  │ HTTP Handlers                      │  │
│  │  - Register, Login, Refresh, Me    │  │
│  │  - OAuth2 callbacks (Google/FB)    │  │
│  │  - Structured logging middleware   │  │
│  └────────────────────────────────────┘  │
│  ┌────────────────────────────────────┐  │
│  │ Auth Service (Business Logic)      │  │
│  │  - Password hashing/verification   │  │
│  │  - JWT generation/validation       │  │
│  │  - OAuth2 user linking             │  │
│  └────────────────────────────────────┘  │
│  ┌────────────────────────────────────┐  │
│  │ Repositories (Data Access)         │  │
│  │  - UserRepository                  │  │
│  │  - OAuthRepository                 │  │
│  │  - RefreshTokenRepository          │  │
│  └────────────────────────────────────┘  │
└──────────────────────────────────────────┘
    ↓
PostgreSQL (userauth database)
```

### Other Services (via gRPC)
- **API Gateway** → `ValidateToken()` to verify JWTs
- **Order Management** → `GetUser()` to fetch user details

---

## Build & Run

### Prerequisites

- Go 1.22+
- PostgreSQL 16+
- .env file with configuration (see `.env.example`)

### Local Development

1. **Setup environment**
   ```bash
   cp .env.example .env
   # Edit .env with your database credentials and JWT secrets
   ```

2. **Install dependencies**
   ```bash
   go mod download
   go mod tidy
   ```

3. **Generate proto stubs (optional for now)**
   ```bash
   # After proto compilation setup:
   protoc --go_out=. --go-grpc_out=. proto/auth.proto
   ```

4. **Run the service**
   ```bash
   go run ./cmd/main.go
   ```

   Service will start on:
   - HTTP: `http://localhost:8080`
   - gRPC: `localhost:50051` (placeholder until proto stubs generated)
   - OpenAPI/Swagger JSON: `http://localhost:8080/swagger.json`

### Docker

1. **Build image**
   ```bash
   docker build -t auth-service .
   ```

2. **Run container**
   ```bash
   docker run --env-file .env -p 8080:8080 -p 50051:50051 auth-service
   ```

---

## API Endpoints

### Public Endpoints (No Auth Required)

#### `POST /auth/register`
Register a new user with email and password.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "secure_password",
  "full_name": "John Doe"
}
```

**Response (201 Created):**
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "full_name": "John Doe",
    "role": "customer",
    "is_verified": true,
    "created_at": "2026-06-08T10:00:00Z"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "a1b2c3d4e5f6..."
}
```

#### `POST /auth/login`
Authenticate with email and password.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "secure_password"
}
```

**Response (200 OK):**
```json
{
  "user": { ... },
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "a1b2c3d4e5f6..."
}
```

#### `POST /auth/refresh`
Get a new access token using a refresh token.

**Request:**
```json
{
  "refresh_token": "a1b2c3d4e5f6..."
}
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### Protected Endpoints (Requires JWT)

#### `GET /auth/me`
Get current authenticated user profile.

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

**Response (200 OK):**
```json
{
  "user": { ... }
}
```

### OAuth2 Endpoints

#### `GET /auth/oauth/google/url`
Get Google OAuth authorization URL.

**Query Parameters:**
- `state` (optional) — CSRF protection token

**Response (200 OK):**
```json
{
  "oauth_url": "https://accounts.google.com/o/oauth2/v2/auth?client_id=..."
}
```

#### `GET /auth/oauth/google/callback?code=...&state=...`
Google OAuth callback endpoint.

**Response (200 OK):**
```json
{
  "user": { ... },
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "a1b2c3d4e5f6..."
}
```

#### `GET /auth/oauth/facebook/url`
Get Facebook OAuth authorization URL.

#### `GET /auth/oauth/facebook/callback?code=...&state=...`
Facebook OAuth callback endpoint.

### Health Check

#### `GET /health`
Service health status.

**Response (200 OK):**
```json
{
  "status": "ok"
}
```

### API Documentation

#### `GET /swagger.json`
OpenAPI 3.0 specification in JSON format.

**Response (200 OK):**
```json
{
  "openapi": "3.0.0",
  "info": {
    "title": "ZapMarket Auth Service API",
    "version": "1.0.0",
    ...
  },
  "paths": {
    "/auth/register": { ... },
    "/auth/login": { ... },
    ...
  }
}
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `zapuser` | PostgreSQL user |
| `DB_PASSWORD` | `zappass123` | PostgreSQL password |
| `DB_NAME` | `userauth` | PostgreSQL database name |
| `JWT_SECRET_KEY` | (required) | Secret key for signing access tokens |
| `JWT_REFRESH_SECRET_KEY` | (required) | Secret key for signing refresh tokens |
| `JWT_ACCESS_EXPIRY_HOURS` | `1` | Access token expiration in hours |
| `JWT_REFRESH_EXPIRY_DAYS` | `7` | Refresh token expiration in days |
| `OAUTH2_GOOGLE_CLIENT_ID` | `""` | Google OAuth client ID |
| `OAUTH2_GOOGLE_CLIENT_SECRET` | `""` | Google OAuth client secret |
| `OAUTH2_GOOGLE_REDIRECT_URL` | `http://localhost:8080/auth/oauth/google/callback` | Google redirect URI |
| `OAUTH2_FACEBOOK_CLIENT_ID` | `""` | Facebook OAuth client ID |
| `OAUTH2_FACEBOOK_CLIENT_SECRET` | `""` | Facebook OAuth client secret |
| `OAUTH2_FACEBOOK_REDIRECT_URL` | `http://localhost:8080/auth/oauth/facebook/callback` | Facebook redirect URI |
| `HTTP_PORT` | `8080` | HTTP server port |
| `GRPC_PORT` | `50051` | gRPC server port |
| `APP_ENV` | `development` | Environment: `development` or `production` |

---

## gRPC Service (Internal)

The auth-service exposes the following gRPC endpoints for internal service-to-service communication:

### `ValidateToken(token) -> User`
Called by API Gateway to validate JWT tokens and retrieve user info.

**Proto Definition:**
```proto
service AuthService {
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  // ... (see proto/auth.proto)
}
```

### Implementation Notes

**Proto Stubs:** The gRPC service is currently a placeholder. To enable full gRPC functionality:

1. Ensure `protoc` compiler is installed:
   ```bash
   brew install protobuf  # macOS
   # or
   apt install protobuf-compiler  # Linux
   ```

2. Generate proto stubs:
   ```bash
   cd services/auth-service
   protoc --go_out=. --go-grpc_out=. proto/auth.proto
   ```

3. Uncomment the gRPC server startup in `cmd/main.go`

---

## Database Schema

The service uses PostgreSQL with the `userauth` database containing:

- **users** — User accounts with email, password_hash, role, verification status
- **oauth_accounts** — Linked OAuth provider accounts
- **refresh_tokens** — Issued refresh tokens with expiration and revocation
- **addresses** — User shipping/billing addresses
- **notification_preferences** — User notification channel preferences

See [db-design.md](../../db-design.md) for full schema details.

---

## Structured Logging

The service implements structured request/response logging for both HTTP and gRPC endpoints using the standard `log/slog` library.

### HTTP Logging
All HTTP endpoints (except `/health`) log:
- **Request**: Method, path, query, user ID (if authenticated), remote address, user agent, referer
- **Response**: Method, path, status code, duration (ms), response body (truncated to 500 chars)

Example log output:
```
time="2026-06-09T10:30:45Z" level=info msg="HTTP request" method=POST path=/auth/login query= user_id=[authenticated] remote_addr=192.168.1.100:54321 user_agent=Mozilla/5.0 (...) referer=
time="2026-06-09T10:30:46Z" level=info msg="HTTP response" method=POST path=/auth/login status=200 duration_ms=125 response_body={"user":{"id":"...","email":"user@example.com",...},"access_token":"...","refresh_token":"..."}
```

### gRPC Logging
All gRPC methods log:
- **Request**: Method name at start of processing
- **Response**: Method name, status (success/error), error message (if applicable), duration (ms)

Example log output:
```
time="2026-06-09T10:30:45Z" level=info msg="gRPC request" method=RegisterUser
time="2026-06-09T10:30:46Z" level=info msg="gRPC response" method=RegisterUser status=success duration_ms=125
```

---

## Security Considerations

1. **Password Hashing** — Bcrypt with cost factor 12
2. **JWT Signing** — HS256 with strong secret keys (change in production)
3. **Token Expiration** — Access tokens expire in 1 hour; refresh tokens in 7 days
4. **Refresh Token Revocation** — Tokens can be manually invalidated
5. **OAuth2 Flow** — Server-side authorization code flow; no implicit grant
6. **HTTPS in Production** — OAuth2 redirect URIs must use HTTPS
7. **Structured Logging** — No sensitive data (tokens, passwords) logged in plain text

---

## Testing

### Manual Testing

```bash
# Health check
curl http://localhost:8080/health

# Register
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "full_name": "John Doe"
  }'

# Login
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'

# Get current user (replace TOKEN with actual access token)
curl -X GET http://localhost:8080/auth/me \
  -H "Authorization: Bearer TOKEN"

# Refresh token
curl -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "TOKEN"}'

# View Swagger/OpenAPI spec
curl http://localhost:8080/swagger.json | jq .
```

### Verifying Logging
When making requests, check the service console output for structured log entries as described in the **Structured Logging** section above.

---

## Future Enhancements (Phase 2+)

- [ ] **Email Verification** — Send verification email before allowing login
- [ ] **Password Reset** — Forgot password flow with email verification
- [ ] **Redis Integration** — Token blacklist, session store, rate limiting
- [ ] **Kafka Events** — Publish `user.registered` events
- [ ] **Two-Factor Authentication** — 2FA via TOTP or SMS
- [ ] **Admin Dashboard** — User management UI
- [ ] **Audit Logging** — Track auth events (login, password change, etc.)
- [ ] **Rate Limiting** — Prevent brute force attacks
- [ ] **mTLS for gRPC** — Secure service-to-service communication
- [ ] **Enhanced OpenAPI** — Add Swagger UI endpoint for interactive documentation

---

## Troubleshooting

### Database Connection Error
- Check PostgreSQL is running: `psql -h localhost -U zapuser -d userauth`
- Verify `.env` file has correct credentials
- Ensure `userauth` database exists

### Invalid JWT Secret
- Set strong JWT secrets in `.env`
- In production, use environment variables or secret management tool

### OAuth2 Callback Fails
- Verify `OAUTH2_*_CLIENT_ID` and `CLIENT_SECRET` are correct
- Ensure redirect URLs match OAuth provider configuration
- Check firewall allows external HTTPS connections

### gRPC Server Not Running
- Proto stubs must be generated first
- Run: `protoc --go_out=. --go-grpc_out=. proto/auth.proto`
- Uncomment gRPC startup in `cmd/main.go`

### No Structured Logs Appearing
- Verify the service was built with the logging middleware changes
- Check that you're hitting endpoints other than `/health` (which is intentionally not logged)
- Ensure the service isn't running in a mode that suppresses info-level logs

---

## References

- [Database Schema Design](../../db-design.md)
- [Architecture Documentation](../../design.md)
- [Go JWT Library](https://github.com/golang-jwt/jwt)
- [OAuth2 Specification](https://tools.ietf.org/html/rfc6749)
- [gRPC Documentation](https://grpc.io/docs/languages/go/)
- [OpenAPI/Swagger Specification](https://swagger.io/specification/)
- [Structured Logging with slog](https://golang.org/pkg/log/slog/)