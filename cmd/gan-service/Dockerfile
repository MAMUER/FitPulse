# syntax=docker/dockerfile:1
FROM golang@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS go-builder

ENV GOPROXY=https://proxy.golang.org,direct
ENV GOTOOLCHAIN=auto

# hadolint ignore=DL3018
RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/gan-service/ ./cmd/gan-service/
COPY internal/ ./internal/
COPY api/ ./api/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s -buildid=" -o "/app/bin/gan-service" "./cmd/gan-service"

FROM python:3.14-slim

WORKDIR /app

COPY cmd/gan-service/requirements.txt /tmp/requirements.txt
RUN pip install \
    --extra-index-url https://pypi.org/simple \
    --index-url https://download.pytorch.org/whl/cpu \
    --no-cache-dir \
    --only-binary :all: \
    filelock==4.2.4 \
    fsspec==2026.9.0 \
    jinja2==3.1.6 \
    markupsafe==3.0.3 \
    mpmath==1.3.0 \
    networkx==3.7 \
    numpy==2.3.5 \
    setuptools==84.2.1 \
    sympy==1.14.0 \
    torch==2.13.0 \
    typing-extensions==4.16.0

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
