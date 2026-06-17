# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder

ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=off

RUN apk add --no-cache git=2.54.0-r0 ca-certificates=20260611-r0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/classifier/ ./cmd/classifier/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY api/ ./api/

RUN CGO_ENABLED=0 GOOS=linux go build \
	-ldflags="-w -s" \
	-o /app/bin/classifier \
	./cmd/classifier

FROM alpine:3.24

# hadolint ignore=DL3018
RUN apk upgrade --no-cache && \
	apk add --no-cache ca-certificates tzdata && \
	addgroup -g 1001 -S appgroup && \
	adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/bin/classifier .

USER appuser

EXPOSE 8001

ENTRYPOINT ["./classifier"]