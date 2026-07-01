# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

WORKDIR /workspace

ENV GOTOOLCHAIN=local
ENV GOWORK=off

RUN apk add --no-cache git

COPY pkg/ ./pkg/
COPY services/settlement-service/go.mod services/settlement-service/go.sum ./services/settlement-service/

WORKDIR /workspace/services/settlement-service
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /workspace
COPY services/settlement-service/ ./services/settlement-service/

WORKDIR /workspace/services/settlement-service
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /app/settlement-service ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/settlement-service /app/settlement-service
COPY services/settlement-service/migrations /app/migrations

EXPOSE 8088

HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8088/health || exit 1

CMD ["/app/settlement-service"]
