#!/usr/bin/env python3
"""Train classifier on processed classifier dataset and export to ONNX.

Usage:
    python scripts/train_classifier.py \
        --dataset datasets/processed/classifier_dataset.csv \
        --output models/classifier.onnx \
        --max-iter 50 \
        --lr 1e-2 \
        --val-split 0.2
"""

import argparse
from pathlib import Path

import numpy as np
import pandas as pd
from sklearn.ensemble import HistGradientBoostingClassifier
from sklearn.metrics import accuracy_score, classification_report
from sklearn.model_selection import train_test_split
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


def load_dataset(path: Path) -> tuple[np.ndarray, np.ndarray, list[str]]:
    df = pd.read_csv(path)
    if "label" not in df.columns:
        raise ValueError("Dataset must contain 'label' column")

    available_features = [c for c in REQUIRED_FEATURES if c in df.columns]
    if not available_features:
        raise ValueError("No usable feature columns found in dataset")

    non_empty = [c for c in available_features if df[c].notna().sum() > 0]
    if not non_empty:
        raise ValueError("All feature columns are empty")

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


def main() -> None:
    parser = argparse.ArgumentParser(description="Train classifier and export to ONNX")
    parser.add_argument("--dataset", type=Path, required=True, help="Path to processed classifier_dataset.csv")
    parser.add_argument("--output", type=Path, default=Path("models/classifier.onnx"), help="Output ONNX path")
    parser.add_argument("--max-iter", type=int, default=50, help="Maximum number of iterations")
    parser.add_argument("--lr", type=float, default=1e-2, help="Learning rate")
    parser.add_argument("--val-split", type=float, default=0.2, help="Validation split")
    parser.add_argument("--seed", type=int, default=42, help="Random seed")
    args = parser.parse_args()

    np.random.seed(args.seed)

    print(f"Loading dataset from {args.dataset}")
    x, y, feature_columns = load_dataset(args.dataset)
    print(f"Dataset shape: {x.shape}, classes: {len(np.unique(y))}")

    unique, counts = np.unique(y, return_counts=True)
    for cls_idx, count in zip(unique, counts):
        print(f"  Class {CLASSES[cls_idx]}: {count} samples")

    min_class_count = counts.min()
    if min_class_count < 2:
        print(f"WARNING: Smallest class has only {min_class_count} sample(s). Disabling stratified split.")
        x_train, x_val, y_train, y_val = train_test_split(x, y, test_size=args.val_split, random_state=args.seed)
    else:
        x_train, x_val, y_train, y_val = train_test_split(x, y, test_size=args.val_split, random_state=args.seed, stratify=y)

    model = HistGradientBoostingClassifier(
        max_iter=args.max_iter,
        learning_rate=args.lr,
        random_state=args.seed,
        verbose=0,
    )

    print("Training HistGradientBoosting...")
    model.fit(x_train, y_train)

    y_pred = model.predict(x_val)
    print(f"Validation accuracy: {accuracy_score(y_val, y_pred):.4f}")
    print("\nClassification report:")
    print(classification_report(y_val, y_pred, target_names=CLASSES, zero_division=0))

    # Save full model as pickle
    args.output.parent.mkdir(parents=True, exist_ok=True)
    model_path = args.output.with_suffix(".pkl")
    joblib.dump(model, model_path)
    print(f"Model saved to {model_path}")

    # Export fallback ONNX from a small sklearn pipeline
    print(f"\nExporting fallback ONNX to {args.output} ...")
    from sklearn.pipeline import Pipeline
    from sklearn.impute import SimpleImputer
    from sklearn.linear_model import LogisticRegression

    sample_size = min(5000, len(x_train))
    rng = np.random.default_rng(args.seed)
    idx = rng.choice(len(x_train), size=sample_size, replace=False)
    x_sample = x_train[idx]
    y_sample = y_train[idx]

    fallback_model = Pipeline([
        ("imputer", SimpleImputer()),
        ("clf", LogisticRegression(max_iter=200, class_weight="balanced")),
    ], memory=None)
    fallback_model.fit(x_sample, y_sample)

    initial_types = [("input", FloatTensorType([1, len(feature_columns)]))]
    onnx_model = convert_sklearn(fallback_model, initial_types=initial_types, options={id(fallback_model): {"zipmap": False}})

    with open(args.output, "wb") as f:
        f.write(onnx_model.SerializeToString())

    print(f"ONNX model exported to {args.output} ({len(onnx_model.SerializeToString())} bytes)")
    print("Done.")


if __name__ == "__main__":
    main()
