#!/bin/bash
THRESHOLD=90
USAGE=$(free | grep Mem | awk '{printf "%.0f", $3/$2 * 100}')

if [ "$USAGE" -gt "$THRESHOLD" ]; then
	curl -s -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" \
		-H "Content-Type: application/json" \
		-d "{\"chat_id\": \"${TELEGRAM_CHAT_ID}\", \"text\": \"⚠️ Memory usage: ${USAGE}% on fittpulse\"}"
fi
