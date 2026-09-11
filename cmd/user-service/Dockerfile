# syntax=docker/dockerfile:1

# ========== Builder stage ==========
FROM golang@sha256:ce864e7223ac17b1775e6fd0b4c0db580c2eb50e7953a427916379e4b92a1628 AS builder

# Set Go environment for module downloading
ENV GOPROXY=https://proxy.golang.org,direct
RUN apk add --no-cache ca-certificates=20260611-r0 git=2.54.0-r0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/user-service/ ./cmd/user-service/
COPY internal/ ./internal/
COPY api/ ./api/

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/bin/user-service \
    ./cmd/user-service

# ========== Runtime stage ==========
FROM alpine@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

# Update libssl3 to fix CVE-2026-45447 (use-after-free in PKCS#7 verification)
# Fixed in libssl3 3.5.7-r0; alpine:3.24 ships 3.5.7-r0 by default
# hadolint ignore=DL3018
RUN apk upgrade --no-cache && \
    apk add --no-cache ca-certificates tzdata && \
    addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/bin/user-service .

USER 1001

HEALTHCHECK --interval=30s --timeout=3s CMD ps | grep -E 'user-service' | grep -v grep || exit 1

EXPOSE 50051

ENTRYPOINT ["./user-service"]
