# syntax=docker/dockerfile:1

# ========== Builder stage ==========
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git=2.52.0-r0 ca-certificates=20260413-r0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/device-connector/ ./cmd/device-connector/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY api/ ./api/

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/bin/device-connector \
    ./cmd/device-connector

# ========== Runtime stage ==========
FROM alpine:3.21

RUN apk --no-cache add ca-certificates=20260413-r0 tzdata=2026b-r0 && \
    addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/bin/device-connector .

USER appuser

EXPOSE 8082

ENTRYPOINT ["./device-connector"]
