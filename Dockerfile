# Multi-stage build for Homebrew Go
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    git \
    curl \
    bash \
    make \
    gcc \
    musl-dev \
    && rm -rf /var/cache/apk/*

# Create brew user
RUN addgroup -g 1000 brew && \
    adduser -D -s /bin/bash -u 1000 -G brew brew

# Create necessary directories
RUN mkdir -p /opt/homebrew/{bin,Cellar,Caskroom,Library/Taps,var/homebrew/locks} && \
    chown -R brew:brew /opt/homebrew

# Copy binary from builder stage
COPY --from=builder /app/build/brew /opt/homebrew/bin/brew

# Create symlink for easy access
RUN ln -s /opt/homebrew/bin/brew /usr/local/bin/brew

# Set environment variables
ENV HOMEBREW_PREFIX=/opt/homebrew
ENV HOMEBREW_REPOSITORY=/opt/homebrew
ENV HOMEBREW_CELLAR=/opt/homebrew/Cellar
ENV HOMEBREW_CASKROOM=/opt/homebrew/Caskroom
ENV PATH="/opt/homebrew/bin:/opt/homebrew/sbin:$PATH"

# Switch to brew user
USER brew
WORKDIR /home/brew

# Set default command
ENTRYPOINT ["/opt/homebrew/bin/brew"]
CMD ["--help"]

# Labels
LABEL maintainer="Homebrew Go Team"
LABEL description="Homebrew package manager written in Go"
LABEL version="3.0.0"