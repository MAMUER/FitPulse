#!/usr/bin/env python3
"""Экспортирует компактный набор блюд для Go-меню (go:embed).

Из 66 301 продукта отбирает сбалансированный по калориям набор, чтобы
`internal/chat` мог строить меню с учётом аллергий и диет без внешних файлов.

Запуск:
    python scripts/export_menu_food.py
"""

from __future__ import annotations

import json
from pathlib import Path

import pandas as pd

PROJECT_ROOT = Path(__file__).resolve().parent.parent
INPUT = PROJECT_ROOT / "datasets" / "processed" / "food_nutrients.csv"
OUTPUT = PROJECT_ROOT / "internal" / "chat" / "food_data.json"

TARGET_ITEMS = 1500


def main() -> None:
    df = pd.read_csv(INPUT)
    df = df[df["kcal"] > 0].dropna(subset=["name"])
    df["kcal"] = pd.to_numeric(df["kcal"], errors="coerce").fillna(0)
    df = df[(df["kcal"] > 50) & (df["kcal"] < 900)]

    # Стратифицируем по калориям, чтобы были и лёгкие, и сытные блюда.
    df["bucket"] = pd.cut(df["kcal"], bins=[0, 150, 300, 500, 900], labels=False)
    per_bucket = max(1, TARGET_ITEMS // 4)
    parts = []
    for _, group in df.groupby("bucket"):
        parts.append(group.sample(n=min(per_bucket, len(group)), random_state=42))
    sample = pd.concat(parts).head(TARGET_ITEMS)

    items = []
    for _, r in sample.iterrows():
        allergens = str(r.get("allergens", "") or "")
        items.append(
            {
                "name": str(r["name"])[:120],
                "kcal": int(r["kcal"]),
                "protein": int(r.get("protein_g", 0) or 0),
                "carbs": int(r.get("carbs_g", 0) or 0),
                "fat": int(r.get("fat_g", 0) or 0),
                "allergens": allergens[:120],
            }
        )

    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    with open(OUTPUT, "w", encoding="utf-8") as fh:
        json.dump(items, fh, ensure_ascii=False, separators=(",", ":"))
    print(f"Saved {len(items)} dishes to {OUTPUT}")


if __name__ == "__main__":
    main()