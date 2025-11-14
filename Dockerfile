# Multi-stage Dockerfile for SECA-CLI
# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Copy vendored dependencies required by replace directives
COPY third_party ./third_party

RUN go mod download

# Copy source code
COPY . .

# Build the application
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X github.com/khanhnv2901/seca-cli/cmd.Version=${VERSION} \
              -X github.com/khanhnv2901/seca-cli/cmd.GitCommit=${GIT_COMMIT} \
              -X github.com/khanhnv2901/seca-cli/cmd.BuildDate=${BUILD_DATE}" \
    -o seca main.go

# Stage 2: Runtime
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1000 seca && \
    adduser -D -u 1000 -G seca seca

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/seca /app/seca

# Copy compliance frameworks
COPY --from=builder /build/internal/compliance /app/internal/compliance

# Create data directory
RUN mkdir -p /app/data && chown -R seca:seca /app

# Switch to non-root user
USER seca

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/v1/health || exit 1

# Default command
ENTRYPOINT ["/app/seca"]
CMD ["serve", "--addr", "0.0.0.0:8080"]
