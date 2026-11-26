#!/usr/bin/env python3
"""Обучает линейную (softmax) модель классификатора и экспортирует веса для Go.

Зачем: `scripts/train_classifier.py` обучает HistGradientBoosting на датасете из
реальных носимых устройств, но в репозитории нет ни сырых датасетов, ни
`classifier_dataset.csv`, поэтому воспроизвести веса нельзя. Плюс экспорт в
`classifier_weights.json` терял параметры нормализации, из-за чего Go-инференс
получал сырые ЧСС/SpO2/температуру и выдавал один и тот же класс.

Этот скрипт:
  1. читает эталонный датасет, размеченный rule-based классификатором
     (`classifier -dump-dataset`, CSV: heart_rate,hrv,spo2,temperature,...);
  2. обучает мультиномиальную логистическую регрессию (softmax) без внешних
     зависимостей — только стандартная библиотека Python;
  3. экспортирует веса вместе с `feature_min`/`feature_max`, чтобы Go применял
     ту же min-max нормализацию, что и обучение.

Запуск:
    python scripts/train_classifier_linear.py \
        --dataset datasets/processed/classifier_distilled.csv \
        --output models/classifier_weights.json

Зависимости: только Python 3.9+ (никаких numpy/sklearn/torch).
"""

from __future__ import annotations

import argparse
import csv
import json
import math
import random
from datetime import datetime, timezone
from pathlib import Path

# Фиксированные физиологические диапазоны. Совпадают с константами в
# scripts/prepare_classifier_dataset.py (HR_MIN..TEMP_MAX), поэтому модель,
# обученная на реальных данных, и модель, обученная на дистиллированном
# датасете, масштабируются одинаково.
FEATURE_KEYS = ["heart_rate", "hrv", "spo2", "temperature"]
FEATURE_MIN = [40.0, 10.0, 88.0, 35.0]
FEATURE_MAX = [200.0, 100.0, 100.0, 40.0]

CLASSES = [
    "recovery",
    "endurance_basic",
    "endurance_threshold",
    "power_hiit",
    "overtraining",
    "illness",
    "unknown",
]


def load_dataset(path: Path) -> tuple[list[list[float]], list[int]]:
    features: list[list[float]] = []
    labels: list[int] = []
    with path.open(newline="") as f:
        reader = csv.DictReader(f)
        for row in reader:
            try:
                x = [float(row[key]) for key in FEATURE_KEYS]
                y = int(float(row["label"]))
            except (KeyError, TypeError, ValueError):
                continue
            if not 0 <= y < len(CLASSES):
                continue
            features.append(x)
            labels.append(y)
    if len(features) < 10:
        raise SystemExit(f"Недостаточно данных в {path}: {len(features)} строк")
    return features, labels


def normalize(x: list[float]) -> list[float]:
    out = []
    for value, low, high in zip(x, FEATURE_MIN, FEATURE_MAX):
        span = high - low
        out.append(0.0 if span <= 0 else (value - low) / span)
    return out


def softmax(logits: list[float]) -> list[float]:
    peak = max(logits)
    exps = [math.exp(value - peak) for value in logits]
    total = sum(exps)
    return [value / total for value in exps]


def _batch_gradient(
    batch: list[int],
    features: list[list[float]],
    labels: list[int],
    weights: list[list[float]],
    bias: list[float],
    n_classes: int,
    n_features: int,
) -> tuple[list[list[float]], list[float]]:
    """Считает суммарный градиент мини-батча."""
    grad_w = [[0.0] * n_features for _ in range(n_classes)]
    grad_b = [0.0] * n_classes

    for idx in batch:
        logits = [
            bias[c] + sum(weights[c][j] * features[idx][j] for j in range(n_features))
            for c in range(n_classes)
        ]
        probs = softmax(logits)
        for c in range(n_classes):
            err = probs[c] - (1.0 if labels[idx] == c else 0.0)
            grad_b[c] += err
            row = grad_w[c]
            for j in range(n_features):
                row[j] += err * features[idx][j]
    return grad_w, grad_b


def _apply_step(
    weights: list[list[float]],
    bias: list[float],
    grad_w: list[list[float]],
    grad_b: list[float],
    scale: float,
    l2: float,
    n_classes: int,
    n_features: int,
) -> None:
    """Обновляет веса и смещения один шаг градиентного спуска."""
    for c in range(n_classes):
        bias[c] -= scale * grad_b[c]
        for j in range(n_features):
            weights[c][j] -= scale * (grad_w[c][j] + l2 * weights[c][j])


def train(
    features: list[list[float]],
    labels: list[int],
    epochs: int,
    batch_size: int,
    lr: float,
    l2: float,
    seed: int,
) -> tuple[list[list[float]], list[float]]:
    n_classes = len(CLASSES)
    n_features = len(FEATURE_KEYS)
    weights = [[0.0] * n_features for _ in range(n_classes)]
    bias = [0.0] * n_classes

    rng = random.Random(seed)
    order = list(range(len(features)))

    for epoch in range(epochs):
        rng.shuffle(order)
        lr_epoch = lr / (1.0 + epoch / max(1, epochs // 2))

        for start in range(0, len(order), batch_size):
            batch = order[start : start + batch_size]
            grad_w, grad_b = _batch_gradient(
                batch, features, labels, weights, bias, n_classes, n_features
            )
            _apply_step(
                weights,
                bias,
                grad_w,
                grad_b,
                lr_epoch / len(batch),
                l2,
                n_classes,
                n_features,
            )

    return weights, bias


def accuracy(features, labels, weights, bias) -> float:
    correct = 0
    for x, y in zip(features, labels):
        logits = [
            bias[c] + sum(weights[c][j] * x[j] for j in range(len(x)))
            for c in range(len(CLASSES))
        ]
        if max(range(len(logits)), key=logits.__getitem__) == y:
            correct += 1
    return correct / len(labels)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dataset", type=Path, required=True)
    parser.add_argument(
        "--output", type=Path, default=Path("models/classifier_weights.json")
    )
    parser.add_argument("--epochs", type=int, default=400)
    parser.add_argument("--batch-size", type=int, default=64)
    parser.add_argument("--lr", type=float, default=0.5)
    parser.add_argument("--l2", type=float, default=1e-4)
    parser.add_argument("--seed", type=int, default=42)
    args = parser.parse_args()

    features, labels = load_dataset(args.dataset)
    normalized = [normalize(x) for x in features]

    counts = dict.fromkeys(CLASSES, 0)
    for label in labels:
        counts[CLASSES[label]] += 1
    print(f"Датасет: {len(labels)} строк, распределение: {counts}")

    weights, bias = train(
        normalized,
        labels,
        epochs=args.epochs,
        batch_size=args.batch_size,
        lr=args.lr,
        l2=args.l2,
        seed=args.seed,
    )

    train_acc = accuracy(normalized, labels, weights, bias)
    print(f"Точность на обучающей выборке: {train_acc:.4f}")
    if train_acc < 0.80:
        raise SystemExit(
            f"Точность {train_acc:.3f} ниже порога 0.80 — модель не выучила правила"
        )

    flat_weights = [float(w) for row in weights for w in row]
    payload = {
        "labels": list(range(len(CLASSES))),
        "weights": flat_weights,
        "intercepts": [float(b) for b in bias],
        "multi_class": 1,
        "feature_keys": FEATURE_KEYS,
        "feature_min": FEATURE_MIN,
        "feature_max": FEATURE_MAX,
        "classes": CLASSES,
        "source": "distilled-rule-based",
        "trained_at": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "samples": len(labels),
        "train_accuracy": round(train_acc, 4),
    }

    args.output.parent.mkdir(parents=True, exist_ok=True)
    with args.output.open("w") as f:
        json.dump(payload, f, indent=2)
    print(f"Веса сохранены: {args.output}")


if __name__ == "__main__":
    main()
