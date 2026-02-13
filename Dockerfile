# Stage 1: Build Go binary
FROM golang:1.26-alpine AS go-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with version info
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X github.com/eenemeene/kitamanager-go/internal/version.GitCommit=${GIT_COMMIT} -X github.com/eenemeene/kitamanager-go/internal/version.BuildTime=${BUILD_TIME}" \
    -o main ./cmd/api

# Stage 2: Final minimal image
FROM alpine:3.23

WORKDIR /app

# Install ca-certificates for HTTPS and create non-root user
RUN apk --no-cache add ca-certificates && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy binary from builder
COPY --from=go-builder /app/main .

# Copy config files
COPY --from=go-builder /app/configs ./configs

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 -O /dev/null http://localhost:8080/api/v1/live || exit 1

# Run the application
CMD ["./main"]
