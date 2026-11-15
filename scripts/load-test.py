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


def percentile(values, pct):
    """Nearest-rank percentile over a sorted-able list of numbers."""
    if not values:
        return None
    ordered = sorted(values)
    index = min(len(ordered) - 1, max(0, round(pct / 100 * len(ordered) + 0.5) - 1))
    return ordered[index]


def print_results(results_file):
    """Parse the k6 JSON stream and print the metrics that were measured.

    k6 `--out json=` emits one JSON object per line: `Metric` definitions and
    `Point` samples. Aggregating the points here means the summary reflects the
    run instead of printing fixed placeholders.
    """
    if not Path(results_file).exists():
        return

    durations = []
    counters = {}
    rates = {}
    with open(results_file, "r", encoding="utf-8", errors="replace") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                record = json.loads(line)
            except json.JSONDecodeError:
                continue
            if record.get("type") != "Point":
                continue
            metric = record.get("metric")
            value = record.get("data", {}).get("value")
            if not metric or value is None:
                continue
            if metric == "http_req_duration":
                durations.append(value)
            elif metric in ("http_reqs", "iterations", "api_errors_total"):
                counters[metric] = counters.get(metric, 0) + value
            elif metric in ("errors", "checks"):
                rates.setdefault(metric, []).append(value)

    if not durations and not counters:
        print(f"{YELLOW}No k6 metrics found in {results_file}{RESET}")
        return

    print("\nKey Metrics:")
    print(f"  HTTP requests : {int(counters.get('http_reqs', 0))}")
    if durations:
        p95 = percentile(durations, 95)
        p99 = percentile(durations, 99)
        print(f"  P95 response  : {p95:.0f} ms")
        print(f"  P99 response  : {p99:.0f} ms")
        print(
            f"  {GREEN}P95 < 2000 ms{RESET}"
            if p95 < 2000
            else f"  {RED}P95 >= 2000 ms{RESET}"
        )
        print(
            f"  {GREEN}P99 < 5000 ms{RESET}"
            if p99 < 5000
            else f"  {RED}P99 >= 5000 ms{RESET}"
        )
    if "errors" in rates and rates["errors"]:
        error_rate = rates["errors"][-1]
        print(f"  Error rate    : {error_rate * 100:.2f}%")
        if error_rate >= 0.1:
            print(f"  {RED}Error rate >= 10%{RESET}")
    if "checks" in rates and rates["checks"]:
        print(f"  Checks passed : {rates['checks'][-1] * 100:.2f}%")
    print()


def find_k6():
    k6_path = shutil.which("k6")
    if not k6_path:
        common_paths = [
            "/usr/local/bin/k6",
            "/usr/bin/k6",
            "/opt/homebrew/bin/k6",
            "~/bin/k6",
            "./k6",
        ]
        if os.name == "nt":
            common_paths = [
                os.path.join(os.environ.get("ProgramFiles", r"C:\Program Files"), "k6", "k6.exe"),
                os.path.join(
                    os.environ.get("LOCALAPPDATA", ""), "Microsoft", "WinGet", "Links", "k6.exe"
                ),
                "k6.exe",
                "./k6.exe",
            ]
        for path in common_paths:
            expanded_path = os.path.expanduser(path)
            if os.path.isfile(expanded_path) and os.access(expanded_path, os.X_OK):
                return expanded_path
    return k6_path


def build_k6_command(k6_path, args):
    cmd = [
        k6_path,
        "run",
        "--out",
        f"json={args.output}",
        str(LOAD_TEST_K6),
        "--env",
        f"BASE_URL={args.base_url}",
        "--env",
        f"K6_DURATION={args.duration}",
        "--env",
        f"K6_VUS={args.vus}",
        "--env",
        f"INSECURE_SKIP_TLS_VERIFY={'true' if args.insecure else 'false'}",
    ]
    return cmd


def run_load_test(cmd):
    try:
        result = subprocess.run(cmd, timeout=1200)

        if result.returncode == 0:
            print(f"\n{GREEN}{BOLD} LOAD TEST COMPLETED SUCCESSFULLY!{RESET}\n")
            return 0
        else:
            print(
                f"\n{RED}{BOLD} LOAD TEST FAILED (exit code {result.returncode}){RESET}\n"
            )
            return result.returncode

    except subprocess.TimeoutExpired:
        print(f"\n{RED}Load test timed out (20 min)!{RESET}\n")
        return 1

    except KeyboardInterrupt:
        print(f"\n{YELLOW} Load test interrupted by user{RESET}\n")
        return 130

    except FileNotFoundError:
        print(f"\n{RED}Error: k6 executable not found{RESET}\n")
        return 1


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
    parser.add_argument(
        "--base-url", default="https://localhost:8443", help="API base URL"
    )
    parser.add_argument(
        "--duration", default="8m", help="Test duration (e.g. 1m, 5m, 30s)"
    )
    parser.add_argument(
        "--vus", type=int, default=50, help="Max number of virtual users"
    )
    parser.add_argument("--insecure", action="store_true", help="Skip TLS verification")
    parser.add_argument(
        "--output", default="results.json", help="Output file for results"
    )
    args = parser.parse_args()

    k6_path = find_k6()
    if not k6_path:
        print(f"{RED}k6 not found in PATH or common locations!{RESET}")
        print(
            f"{YELLOW}Install from: https://k6.io/docs/getting-started/installation/{RESET}"
        )
        if os.name == "nt":
            print(
                f"{YELLOW}Windows: winget install k6 --source winget "
                f"(k6.exe must be on PATH){RESET}\n"
            )
        else:
            print(
                f"{YELLOW}Or download and place in one of: /usr/local/bin/k6, "
                f"/usr/bin/k6, /opt/homebrew/bin/k6{RESET}\n"
            )
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

    cmd = build_k6_command(k6_path, args)
    print(f"{YELLOW}Running load test...{RESET}\n")

    exit_code = run_load_test(cmd)
    # Print the measured summary whenever k6 produced one — including threshold
    # failures (k6 exits 99), where the metrics matter most.
    if Path(args.output).exists():
        print_results(args.output)
    sys.exit(exit_code)


if __name__ == "__main__":
    main()
