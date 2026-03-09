FROM golang:1.22 AS builder

WORKDIR /app
COPY main.go .
RUN go build -o notification-service main.go

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/notification-service /app/notification-service

EXPOSE 8081

CMD ["/app/notification-service"]
