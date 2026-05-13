FROM python:3.11-slim

# Security: non-root user
RUN groupadd -r mluser && useradd -r -g mluser -d /app -s /sbin/nologin mluser

WORKDIR /app

# Установка зависимостей
COPY cmd/ml-generator/requirements.txt .
RUN python -m pip install --upgrade pip setuptools wheel
RUN pip install --no-cache-dir --retries 5 --timeout 120 -r requirements.txt

# Копирование кода
COPY cmd/ml-generator/main.py .
COPY cmd/ml-generator/train_gan.py .

# Создание директории для моделей
RUN mkdir -p /app/models

# Копирование моделей из корневой директории
COPY models/generator.keras /app/models/generator.keras
COPY models/generator.h5 /app/models/generator.h5

# Security: ownership
RUN chown -R mluser:mluser /app
USER mluser

EXPOSE 8002

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8002/health')" || exit 1

CMD ["python", "-m", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8002"]