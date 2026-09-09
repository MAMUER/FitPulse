# syntax=docker/dockerfile:1

# ========== Builder stage ==========
FROM golang@sha256:1b0e4d34c19d4749a3b979149e8044917b4b0e57ee4e5c59901fd8b2356a0b7f AS builder

# Set Go environment for module downloading
ENV GOPROXY=https://proxy.golang.org,direct

RUN apk add --no-cache ca-certificates=20260611-r0 git=2.54.0-r0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/training-service/ ./cmd/training-service/
COPY internal/ ./internal/
COPY api/ ./api/

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/bin/training-service \
    ./cmd/training-service

# ========== Runtime stage ==========
FROM alpine@sha256:24aa3c2e3b6f4425d2b3c1e3289d2d8f3f3b2e1d5c6a7b8c9d0e1f2a3b4c5d6e

# Update libssl3 to fix CVE-2026-45447 (use-after-free in PKCS#7 verification)
# Fixed in libssl3 3.5.7-r0; alpine:3.24 ships 3.5.7-r0 by default
# hadolint ignore=DL3018
RUN apk upgrade --no-cache && \
    apk add --no-cache ca-certificates tzdata && \
    addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/bin/training-service .

USER 1001

EXPOSE 50053

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["sh", "-c", "cat > /dev/tcp/localhost/50053"]

ENTRYPOINT ["./training-service"]
