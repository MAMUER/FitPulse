# Python 3.14 — последняя стабильная версия на July 2026
FROM python:3.14-slim AS builder

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

# Обновляем pip до актуальной версии (>= 26.1 на July 2026)
RUN pip install --no-cache-dir --only-binary :all: --upgrade "pip>=26.1" "setuptools>=83.0" "wheel>=0.47"

# Install dependencies (cached layer) — использует lock файл для reproducibility
COPY cmd/ml_generator/requirements.lock.txt /tmp/requirements.lock.txt
RUN pip install --no-cache-dir --only-binary :all: --retries 5 --timeout 120 -r /tmp/requirements.lock.txt

# Runtime stage
FROM python:3.14-slim AS runtime

RUN groupadd -r mluser -g 1001 && \
    useradd -r -u 1001 -g mluser -d /app -s /sbin/nologin mluser

WORKDIR /app

# Copy venv from builder
COPY --from=builder /opt/venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Create models directory
RUN mkdir -p /app/models && chown -R 1001:1001 /app/models

USER 1001

# Copy code
COPY --chown=1001:1001 cmd /app/cmd

# Copy models if they exist (optional — модель обучается отдельно)
COPY --chown=1001:1001 models/generator.onnx /app/models/generator.onnx

EXPOSE 8002

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
   CMD ["python", "-c", "import urllib.request; urllib.request.urlopen('http://localhost:8002/health')"]

WORKDIR /app/cmd/ml_generator
CMD ["python", "-m", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8002", "--loop", "uvloop"]