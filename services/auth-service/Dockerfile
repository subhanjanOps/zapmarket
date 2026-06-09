# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go.work and service module
COPY go.work go.work.sum* ./
COPY services/auth-service ./services/auth-service
COPY pkg ./pkg

# Download dependencies
RUN cd services/auth-service && go mod download

# Build the binary
RUN cd services/auth-service && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o auth-service ./cmd/main.go

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS/OAuth2
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/services/auth-service/auth-service .

# Copy .env template (optional)
COPY services/auth-service/.env.example .env.example

# Expose ports
EXPOSE 8080 50051

# Health check
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the service
CMD ["./auth-service"]
