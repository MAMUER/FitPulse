#!/usr/bin/env python3
"""
Fitness Platform — API Test Suite (cross-platform)

Usage:
    python scripts/api-test.py
    python scripts/api-test.py --base-url https://localhost:8443
"""

import os
import sys
import json
import random
import argparse
import http.client
import ssl
import urllib.request
import urllib.error
from urllib.parse import urlparse
from datetime import datetime

# === Configuration ===
DEFAULT_BASE_URL = "https://localhost:8443"
TEST_PASSWORD = os.getenv("TEST_PASSWORD", "TestPass123!")
GREEN = "\033[92m"
YELLOW = "\033[93m"
RED = "\033[91m"
CYAN = "\033[96m"
GRAY = "\033[90m"
RESET = "\033[0m"
BOLD = "\033[1m"


class TestRunner:
    def __init__(self, base_url):
        self.base_url = base_url.rstrip("/")
        self.token = None
        self.passed = 0
        self.failed = 0
        self.skipped = 0
        self.results = []
        # Disable SSL verification for self-signed certs
        self.ctx = ssl.create_default_context()
        self.ctx.check_hostname = False
        self.ctx.verify_mode = ssl.CERT_NONE

    def _make_request(self, method, request_path, data=None, headers=None):
        req = urllib.request.Request(
            self.base_url + request_path,
            data=data,
            headers=headers or {},
            method=method,
        )
        try:
            with urllib.request.urlopen(req, context=self.ctx, timeout=30) as resp:
                return resp.status, json.loads(resp.read().decode("utf-8", errors="replace"))
        except urllib.error.HTTPError as e:
            body = {}
            try:
                body = json.loads(e.read().decode("utf-8", errors="replace"))
            except Exception:
                pass
            return e.code, body
        except Exception as e:
            return None, {"error": str(e)}

    def request(self, method, path, body=None, token=None, expected_status=200):
        """Send HTTP request and return (status_code, body_dict)."""
        url = f"{self.base_url}{path}"
        parsed = urlparse(url)
        if parsed.scheme not in {"http", "https"} or not parsed.hostname:
            return None, {"error": "unsupported or invalid API URL"}

        headers = {"Content-Type": "application/json"}
        if token:
            headers["Authorization"] = f"Bearer {token}"

        data = json.dumps(body).encode("utf-8") if body else None
        request_path = parsed.path or "/"
        if parsed.query:
            request_path = f"{request_path}?{parsed.query}"

        return self._make_request(method, request_path, data=data, headers=headers)

    def test(self, name, method, path, body=None, expected=200, token=None):
        """Run a single test case."""
        num = self.passed + self.failed + self.skipped + 1
        print(f"  [{num}] {name} ", end="", flush=True)

        status, resp_body = self.request(method, path, body, token=token, expected_status=expected)

        if status is None:
            print(f"{RED}ERROR (connection){RESET}")
            self.failed += 1
            self.results.append(f"FAIL | {method} {path} | CONNECTION ERROR")
            return resp_body

        ok = status == expected
        preview = str(resp_body)[:120]

        if ok:
            self.passed += 1
            print(f"{GREEN}PASS ({status}){RESET}")
        else:
            self.failed += 1
            print(f"{RED}FAIL (exp:{expected} got:{status}) — {preview}{RESET}")

        self.results.append(
            f"{'PASS' if ok else 'FAIL'} | {method} {path} | Exp:{expected} | Act:{status} | {preview}"
        )
        return resp_body


def section(title):
    print(f"\n{CYAN}=== {title} ==={RESET}")


def main():
    parser = argparse.ArgumentParser(description="Fitness Platform API Test Suite")
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL, help="API base URL")
    args = parser.parse_args()

    t = TestRunner(args.base_url)
    test_email = f"apitest-{random.randint(1000, 9999)}@example.com"

    print(f"\n{BOLD}{CYAN}{'=' * 50}{RESET}")
    print(f"{BOLD}{CYAN}   FITNESS PLATFORM — API TEST SUITE{RESET}")
    print(f"{BOLD}{CYAN}{'=' * 50}{RESET}")
    print(f"  Base URL : {args.base_url}")
    print(f"  Test User: {test_email}")
    print()

    # 0. Health
    section("0. HEALTH")
    t.test("Health", "GET", "/health", expected=200)

    # 1. Auth
    section("1. AUTH")
    reg_body = {
        "email": test_email,
        "password": TEST_PASSWORD,
        "full_name": "API Test User",
        "role": "client",
    }
    resp = t.test("Register", "POST", "/api/v1/register", body=reg_body, expected=200)

    # Extract verification token
    verify_token = ""
    if isinstance(resp, dict) and resp.get("message"):
        import re
        match = re.search(r"token \(dev only\):\s*([a-f0-9]+)", resp["message"])
        if match:
            verify_token = match.group(1)

    if verify_token:
        t.test("Confirm Email", "POST", "/api/v1/auth/confirm",
               body={"token": verify_token}, expected=200)

    t.test("Register (dup)", "POST", "/api/v1/register", body=reg_body, expected=409)
    t.test("Register (bad email)", "POST", "/api/v1/register",
           body={"email": "bad", "password": TEST_PASSWORD, "full_name": "B", "role": "client"}, expected=400)
    t.test("Register (short pw)", "POST", "/api/v1/register",
           body={"email": "s@e.com", "password": "123", "full_name": "S", "role": "client"}, expected=400)

    # Login
    login_resp = t.test("Login", "POST", "/api/v1/login",
                        body={"email": test_email, "password": TEST_PASSWORD}, expected=200)
    if isinstance(login_resp, dict) and login_resp.get("access_token"):
        t.token = login_resp["access_token"]

    t.test("Login (wrong pw)", "POST", "/api/v1/login",
           body={"email": test_email, "password": "wrong"}, expected=401)
    t.test("Login (empty email)", "POST", "/api/v1/login",
           body={"email": "", "password": TEST_PASSWORD}, expected=400)

    if not t.token:
        print(f"\n{RED}No token obtained. Skipping auth tests.{RESET}")
        sys.exit(1)

    # 2. Profile
    section("2. PROFILE")
    t.test("Get Profile", "GET", "/api/v1/profile", token=t.token, expected=200)
    t.test("Update Profile", "PUT", "/api/v1/profile",
           body={"full_name": "API Test User", "age": 28, "gender": "male",
                 "height_cm": 180, "weight_kg": 75.5,
                 "fitness_level": "intermediate", "goals": ["weight_loss", "endurance"],
                 "contraindications": ["knee"], "nutrition": "balanced", "sleep_hours": 7.5},
           token=t.token, expected=200)
    t.test("Get Profile (after)", "GET", "/api/v1/profile", token=t.token, expected=200)

    # 3. Biometrics
    section("3. BIOMETRICS")
    now = datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
    t.test("Add Biometric (HR)", "POST", "/api/v1/biometrics",
           body={"metric_type": "heart_rate", "value": 72.0, "timestamp": now},
           token=t.token, expected=201)
    t.test("Add Biometric (SpO2)", "POST", "/api/v1/biometrics",
           body={"metric_type": "spo2", "value": 98.0, "timestamp": now},
           token=t.token, expected=201)
    t.test("Add Biometric (neg)", "POST", "/api/v1/biometrics",
           body={"metric_type": "heart_rate", "value": -10.0, "timestamp": now},
           token=t.token, expected=400)
    t.test("Get Biometrics", "GET", "/api/v1/biometrics?metric_type=heart_rate&limit=10",
           token=t.token, expected=200)

    # Logout
    t.test("Logout", "POST", "/api/v1/logout", token=t.token, expected=200)
    t.token = None

    # 4. Post-logout — требование #4: сервер возвращает 404 вместо 401/403
    section("4. POST-LOGOUT")
    t.test("Profile (no token)", "GET", "/api/v1/profile", expected=404)
    t.test("Biometrics (no token)", "GET", "/api/v1/biometrics", expected=404)

    # Re-login
    lr = t.test("Re-login", "POST", "/api/v1/login",
                body={"email": test_email, "password": TEST_PASSWORD}, expected=200)
    if isinstance(lr, dict) and lr.get("access_token"):
        t.token = lr["access_token"]

    # 5. Training
    section("5. TRAINING")
    t.test("Get Plans", "GET", "/api/v1/training/plans", token=t.token, expected=200)
    t.test("Get Progress", "GET", "/api/v1/training/progress", token=t.token, expected=200)

    # 6. ML
    section("6. ML")
    # ML может вернуть 202 (async job) или 200 (sync result)
    ml_resp = t.test("ML Classify", "POST", "/api/v1/ml/classify",
                     token=t.token, expected=200)
    if isinstance(ml_resp, dict) and ml_resp.get("job_id"):
        print(f"       {GRAY}job_id: {ml_resp['job_id']}{RESET}")

    # 7. Security
    section("7. SECURITY")
    t.token = None
    t.test("Profile (no token)", "GET", "/api/v1/profile", expected=404)
    t.test("Training (no token)", "GET", "/api/v1/training/plans", expected=404)

    # Summary
    total = t.passed + t.failed + t.skipped
    print(f"\n{CYAN}{'=' * 50}{RESET}")
    print(f"{CYAN}  SUMMARY{RESET}")
    print(f"{CYAN}{'=' * 50}{RESET}")
    print(f"  {GREEN}Passed : {t.passed}/{total}{RESET}")
    print(f"  {RED}Failed : {t.failed}/{total}{RESET}")
    if t.skipped > 0:
        print(f"  {GRAY}Skipped: {t.skipped}/{total}{RESET}")

    if t.failed > 0:
        print(f"\n{RED}  FAILURES:{RESET}")
        for r in t.results:
            if "FAIL" in r:
                print(f"    {r}")

    print()
    if t.failed == 0:
        print(f"{GREEN}{BOLD}  ALL TESTS PASSED!{RESET}\n")
        sys.exit(0)
    else:
        print(f"{RED}{BOLD}  SOME TESTS FAILED!{RESET}\n")
        sys.exit(1)


if __name__ == "__main__":
    main()
