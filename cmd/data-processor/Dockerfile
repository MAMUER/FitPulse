# syntax=docker/dockerfile:1

FROM golang:1.27.2@sha256:5bc7f572bbaa98885a3a1fd9c0aa76b59e3e14e8628bfc316bbfd0c701e4818c AS builder

ENV GOPROXY=https://proxy.golang.org,direct

RUN apk add --no-cache ca-certificates=20260611-r0 git=2.54.0-r0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/data-processor/ ./cmd/data-processor/
COPY internal/ ./internal/
COPY api/ ./api/

RUN CGO_ENABLED=0 GOOS=linux go build \
	-ldflags="-w -s -buildid=" \
	-o /app/bin/data-processor \
	./cmd/data-processor

FROM alpine@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

ARG APP_UID=1001
ARG APP_GID=1001

# hadolint ignore=DL3018
RUN apk upgrade --no-cache && \
	apk add --no-cache ca-certificates curl tzdata && \
	addgroup -g "${APP_GID}" -S appgroup && \
	adduser -u "${APP_UID}" -S appuser -G appgroup

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/bin/data-processor .

USER ${APP_UID}

EXPOSE 8084 9092

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
	CMD ["curl", "-f", "http://localhost:8084/health"]

ENTRYPOINT ["./data-processor"]
