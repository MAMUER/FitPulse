#!/usr/bin/env python3
"""Prepare classifier dataset from existing processed data or generate synthetic fallback."""

from __future__ import annotations

import json
import os
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


def validate_raw_dataset(df: pd.DataFrame, normalized: bool = False) -> pd.DataFrame:
    df = df.copy()
    required = [c for c in FEATURE_COLUMNS if c in df.columns]
    if not required:
        raise ValueError("No required feature columns found in dataset")

    df = df.dropna(subset=required, how="all")

    # У нормализованного датасета heart_rate уже в [0,1] — колонку трогать нельзя.
    if not normalized and "heart_rate" in df.columns:
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


def clean_dataset(df: pd.DataFrame, normalized: bool = False) -> pd.DataFrame:
    subset = [c for c in FEATURE_COLUMNS if c in df.columns] + ["label"]

    # Полный drop_duplicates на десятках миллионов строк занимает минуты.
    # По умолчанию пропускаем: дубликаты легитимны (повторные измерения).
    if os.environ.get("DEDUP", "0") == "1":
        print("  удаление дубликатов ...", flush=True)
        df = df.drop_duplicates(subset=subset, keep="first")

    print("  удаление all-NaN строк ...", flush=True)
    df = df.dropna(subset=subset, how="all")

    # Фильтр выбросов применяем только к сырым значениям: у нормализованного
    # датасета признаки уже в [0,1], и сырые пороги отбросили бы всё.
    if not normalized:
        for col, (low, high) in VALID_RANGES.items():
            if col in df.columns:
                series = df[col]
                df = df[series.isna() | ((series >= low) & (series <= high))]
    return df


def load_main_dataset() -> tuple[pd.DataFrame, bool]:
    """Загружает основной датасет и определяет, нормализован ли он."""
    if not INPUT_PATH.exists():
        print("No existing dataset found. Generating synthetic dataset.", flush=True)
        return generate_synthetic_dataset(), False

    print(f"Loading existing dataset from {INPUT_PATH}", flush=True)
    df = pd.read_csv(INPUT_PATH)
    print(f"Loaded {len(df)} samples", flush=True)
    normalized = bool("heart_rate" in df.columns and df["heart_rate"].max() <= 1.0)
    return df, normalized


def merge_sleep_dataset(df: pd.DataFrame, normalized: bool) -> pd.DataFrame:
    """Подмешивает sleep_dataset.csv (сон + SpO2 из PhysioNet)."""
    sleep_path = INPUT_PATH.parent / "sleep_dataset.csv"
    if not sleep_path.exists():
        return df

    sleep_df = pd.read_csv(sleep_path)
    if sleep_df.empty:
        return df

    if normalized:
        sleep_df = normalize_with_scaler(sleep_df)

    merged = pd.concat([df, sleep_df[FEATURE_COLUMNS + ["label"]]], ignore_index=True)
    print(f"Добавлен sleep_dataset.csv: +{len(sleep_df)} строк", flush=True)
    return merged


def normalize_with_scaler(df: pd.DataFrame) -> pd.DataFrame:
    """Приводит sleep-датасет к той же [0,1]-шкале, что и основной."""
    scaler_path = PROJECT_ROOT / "models" / "classifier_scaler.json"
    if not scaler_path.exists():
        return df
    with open(scaler_path, encoding="utf-8") as fh:
        scaler = json.load(fh)
    for col in FEATURE_COLUMNS:
        if col in df.columns:
            low = scaler[col]["min"]
            span = scaler[col]["max"] - low
            if span > 0:
                df[col] = (df[col] - low) / span
    return df


def finalize_dataset(df: pd.DataFrame, normalized: bool) -> pd.DataFrame:
    """Чистит, нормализует и балансирует датасет."""
    df = validate_raw_dataset(df, normalized)
    df = clean_dataset(df, normalized)
    print(f"After cleaning: {len(df)} samples", flush=True)

    if len(df) < MIN_DATASET_SIZE:
        print(
            f"WARNING: Dataset too small after cleaning ({len(df)} < {MIN_DATASET_SIZE}). Generating synthetic fallback.",
            flush=True,
        )
        return balance_dataset(generate_synthetic_dataset())

    if normalized:
        print("Признаки уже в [0,1] — повторная нормализация пропущена", flush=True)
    else:
        df = normalize_features(df)

    for col in FEATURE_COLUMNS:
        if col not in df.columns:
            df[col] = np.nan

    return balance_dataset(df[FEATURE_COLUMNS + ["label"]])


def main() -> None:
    print("Preparing classifier dataset...", flush=True)

    df, normalized = load_main_dataset()
    df = merge_sleep_dataset(df, normalized)
    df = finalize_dataset(df, normalized)

    OUTPUT_PATH.parent.mkdir(parents=True, exist_ok=True)
    df.to_csv(OUTPUT_PATH, index=False)
    print(f"Dataset saved to {OUTPUT_PATH} ({len(df)} samples)")

    print("\nClass distribution:")
    print(df["label"].value_counts().to_string())
    print("\nFeature stats:")
    print(df[FEATURE_COLUMNS].describe().to_string())


if __name__ == "__main__":
    main()
