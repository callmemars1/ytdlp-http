# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# Runtime stage
FROM alpine:3.20

# Install yt-dlp, ffmpeg, and other dependencies
RUN apk add --no-cache \
    yt-dlp \
    ffmpeg \
    python3 \
    py3-pip \
    ca-certificates \
    tzdata

# Update yt-dlp to latest version
RUN yt-dlp -U

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Create directories with proper permissions
RUN mkdir -p /app/downloads /tmp/ytdlp-downloads && \
    chown -R appuser:appgroup /app /tmp/ytdlp-downloads

# Switch to non-root user
USER appuser

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder --chown=appuser:appgroup /app/main .

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Set environment variables
ENV GIN_MODE=release
ENV TZ=UTC

# Run the application
CMD ["./main"]