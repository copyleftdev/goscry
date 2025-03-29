FROM golang:1.24.1-alpine AS builder

# Install git, dependencies, and Chrome build dependencies
RUN apk add --no-cache git ca-certificates tzdata && \
    apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/goscry \
    -ldflags="-w -s -X main.Version=$(git describe --tags --always --dirty || echo 'dev')" \
    ./cmd/goscry/

# Use a Chrome-enabled base image for the final stage
FROM zenika/alpine-chrome:latest

# Switch to root for installation steps
USER root

# Install required packages
RUN apk add --no-cache ca-certificates tzdata netcat-openbsd

# Add non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy binary from builder stage
COPY --from=builder /go/bin/goscry /usr/local/bin/goscry

# Copy config template and create config directory
COPY config.example.yaml /etc/goscry/config.yaml
RUN mkdir -p /var/lib/goscry && \
    chown -R appuser:appgroup /var/lib/goscry /etc/goscry

# Copy entrypoint script
COPY docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# Set Chrome executable path in the environment
ENV GOSCRY_BROWSER_EXECUTABLEPATH=/usr/bin/chromium-browser
ENV GOSCRY_BROWSER_HEADLESS=true

# Use non-root user for better security
USER appuser

# Create a volume for persistent data
VOLUME /var/lib/goscry

# Expose API port
EXPOSE 8080

# Set the entrypoint
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]

# Default command
CMD ["goscry", "-config", "/etc/goscry/config.yaml"]

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1
