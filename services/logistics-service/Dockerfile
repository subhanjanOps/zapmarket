FROM golang:1.25-alpine AS builder

WORKDIR /workspace

ENV GOTOOLCHAIN=local
ENV GOWORK=off

RUN apk add --no-cache git

COPY pkg/ ./pkg/
COPY services/logistics-service/ ./services/logistics-service/

WORKDIR /workspace/services/logistics-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/logistics-service ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/logistics-service /app/logistics-service
COPY services/logistics-service/migrations /app/migrations

EXPOSE 8092

HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8092/health || exit 1

CMD ["/app/logistics-service"]
