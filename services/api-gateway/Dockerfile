# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

WORKDIR /workspace

ENV GOTOOLCHAIN=local
ENV GOWORK=off

RUN apk add --no-cache git

COPY pkg/ ./pkg/
COPY services/api-gateway/go.mod services/api-gateway/go.sum ./services/api-gateway/

WORKDIR /workspace/services/api-gateway
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /workspace
COPY services/api-gateway/ ./services/api-gateway/

WORKDIR /workspace/services/api-gateway
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /app/api-gateway .


FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/api-gateway /app/api-gateway
COPY --from=builder /workspace/services/api-gateway/migrations /app/migrations

EXPOSE 8000

HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8000/health || exit 1

CMD ["/app/api-gateway"]
