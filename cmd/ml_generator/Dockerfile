# Python 3.14 slim — минимальный образ
FROM python@sha256:cad9a2c871761c413caa6fdd6441c783451e740a48aaeba60ae62a8b53525ef6 AS builder

# Security: non-root user
RUN groupadd -r mluser -g 1001 && \
    useradd -r -u 1001 -g mluser -d /app -s /sbin/nologin mluser

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential=12.9 \
    && rm -rf /var/lib/apt/lists/*

# Create virtual environment
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Install dependencies (cached layer) — использует lock файл для reproducibility
COPY cmd/ml_generator/requirements.lock.txt /tmp/requirements.lock.txt
RUN pip install --no-cache-dir --only-binary :all: --retries 5 \
    --timeout 120 --require-hashes -r /tmp/requirements.lock.txt

# Runtime stage
FROM python@sha256:cad9a2c871761c413caa6fdd6441c783451e740a48aaeba60ae62a8b53525ef6 AS runtime

ARG APP_UID=1001
ARG APP_GID=1001

RUN groupadd -r mluser -g "${APP_GID}" && \
    useradd -r -u "${APP_UID}" -g mluser -d /app -s /sbin/nologin mluser

WORKDIR /app

# Copy venv from builder
COPY --from=builder /opt/venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Create models directory
RUN mkdir -p /app/models && chown -R "${APP_UID}:${APP_GID}" /app/models

USER ${APP_UID}

# Copy code
COPY --chown=${APP_UID}:${APP_GID} cmd /app/cmd

# Copy models if they exist (optional — модель обучается отдельно)
COPY --chown=${APP_UID}:${APP_GID} models/generator.onnx /app/models/generator.onnx

EXPOSE 8002

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
   CMD ["python", "-c", "import urllib.request; urllib.request.urlopen('http://localhost:8002/health')"]

WORKDIR /app/cmd/ml_generator
CMD ["python", "run_with_otel.py"]