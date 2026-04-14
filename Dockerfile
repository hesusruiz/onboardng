# -------------------------------------------------------------------------
# STAGE 1: the Alpine sqlite3_rsync
# -------------------------------------------------------------------------
FROM hesusruiz/sqlite3-rsync-alpine:3.51.2 AS sqlite3_rsync

# Stage 2: Build the OnboardNG binary
FROM golang:1.25.7-alpine AS onboardngbuilder

# Install build tools for CGO and sqlite tools
RUN apk update && \
    apk add --no-cache \
    build-base \
    musl-dev \
    linux-headers \
    git \
    gcc \
    sqlite-tools && \
    rm -rf /var/cache/apk/*

WORKDIR /app

# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary with CGO enabled
# -ldflags="-w -s" strips debug information and symbols, reducing the binary size
RUN go build -ldflags="-w -s" -o /onboardng .

# Final stage
FROM alpine/curl:latest

WORKDIR /
COPY --from=onboardngbuilder /onboardng /onboardng
COPY src /src
COPY docs /docs
COPY dist/browser /dist/browser
COPY --from=sqlite3_rsync --chmod=755 /usr/local/bin/sqlite3_rsync /usr/local/bin/sqlite3_rsync

HEALTHCHECK \
    --interval=60s \
    --timeout=5s \
    --start-period=10s \
    --start-interval=3s \
    --retries=3 \
    CMD curl -f http://localhost:7777/health || exit 1

# Expose the port the server runs on
EXPOSE 7777

# Run the binary
ENTRYPOINT ["/onboardng"]