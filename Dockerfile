# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o queue-server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/queue-server .
COPY --from=builder /app/worker .

# Create data directory for WAL
RUN mkdir -p /app/data

EXPOSE 8080

CMD ["./queue-server"]
