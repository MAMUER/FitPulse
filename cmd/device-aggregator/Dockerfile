FROM golang@sha256:f86f1a6701e3dcc445fec097a42f78b758f15950ccf032c2d3e54e2754d32fdb AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/device-aggregator/ ./cmd/device-aggregator/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s -buildid=" -o /app/bin/device-aggregator ./cmd/device-aggregator

FROM alpine@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

ARG APP_UID=1001
ARG APP_GID=1001

# Update libssl3 to fix CVE-2026-45447 (use-after-free in PKCS#7 verification)
# Fixed in libssl3 3.5.7-r0; alpine:3.24 ships 3.5.7-r0 by default
# hadolint ignore=DL3018
RUN apk upgrade --no-cache && \
  apk --no-cache add ca-certificates tzdata && \
  addgroup -g "${APP_GID}" -S appgroup && \
  adduser -u "${APP_UID}" -S appuser -G appgroup

WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /app/bin/device-aggregator .

USER ${APP_UID}
EXPOSE 8083
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["sh", "-c", "cat > /dev/tcp/localhost/8083"]
ENTRYPOINT ["./device-aggregator"]