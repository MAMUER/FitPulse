# syntax=docker/dockerfile:1
# ========== Builder stage ==========
FROM node@sha256:ef24c5053d50fdc3e4e56eb4e7ddb7861874ab0fdc797046ba897581deb8e868 AS builder
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci --ignore-scripts --no-audit --no-fund
COPY . .
RUN npm run build

# ========== Runtime stage ==========
FROM nginx:1.27-alpine AS runtime
# hadolint ignore=DL3009
RUN apk upgrade --no-cache && \
  addgroup -g 1001 -S appgroup && \
  adduser -u 1001 -S appuser -G appgroup -s /sbin/nologin
WORKDIR /usr/share/nginx/html
COPY --from=builder --chown=appuser:appgroup /app/dist .
USER 1001
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["sh", "-c", "cat > /dev/tcp/localhost/3000"]
ENTRYPOINT ["nginx", "-g", "daemon off;"]
