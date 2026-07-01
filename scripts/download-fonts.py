#!/usr/bin/env python3
"""Download self-hosted fonts from Google Fonts and generate fonts.css."""

import os
import re
import sys
from urllib.parse import unquote, urlparse

import urllib.request


def download_fonts(family, weights, output_dir):
    family_param = family.replace(" ", "+")
    weights_param = ";".join(str(w) for w in weights)
    css_url = (
        "https://fonts.googleapis.com/css2"
        f"?family={family_param}:wght@{weights_param}&display=swap"
    )

    print(f"Fetching {family} from Google Fonts...")
    req = urllib.request.Request(
        css_url,
        headers={
            "User-Agent": (
                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
                "AppleWebKit/537.36 (KHTML, like Gecko) "
                "Chrome/120.0.0.0 Safari/537.36"
            )
        },
    )
    with urllib.request.urlopen(req, timeout=30) as resp:
        css = resp.read().decode("utf-8")

    os.makedirs(output_dir, exist_ok=True)

    def replace_url(match):
        font_url = match.group(1)
        parsed = urlparse(font_url)
        filename = unquote(os.path.basename(parsed.path))
        local_path = os.path.join(output_dir, filename)

        if not os.path.exists(local_path):
            print(f"  Downloading {filename}...")
            urllib.request.urlretrieve(font_url, local_path)
        else:
            print(f"  Skipping {filename}")

        return f"url('{filename}')"

    local_css = re.sub(
        r"""url\(["']?(https://fonts\.gstatic\.com/[^)"']+\.woff2)["']?\)""",
        replace_url,
        css,
    )
    return local_css


def main():
    output_dir = sys.argv[1] if len(sys.argv) > 1 else "web/static/fonts"

    parts = []
    parts.append(download_fonts("JetBrains Mono", [400, 700], output_dir))
    parts.append(download_fonts("Inter", [400, 500, 600, 700], output_dir))

    fonts_css = "\n".join(parts)
    fonts_css_path = os.path.join(output_dir, "fonts.css")

    with open(fonts_css_path, "w", encoding="utf-8") as f:
        f.write(fonts_css)

    count = fonts_css.count("@font-face")
    print(f"\nGenerated fonts.css with {count} @font-face rules")


if __name__ == "__main__":
    main()
