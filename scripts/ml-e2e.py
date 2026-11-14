#!/usr/bin/env python3
"""Real ML E2E smoke test for the deployed classifier and GAN service.

The test is intended to run from the CI runner against port-forwarded
ClusterIP services (kubectl port-forward), so it does not depend on the
ingress or on NetworkPolicies. Standard library only.
"""

import argparse
import json
import sys
import urllib.error
import urllib.request


def request(method, url, payload=None, timeout=30):
    data = json.dumps(payload).encode("utf-8") if payload is not None else None
    req = urllib.request.Request(
        url, data=data, headers={"Content-Type": "application/json"}, method=method
    )
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            body = resp.read().decode("utf-8") or "{}"
            try:
                return resp.status, json.loads(body)
            except json.JSONDecodeError:
                return resp.status, {"_raw": body}
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8") or "{}"
        try:
            body = json.loads(body)
        except json.JSONDecodeError:
            body = {"_raw": body}
        return exc.code, body


def main():
    parser = argparse.ArgumentParser(description="FitPulse ML E2E smoke test")
    parser.add_argument("--classifier-url", default="http://127.0.0.1:18001")
    parser.add_argument("--gan-url", default="http://127.0.0.1:18002")
    args = parser.parse_args()

    # 1. Classifier health + model loaded.
    status, body = request("GET", f"{args.classifier_url}/health", timeout=15)
    if status != 200:
        print(f"FAIL classifier /health: {status} {body}")
        sys.exit(1)
    if body.get("model_loaded") is False:
        print(f"FAIL classifier model not loaded: {body}")
        sys.exit(1)
    print(f"OK classifier /health: model_loaded={body.get('model_loaded')}")

    # 2. Classifier inference.
    payload = {
        "physiological_data": {
            "heart_rate": 72,
            "heart_rate_variability": 50,
            "spo2": 98,
            "temperature": 36.6,
            "blood_pressure_systolic": 120,
            "blood_pressure_diastolic": 80,
            "sleep_hours": 7,
        },
        "user_profile": {
            "gender": "male",
            "age": 30,
            "fitness_level": "intermediate",
            "weight": 75,
            "height": 180,
            "health_conditions": [],
            "goals": ["weight_loss"],
        },
        "user_id": "ml-e2e",
    }
    status, body = request(
        "POST", f"{args.classifier_url}/classify", payload=payload, timeout=60
    )
    if status != 200 or not body.get("state"):
        print(f"FAIL classifier /classify: {status} {body}")
        sys.exit(1)
    print(
        f"OK classifier /classify: state={body.get('state')} "
        f"predicted_class={body.get('predicted_class')}"
    )

    # 3. GAN service health.
    status, body = request("GET", f"{args.gan_url}/health", timeout=15)
    if status != 200 or body.get("status") != "healthy":
        print(f"FAIL gan /health: {status} {body}")
        sys.exit(1)
    print(f"OK gan /health: {body}")

    # 4. GAN plan generation (loads the local model; can take a while).
    gan_payload = {
        "class_index": 1,
        "age": 30,
        "gender": 1,
        "weight": 75,
        "height": 180,
        "goal": 1,
        "num_samples": 1,
    }
    status, body = request(
        "POST", f"{args.gan_url}/gan/generate", payload=gan_payload, timeout=120
    )
    if status != 200 or not body.get("plans"):
        print(f"FAIL gan /gan/generate: {status} {body}")
        sys.exit(1)
    print(f"OK gan /gan/generate: plans={len(body['plans'])}")
    print("ML E2E PASSED")


if __name__ == "__main__":
    main()
