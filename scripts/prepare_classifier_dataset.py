#!/usr/bin/env python3
"""Prepare classifier dataset from existing processed data or generate synthetic fallback."""

from __future__ import annotations

import json
from pathlib import Path
from typing import NamedTuple

import numpy as np
import pandas as pd

PROJECT_ROOT = Path(__file__).resolve().parent.parent
INPUT_PATH = PROJECT_ROOT / "datasets" / "processed" / "classifier_dataset.csv"
OUTPUT_PATH = INPUT_PATH
SCALER_PATH = PROJECT_ROOT / "models" / "classifier_scaler.json"

FEATURE_COLUMNS = [
    "heart_rate",
    "hrv",
    "spo2",
    "temperature",
    "systolic_pressure",
    "diastolic_pressure",
    "sleep_hours",
]

CLASSES = [
    "recovery",
    "endurance_basic",
    "endurance_threshold",
    "power_hiit",
    "overtraining",
    "illness",
    "unknown",
]

HR_MIN, HR_MAX = 40.0, 200.0
HRV_MIN, HRV_MAX = 10.0, 100.0
SPO2_MIN, SPO2_MAX = 88.0, 100.0
TEMP_MIN, TEMP_MAX = 35.0, 40.0
SYS_MIN, SYS_MAX = 80.0, 200.0
DIA_MIN, DIA_MAX = 50.0, 130.0
SLEEP_MIN, SLEEP_MAX = 0.0, 12.0

MIN_DATASET_SIZE = 10000


def validate_raw_dataset(df: pd.DataFrame) -> pd.DataFrame:
    df = df.copy()
    required = [c for c in FEATURE_COLUMNS if c in df.columns]
    if not required:
        raise ValueError("No required feature columns found in dataset")

    df = df.dropna(subset=required, how="all")

    if "heart_rate" in df.columns:
        hr = df["heart_rate"].dropna()
        if not hr.empty and hr.max() < 1.0:
            df = df.drop(columns=["heart_rate"], errors="ignore")

    return df


def normalize_features(df: pd.DataFrame) -> pd.DataFrame:
    """Min-max нормализация с сохранением параметров в models/classifier_scaler.json."""
    df = df.copy()
    scaler: dict[str, dict[str, float]] = {}
    for col in FEATURE_COLUMNS:
        if col not in df.columns:
            df[col] = np.nan
        col_min = df[col].min()
        col_max = df[col].max()
        if pd.notna(col_min) and pd.notna(col_max) and col_max > col_min:
            df[col] = (df[col] - col_min) / (col_max - col_min)
            scaler[col] = {"min": float(col_min), "max": float(col_max)}
        else:
            scaler[col] = {"min": 0.0, "max": 1.0}
    SCALER_PATH.parent.mkdir(parents=True, exist_ok=True)
    with open(SCALER_PATH, "w") as f:
        json.dump(scaler, f, indent=2)
    print(f"Scaler saved to {SCALER_PATH}")
    return df


class Metrics(NamedTuple):
    """Физиологические признаки одной записи."""

    hr: float
    hrv: float
    spo2: float
    temp: float
    sleep: float


def _is_illness(m: Metrics) -> bool:
    return m.temp >= 38.0 or m.spo2 <= 93.0


def _is_recovery(m: Metrics) -> bool:
    return m.hr < 60 and m.hrv > 70 and m.sleep >= 7


def _is_overtraining(m: Metrics) -> bool:
    return m.hrv < 25 or m.sleep < 5


def _is_power_hiit(m: Metrics) -> bool:
    return m.hr >= 160 and m.hrv >= 45


def _is_endurance_threshold(m: Metrics) -> bool:
    return m.hr >= 140 and m.hrv >= 35


def _is_endurance_basic(m: Metrics) -> bool:
    return m.hr >= 110


# Порядок важен: первое сработавшее правило определяет класс.
LABEL_RULES = (
    ("illness", _is_illness),
    ("recovery", _is_recovery),
    ("overtraining", _is_overtraining),
    ("power_hiit", _is_power_hiit),
    ("endurance_threshold", _is_endurance_threshold),
    ("endurance_basic", _is_endurance_basic),
)


def rule_based_label(row: pd.Series) -> str:
    metrics = Metrics(
        hr=float(row.get("heart_rate", 0) or 0),
        hrv=float(row.get("hrv", 0) or 0),
        spo2=float(row.get("spo2", 100) or 100),
        temp=float(row.get("temperature", 37) or 37),
        sleep=float(row.get("sleep_hours", 7) or 7),
    )
    for label, predicate in LABEL_RULES:
        if predicate(metrics):
            return label
    return "unknown"


def generate_synthetic_dataset(n_samples: int = 2000, seed: int = 42) -> pd.DataFrame:
    rng = np.random.default_rng(seed)
    data = {
        "heart_rate": rng.uniform(HR_MIN, HR_MAX, size=n_samples),
        "hrv": rng.uniform(HRV_MIN, HRV_MAX, size=n_samples),
        "spo2": rng.uniform(SPO2_MIN, SPO2_MAX, size=n_samples),
        "temperature": rng.uniform(TEMP_MIN, TEMP_MAX, size=n_samples),
        "systolic_pressure": rng.uniform(SYS_MIN, SYS_MAX, size=n_samples),
        "diastolic_pressure": rng.uniform(DIA_MIN, DIA_MAX, size=n_samples),
        "sleep_hours": rng.uniform(SLEEP_MIN, SLEEP_MAX, size=n_samples),
    }
    df = pd.DataFrame(data)
    df["label"] = df.apply(rule_based_label, axis=1)
    return df


def balance_dataset(
    df: pd.DataFrame, per_class: int = 20000, max_oversample: int = 5
) -> pd.DataFrame:
    """Балансирует классы, сохраняя реальное распределение признаков.

    Реальные датасеты (WESAD/ADARP/PPG_DaLiA и др.) дают сильный перекос:
    тысячи «endurance_basic» и единицы «recovery»/«illness». Без балансировки
    линейная модель (и бустинг) предсказывают только мажоритарные классы.
    """
    parts = []
    for label, group in df.groupby("label"):
        if len(group) >= per_class:
            parts.append(group.sample(n=per_class, random_state=42))
        else:
            target = min(per_class, len(group) * max_oversample)
            parts.append(group.sample(n=target, replace=True, random_state=42))
    balanced = pd.concat(parts, ignore_index=True)
    balanced = balanced.sample(frac=1.0, random_state=42).reset_index(drop=True)
    print(
        "Балансировка классов (до %d на класс): %s"
        % (per_class, balanced["label"].value_counts().to_dict())
    )
    return balanced


# Физиологически возможные диапазоны. Публичные датасеты (WESAD, ADARP и др.)
# содержат пропуски в виде 0 и выбросы (в ЧСС попадают метки времени), из-за
# которых min-max ломается: max heart_rate ~1.6e12. Такие строки отбрасываем.
VALID_RANGES = {
    "heart_rate": (25.0, 230.0),
    "hrv": (1.0, 400.0),
    "spo2": (70.0, 100.0),
    "temperature": (30.0, 43.0),
    "systolic_pressure": (70.0, 260.0),
    "diastolic_pressure": (40.0, 160.0),
    "sleep_hours": (0.0, 24.0),
}


def clean_dataset(df: pd.DataFrame) -> pd.DataFrame:
    df = df.copy()
    subset = [c for c in FEATURE_COLUMNS if c in df.columns] + ["label"]
    df = df.drop_duplicates(subset=subset, keep="first")
    df = df.dropna(subset=FEATURE_COLUMNS, how="all")
    for col, (low, high) in VALID_RANGES.items():
        if col in df.columns:
            series = df[col]
            # NaN допустим: признак может отсутствовать в источнике.
            df = df[series.isna() | ((series >= low) & (series <= high))]
    return df


def main() -> None:
    print("Preparing classifier dataset...", flush=True)

    already_normalized = False
    if INPUT_PATH.exists():
        print(f"Loading existing dataset from {INPUT_PATH}", flush=True)
        df = pd.read_csv(INPUT_PATH)
        print(f"Loaded {len(df)} samples", flush=True)

        df = validate_raw_dataset(df)
        df = clean_dataset(df)
        print(f"After cleaning: {len(df)} samples", flush=True)

        if len(df) < MIN_DATASET_SIZE:
            print(
                f"WARNING: Dataset too small after cleaning ({len(df)} < {MIN_DATASET_SIZE}). Generating synthetic fallback.",
                flush=True,
            )
            df = generate_synthetic_dataset()
        elif "heart_rate" in df.columns and df["heart_rate"].max() <= 1.0:
            # Повторный запуск не должен сжимать признаки второй раз.
            already_normalized = True
            print("Признаки уже в [0,1] — повторная нормализация пропущена", flush=True)
    else:
        print("No existing dataset found. Generating synthetic dataset.", flush=True)
        df = generate_synthetic_dataset()

    if already_normalized:
        df = df.copy()
    else:
        df = normalize_features(df)

    for col in FEATURE_COLUMNS:
        if col not in df.columns:
            df[col] = np.nan

    df = df[FEATURE_COLUMNS + ["label"]]
    df = balance_dataset(df)

    OUTPUT_PATH.parent.mkdir(parents=True, exist_ok=True)
    df.to_csv(OUTPUT_PATH, index=False)
    print(f"Dataset saved to {OUTPUT_PATH} ({len(df)} samples)")

    print("\nClass distribution:")
    print(df["label"].value_counts().to_string())
    print("\nFeature stats:")
    print(df[FEATURE_COLUMNS].describe().to_string())


if __name__ == "__main__":
    main()
