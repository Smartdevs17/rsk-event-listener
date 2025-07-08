FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o bin/rsk-event-listener cmd/listener/main.go

# Final image
FROM alpine:3.18

WORKDIR /app

# Copy binary and config
COPY --from=builder /app/bin/rsk-event-listener /app/rsk-event-listener
COPY config /app/config
COPY data /app/data
COPY .env /app/.env

# Expose ports (API and metrics)
EXPOSE 8081 9090

# Run the application
ENTRYPOINT ["/app/rsk-event-listener", "--config", "/app/config/production.yaml"]