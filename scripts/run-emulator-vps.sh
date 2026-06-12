#!/bin/bash
# scripts/run-emulator-vps.sh

set -e

echo "========================================"
echo "  DEVICE EMULATOR RUNNER (VPS)"
echo "========================================"

# Загружаем переменные окружения
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

# Получаем UUID администратора из PostgreSQL
USER_ID=$(docker exec fitness-postgres psql -U ${POSTGRES_USER:-postgres} -d ${POSTGRES_DB:-fitness} -t -c "SELECT id FROM users WHERE email='admin@fitpulse.local'" | tr -d '[:space:]')

if [ -z "$USER_ID" ]; then
    echo "ERROR: Admin user not found!"
    exit 1
fi

echo "User ID: $USER_ID"
echo ""

# Проверяем, есть ли уже устройство для этого пользователя
EXISTING_DEVICE=$(docker exec fitness-postgres psql -U ${POSTGRES_USER:-postgres} -d ${POSTGRES_DB:-fitness} -t -c "SELECT id FROM devices WHERE user_id='$USER_ID' LIMIT 1" | tr -d '[:space:]')

if [ -n "$EXISTING_DEVICE" ]; then
    echo "Device already exists: $EXISTING_DEVICE"
    echo "Starting emulator with existing device..."
    
    # Получаем токен устройства
    DEVICE_TOKEN=$(docker exec fitness-postgres psql -U ${POSTGRES_USER:-postgres} -d ${POSTGRES_DB:-fitness} -t -c "SELECT token FROM devices WHERE id='$EXISTING_DEVICE'" | tr -d '[:space:]')
    
    # Запускаем эмулятор с существующим устройством
    docker run -d --name fitness-device-emulator \
        --network fitness-network \
        -e USER_ID="$USER_ID" \
        -e DEVICE_ID="$EXISTING_DEVICE" \
        -e DEVICE_TOKEN="$DEVICE_TOKEN" \
        -e DEVICE_TYPE="samsung_galaxy_watch" \
        -e CONNECTOR_URL="http://device-connector:8082" \
        -e SYNC_INTERVAL="30s" \
        ghcr.io/mamuer/project/device-emulator:latest
else
    echo "No device found. Registering new device..."
    
    # Запускаем эмулятор с авторегистрацией
    docker run -d --name fitness-device-emulator \
        --network fitness-network \
        -e USER_ID="$USER_ID" \
        -e DEVICE_TYPE="samsung_galaxy_watch" \
        -e CONNECTOR_URL="http://device-connector:8082" \
        -e SYNC_INTERVAL="30s" \
        -e AUTO_REGISTER="true" \
        ghcr.io/mamuer/project/device-emulator:latest
fi

echo ""
echo "✅ Emulator started!"
echo "To view logs: docker logs -f fitness-device-emulator"
echo "To stop: docker stop fitness-device-emulator && docker rm fitness-device-emulator"