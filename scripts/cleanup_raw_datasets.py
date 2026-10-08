#!/usr/bin/env python3
"""Remove .DS_Store files, MUSE folders, and tags_*.csv files from raw datasets."""

from __future__ import annotations

import shutil
from pathlib import Path

RAW_ROOT = Path(__file__).resolve().parent.parent / "datasets" / "raw"


def main() -> None:
    removed_ds = 0
    removed_dirs = 0
    removed_tags = 0

    for path in RAW_ROOT.rglob("*"):
        if not path.exists():
            continue
        if path.is_dir() and path.name == "MUSE":
            shutil.rmtree(path)
            removed_dirs += 1
            continue
        if path.is_file():
            if path.name == ".DS_Store":
                path.unlink()
                removed_ds += 1
            elif (
                path.name.startswith("tags_") and path.suffix.lower() == ".csv"
            ) or path.suffix.lower() == ".txt":
                path.unlink()
                removed_tags += 1

    print(f"CLEANUP: removed {removed_ds} .DS_Store files")
    print(f"CLEANUP: removed {removed_dirs} MUSE directories")
    print(f"CLEANUP: removed {removed_tags} tags_*.csv files")


if __name__ == "__main__":
    main()
