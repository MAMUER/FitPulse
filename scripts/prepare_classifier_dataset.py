#!/usr/bin/env python3
"""Prepare classifier dataset from existing processed data or generate synthetic fallback."""

from __future__ import annotations

from pathlib import Path

import numpy as np
import pandas as pd

PROJECT_ROOT = Path(__file__).resolve().parent.parent
INPUT_PATH = PROJECT_ROOT / "datasets" / "processed" / "classifier_dataset.csv"
OUTPUT_PATH = INPUT_PATH

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
    df = df.copy()
    for col in FEATURE_COLUMNS:
        if col not in df.columns:
            df[col] = np.nan
            continue
        col_min = df[col].min()
        col_max = df[col].max()
        if col_max > col_min:
            df[col] = (df[col] - col_min) / (col_max - col_min)
    return df


def rule_based_label(row: pd.Series) -> str:
    hr = float(row.get("heart_rate", 0) or 0)
    hrv = float(row.get("hrv", 0) or 0)
    spo2 = float(row.get("spo2", 100) or 100)
    temp = float(row.get("temperature", 37) or 37)
    sleep = float(row.get("sleep_hours", 7) or 7)

    labels = [
        ("illness", temp >= 38.0 or spo2 <= 93.0),
        ("recovery", hr < 60 and hrv > 70 and sleep >= 7),
        ("overtraining", hrv < 25 or sleep < 5),
        ("power_hiit", hr >= 160 and hrv >= 45),
        ("endurance_threshold", hr >= 140 and hrv >= 35),
        ("endurance_basic", hr >= 110),
    ]
    for label, condition in labels:
        if condition:
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


def clean_dataset(df: pd.DataFrame) -> pd.DataFrame:
    df = df.copy()
    subset = [c for c in FEATURE_COLUMNS if c in df.columns] + ["label"]
    df = df.drop_duplicates(subset=subset, keep="first")
    df = df.dropna(subset=FEATURE_COLUMNS, how="all")
    return df


def main() -> None:
    print("Preparing classifier dataset...")

    if INPUT_PATH.exists():
        print(f"Loading existing dataset from {INPUT_PATH}")
        df = pd.read_csv(INPUT_PATH)
        print(f"Loaded {len(df)} samples")

        df = validate_raw_dataset(df)
        df = clean_dataset(df)
        print(f"After cleaning: {len(df)} samples")

        if len(df) < MIN_DATASET_SIZE:
            print(
                f"WARNING: Dataset too small after cleaning ({len(df)} < {MIN_DATASET_SIZE}). Generating synthetic fallback."
            )
            df = generate_synthetic_dataset()
    else:
        print("No existing dataset found. Generating synthetic dataset.")
        df = generate_synthetic_dataset()

    df = normalize_features(df)

    for col in FEATURE_COLUMNS:
        if col not in df.columns:
            df[col] = np.nan

    df = df[FEATURE_COLUMNS + ["label"]]

    OUTPUT_PATH.parent.mkdir(parents=True, exist_ok=True)
    df.to_csv(OUTPUT_PATH, index=False)
    print(f"Dataset saved to {OUTPUT_PATH} ({len(df)} samples)")

    print("\nClass distribution:")
    print(df["label"].value_counts().to_string())
    print("\nFeature stats:")
    print(df[FEATURE_COLUMNS].describe().to_string())


if __name__ == "__main__":
    main()
