FROM golang:1.25-alpine AS builder

WORKDIR /workspace

ENV GOTOOLCHAIN=local
ENV GOWORK=off

RUN apk add --no-cache git

COPY pkg/ ./pkg/
COPY services/inventory-service/ ./services/inventory-service/

WORKDIR /workspace/services/inventory-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/inventory-service .


FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/inventory-service /app/inventory-service
COPY services/inventory-service/migrations /app/migrations

EXPOSE 8082 50053

HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8082/health || exit 1

CMD ["/app/inventory-service"]
