# syntax=docker/dockerfile:1

# ========== Builder stage ==========
FROM golang:1.26-alpine AS builder

# Set Go environment for module downloading
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=off

WORKDIR /app

RUN apk add --no-cache git=2.54.0-r0 ca-certificates=20260611-r0

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/biometric-service/ ./cmd/biometric-service/
COPY internal/ ./internal/
COPY api/ ./api/

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/bin/biometric-service \
    ./cmd/biometric-service

# ========== Runtime stage ==========
FROM alpine:3.24

# Update libssl3 to fix CVE-2026-45447 (use-after-free in PKCS#7 verification)
# Fixed in libssl3 3.5.7-r0; alpine:3.23 ships 3.5.6-r0 by default
# hadolint ignore=DL3018
RUN apk upgrade --no-cache && \
    apk add --no-cache ca-certificates tzdata && \
    addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/bin/biometric-service .

USER appuser

EXPOSE 50052

ENTRYPOINT ["./biometric-service"]
