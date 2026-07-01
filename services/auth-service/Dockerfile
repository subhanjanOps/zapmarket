# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

WORKDIR /workspace

ENV GOTOOLCHAIN=local
ENV GOWORK=off

RUN apk add --no-cache git

COPY pkg/ ./pkg/
COPY services/auth-service/go.mod services/auth-service/go.sum ./services/auth-service/

WORKDIR /workspace/services/auth-service
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /workspace
COPY services/auth-service/ ./services/auth-service/

WORKDIR /workspace/services/auth-service
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /app/auth-service ./cmd/main.go


FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/auth-service /app/auth-service
COPY services/auth-service/migrations /app/migrations

EXPOSE 8080 50051

HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

CMD ["/app/auth-service"]
