# Используем Python 3.12, так как TensorFlow 2.19 не поддерживает 3.13+
# См.: https://pypi.org/project/tensorflow/2.19.0/#files
FROM python:3.12-slim

# Security: non-root user and venv setup in one RUN
RUN groupadd -r mluser && \
    useradd -r -g mluser -d /app -s /sbin/nologin mluser && \
    python -m venv /opt/venv && \
    chown -R mluser:mluser /opt/venv

WORKDIR /app

# Create models directory before switching user
RUN mkdir -p /app/models && chown -R mluser:mluser /app

USER mluser
ENV PATH="/opt/venv/bin:$PATH"

# Install dependencies (cached layer)
COPY --chown=mluser:mluser cmd/ml_classifier/requirements.txt /tmp/requirements.txt
RUN python -m pip install --no-cache-dir --upgrade pip==25.2 setuptools==81.0 wheel==0.45.1 && \
    python -m pip install --no-cache-dir --retries 5 --timeout 120 -r /tmp/requirements.txt

# Copy code (preserves project structure)
COPY --chown=mluser:mluser cmd /app/cmd

# Copy all models from models/ folder (including timestamped versions).
# At container startup, main.py will select the latest timestamped model
# and update classifier.keras. Old timestamped models will be deleted automatically.
COPY --chown=mluser:mluser models/ /app/models/

EXPOSE 8001

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
   CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8001/health')" || exit 1

WORKDIR /app/cmd/ml_classifier
CMD ["python", "-m", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8001"]
