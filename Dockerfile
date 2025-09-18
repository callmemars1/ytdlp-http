# Build stage
FROM ubuntu:22.04 AS builder

WORKDIR /app

# Install Go
RUN apt-get update && apt-get install -y \
    wget \
    tar \
    && rm -rf /var/lib/apt/lists/*

RUN wget -O go.tar.gz https://golang.org/dl/go1.25.0.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go.tar.gz \
    && rm go.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# Runtime stage
FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    ffmpeg \
    curl \
    ca-certificates \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Install latest yt-dlp
RUN pip3 install -U yt-dlp

# Create non-root user
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -s /bin/bash -m appuser

# Create directories with proper permissions
RUN mkdir -p /app/downloads /tmp/ytdlp-downloads && \
    chown -R appuser:appgroup /app /tmp/ytdlp-downloads

# Switch to non-root user
USER appuser

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder --chown=appuser:appgroup /app/main .

# Set environment variables
ENV GIN_MODE=release
ENV TZ=UTC

# Run the application
CMD ["./main"]