#!/usr/bin/env python3
"""Готовит датасет менструального цикла для персонализации.

Источники:
  - datasets/raw/Menstrual_cycle_data_with_factors/menstrual_cycle_dataset_with_factors.csv
  - datasets/raw/Menstrual_Health_&_Productivity/{Period_Log.csv,User_Profile.csv}

Результат: datasets/processed/menstrual_dataset.csv
    user_id, age, bmi, stress, sleep_hours, cycle_length, period_length, phase, flow

Запуск:
    python scripts/prepare_menstrual_datasets.py
"""

from __future__ import annotations

from pathlib import Path

import pandas as pd

PROJECT_ROOT = Path(__file__).resolve().parent.parent
RAW_ROOT = PROJECT_ROOT / "datasets" / "raw"
OUTPUT = PROJECT_ROOT / "datasets" / "processed" / "menstrual_dataset.csv"


def load_factors() -> pd.DataFrame:
    path = RAW_ROOT / "Menstrual_cycle_data_with_factors" / "menstrual_cycle_dataset_with_factors.csv"
    if not path.exists():
        return pd.DataFrame()
    df = pd.read_csv(path)
    return pd.DataFrame(
        {
            "user_id": df["User ID"].astype(str),
            "age": pd.to_numeric(df["Age"], errors="coerce"),
            "bmi": pd.to_numeric(df["BMI"], errors="coerce"),
            "stress": pd.to_numeric(df["Stress Level"], errors="coerce"),
            "sleep_hours": pd.to_numeric(df["Sleep Hours"], errors="coerce"),
            "cycle_length": pd.to_numeric(df["Cycle Length"], errors="coerce"),
            "period_length": pd.to_numeric(df["Period Length"], errors="coerce"),
            "phase": "unknown",
            "flow": "",
        }
    )


def load_period_log() -> pd.DataFrame:
    path = RAW_ROOT / "Menstrual_Health_&_Productivity" / "Period_Log.csv"
    if not path.exists():
        return pd.DataFrame()
    df = pd.read_csv(path)
    return pd.DataFrame(
        {
            "user_id": df["user_id"].astype(str),
            "age": float("nan"),
            "bmi": float("nan"),
            "stress": pd.to_numeric(df.get("stress_score_cycle"), errors="coerce"),
            "sleep_hours": pd.to_numeric(df.get("sleep_hours_cycle"), errors="coerce"),
            "cycle_length": pd.to_numeric(df.get("cycle_length_days"), errors="coerce"),
            "period_length": float("nan"),
            "phase": df.get("cycle_phase", ""),
            "flow": df.get("flow_level", ""),
        }
    )


def main() -> None:
    frames = [f for f in (load_factors(), load_period_log()) if not f.empty]
    if not frames:
        raise SystemExit("Нет данных менструального цикла")

    df = pd.concat(frames, ignore_index=True)
    df = df.dropna(subset=["age", "cycle_length"], how="all")
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    df.to_csv(OUTPUT, index=False)
    print(f"Saved {len(df)} rows to {OUTPUT}", flush=True)
    print(df[["age", "bmi", "stress", "sleep_hours", "cycle_length"]].describe().to_string(), flush=True)


if __name__ == "__main__":
    main()