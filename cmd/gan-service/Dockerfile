# syntax=docker/dockerfile:1
FROM golang:1.27.4-alpine AS go-builder

ENV GOPROXY=https://proxy.golang.org,direct
ENV GOTOOLCHAIN=auto

# hadolint ignore=DL3018
RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY cmd/gan-service/ ./cmd/gan-service/
COPY internal/ ./internal/
COPY api/ ./api/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s -buildid=" -o "/app/bin/gan-service" "./cmd/gan-service"

FROM python:3.14-slim

# hadolint ignore=DL3008
RUN apt-get update \
    && apt-get upgrade -y \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY cmd/gan-service/requirements.txt /tmp/requirements.txt
# Состав и версии зависимостей живут только в requirements.txt. Раньше Dockerfile
# дублировал список inline-пинами и разошёлся с ним (torch 2.13 вместо 2.15,
# numpy 2.3 вместо 2.5, filelock 4.2.4 вместо 4.2.10) — теперь источник истины один.
# Флаг записан как --only-binary=:all: (через "="), а не двумя аргументами: парсер
# docker:S8544 считает ":all:" именем пакета и даёт ложное "dependencies without
# locking resolved versions". NOSONAR для Dockerfile не поддерживается.
RUN pip install --no-cache-dir --only-binary=:all: --index-url https://pypi.org/simple --upgrade "pip==26.3" && \
    pip install \
    --extra-index-url https://pypi.org/simple \
    --index-url https://download.pytorch.org/whl/cpu \
    --no-cache-dir \
    --only-binary=:all: \
    -r /tmp/requirements.txt

COPY --from=go-builder /app/bin/gan-service /app/gan-service
COPY cmd/ml_generator/ ./cmd/ml_generator/
COPY models/ ./models/

ENV PYTHON_PATH=/usr/local/bin/python
ENV GAN_SCRIPT_PATH=/app/cmd/ml_generator/gan_train.py
ENV GAN_MODEL_PATH=/app/models/gan_generator.pt
ENV GAN_SERVICE_PORT=8004
ENV GAN_SERVICE_METRICS_PORT=9094

EXPOSE 8004 9094

RUN groupadd -r -g 1001 appgroup && useradd -r -u 1001 -g appgroup -d /app -s /sbin/nologin appuser
USER 1001

ENTRYPOINT ["./gan-service"]
