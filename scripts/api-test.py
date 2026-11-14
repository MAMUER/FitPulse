#!/usr/bin/env python3
"""
Fitness Platform — API Test Suite (cross-platform)

Usage:
    python scripts/api-test.py
    python scripts/api-test.py --base-url https://localhost:8443
    python scripts/api-test.py --insecure
"""

import argparse
import http.client
import json
import os
import random
import re
import ssl
import subprocess
import sys
import time
from datetime import datetime, timezone
from urllib.parse import urlsplit

# === Configuration ===
DEFAULT_BASE_URL = "https://localhost:8443"
TEST_PASSWORD = os.getenv("TEST_PASSWORD", "TestPass123!")
REGISTER_PATH = "/api/v1/register"
LOGIN_PATH = "/api/v1/login"
PROFILE_PATH = "/api/v1/profile"
BIOMETRICS_PATH = "/api/v1/biometrics"
GREEN = "\033[92m"
YELLOW = "\033[93m"
RED = "\033[91m"
CYAN = "\033[96m"
GRAY = "\033[90m"
RESET = "\033[0m"
BOLD = "\033[1m"


class TestRunner:
    def __init__(self, base_url, insecure=False, confirm_email=False):
        self.base_url = base_url.rstrip("/")
        self.token = None
        self.last_set_cookie = ""
        self.confirm_email = confirm_email
        self.passed = 0
        self.failed = 0
        self.skipped = 0
        self.results = []
        self.last_status = None
        self.parsed_base_url = urlsplit(self.base_url)
        self.host = self.parsed_base_url.hostname
        try:
            port = self.parsed_base_url.port
            port_error = False
        except ValueError:
            port = None
            port_error = True
        if port is not None:
            self.port = port
        else:
            self.port = 443 if self.parsed_base_url.scheme == "https" else 80
        self.path_prefix = self.parsed_base_url.path.rstrip("/")
        self.invalid_base_url = (
            self.parsed_base_url.scheme not in {"http", "https"}
            or not self.parsed_base_url.hostname
            or port_error
        )
        self.ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
        self.ctx.minimum_version = ssl.TLSVersion.TLSv1_2
        self.ctx.check_hostname = True
        self.ctx.verify_mode = ssl.CERT_REQUIRED
        if insecure:
            self.ctx.check_hostname = False
            self.ctx.verify_mode = ssl.CERT_NONE

    def _make_request(self, method, request_path, data=None, headers=None):
        path = request_path if request_path.startswith("/") else f"/{request_path}"
        if self.path_prefix:
            path = f"{self.path_prefix}{path}"

        connection_cls = (
            http.client.HTTPSConnection
            if self.parsed_base_url.scheme == "https"
            else http.client.HTTPConnection
        )
        connection_kwargs = {"timeout": 30}
        if self.parsed_base_url.scheme == "https":
            connection_kwargs["context"] = self.ctx

        try:
            conn = connection_cls(self.host, self.port, **connection_kwargs)
            try:
                conn.request(method, path, body=data, headers=headers or {})
                resp = conn.getresponse()
                self.last_set_cookie = "; ".join(resp.msg.get_all("Set-Cookie") or [])
                body = resp.read().decode("utf-8", errors="replace")
                if not body:
                    return resp.status, {}
                try:
                    return resp.status, json.loads(body)
                except json.JSONDecodeError:
                    # Some gateways/proxies return plain text (e.g. "OK",
                    # "404 page not found", ModSecurity block pages). Keep the
                    # raw body so the test reports a normal FAIL instead of
                    # crashing with a JSONDecodeError.
                    return resp.status, {"_raw": body}
            finally:
                conn.close()
        except (OSError, http.client.HTTPException) as e:
            return None, {"error": str(e)}

    def request(self, method, path, body=None, token=None):
        """Send HTTP request and return (status_code, body_dict)."""
        url = f"{self.base_url}{path}"
        parsed = urlsplit(url)
        if self.invalid_base_url:
            return None, {"error": "unsupported or invalid API URL"}
        # Blocks unsafe URL schemes (e.g. file://) from reaching the HTTP client below.

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

        status, resp_body = self.request(method, path, body, token=token)
        # Auth endpoints are rate-limited per IP (5/min). A CI runner can hit
        # the limit after a few runs, and a 429 is not a product failure:
        # wait for the window to reset and retry instead of crashing.
        for _ in range(2):
            if status != 429:
                break
            print("rate-limited (429), waiting 65s...", end="", flush=True)
            time.sleep(65)
            status, resp_body = self.request(method, path, body, token=token)
        self.last_status = status
        # The gateway stores the access token in an HttpOnly cookie, not in the
        # JSON body. Capture it from Set-Cookie so authenticated tests work.
        if status == 200 and self.last_set_cookie:
            m = re.search(r"(?:^|;\s*)access_token=([^;]+)", self.last_set_cookie)
            if m:
                self.token = m.group(1)

        if status is None:
            error = resp_body.get("error", "unknown connection error")
            print(f"{RED}ERROR (connection: {error}){RESET}")
            self.failed += 1
            self.results.append(f"FAIL | {method} {path} | CONNECTION ERROR: {error}")
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
            f"{'PASS' if ok else 'FAIL'} | {method} {path} | "
            f"Exp:{expected} | Act:{status} | {preview}"
        )
        return resp_body


def section(title):
    print(f"\n{CYAN}=== {title} ==={RESET}")


def test_health(t):
    section("0. HEALTH")
    body = t.test("Health", "GET", "/health", expected=200)
    if isinstance(body, dict) and body.get("status") != "ok":
        t.results.append(f"FAIL | GET /health | status={body.get('status')!r}")
        t.failed += 1
        t.passed -= 1
        print(f"{RED}Health status is {body.get('status')!r}{RESET}")
    return t.last_status == 200


def confirm_test_user_email(email: str) -> None:
    """Confirm the test user's email via kubectl (CI only).

    The API test runs against the production cluster; the registration email is
    not delivered to a real mailbox. The test job has the kubeconfig, so mark
    the user as confirmed directly in Postgres. This is a test-only helper and
    never runs without --confirm-email.
    """
    script = (
        'export PGPASSWORD="$(cat /etc/secrets/POSTGRES_PASSWORD)"\n'
        'U="$(cat /etc/secrets/POSTGRES_USER)"\n'
        'D="$(cat /etc/secrets/POSTGRES_DB)"\n'
        'psql -U "$U" -d "$D" -v ON_ERROR_STOP=1 -c "UPDATE users SET email_confirmed = TRUE '
        f"WHERE email_hash = encode(digest(lower('{email}'), 'sha256'), 'hex')\""
    )
    try:
        subprocess.run(
            [
                "kubectl",
                "exec",
                "-i",
                "-n",
                "fitness-platform-production",
                "postgres-0",
                "--",
                "sh",
                "-s",
            ],
            input=script,
            text=True,
            check=True,
            capture_output=True,
            timeout=30,
        )
        print("  Confirmed test user email via kubectl")
    except Exception as exc:  # best-effort: the login test will surface the failure
        print(f"  Warning: could not confirm test user email: {exc}")


def test_auth(t, test_email):
    section("1. AUTH")
    reg_body = {
        "email": test_email,
        "password": TEST_PASSWORD,
        "full_name": "API Test User",
        "role": "client",
    }
    t.test("Register", "POST", REGISTER_PATH, body=reg_body, expected=200)
    if t.confirm_email:
        confirm_test_user_email(test_email)

    t.test("Register (dup)", "POST", REGISTER_PATH, body=reg_body, expected=409)
    t.test(
        "Register (bad email)",
        "POST",
        REGISTER_PATH,
        body={
            "email": "bad",
            "password": TEST_PASSWORD,
            "full_name": "B",
            "role": "client",
        },
        expected=400,
    )
    t.test(
        "Register (short pw)",
        "POST",
        REGISTER_PATH,
        body={
            "email": "s@e.com",
            "password": "123",
            "full_name": "S",
            "role": "client",
        },
        expected=400,
    )

    t.test(
        "Login",
        "POST",
        LOGIN_PATH,
        body={"email": test_email, "password": TEST_PASSWORD},
        expected=200,
    )
    # test() captures the access_token from the Set-Cookie header. If the test
    # user is still unconfirmed, fall back to the seeded admin so the remaining
    # authenticated tests can run (the Login FAIL above is still reported).
    if (
        not t.token
        and os.getenv("SEED_ADMIN_EMAIL")
        and os.getenv("SEED_ADMIN_PASSWORD")
    ):
        t.test(
            "Login (admin fallback)",
            "POST",
            LOGIN_PATH,
            body={
                "email": os.getenv("SEED_ADMIN_EMAIL"),
                "password": os.getenv("SEED_ADMIN_PASSWORD"),
            },
            expected=200,
        )

    t.test(
        "Login (wrong pw)",
        "POST",
        LOGIN_PATH,
        body={"email": test_email, "password": "wrong"},
        expected=401,
    )
    t.test(
        "Login (empty email)",
        "POST",
        LOGIN_PATH,
        body={"email": "", "password": TEST_PASSWORD},
        expected=400,
    )

    if not t.token:
        print(f"\n{RED}No token obtained. Skipping auth tests.{RESET}")
        sys.exit(1)


def test_profile(t):
    section("2. PROFILE")
    t.test("Get Profile", "GET", PROFILE_PATH, token=t.token, expected=200)
    t.test(
        "Update Profile",
        "PUT",
        PROFILE_PATH,
        body={
            "full_name": "API Test User",
            "age": 28,
            "gender": "male",
            "height_cm": 180,
            "weight_kg": 75.5,
            "fitness_level": "intermediate",
            "goals": ["weight_loss", "endurance"],
            "contraindications": ["knee"],
            "nutrition": "balanced",
            "sleep_hours": 7.5,
        },
        token=t.token,
        expected=200,
    )
    t.test("Get Profile (after)", "GET", PROFILE_PATH, token=t.token, expected=200)


def test_biometrics(t):
    section("3. BIOMETRICS")
    now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    t.test(
        "Add Biometric (HR)",
        "POST",
        BIOMETRICS_PATH,
        body={"metric_type": "heart_rate", "value": 72.0, "timestamp": now},
        token=t.token,
        expected=201,
    )
    t.test(
        "Add Biometric (SpO2)",
        "POST",
        BIOMETRICS_PATH,
        body={"metric_type": "spo2", "value": 98.0, "timestamp": now},
        token=t.token,
        expected=201,
    )
    t.test(
        "Add Biometric (neg)",
        "POST",
        BIOMETRICS_PATH,
        body={"metric_type": "heart_rate", "value": -10.0, "timestamp": now},
        token=t.token,
        expected=400,
    )
    t.test(
        "Get Biometrics",
        "GET",
        f"{BIOMETRICS_PATH}?metric_type=heart_rate&limit=10",
        token=t.token,
        expected=200,
    )

    t.test("Logout", "POST", "/api/v1/logout", token=t.token, expected=200)
    t.token = None


def test_post_logout(t):
    section("4. POST-LOGOUT")
    t.test("Profile (no token)", "GET", PROFILE_PATH, expected=404)
    t.test("Biometrics (no token)", "GET", BIOMETRICS_PATH, expected=404)

    t.test(
        "Re-login",
        "POST",
        LOGIN_PATH,
        body={"email": t.test_email, "password": TEST_PASSWORD},
        expected=200,
    )
    # test() captures the new access token from the cookie.


def test_training(t):
    section("5. TRAINING")
    t.test("Get Plans", "GET", "/api/v1/training/plans", token=t.token, expected=200)
    t.test(
        "Get Progress", "GET", "/api/v1/training/progress", token=t.token, expected=200
    )


def test_chat(t):
    section("6. Chat")
    chat_resp = t.test(
        "Chat",
        "POST",
        "/api/v1/chat",
        token=t.token,
        body={"message": "plan"},
        expected=200,
    )
    if isinstance(chat_resp, dict) and chat_resp.get("answer"):
        print(f"       {GRAY}answer: {chat_resp['answer'][:50]}...{RESET}")


def test_totp(t):
    section("7. TOTP / 2FA")
    totp_setup = t.test(
        "TOTP Setup", "POST", "/api/v1/auth/2fa/setup", token=t.token, expected=200
    )
    if (
        isinstance(totp_setup, dict)
        and totp_setup.get("secret")
        and totp_setup.get("backup_codes")
    ):
        secret = totp_setup["secret"]
        backup_codes = totp_setup["backup_codes"]
        t.test(
            "TOTP Confirm (invalid code)",
            "POST",
            "/api/v1/auth/2fa/confirm",
            body={
                "passcode": "000000",
                "temp_secret": secret,
                "backup_codes": backup_codes,
            },
            token=t.token,
            expected=400,
        )
        t.test(
            "TOTP Status", "GET", "/api/v1/auth/2fa/status", token=t.token, expected=200
        )
    else:
        t.skipped += 1
        print(f"       {YELLOW}SKIP: TOTP setup unavailable{RESET}")


def test_security(t):
    section("8. SECURITY")
    t.token = None
    t.test("Profile (no token)", "GET", PROFILE_PATH, expected=404)
    t.test("Training (no token)", "GET", "/api/v1/training/plans", expected=404)


def print_summary(t):
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


def main():
    parser = argparse.ArgumentParser(description="Fitness Platform API Test Suite")
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL, help="API base URL")
    parser.add_argument(
        "--insecure", action="store_true", help="Disable SSL certificate verification"
    )
    parser.add_argument(
        "--confirm-email",
        action="store_true",
        help="Confirm the test user's email via kubectl (CI only)",
    )
    args = parser.parse_args()

    t = TestRunner(
        args.base_url, insecure=args.insecure, confirm_email=args.confirm_email
    )
    test_email = f"apitest-{random.SystemRandom().randint(1000, 9999)}@example.com"
    t.test_email = test_email

    print(f"\n{BOLD}{CYAN}{'=' * 50}{RESET}")
    print(f"{BOLD}{CYAN}   FITNESS PLATFORM — API TEST SUITE{RESET}")
    print(f"{BOLD}{CYAN}{'=' * 50}{RESET}")
    print(f"  Base URL : {args.base_url}")
    print(f"  Test User: {test_email}")
    print()

    if not test_health(t):
        print(f"\n{RED}Health check failed; skipping the remaining API tests.{RESET}")
        print_summary(t)
        return
    test_auth(t, test_email)
    test_profile(t)
    test_biometrics(t)
    test_post_logout(t)
    test_training(t)
    test_chat(t)
    test_totp(t)
    test_security(t)
    print_summary(t)


if __name__ == "__main__":
    main()
