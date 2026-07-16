# syntax=docker/dockerfile:1
# ========== Builder stage ==========
FROM golang:1.26-alpine AS builder
# Set Go environment for module downloading
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=off
RUN apk add --no-cache git=2.54.0-r0 ca-certificates=20260611-r0
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
# Web files — copy early so changes always trigger rebuild
COPY web/ ./web/
COPY cmd/gateway/ ./cmd/gateway/
COPY internal/ ./internal/
COPY api/ ./api/
RUN CGO_ENABLED=0 GOOS=linux go build \
-ldflags="-w -s" \
-o /app/bin/gateway \
./cmd/gateway
# ========== Runtime stage ==========
FROM alpine:3.24
# hadolint ignore=DL3018
RUN apk upgrade --no-cache && \
    apk add --no-cache ca-certificates tzdata curl=8.20.0-r1
RUN addgroup -g 1001 -S appgroup && \
adduser -u 1001 -S appuser -G appgroup
WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /app/bin/gateway .
COPY --from=builder --chown=appuser:appgroup /app/web ./web
USER appuser
EXPOSE 8080 8443
ENTRYPOINT ["./gateway"]