# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

WORKDIR /workspace

ENV GOTOOLCHAIN=local
ENV GOWORK=off

RUN apk add --no-cache git

COPY pkg/ ./pkg/
COPY services/notification-service/go.mod services/notification-service/go.sum ./services/notification-service/

WORKDIR /workspace/services/notification-service
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /workspace
COPY services/notification-service/ ./services/notification-service/

WORKDIR /workspace/services/notification-service
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /app/notification-service .


FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/notification-service /app/notification-service

EXPOSE 8085

HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8085/health || exit 1

CMD ["/app/notification-service"]
