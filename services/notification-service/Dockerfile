FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum ./
COPY pkg/ ./pkg/
COPY services/notification-service/ ./services/notification-service/
RUN cd services/notification-service && go build -o /notification-service .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /notification-service /notification-service
ENTRYPOINT ["/notification-service"]
