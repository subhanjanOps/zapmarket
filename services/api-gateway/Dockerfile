FROM golang:1.25-alpine AS builder

WORKDIR /workspace

ENV GOTOOLCHAIN=local
ENV GOWORK=off

RUN apk add --no-cache git

COPY pkg/ ./pkg/
COPY services/api-gateway/ ./services/api-gateway/

WORKDIR /workspace/services/api-gateway
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/api-gateway .


FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/api-gateway /app/api-gateway

EXPOSE 8000

HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8000/health || exit 1

CMD ["/app/api-gateway"]
