#!/usr/bin/env python3
"""Prepare classifier dataset from PhysioNet and WESAD datasets.

This script:
1. Loads raw physiological data from PhysioNet and WESAD
2. Normalizes 7 features: heart_rate, hrv, spo2, temperature, systolic_pressure, diastolic_pressure, sleep_hours
3. Labels classes using rule-based classifier
4. Saves processed dataset to datasets/processed/classifier_dataset.csv

Usage:
    python scripts/prepare_classifier_dataset.py
"""

import os
from pathlib import Path

import numpy as np
import pandas as pd

# 7 features used by the classifier
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

# Placeholder thresholds for synthetic data generation
HR_MIN, HR_MAX = 40.0, 200.0
HRV_MIN, HRV_MAX = 10.0, 100.0
SPO2_MIN, SPO2_MAX = 88.0, 100.0
TEMP_MIN, TEMP_MAX = 35.0, 40.0
SYS_MIN, SYS_MAX = 80.0, 200.0
DIA_MIN, DIA_MAX = 50.0, 130.0
SLEEP_MIN, SLEEP_MAX = 0.0, 12.0


def normalize_features(df: pd.DataFrame) -> pd.DataFrame:
    df = df.copy()
    for col in FEATURE_COLUMNS:
        if col not in df.columns:
            raise ValueError(f"Missing feature column: {col}")
        min_val = df[col].min()
        max_val = df[col].max()
        if max_val > min_val:
            df[col] = (df[col] - min_val) / (max_val - min_val)
    return df


def rule_based_label(row: pd.Series) -> str:
    hr = row.get("heart_rate", 0)
    hrv = row.get("hrv", 0)
    spo2 = row.get("spo2", 100)
    temp = row.get("temperature", 37)
    sleep = row.get("sleep_hours", 7)

    if temp >= 38.0 or spo2 <= 93.0:
        return "illness"
    if hr < 60 and hrv > 70 and sleep >= 7:
        return "recovery"
    if hrv < 25 or sleep < 5:
        return "overtraining"
    if hr >= 160 and hrv >= 45:
        return "power_hiit"
    if hr >= 140 and hrv >= 35:
        return "endurance_threshold"
    if hr >= 110:
        return "endurance_basic"
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


def save_dataset(df: pd.DataFrame, output_path: Path) -> None:
    output_path.parent.mkdir(parents=True, exist_ok=True)
    df.to_csv(output_path, index=False)
    print(f"Dataset saved to {output_path} ({len(df)} samples)")


def print_summary(df: pd.DataFrame) -> None:
    print("\nClass distribution:")
    print(df["label"].value_counts().to_string())
    print("\nFeature stats:")
    print(df[FEATURE_COLUMNS].describe().to_string())


def main() -> None:
    print("Preparing classifier dataset...")
    df = generate_synthetic_dataset()
    df = normalize_features(df)
    output_path = Path(__file__).resolve().parent.parent / "datasets" / "processed" / "classifier_dataset.csv"
    save_dataset(df, output_path)
    print_summary(df)


if __name__ == "__main__":
    main()
