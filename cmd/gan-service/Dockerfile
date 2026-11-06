# syntax=docker/dockerfile:1
FROM golang@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS go-builder

ENV GOPROXY=https://proxy.golang.org,direct
ENV GOTOOLCHAIN=auto

RUN apk add --no-cache git=2.54.0-r0 ca-certificates=20261020-r0

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
RUN pip install --no-cache-dir --only-binary :all: --require-hashes -r /tmp/requirements.txt --index-url https://download.pytorch.org/whl/cpu --extra-index-url https://pypi.org/simple

COPY --from=go-builder /app/bin/gan-service /app/gan-service
COPY cmd/ml_generator/ ./cmd/ml_generator/
COPY models/ ./models/

ENV PYTHON_PATH=/usr/local/bin/python
ENV GAN_SCRIPT_PATH=/app/cmd/ml_generator/gan_train.py
ENV GAN_MODEL_PATH=/app/models/gan_generator.pt
ENV GAN_SERVICE_PORT=8004
ENV GAN_SERVICE_METRICS_PORT=9094

EXPOSE 8004 9094

RUN addgroup -g 1001 -S appgroup && adduser -u 1001 -S appuser -G appgroup
USER 1001

ENTRYPOINT ["./gan-service"]
