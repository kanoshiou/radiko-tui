# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the server binary (noaudio mode - no CGO needed)
RUN CGO_ENABLED=0 GOOS=linux go build -tags noaudio -ldflags="-s -w" -o radiko-server .

# Runtime stage
FROM alpine:latest

# Install ffmpeg for audio decoding
RUN apk add --no-cache ffmpeg ca-certificates tzdata

# Create non-root user for security
RUN adduser -D -u 1000 radiko
USER radiko

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/radiko-server .

# Default port
EXPOSE 8080

# Default environment variables
ENV PORT=8080
ENV GRACE_SECONDS=30

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT}/api/status || exit 1

# Run server
ENTRYPOINT ["./radiko-server"]
CMD ["-server", "-port", "8080", "-grace", "30"]
