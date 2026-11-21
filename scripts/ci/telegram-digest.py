#!/usr/bin/env python3
"""Telegram digest of every GitHub Actions annotation for a commit.

GitHub turns every ``::warning::``, ``::error::``, ``::notice::`` and
``::deprecation::`` command (plus tool-generated annotations: CodeQL, Trivy,
SonarCloud, Super-Linter, ...) into a *check-run annotation*.  This script
collects all of them for one commit and posts a single grouped digest to
Telegram, so warnings and errors from the whole pipeline land in one place.

Standard library only - it must run on a bare ubuntu-latest runner.

Required environment:
    GITHUB_TOKEN       token with ``checks: read`` (``github.token`` is enough)
    REPO               owner/repo
    SHA                commit SHA
    TELEGRAM_BOT_TOKEN bot token
    TELEGRAM_CHAT_ID   target chat

Optional:
    GITHUB_API_URL     default https://api.github.com
    SERVER_URL         default https://github.com
    RUN_ID             workflow run id (for the link)
    TELEGRAM_DRY_RUN   1 - print the digest instead of sending it

The script never fails the workflow: digest delivery is best-effort.
"""

from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request

API_URL = os.environ.get("GITHUB_API_URL", "https://api.github.com").rstrip("/")
SERVER_URL = os.environ.get("SERVER_URL") or os.environ.get(
    "GITHUB_SERVER_URL", "https://github.com"
)
SERVER_URL = SERVER_URL.rstrip("/")
REPO = os.environ.get("REPO") or os.environ.get("GITHUB_REPOSITORY", "")
SHA = os.environ.get("SHA") or os.environ.get("GITHUB_SHA", "")
RUN_ID = os.environ.get("RUN_ID") or os.environ.get("GITHUB_RUN_ID", "")
TOKEN = os.environ.get("GITHUB_TOKEN") or os.environ.get("GH_TOKEN", "")
BOT_TOKEN = os.environ.get("TELEGRAM_BOT_TOKEN", "")
CHAT_ID = os.environ.get("TELEGRAM_CHAT_ID", "")
DRY_RUN = os.environ.get("TELEGRAM_DRY_RUN", "").lower() not in ("", "0", "false", "no")

# Telegram hard limit is 4096 chars; stay well below it.
MAX_MESSAGE_CHARS = 3500
# Keep the digest readable: cap items per level and clip long messages.
MAX_ITEMS_PER_LEVEL = 150
MAX_ITEM_CHARS = 300
MAX_TITLE_CHARS = 60

LEVEL_ORDER = ["failure", "error", "warning", "notice"]
LEVEL_ICON = {
    "failure": "\u274c",
    "error": "\u274c",
    "warning": "\u26a0\ufe0f",
    "notice": "\u2139\ufe0f",
}
LEVEL_TITLE = {
    "failure": "FAILURES",
    "error": "ERRORS",
    "warning": "WARNINGS",
    "notice": "NOTICES / DEPRECATIONS",
}


def log(message: str) -> None:
    """Diagnostics go to stderr so they never pollute step output parsing."""
    print(message, file=sys.stderr)


def api_get(path: str, collection_key: str | None = None) -> list[dict]:
    """GET a paginated GitHub API endpoint and return every item.

    Some endpoints (``/commits/{ref}/check-runs``) wrap the payload in an
    object instead of returning a bare array - ``collection_key`` names the
    field holding the list.
    """
    url = f"{API_URL}{path}"
    items: list[dict] = []
    headers = {
        "Authorization": f"Bearer {TOKEN}",
        "Accept": "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
        "User-Agent": "fitpulse-ci-digest",
    }
    while url:
        request = urllib.request.Request(url, headers=headers)
        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                payload = json.load(response)
                link_header = response.headers.get("Link", "")
        except urllib.error.HTTPError as error:  # noqa: PERF203 - report and stop
            log(f"GitHub API {error.code} for {url}: {error.reason}")
            break
        except (urllib.error.URLError, TimeoutError) as error:
            log(f"GitHub API unreachable for {url}: {error}")
            break

        if isinstance(payload, list):
            items.extend(payload)
        elif collection_key and isinstance(payload.get(collection_key), list):
            items.extend(payload[collection_key])
        else:
            items.append(payload)

        url = ""
        for part in link_header.split(","):
            if 'rel="next"' in part:
                url = part.split(";")[0].strip().strip("<>")
                break
    return items


def collect_annotations() -> list[dict]:
    """Return every annotation of every check-run of the commit."""
    check_runs = api_get(
        f"/repos/{REPO}/commits/{SHA}/check-runs?filter=latest&per_page=100",
        collection_key="check_runs",
    )
    annotations: list[dict] = []
    for check_run in check_runs:
        check_name = check_run.get("name") or "unknown"
        run_id = check_run.get("id")
        if run_id is None:
            continue
        for item in api_get(
            f"/repos/{REPO}/check-runs/{run_id}/annotations?per_page=100"
        ):
            annotations.append(
                {
                    "level": (item.get("annotation_level") or "notice").lower(),
                    "check": check_name,
                    "title": item.get("title") or "",
                    "message": item.get("message") or "",
                    "path": item.get("path") or "",
                    "line": item.get("start_line"),
                }
            )
    return annotations


def escape_html(text: str) -> str:
    return text.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def clip(text: str, limit: int) -> str:
    """Collapse whitespace and clip to ``limit`` characters."""
    flat = " ".join((text or "").split())
    if len(flat) <= limit:
        return flat
    return flat[: max(1, limit - 1)].rstrip() + "\u2026"


def group_by_level(annotations: list[dict]) -> dict[str, list[dict]]:
    grouped: dict[str, list[dict]] = {level: [] for level in LEVEL_ORDER}
    for annotation in annotations:
        level = annotation["level"]
        if level not in grouped:
            # Unknown levels are surfaced as warnings rather than dropped.
            level = "notice"
        grouped[level].append(annotation)
    return grouped


def location(annotation: dict) -> str:
    path = annotation.get("path") or ""
    line = annotation.get("line")
    if not path:
        return ""
    return f"{path}:{line}" if line else path


def render_item(annotation: dict) -> str:
    check = clip(annotation["check"], MAX_TITLE_CHARS)
    where = location(annotation)
    detail = clip(annotation["message"], MAX_ITEM_CHARS)

    parts = [f"\u2022 <b>{escape_html(check)}</b>"]
    if where:
        parts.append(f"<code>{escape_html(clip(where, 120))}</code>")
    if detail:
        parts.append(escape_html(detail))
    return " ".join(parts)


def plural(count: int, word: str) -> str:
    return f"{count} {word}" if count == 1 else f"{count} {word}s"


def build_header(grouped: dict[str, list[dict]]) -> str:
    counts = ", ".join(
        plural(len(grouped[level]), level) for level in LEVEL_ORDER if grouped[level]
    )
    summary = counts if counts else "no annotations"
    return "\U0001f9fe <b>CI digest</b> \u2014 " + summary


def build_run_url() -> str:
    """Link to the workflow run, falling back to the commit and then to Actions."""
    if RUN_ID:
        return f"{SERVER_URL}/{REPO}/actions/runs/{RUN_ID}"
    if SHA:
        return f"{SERVER_URL}/{REPO}/commit/{SHA}"
    return f"{SERVER_URL}/{REPO}/actions"


def level_blocks(level: str, items: list[dict]) -> list[str]:
    """Render one level section: header, items and overflow marker."""
    blocks = [f"{LEVEL_ICON[level]} <b>{LEVEL_TITLE[level]} ({len(items)})</b>"]
    blocks.extend(render_item(annotation) for annotation in items[:MAX_ITEMS_PER_LEVEL])
    hidden = len(items) - MAX_ITEMS_PER_LEVEL
    if hidden > 0:
        blocks.append(f"\u2026 и ещё {hidden}")
    return blocks


def pack_blocks(blocks: list[str]) -> list[str]:
    """Pack blocks into chunks that fit into a single Telegram message."""
    chunks: list[str] = []
    current = ""
    for block in blocks:
        candidate = f"{current}\n{block}" if current else block
        if current and len(candidate) > MAX_MESSAGE_CHARS:
            chunks.append(current)
            current = block
        else:
            current = candidate
    if current:
        chunks.append(current)
    return chunks


def build_messages(annotations: list[dict]) -> list[str]:
    """Render the digest, split into Telegram-sized chunks."""
    grouped = group_by_level(annotations)

    blocks: list[str] = []
    for level in LEVEL_ORDER:
        if grouped[level]:
            blocks.extend(level_blocks(level, grouped[level]))
    if not blocks:
        blocks.append("✅ В логах прогона нет warnings/errors/notes")
    blocks.append(f'<a href="{escape_html(build_run_url())}">Открыть запуск</a>')

    chunks = pack_blocks(blocks)
    header = build_header(grouped)
    if len(chunks) == 1:
        return [f"{header}\n\n{chunks[0]}"]
    return [
        f"{header} ({index}/{len(chunks)})\n\n{body}"
        for index, body in enumerate(chunks, start=1)
    ]


def send_telegram(text: str) -> None:
    if DRY_RUN:
        try:
            print(text)
        except UnicodeEncodeError:
            # Windows consoles default to cp1251/cp866 and cannot render emoji.
            print(text.encode("ascii", "replace").decode("ascii"))
        return
    if not BOT_TOKEN or not CHAT_ID:
        log("TELEGRAM_BOT_TOKEN / TELEGRAM_CHAT_ID not set - skipping digest")
        return

    payload = json.dumps(
        {
            "chat_id": CHAT_ID,
            "text": text,
            "parse_mode": "HTML",
            "disable_web_page_preview": True,
        }
    ).encode("utf-8")
    request = urllib.request.Request(
        f"https://api.telegram.org/bot{BOT_TOKEN}/sendMessage",
        data=payload,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            body = json.load(response)
    except urllib.error.HTTPError as error:
        log(f"Telegram API {error.code}: {error.reason}")
        return
    except (urllib.error.URLError, TimeoutError) as error:
        log(f"Telegram API unreachable: {error}")
        return

    if not body.get("ok"):
        log(f"Telegram rejected the message: {body}")


def main() -> None:
    if not REPO or not SHA:
        log("REPO/SHA are not set - nothing to collect")
        return
    if not TOKEN:
        log("GITHUB_TOKEN is not set - cannot read check-runs")
        return

    annotations = collect_annotations()
    log(f"collected {len(annotations)} annotations for {REPO}@{SHA[:8]}")

    for message in build_messages(annotations):
        send_telegram(message)


if __name__ == "__main__":
    main()
