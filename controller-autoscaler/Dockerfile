# Stage 1: build binary
FROM golang:1.25 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o  nginx-controller-autoscaler main.go

# Stage 2: run minimal image
FROM alpine:3.19
WORKDIR /root/
COPY --from=builder /app/nginx-controller-autoscaler .
USER 65532:65532  # non-root user

ENTRYPOINT ["./nginx-controller-autoscaler"]
