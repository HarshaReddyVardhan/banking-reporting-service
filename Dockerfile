# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o reporting-service ./cmd/server

# Run Stage
FROM alpine:3.18

WORKDIR /app

# Install runtime dependencies (check for vulnerability scans as per requirements)
RUN apk add --no-cache ca-certificates tzdata

# Create a non-root user
RUN adduser -D -g '' appuser
USER appuser

# Copy the binary from builder
COPY --from=builder /app/reporting-service .

# Copy config if needed (or rely on env vars/volume mount)
# COPY --from=builder /app/config ./config

# Expose port
EXPOSE 8080

CMD ["./reporting-service"]
