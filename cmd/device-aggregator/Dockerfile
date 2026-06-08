FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/device-aggregator/ ./cmd/device-aggregator/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/device-aggregator ./cmd/device-aggregator

FROM alpine:3.23
# hadolint ignore=DL3018
RUN apk --no-cache add ca-certificates tzdata
RUN addgroup -g 1001 -S appgroup && adduser -u 1001 -S appuser -G appgroup

WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /app/bin/device-aggregator .

USER appuser
EXPOSE 8083
ENTRYPOINT ["./device-aggregator"]