FROM golang:1.25-alpine AS builder

WORKDIR /workspace

ENV GOTOOLCHAIN=local
ENV GOWORK=off

RUN apk add --no-cache git

COPY pkg/ ./pkg/
COPY services/wishlist-service/ ./services/wishlist-service/

WORKDIR /workspace/services/wishlist-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/wishlist-service ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/wishlist-service /app/wishlist-service
COPY services/wishlist-service/migrations /app/migrations

EXPOSE 8089

HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8089/health || exit 1

CMD ["/app/wishlist-service"]
