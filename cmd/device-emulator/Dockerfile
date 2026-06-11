# syntax=docker/dockerfile:1

# ========== Builder stage ==========
FROM golang:1.26-alpine AS builder

# Set Go environment for module downloading
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=off

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/device-emulator/ ./cmd/device-emulator/
COPY internal/wearableemulator/ ./internal/wearableemulator/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/bin/device-emulator \
    ./cmd/device-emulator

# ========== Runtime stage ==========
FROM alpine:3.23

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/bin/device-emulator .

USER appuser

# Device emulator doesn't expose ports, он подключается к device-connector
ENTRYPOINT ["./device-emulator"]
