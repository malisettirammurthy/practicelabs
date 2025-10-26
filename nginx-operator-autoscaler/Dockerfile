# ── Build stage
FROM golang:1.25 AS builder
WORKDIR /workspace

# Copy module files first and pre-download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy rest of the code
COPY . .

# Force module verification
RUN go mod verify

# Build the manager binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o manager ./cmd/manager

# ── Runtime stage
FROM alpine:3.19
WORKDIR /root/
COPY --from=builder /workspace/manager .
USER 65532:65532
ENTRYPOINT ["./manager"]

