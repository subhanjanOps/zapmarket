# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

WORKDIR /workspace

ENV GOTOOLCHAIN=local
ENV GOWORK=off

RUN apk add --no-cache git

COPY pkg/ ./pkg/
COPY services/product-catalog-service/go.mod services/product-catalog-service/go.sum ./services/product-catalog-service/

WORKDIR /workspace/services/product-catalog-service
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /workspace
COPY services/product-catalog-service/ ./services/product-catalog-service/

WORKDIR /workspace/services/product-catalog-service
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /app/product-catalog-service .


FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/product-catalog-service /app/product-catalog-service
COPY services/product-catalog-service/migrations /app/migrations

EXPOSE 8081 50052

HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- "http://localhost:8081/api/v1/categories?limit=1" || exit 1

CMD ["/app/product-catalog-service"]
