FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/device-aggregator/ ./cmd/device-aggregator/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/device-aggregator ./cmd/device-aggregator

FROM alpine:3.23

# Update libssl3 to fix CVE-2026-45447 (use-after-free in PKCS#7 verification)
# Fixed in libssl3 3.5.7-r0; alpine:3.23 ships 3.5.6-r0 by default
# hadolint ignore=DL3018
RUN apk upgrade --no-cache && \
    apk --no-cache add ca-certificates tzdata
RUN addgroup -g 1001 -S appgroup && adduser -u 1001 -S appuser -G appgroup

WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /app/bin/device-aggregator .

USER appuser
EXPOSE 8083
ENTRYPOINT ["./device-aggregator"]