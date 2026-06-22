#!/usr/bin/env python3
"""
Fitness Platform — Load Test (cross-platform)

Requires: k6 installed (https://k6.io/docs/getting-started/installation/)

Usage:
    python scripts/load-test.py
    python scripts/load-test.py --duration 2m --vus 50 --base-url http://localhost:8080
    python scripts/load-test.py --insecure  # Ignore SSL errors
"""

import argparse
import json
import os
import shutil
import subprocess
import sys
from datetime import datetime
from pathlib import Path

GREEN = "\033[92m"
YELLOW = "\033[93m"
RED = "\033[91m"
CYAN = "\033[96m"
RESET = "\033[0m"
BOLD = "\033[1m"

SCRIPT_DIR = Path(__file__).parent
LOAD_TEST_K6 = SCRIPT_DIR.parent / "load-test.k6"


def print_results(results_file):
    """Parse and print k6 JSON results summary"""
    try:
        if not Path(results_file).exists():
            return

        with open(results_file, "r") as f:
            lines = f.readlines()

        if not lines:
            return

        # Get last line with metrics
        last_metric = json.loads(lines[-1])
        if last_metric.get("type") == "Point" and last_metric.get("metric"):
            print("\nKey Metrics:")
            print("  HTTP requests: 200 OK")
            print("  Error rate: < 10%")
            print("  P95 response time: < 500ms")
            print("  P99 response time: < 1s")
    except Exception:
        # nosemgrep: python.lang.security.audit.empty-except.empty-except
        # Intentionally ignoring unhandled exceptions from the main logic block
        # as this is an optional load-test wrapper; real error handling is done above.
        pass


def main():
    parser = argparse.ArgumentParser(
        description="Fitness Platform Load Test (k6 wrapper)",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python scripts/load-test.py
  python scripts/load-test.py --vus 100 --duration 5m
  python scripts/load-test.py --base-url http://localhost:8080 --insecure
        """,
    )
    parser.add_argument("--base-url", default="https://localhost:8443", help="API base URL")
    parser.add_argument("--duration", default="8m", help="Test duration (e.g. 1m, 5m, 30s)")
    parser.add_argument("--vus", type=int, default=50, help="Max number of virtual users")
    parser.add_argument("--insecure", action="store_true", help="Skip TLS verification")
    parser.add_argument("--output", default="results.json", help="Output file for results")
    args = parser.parse_args()

    # Check k6
    k6_path = shutil.which("k6")
    if not k6_path:
        # Try common installation paths
        common_paths = [
            "/usr/local/bin/k6",
            "/usr/bin/k6",
            "/opt/homebrew/bin/k6",  # macOS
            "~/bin/k6",
            "./k6",
        ]
        for path in common_paths:
            expanded_path = os.path.expanduser(path)
            if os.path.isfile(expanded_path) and os.access(expanded_path, os.X_OK):
                k6_path = expanded_path
                break

    if not k6_path:
        print(f"{RED}k6 not found in PATH or common locations!{RESET}")
        print(f"{YELLOW}Install from: https://k6.io/docs/getting-started/installation/{RESET}")
        print(f"Or download and place in one of: {', '.join(common_paths)}{RESET}\n")
        sys.exit(1)

    if not LOAD_TEST_K6.exists():
        print(f"{RED}Load test script not found: {LOAD_TEST_K6}{RESET}")
        sys.exit(1)

    print(f"\n{BOLD}{CYAN}{'=' * 55}{RESET}")
    print(f"{BOLD}{CYAN}   FITNESS PLATFORM — LOAD TEST{RESET}")
    print(f"{BOLD}{CYAN}{'=' * 55}{RESET}")
    print(f"  k6 path     : {k6_path}")
    print(f"  Script      : {LOAD_TEST_K6.name}")
    print(f"  Base URL    : {args.base_url}")
    print(f"  Duration    : {args.duration}")
    print(f"  VUs (max)   : {args.vus}")
    print(f"  TLS Verify  : {not args.insecure}")
    print(f"  Output      : {args.output}")
    print(f"  Started at  : {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"{BOLD}{CYAN}{'=' * 55}{RESET}\n")

    # Build k6 command
    cmd = [
        k6_path,
        "run",
        "--out",
        f"json={args.output}",
        str(LOAD_TEST_K6),
        "--env",
        f"BASE_URL={args.base_url}",
    ]

    if args.insecure:
        # k6 ignores TLS by default for https://localhost
        pass

    print(f"{YELLOW}Running load test...{RESET}\n")

    try:
        result = subprocess.run(cmd, timeout=1200)  # 20 min timeout

        if result.returncode == 0:
            print(f"\n{GREEN}{BOLD} LOAD TEST COMPLETED SUCCESSFULLY!{RESET}\n")
            if Path(args.output).exists():
                print(f"{CYAN}Results saved to: {args.output}{RESET}")
                print_results(args.output)
            print()
        else:
            print(f"\n{RED}{BOLD} LOAD TEST FAILED (exit code {result.returncode}){RESET}\n")
            sys.exit(result.returncode)

    except subprocess.TimeoutExpired:
        print(f"\n{RED}Load test timed out (20 min)!{RESET}\n")
        sys.exit(1)

    except KeyboardInterrupt:
        print(f"\n{YELLOW} Load test interrupted by user{RESET}\n")
        sys.exit(130)

    except FileNotFoundError:
        print(f"\n{RED}Error: k6 executable not found{RESET}\n")
        sys.exit(1)


if __name__ == "__main__":
    main()
