#!/usr/bin/env python3
"""Train classifier on processed classifier dataset and export to ONNX + Go weights.

Usage:
    python scripts/train_classifier.py \
        --dataset datasets/processed/classifier_dataset.csv \
        --output models/classifier.onnx \
        --max-iter 50 \
        --lr 1e-2 \
        --val-split 0.2 \
        --test-split 0.1
"""

import argparse
import json
from datetime import datetime, timezone
from pathlib import Path

import numpy as np
import pandas as pd
from sklearn.ensemble import HistGradientBoostingClassifier
from sklearn.linear_model import LogisticRegression
from sklearn.metrics import accuracy_score, classification_report
from sklearn.model_selection import train_test_split
from sklearn.pipeline import Pipeline
from sklearn.preprocessing import SimpleImputer
from skl2onnx import convert_sklearn
from skl2onnx.common.data_types import FloatTensorType
import joblib


CLASSES = [
    "recovery",
    "endurance_basic",
    "endurance_threshold",
    "power_hiit",
    "overtraining",
    "illness",
    "unknown",
]

REQUIRED_FEATURES = [
    "heart_rate",
    "hrv",
    "spo2",
    "temperature",
    "systolic_pressure",
    "diastolic_pressure",
    "sleep_hours",
]


def validate_dataset(df: pd.DataFrame) -> None:
    if "label" not in df.columns:
        raise ValueError("Dataset must contain 'label' column")

    available_features = [c for c in REQUIRED_FEATURES if c in df.columns]
    if not available_features:
        raise ValueError("No usable feature columns found in dataset")

    non_empty = [c for c in available_features if df[c].notna().sum() > 0]
    if not non_empty:
        raise ValueError("All feature columns are empty")

    unique_labels = df["label"].unique()
    if len(unique_labels) < 2:
        raise ValueError(f"Need at least 2 classes for training, got {len(unique_labels)}: {unique_labels}")


def load_dataset(path: Path) -> tuple[np.ndarray, np.ndarray, list[str]]:
    df = pd.read_csv(path)
    validate_dataset(df)

    available_features = [c for c in REQUIRED_FEATURES if c in df.columns]
    non_empty = [c for c in available_features if df[c].notna().sum() > 0]
    feature_columns = non_empty
    print(f"Using features: {feature_columns}")

    x = df[feature_columns].to_numpy(dtype=np.float32)
    label_to_idx = {label: idx for idx, label in enumerate(CLASSES)}
    y_raw = df["label"].map(label_to_idx)

    unmapped = y_raw.isna().sum()
    if unmapped > 0:
        print(f"WARNING: {unmapped} rows have unmapped labels, dropping them")

    valid_mask = y_raw.notna().to_numpy()
    x = x[valid_mask]
    y = y_raw[valid_mask].to_numpy(dtype=np.int64)

    unique_labels = np.unique(y)
    if len(unique_labels) < 2:
        raise ValueError(f"Need at least 2 classes for training, got {len(unique_labels)}: {unique_labels}")

    return x, y, feature_columns


def export_go_weights(model, output_path: Path) -> None:
    coef = model.named_steps["clf"].coef_
    labels = list(range(len(CLASSES)))

    weights = []
    for cls_idx in range(coef.shape[0]):
        for feat_idx in range(coef.shape[1]):
            weights.append(float(coef[cls_idx, feat_idx]))

    weights_data = {
        "labels": labels,
        "weights": weights,
        "multi_class": 1,
    }

    with open(output_path, "w") as f:
        json.dump(weights_data, f)

    print(f"Go weights exported to {output_path}")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Train classifier and export to ONNX + Go weights")
    parser.add_argument("--dataset", type=Path, required=True, help="Path to processed classifier_dataset.csv")
    parser.add_argument("--output", type=Path, default=Path("models/classifier.onnx"), help="Output ONNX path")
    parser.add_argument("--max-iter", type=int, default=50, help="Maximum number of iterations")
    parser.add_argument("--lr", type=float, default=1e-2, help="Learning rate")
    parser.add_argument("--val-split", type=float, default=0.2, help="Validation split")
    parser.add_argument("--test-split", type=float, default=0.1, help="Test split (holdout)")
    parser.add_argument("--seed", type=int, default=42, help="Random seed")
    parser.add_argument("--metrics-output", type=Path, default=Path("models/train_metrics.json"), help="Path to save metrics JSON")
    return parser.parse_args()


def split_dataset(x, y, args):
    use_stratify = counts.min() >= 2 if (counts := np.unique(y, return_counts=True)[1]).size else False

    if args.test_split > 0:
        x_train, x_temp, y_train, y_temp = train_test_split(
            x, y, test_size=args.val_split + args.test_split, random_state=args.seed,
            stratify=y if use_stratify else None
        )
        relative_test_size = args.test_split / (args.val_split + args.test_split)
        x_val, x_test, y_val, y_test = train_test_split(
            x_temp, y_temp, test_size=relative_test_size, random_state=args.seed,
            stratify=y_temp if len(np.unique(y_temp)) > 1 and use_stratify else None
        )
        print(f"Split: train={len(y_train)}, val={len(y_val)}, test={len(y_test)}")
    else:
        x_train, x_val, y_train, y_val = train_test_split(
            x, y, test_size=args.val_split, random_state=args.seed,
            stratify=y if use_stratify else None
        )
        x_test, y_test = None, None
        print(f"Split: train={len(y_train)}, val={len(y_val)}")

    return x_train, x_val, x_test, y_train, y_val, y_test


def train_model(x_train, y_train, args):
    model = HistGradientBoostingClassifier(
        max_iter=args.max_iter,
        learning_rate=args.lr,
        random_state=args.seed,
        verbose=0,
    )
    print("Training HistGradientBoosting...")
    model.fit(x_train, y_train)
    return model


def evaluate_model(model, x_val, y_val, x_test, y_test):
    y_pred_val = model.predict(x_val)
    val_accuracy = accuracy_score(y_val, y_pred_val)
    print(f"Validation accuracy: {val_accuracy:.4f}")
    print("\nClassification report (validation):")
    print(classification_report(y_val, y_pred_val, target_names=CLASSES, zero_division=0))

    metrics_data = {
        "timestamp": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "val_accuracy": float(val_accuracy),
        "val_samples": int(len(y_val)),
    }

    if x_test is not None:
        y_pred_test = model.predict(x_test)
        test_accuracy = accuracy_score(y_test, y_pred_test)
        print(f"Test accuracy: {test_accuracy:.4f}")
        print("\nClassification report (test):")
        print(classification_report(y_test, y_pred_test, target_names=CLASSES, zero_division=0))
        metrics_data["test_accuracy"] = float(test_accuracy)
        metrics_data["test_samples"] = int(len(y_test))

    return metrics_data


def build_stratified_sample(x_train, y_train, sample_size, seed):
    rng = np.random.default_rng(seed)
    idx = []
    per_class = sample_size // len(CLASSES)
    for cls in range(len(CLASSES)):
        cls_mask = y_train == cls
        cls_indices = np.nonzero(cls_mask)[0]
        if len(cls_indices) == 0:
            continue
        n_sample = min(per_class, len(cls_indices))
        idx.extend(rng.choice(cls_indices, size=n_sample, replace=False).tolist())
    idx = np.array(idx, dtype=int)
    if len(idx) < sample_size:
        remaining = rng.choice(len(x_train), size=sample_size - len(idx), replace=False)
        idx = np.concatenate([idx, remaining])
    return idx


def export_fallback_onnx(x_train, y_train, feature_columns, args):
    sample_size = min(5000, len(x_train))
    if np.unique(y_train).size >= 2 and np.min(np.unique(y_train, return_counts=True)[1]) >= 2:
        idx = build_stratified_sample(x_train, y_train, sample_size, args.seed)
    else:
        rng = np.random.default_rng(args.seed)
        idx = rng.choice(len(x_train), size=sample_size, replace=False)

    x_sample = x_train[idx]
    y_sample = y_train[idx]
    print(f"Fallback sample distribution: {dict(zip(*np.unique(y_sample, return_counts=True)))}")

    fallback_model = Pipeline([
        ("imputer", SimpleImputer()),
        ("clf", LogisticRegression(max_iter=200, class_weight="balanced", random_state=args.seed)),
    ], memory=None)
    fallback_model.fit(x_sample, y_sample)

    initial_types = [("input", FloatTensorType([1, len(feature_columns)]))]
    onnx_model = convert_sklearn(fallback_model, initial_types=initial_types, options={id(fallback_model): {"zipmap": False}})

    with open(args.output, "wb") as f:
        f.write(onnx_model.SerializeToString())

    print(f"ONNX model exported to {args.output} ({len(onnx_model.SerializeToString())} bytes)")
    return fallback_model


def save_metrics(metrics_data, metrics_output: Path) -> None:
    metrics_output.parent.mkdir(parents=True, exist_ok=True)
    with open(metrics_output, "w") as f:
        json.dump(metrics_data, f, indent=2)
    print(f"Metrics exported to {metrics_output}")


def main() -> None:
    args = parse_args()
    np.random.seed(args.seed)

    print(f"Loading dataset from {args.dataset}")
    x, y, feature_columns = load_dataset(args.dataset)
    print(f"Dataset shape: {x.shape}, classes: {len(np.unique(y))}")

    unique, counts = np.unique(y, return_counts=True)
    for cls_idx, count in zip(unique, counts):
        print(f"  Class {CLASSES[cls_idx]}: {count} samples")

    x_train, x_val, x_test, y_train, y_val, y_test = split_dataset(x, y, args)
    model = train_model(x_train, y_train, args)
    metrics_data = evaluate_model(model, x_val, y_val, x_test, y_test)

    args.output.parent.mkdir(parents=True, exist_ok=True)
    model_path = args.output.with_suffix(".pkl")
    joblib.dump(model, model_path)
    print(f"Model saved to {model_path}")

    print(f"\nExporting fallback ONNX to {args.output} ...")
    fallback_model = export_fallback_onnx(x_train, y_train, feature_columns, args)

    go_weights_path = args.output.parent / "classifier_weights.json"
    export_go_weights(fallback_model, go_weights_path)

    metrics_data.update({
        "train_samples": int(len(y_train)),
        "features": feature_columns,
        "classes": CLASSES,
        "class_distribution": {CLASSES[i]: int(c) for i, c in zip(unique, counts)},
    })
    save_metrics(metrics_data, args.metrics_output)
    print("Done.")


if __name__ == "__main__":
    main()
