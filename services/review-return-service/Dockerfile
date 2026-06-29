FROM golang:1.25-alpine AS builder

WORKDIR /workspace

ENV GOTOOLCHAIN=local
ENV GOWORK=off

RUN apk add --no-cache git

COPY pkg/ ./pkg/
COPY services/review-return-service/ ./services/review-return-service/

WORKDIR /workspace/services/review-return-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/review-return-service ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/review-return-service /app/review-return-service
COPY services/review-return-service/migrations /app/migrations

EXPOSE 8090

HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8090/health || exit 1

CMD ["/app/review-return-service"]
