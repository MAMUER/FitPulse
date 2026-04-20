# syntax=docker/dockerfile:1

# ========== Builder stage ==========
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Кэшируем загрузку зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем только нужный код
COPY cmd/user-service/ ./cmd/user-service/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY api/ ./api/

# Собираем статический бинарник
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/bin/user-service \
    ./cmd/user-service

# ========== Runtime stage ==========
FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata

# Создаём непривилегированного пользователя
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/bin/user-service .

USER appuser

EXPOSE 50051

ENTRYPOINT ["./user-service"]
