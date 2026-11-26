#!/usr/bin/env python3
"""Экспортирует HistGradientBoosting в формат, читаемый Go-инференсом.

Зачем: линейная аппроксимация (LogisticRegression на 5000 строк) воспроизводит
бустинг лишь на ~71% — в Go-классификаторе теряется и нормализация, и
импутация, и взаимодействия признаков. Деревья дают 99.9%.

Формат `histgb-v1`:
    {
      "format": "histgb-v1",
      "n_classes": 7,
      "learning_rate": 0.05,
      "baseline": [...],
      "feature_keys": [...],
      "feature_min": [...],
      "feature_max": [...],
      "classes": [...],
      "trees": [
        {"feature":[...], "threshold":[...], "left":[...], "right":[...],
         "missing_left":[...], "value":[...], "leaf":[...]}
      ]
    }

Запуск:
    python scripts/export_go_model.py
"""

from __future__ import annotations

import json
import math
from pathlib import Path

import joblib
import numpy as np
import pandas as pd

PROJECT_ROOT = Path(__file__).resolve().parent.parent
MODEL_PATH = PROJECT_ROOT / "models" / "classifier.pkl"
DATASET_PATH = PROJECT_ROOT / "datasets" / "processed" / "classifier_dataset.csv"
TREES_PATH = PROJECT_ROOT / "models" / "classifier_trees.json"
GOLDEN_PATH = PROJECT_ROOT / "models" / "classifier_golden.json"


def finite(value) -> float | None:
    """JSON не поддерживает NaN/Infinity — узлы-листья имеют threshold=inf."""
    value = float(value)
    if math.isnan(value) or math.isinf(value):
        return None
    return value


def threshold_value(node, is_leaf) -> float | None:
    """Порог внутреннего узла может быть +inf; JSON не поддерживает Infinity.

    Заменяем на 1e308, иначе Go читает 0 и уходит в другую ветку.
    У листьев порог не используется — пишем null.
    """
    if bool(is_leaf):
        return None
    value = float(node)
    if math.isnan(value):
        return 0.0
    if math.isinf(value):
        return 1e308
    return value


def export_trees(
    model,
    feature_columns: list[str],
    classes: list[str],
    scaler: dict,
) -> None:
    trees = []
    for stage in model._predictors:
        for predictor in stage:
            nodes = predictor.nodes
            leaves = nodes["is_leaf"].astype(bool)
            trees.append(
                {
                    "feature": nodes["feature_idx"].astype(int).tolist(),
                    "threshold": [
                        threshold_value(nodes["num_threshold"][i], leaves[i])
                        for i in range(nodes.size)
                    ],
                    "left": nodes["left"].astype(int).tolist(),
                    "right": nodes["right"].astype(int).tolist(),
                    "missing_left": [
                        bool(v) for v in nodes["missing_go_to_left"].astype(bool)
                    ],
                    "value": [finite(v) for v in nodes["value"]],
                    "leaf": leaves.tolist(),
                }
            )

    payload = {
        "format": "histgb-v1",
        "n_classes": int(model.n_trees_per_iteration_),
        "learning_rate": float(model.learning_rate),
        "baseline": [float(v) for v in model._baseline_prediction.ravel()],
        "feature_keys": feature_columns,
        "feature_min": [float(scaler[c]["min"]) for c in feature_columns],
        "feature_max": [float(scaler[c]["max"]) for c in feature_columns],
        "classes": classes,
        "trees": trees,
    }

    with open(TREES_PATH, "w", encoding="utf-8") as fh:
        json.dump(payload, fh, separators=(",", ":"))
    print(f"Trees exported to {TREES_PATH} ({len(trees)} trees)")


def export_golden_set(
    model,
    df: pd.DataFrame,
    feature_columns: list[str],
    scaler: dict,
    classes: list[str],
    n: int = 400,
) -> None:
    """Сохраняет сырые значения + ожидаемый класс для Go-теста."""
    sample = df.sample(n=min(n, len(df)), random_state=42)
    cases = []

    for _, row in sample.iterrows():
        raw = {}
        for col in feature_columns:
            norm = row[col]
            if pd.isna(norm):
                continue  # отсутствующее значение — Go должен импутировать
            raw[col] = (
                float(norm) * (scaler[col]["max"] - scaler[col]["min"])
                + scaler[col]["min"]
            )

        x = [
            (
                (raw[col] - scaler[col]["min"])
                / (scaler[col]["max"] - scaler[col]["min"])
                if col in raw
                else np.nan
            )
            for col in feature_columns
        ]
        predicted = int(model.predict(np.array([x]))[0])
        cases.append({"features": raw, "class": predicted})

    with open(GOLDEN_PATH, "w", encoding="utf-8") as fh:
        json.dump(
            {
                "feature_keys": feature_columns,
                "classes": classes,
                "cases": cases,
            },
            fh,
            separators=(",", ":"),
        )
    print(f"Golden set exported to {GOLDEN_PATH} ({len(cases)} cases)")


def main() -> None:
    model = joblib.load(MODEL_PATH)
    df = pd.read_csv(DATASET_PATH)

    with open(
        PROJECT_ROOT / "models" / "classifier_scaler.json", encoding="utf-8"
    ) as fh:
        scaler = json.load(fh)

    if hasattr(model, "feature_names_in_"):
        feature_columns = list(model.feature_names_in_)
    else:
        # Тот же порядок, что в train_classifier.py: только непустые признаки.
        required = [
            "heart_rate",
            "hrv",
            "spo2",
            "temperature",
            "systolic_pressure",
            "diastolic_pressure",
            "sleep_hours",
        ]
        feature_columns = [
            c for c in required if c in df.columns and df[c].notna().sum() > 0
        ]
    classes = [str(c) for c in model.classes_]

    export_trees(model, feature_columns, classes, scaler)
    export_golden_set(model, df, feature_columns, scaler, classes)


if __name__ == "__main__":
    main()
