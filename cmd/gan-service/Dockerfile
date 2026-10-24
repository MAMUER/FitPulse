FROM python:3.12-slim

WORKDIR /app

RUN pip install --no-cache-dir --only-binary :all: numpy==2.2.0 torch==2.5.1 --index-url https://download.pytorch.org/whl/cpu

COPY cmd/gan-service/gan-service .
COPY cmd/ml_generator/ ./cmd/ml_generator/
COPY models/ ./models/

ENV PYTHON_PATH=/usr/local/bin/python
ENV GAN_SCRIPT_PATH=/app/cmd/ml_generator/gan_train.py
ENV GAN_MODEL_PATH=/app/models/gan_generator.pt
ENV GAN_SERVICE_PORT=8004
ENV GAN_SERVICE_METRICS_PORT=9094

EXPOSE 8004 9094

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

ENTRYPOINT ["./gan-service"]
