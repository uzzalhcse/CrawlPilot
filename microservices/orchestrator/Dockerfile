# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY orchestrator/go.mod orchestrator/go.sum ./orchestrator/
COPY shared/go.mod shared/go.sum ./shared/

# Download dependencies
WORKDIR /app/orchestrator
RUN go mod download

# Copy source code
WORKDIR /app
COPY orchestrator/ ./orchestrator/
COPY shared/ ./shared/

# Build
WORKDIR /app/orchestrator
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o orchestrator ./cmd/orchestrator

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary and config
COPY --from=builder /app/orchestrator/orchestrator .
COPY orchestrator/config.yaml .

# Expose port
EXPOSE 8080

# Run
CMD ["./orchestrator"]
