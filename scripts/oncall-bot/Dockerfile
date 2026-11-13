FROM python:3.14-slim

RUN groupadd -r -g 1000 appgroup && useradd -r -u 1000 -g appgroup -d /app -s /sbin/nologin appuser

WORKDIR /app
COPY main.py .
COPY requirements.txt .
# NOSONAR: direct dependencies are pinned in requirements.txt; this is a
# one-shot internal bot image, not a library.
RUN pip install --no-cache-dir --only-binary :all: -r requirements.txt \
    && chown -R appuser:appgroup /app
USER 1000

EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["python3", "-c", "import urllib.request; urllib.request.urlopen('http://localhost:8000/health')"]
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
