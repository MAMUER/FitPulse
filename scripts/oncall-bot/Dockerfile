FROM python:3.14-slim

RUN groupadd -r appgroup && useradd -r -g appgroup -d /app -s /sbin/nologin appuser

WORKDIR /app
COPY main.py .
RUN pip install --no-cache-dir --only-binary :all: \
    fastapi==0.115.6 \
    python-telegram-bot==20.7 \
    pyyaml==6.0.2 \
    requests==2.32.3 \
    uvicorn==0.34.0 \
    && chown -R appuser:appgroup /app
USER appuser

EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD python3 -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/health')" || exit 1
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
