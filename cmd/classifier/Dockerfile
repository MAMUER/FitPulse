# syntax=docker/dockerfile:1

FROM golang:1.27.3-alpine AS builder

ENV GOPROXY=https://proxy.golang.org,direct
ENV GOTOOLCHAIN=auto

# git and ca-certificates are build-time only and not shipped in the runtime stage
# hadolint ignore=DL3018
RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY cmd/classifier/ ./cmd/classifier/
COPY internal/ ./internal/
COPY api/ ./api/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
	-ldflags="-w -s -buildid=" \
	-o /app/bin/classifier \
	./cmd/classifier

FROM alpine:3.24

ARG APP_UID=1001
ARG APP_GID=1001

# hadolint ignore=DL3018
RUN apk upgrade --no-cache && \
	apk add --no-cache ca-certificates curl tzdata && \
	addgroup -g "${APP_GID}" -S appgroup && \
	adduser -u "${APP_UID}" -S appuser -G appgroup

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/bin/classifier .

USER ${APP_UID}

EXPOSE 8001 9091

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
	CMD ["curl", "-f", "http://localhost:8001/health"]

ENTRYPOINT ["./classifier"]