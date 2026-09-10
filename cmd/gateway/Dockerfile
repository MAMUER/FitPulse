# syntax=docker/dockerfile:1
# ========== Web builder stage ==========
FROM node@sha256:ef24c5053d50fdc3e4e56eb4e7ddb7861874ab0fdc797046ba897581deb8e868 AS web-builder
WORKDIR /app/web
RUN npm install --ignore-scripts -g npm@12.0.2 && npm --version
COPY web/package.json web/package-lock.json* ./
RUN npm ci --ignore-scripts --no-audit --no-fund
COPY web/ .
RUN npm run build

# ========== Go builder stage ==========
FROM golang@sha256:ce864e7223ac17b1775e6fd0b4c0db580c2eb50e7953a427916379e4b92a1628 AS builder
# Set Go environment for module downloading
ENV GOPROXY=https://proxy.golang.org,direct
RUN apk add --no-cache ca-certificates=20260611-r0 git=2.54.0-r0
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
FROM alpine@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b
# hadolint ignore=DL3018
RUN apk upgrade --no-cache && \
    apk add --no-cache ca-certificates curl tzdata && \
    addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup
WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /app/bin/gateway .
COPY --from=web-builder --chown=appuser:appgroup /app/web/dist ./web/dist
USER 1001
EXPOSE 8080 8443
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["curl", "-f", "http://localhost:8080/health"]
ENTRYPOINT ["./gateway"]