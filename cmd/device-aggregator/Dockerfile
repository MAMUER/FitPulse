FROM golang:1.27-alpine AS builder

ENV GOPROXY=https://proxy.golang.org,direct
ENV GOTOOLCHAIN=local

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/device-aggregator/ ./cmd/device-aggregator/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s -buildid=" -o /app/bin/device-aggregator ./cmd/device-aggregator

FROM alpine:3.24

ARG APP_UID=1001
ARG APP_GID=1001

# hadolint ignore=DL3018
RUN apk add --no-cache ca-certificates tzdata && \
  addgroup -g "${APP_GID}" -S appgroup && \
  adduser -u "${APP_UID}" -S appuser -G appgroup

WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /app/bin/device-aggregator .

USER ${APP_UID}
EXPOSE 8083
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["sh", "-c", "cat > /dev/tcp/localhost/8083"]
ENTRYPOINT ["./device-aggregator"]