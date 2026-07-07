# Python 3.14 — последняя стабильная версия на July 2026
FROM python:3.14-slim AS builder

# Security: non-root user
RUN groupadd -r mluser && \
    useradd -r -g mluser -d /app -s /sbin/nologin mluser

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Create virtual environment
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Обновляем pip до актуальной версии (>= 26.1 на July 2026)
RUN pip install --no-cache-dir --upgrade "pip>=26.1" "setuptools>=83.0" "wheel>=0.47"

# Install dependencies (cached layer) — использует ranges из requirements.txt
COPY cmd/ml_generator/requirements.txt /tmp/requirements.txt
RUN pip install --no-cache-dir --retries 5 --timeout 120 -r /tmp/requirements.txt

# Runtime stage
FROM python:3.14-slim AS runtime

RUN groupadd -r mluser && \
    useradd -r -g mluser -d /app -s /sbin/nologin mluser

WORKDIR /app

# Copy venv from builder
COPY --from=builder /opt/venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Create models directory
RUN mkdir -p /app/models && chown -R mluser:mluser /app/models

USER mluser

# Copy code
COPY --chown=mluser:mluser cmd /app/cmd

# Copy models if they exist (optional — модель обучается отдельно)
COPY --chown=mluser:mluser models/generator.onnx /app/models/generator.onnx

EXPOSE 8002

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
   CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8002/health')" || exit 1

WORKDIR /app/cmd/ml_generator
CMD ["python", "-m", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8002", "--loop", "uvloop"]