#!/usr/bin/env python3
"""Remove stale .osv-scanner.toml ignores when a fixed release is available."""

from __future__ import annotations

import argparse
import json
import re
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
CONFIG = ROOT / ".osv-scanner.toml"
GO_MOD = ROOT / "go.mod"
NPM_LOCK = ROOT / "web" / "package-lock.json"
OSV_API = "https://api.osv.dev/v1/vulns/{vuln_id}"
BLOCK_RE = re.compile(r"^\[\[IgnoredVulns\]\]\s*$")
ID_RE = re.compile(r"""^\s*id\s*=\s*["']([^"']+)["']\s*$""")
GO_REQUIRE_RE = re.compile(r"^[ \t]*([^ \t\r\n]+)[ \t]+(v[^ \t\r\n]+)", re.MULTILINE)


def parse_blocks(text: str) -> tuple[list[str], list[dict]]:
    lines = text.splitlines()
    starts = [index for index, line in enumerate(lines) if BLOCK_RE.match(line)]
    preamble = lines[: starts[0]] if starts else lines
    blocks = []
    for position, start in enumerate(starts):
        end = starts[position + 1] if position + 1 < len(starts) else len(lines)
        raw = lines[start:end]
        block_id = None
        for line in raw:
            match = ID_RE.match(line)
            if match:
                block_id = match.group(1)
                break
        blocks.append({"id": block_id, "lines": raw})
    return preamble, blocks


def semver_key(version: str) -> tuple[list[int], str]:
    value = version.strip()
    if value[:1] in {"v", "V"}:
        value = value[1:]
    value = value.split("+", 1)[0]
    core, _, prerelease = value.partition("-")
    numbers = []
    for part in core.split("."):
        try:
            numbers.append(int(part))
        except ValueError:
            numbers.append(0)
    while len(numbers) < 3:
        numbers.append(0)
    return numbers, prerelease


def compare_decimal(left: str, right: str) -> int:
    return 1 if int(left) > int(right) else -1


def compare_identifier(left: str, right: str) -> int:
    if left == right:
        return 0
    left_numeric = left.isdigit()
    right_numeric = right.isdigit()
    if left_numeric and right_numeric:
        return compare_decimal(left, right)
    if left_numeric != right_numeric:
        return -1 if left_numeric else 1
    return 1 if left > right else -1


def compare_prerelease(left: str, right: str) -> int:
    if not left and not right:
        return 0
    if not left:
        return 1
    if not right:
        return -1
    left_ids = left.split(".")
    right_ids = right.split(".")
    for left_id, right_id in zip(left_ids, right_ids):
        result = compare_identifier(left_id, right_id)
        if result:
            return result
    if len(left_ids) == len(right_ids):
        return 0
    return 1 if len(left_ids) > len(right_ids) else -1


def compare_versions(left: str, right: str) -> int:
    left_numbers, left_pre = semver_key(left)
    right_numbers, right_pre = semver_key(right)
    if left_numbers != right_numbers:
        return 1 if left_numbers > right_numbers else -1
    return compare_prerelease(left_pre, right_pre)


def at_least(version: str, lower: str | None) -> bool:
    return lower in (None, "", "0") or compare_versions(version, lower) >= 0


def range_affects(version: str, version_range: dict) -> bool:
    introduced: str | None = None
    for event in version_range.get("events", []):
        if "introduced" in event:
            introduced = event["introduced"]
        elif "fixed" in event:
            if at_least(version, introduced) and compare_versions(version, event["fixed"]) < 0:
                return True
            introduced = None
        elif "last_affected" in event:
            if at_least(version, introduced) and compare_versions(version, event["last_affected"]) <= 0:
                return True
            introduced = None
    if introduced is not None and at_least(version, introduced):
        return True
    return False


def go_versions(module: str) -> list[str] | None:
    if not GO_MOD.is_file():
        return None
    text = GO_MOD.read_text(encoding="utf-8")
    versions = [match.group(2) for match in GO_REQUIRE_RE.finditer(text) if match.group(1) == module]
    return versions


def npm_versions(package: str) -> list[str] | None:
    if not NPM_LOCK.is_file():
        return None
    try:
        data = json.loads(NPM_LOCK.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return None
    versions = []
    for path, meta in data.get("packages", {}).items():
        if path == f"node_modules/{package}" or path.endswith(f"/node_modules/{package}"):
            version = meta.get("version")
            if version:
                versions.append(version)
    legacy = data.get("dependencies", {}).get(package, {})
    if not versions and legacy.get("version"):
        versions.append(legacy["version"])
    return versions


def installed_versions(package: dict) -> list[str] | None:
    name = package.get("name", "")
    ecosystem = package.get("ecosystem")
    if ecosystem == "Go":
        return go_versions(name)
    if ecosystem == "npm":
        return npm_versions(name)
    return None


def fetch_advisory(vuln_id: str) -> dict | None:
    request = urllib.request.Request(
        OSV_API.format(vuln_id=vuln_id),
        headers={"User-Agent": "fitpulse-osv-ignore-sync"},
    )
    for attempt in range(3):
        try:
            with urllib.request.urlopen(request, timeout=20) as response:
                return json.loads(response.read().decode("utf-8"))
        except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as error:
            if attempt == 2:
                print(f"warning: could not fetch {vuln_id}: {error}", file=sys.stderr)
                return None
            time.sleep(2 * (attempt + 1))
    return None


def has_fix(advisory: dict) -> bool:
    return any(
        "fixed" in event
        for affected in advisory.get("affected", [])
        for version_range in affected.get("ranges", [])
        for event in version_range.get("events", [])
    )


def ignore_is_stale(advisory: dict) -> bool:
    if not has_fix(advisory):
        return False
    evaluated = False
    for affected in advisory.get("affected", []):
        versions = installed_versions(affected.get("package", {}))
        if not versions:
            continue
        evaluated = True
        for version in versions:
            for version_range in affected.get("ranges", []):
                if range_affects(version, version_range):
                    return False
    return evaluated


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--apply", action="store_true", help="rewrite the config file")
    parser.add_argument("--check", action="store_true", help="fail if stale ignores exist")
    args = parser.parse_args()
    check_only = args.check or not args.apply

    if not CONFIG.is_file():
        print("warning: .osv-scanner.toml not found", file=sys.stderr)
        return 0

    preamble, blocks = parse_blocks(CONFIG.read_text(encoding="utf-8"))
    stale_ids = []
    kept_blocks = []
    for block in blocks:
        vuln_id = block["id"]
        if not vuln_id:
            kept_blocks.append(block)
            continue
        advisory = fetch_advisory(vuln_id)
        if advisory and ignore_is_stale(advisory):
            stale_ids.append(vuln_id)
            continue
        kept_blocks.append(block)

    if not stale_ids:
        print("OSV ignore list is up to date")
        return 0

    print(f"Stale OSV ignore entries: {', '.join(stale_ids)}")
    if check_only:
        return 1

    lines = list(preamble)
    for block in kept_blocks:
        lines.extend(block["lines"])
    updated = "\n".join(lines)
    updated = re.sub(r"\n{3,}", "\n\n", updated).rstrip("\n") + "\n"
    CONFIG.write_text(updated, encoding="utf-8")
    print(f"Removed stale OSV ignore entries: {', '.join(stale_ids)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
