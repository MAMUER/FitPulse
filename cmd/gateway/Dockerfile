# syntax=docker/dockerfile:1
# ========== Web builder stage ==========
FROM node:26-alpine AS web-builder
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci || npm install
COPY web/ .
RUN npm run build

# ========== Go builder stage ==========
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
    apk add --no-cache ca-certificates tzdata curl
RUN addgroup -g 1001 -S appgroup && \
adduser -u 1001 -S appuser -G appgroup
WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /app/bin/gateway .
COPY --from=web-builder --chown=appuser:appgroup /app/web/dist ./web/dist
USER appuser
EXPOSE 8080 8443
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1
ENTRYPOINT ["./gateway"]