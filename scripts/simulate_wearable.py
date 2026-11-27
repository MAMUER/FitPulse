#!/usr/bin/env python3
"""Симулятор OpenWearables-вебхука — проверка без реального устройства.

Отправляет подписанный payload в biometric-service (или в gateway-прокси)
и печатает ответ. Позволяет проверить цепочку:
    устройство -> webhook -> biometric_data -> classifier/GAN.

Запуск:
    python scripts/simulate_wearable.py \
        --url http://127.0.0.1:8085/api/v1/webhook/open-wearables \
        --secret <OPEN_WEARABLES_WEBHOOK_SECRET> \
        --user-id <UUID>

Проверка, что данные дошли:
    psql -c "SELECT metric_type, value, created_at FROM biometric_data ORDER BY created_at DESC LIMIT 10;"
"""

from __future__ import annotations

import argparse
import hashlib
import hmac
import json
import time
import urllib.request
import uuid


def sign(secret: bytes, body: bytes) -> str:
    mac = hmac.new(secret, body, hashlib.sha256)
    return mac.hexdigest()


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument(
        "--url",
        default="http://127.0.0.1:8085/api/v1/integrations/open-wearables/webhook",
    )
    ap.add_argument("--secret", default="")
    ap.add_argument("--user-id", default="")
    ap.add_argument("--source", default="open_wearables")
    args = ap.parse_args()

    user_id = args.user_id or str(uuid.uuid4())
    ts = str(int(time.time()))
    payload = {
        "user_id": user_id,
        "source": args.source,
        "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "nonce": uuid.uuid4().hex,
        "metrics": [
            {"metric_type": "heart_rate", "value": 62.0, "unit": "bpm"},
            {"metric_type": "hrv", "value": 45.0, "unit": "ms"},
            {"metric_type": "spo2", "value": 97.5, "unit": "%"},
            {"metric_type": "temperature", "value": 33.5, "unit": "C"},
            {"metric_type": "sleep", "value": 7.2, "unit": "h"},
        ],
    }
    body = json.dumps(payload).encode()

    req = urllib.request.Request(args.url, data=body, method="POST")
    req.add_header("Content-Type", "application/json")
    req.add_header("X-Webhook-Id", "sim-" + uuid.uuid4().hex)
    req.add_header("X-Webhook-Timestamp", ts)
    if args.secret:
        req.add_header("X-Open-Wearables-Signature", sign(args.secret.encode(), body))

    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            print("status:", resp.status)
            print("body:", resp.read().decode())
    except Exception as exc:
        print("error:", exc)
        return

    print("\nПроверка в БД:")
    print(
        '  psql -c "SELECT metric_type, value, created_at FROM biometric_data ORDER BY created_at DESC LIMIT 10;"'
    )


if __name__ == "__main__":
    main()
