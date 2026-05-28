# Backend Dockerfile — multi-stage build for minimal production image
# Stage 1: Build
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Download dependencies first (layer cache)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags="-w -s" -o dmtool ./cmd/api

# Stage 2: Minimal runtime image
FROM alpine:3.19

WORKDIR /app

# Install CA certs and curl (for health check)
RUN apk add --no-cache ca-certificates curl

COPY --from=builder /app/dmtool .

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

CMD ["./dmtool"]
