# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

WORKDIR /workspace

ENV GOTOOLCHAIN=local
ENV GOWORK=off

RUN apk add --no-cache git

COPY pkg/ ./pkg/
COPY services/inventory-service/go.mod services/inventory-service/go.sum ./services/inventory-service/

WORKDIR /workspace/services/inventory-service
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /workspace
COPY services/inventory-service/ ./services/inventory-service/

WORKDIR /workspace/services/inventory-service
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /app/inventory-service .


FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/inventory-service /app/inventory-service
COPY services/inventory-service/migrations /app/migrations

EXPOSE 8082 50053

HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8082/health || exit 1

CMD ["/app/inventory-service"]
