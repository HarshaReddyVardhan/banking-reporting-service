# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
# COPY go.mod go.sum ./
# RUN go mod download
# For now, since go.sum might not exist yet, we just copy go.mod and run tidy later
COPY go.mod ./
# RUN go mod download 

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/server \
    ./cmd/api

# Runtime stage
FROM gcr.io/distroless/static:nonroot

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=builder /app/server /server

# Copy migrations
COPY --from=builder /app/migrations /migrations

# Use non-root user
USER nonroot:nonroot

# Expose ports
EXPOSE 8080

ENTRYPOINT ["/server"]
