FROM golang@sha256:1b0e4d34c19d4749a3b979149e8044917b4b0e57ee4e5c59901fd8b2356a0b7f AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/device-aggregator/ ./cmd/device-aggregator/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/device-aggregator ./cmd/device-aggregator

FROM alpine@sha256:24aa3c2e3b6f4425d2b3c1e3289d2d8f3f3b2e1d5c6a7b8c9d0e1f2a3b4c5d6e

# Update libssl3 to fix CVE-2026-45447 (use-after-free in PKCS#7 verification)
# Fixed in libssl3 3.5.7-r0; alpine:3.24 ships 3.5.7-r0 by default
# hadolint ignore=DL3018
RUN apk upgrade --no-cache && \
    apk --no-cache add ca-certificates tzdata && \
    addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /app/bin/device-aggregator .

USER 1001
EXPOSE 8083
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["sh", "-c", "cat > /dev/tcp/localhost/8083"]
ENTRYPOINT ["./device-aggregator"]