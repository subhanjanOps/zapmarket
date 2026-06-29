FROM golang:1.25-alpine AS builder

WORKDIR /workspace

ENV GOTOOLCHAIN=local
ENV GOWORK=off

RUN apk add --no-cache git

COPY pkg/ ./pkg/
COPY services/promotions-service/ ./services/promotions-service/

WORKDIR /workspace/services/promotions-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/promotions-service ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/promotions-service /app/promotions-service
COPY services/promotions-service/migrations /app/migrations

EXPOSE 8091

HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8091/health || exit 1

CMD ["/app/promotions-service"]
