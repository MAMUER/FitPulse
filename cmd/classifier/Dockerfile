# syntax=docker/dockerfile:1

FROM golang@sha256:1b0e4d34c19d4749a3b979149e8044917b4b0e57ee4e5c59901fd8b2356a0b7f AS builder

ENV GOPROXY=https://proxy.golang.org,direct

RUN apk add --no-cache ca-certificates=20260611-r0 git=2.54.0-r0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/classifier/ ./cmd/classifier/
COPY internal/ ./internal/
COPY api/ ./api/

RUN CGO_ENABLED=0 GOOS=linux go build \
	-ldflags="-w -s" \
	-o /app/bin/classifier \
	./cmd/classifier

FROM alpine@sha256:24aa3c2e3b6f4425d2b3c1e3289d2d8f3f3b2e1d5c6a7b8c9d0e1f2a3b4c5d6e

# hadolint ignore=DL3018
RUN apk upgrade --no-cache && \
	apk add --no-cache ca-certificates curl tzdata && \
	addgroup -g 1001 -S appgroup && \
	adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/bin/classifier .

USER 1001

EXPOSE 8001 9091

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["curl", "-f", "http://localhost:8001/health"]

ENTRYPOINT ["./classifier"]