# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY worker/go.mod worker/go.sum ./worker/
COPY shared/go.mod shared/go.sum ./shared/

# Download dependencies
WORKDIR /app/worker
RUN go mod download

# Copy source code
WORKDIR /app
COPY worker/ ./worker/
COPY shared/ ./shared/

# Build
WORKDIR /app/worker
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o worker ./cmd/worker

# Runtime stage
FROM mcr.microsoft.com/playwright:v1.40.0-jammy

WORKDIR /root/

# Install Playwright browsers
RUN npx playwright install chromium

# Copy binary and config
COPY --from=builder /app/worker/worker .
COPY worker/config.yaml .

# Expose port
EXPOSE 8080

# Run
CMD ["./worker"]
