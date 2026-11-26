#!/usr/bin/env python3
"""Готовит компактный датасет продуктов (калории/нутриенты/аллергены).

Источники:
  - Open Food Facts  — datasets/raw/food/food.parquet (7.3 ГБ, читаем чанками)
  - USDA FoodData Central — datasets/raw/usda/foundation.zip, sr-legacy.zip

Результат: datasets/processed/food_nutrients.csv
    source, name, kcal, protein_g, carbs_g, fat_g, allergens

Запуск:
    python scripts/prepare_food_datasets.py
"""

from __future__ import annotations

import json
import zipfile
from pathlib import Path

import numpy as np
import pandas as pd

PROJECT_ROOT = Path(__file__).resolve().parent.parent
RAW_ROOT = PROJECT_ROOT / "datasets" / "raw"
OUTPUT = PROJECT_ROOT / "datasets" / "processed" / "food_nutrients.csv"

MAX_ROWS_PER_SOURCE = 50000


def _num(value) -> float:
    try:
        v = float(value)
    except (TypeError, ValueError):
        return 0.0
    return v if v == v else 0.0


def _row_from_nutriments(name: str, nutriments: dict, allergens: str, source: str) -> dict:
    return {
        "source": source,
        "name": str(name)[:200],
        "kcal": _num(nutriments.get("energy-kcal")),
        "protein_g": _num(nutriments.get("proteins")),
        "carbs_g": _num(nutriments.get("carbohydrates")),
        "fat_g": _num(nutriments.get("fat")),
        "allergens": allergens,
    }




def _nutriments_to_dict(value) -> dict:
    """OFF хранит nutriments как массив {name, 100g, ...}."""
    if isinstance(value, dict):
        return value
    if isinstance(value, str):
        try:
            value = json.loads(value)
        except Exception:
            return {}
        return value if isinstance(value, dict) else {}
    if isinstance(value, (list, tuple, np.ndarray)):
        out = {}
        for item in value:
            if isinstance(item, dict) and item.get("name"):
                out[str(item["name"])] = item.get("100g")
        return out
    return {}

def process_open_food_facts() -> list[dict]:
    path = RAW_ROOT / "food" / "food.parquet"
    if not path.exists():
        print("Open Food Facts: missing", flush=True)
        return []

    import pyarrow.parquet as pq

    pf = pq.ParquetFile(path)
    cols = ["product_name", "allergens_tags", "nutriments"]
    rows: list[dict] = []
    for batch in pf.iter_batches(batch_size=20000, columns=cols):
        bdf = batch.to_pandas()
        for _, record in bdf.iterrows():
            name = record.get("product_name")
            if isinstance(name, (list, tuple, np.ndarray)):
                name = name[0] if len(name) else None
            nutriments = _nutriments_to_dict(record.get("nutriments"))
            if not name or not nutriments:
                continue
            allergens = record.get("allergens_tags")
            allergens_text = ", ".join(str(a) for a in allergens) if allergens is not None else ""
            rows.append(_row_from_nutriments(name, nutriments, allergens_text, "openfoodfacts"))
            if len(rows) >= MAX_ROWS_PER_SOURCE:
                break
        if len(rows) >= MAX_ROWS_PER_SOURCE:
            break
    print(f"Open Food Facts: {len(rows)} rows", flush=True)
    return rows


def process_usda() -> list[dict]:
    rows: list[dict] = []
    for zip_name in ("foundation.zip", "sr-legacy.zip"):
        path = RAW_ROOT / "usda" / zip_name
        if not path.exists():
            continue
        with zipfile.ZipFile(path) as zf:
            csv_names = [n for n in zf.namelist() if n.lower().endswith(".csv")]
            for csv_name in csv_names:
                with zf.open(csv_name) as fh:
                    df = pd.read_csv(fh, low_memory=False)
                if "description" not in df.columns:
                    continue
                for _, r in df.head(MAX_ROWS_PER_SOURCE).iterrows():
                    rows.append(
                        {
                            "source": f"usda:{zip_name}",
                            "name": str(r["description"])[:200],
                            "kcal": _num(r.get("energy_kcal")),
                            "protein_g": _num(r.get("protein_g")),
                            "carbs_g": _num(r.get("carbohydrate_g")),
                            "fat_g": _num(r.get("fat_g")),
                            "allergens": "",
                        }
                    )
    print(f"USDA: {len(rows)} rows", flush=True)
    return rows


def main() -> None:
    rows = process_open_food_facts() + process_usda()
    if not rows:
        raise SystemExit("Нет данных о продуктах")

    df = pd.DataFrame(rows)
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    df.to_csv(OUTPUT, index=False)
    print(f"Saved {len(df)} rows to {OUTPUT}", flush=True)
    print(df["source"].value_counts().to_string(), flush=True)


if __name__ == "__main__":
    main()