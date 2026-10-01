#!/usr/bin/env python3
"""Train MLP classifier on processed classifier dataset and export to ONNX.

Usage:
    python scripts/train_classifier.py \
        --dataset datasets/processed/classifier_dataset.csv \
        --output models/classifier.onnx \
        --epochs 100 \
        --lr 1e-3
"""

import argparse
from pathlib import Path

import numpy as np
import pandas as pd
import torch
import torch.nn as nn
import torch.optim as optim
from sklearn.metrics import accuracy_score, classification_report
from sklearn.model_selection import train_test_split
from torch.utils.data import DataLoader, TensorDataset


CLASSES = [
    "recovery",
    "endurance_basic",
    "endurance_threshold",
    "power_hiit",
    "overtraining",
    "illness",
    "unknown",
]

FEATURE_COLUMNS = [
    "heart_rate",
    "hrv",
    "spo2",
    "temperature",
    "systolic_pressure",
    "diastolic_pressure",
    "sleep_hours",
]


class ClassifierMLP(nn.Module):
    def __init__(self, input_dim: int, hidden_dim: int = 64, num_classes: int = len(CLASSES)) -> None:
        super().__init__()
        self.net = nn.Sequential(
            nn.Linear(input_dim, hidden_dim),
            nn.ReLU(),
            nn.Linear(hidden_dim, hidden_dim // 2),
            nn.ReLU(),
            nn.Linear(hidden_dim // 2, num_classes),
        )

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return self.net(x)


def load_dataset(path: Path) -> tuple[np.ndarray, np.ndarray]:
    df = pd.read_csv(path)
    if "label" not in df.columns:
        raise ValueError("Dataset must contain 'label' column")

    x = df[FEATURE_COLUMNS].to_numpy(dtype=np.float32)
    label_to_idx = {label: idx for idx, label in enumerate(CLASSES)}
    y = df["label"].map(label_to_idx).to_numpy(dtype=np.int64)
    return x, y


def train_epoch(model: nn.Module, loader: DataLoader, criterion: nn.Module, optimizer: optim.Optimizer, device: torch.device) -> tuple[float, float]:
    model.train()
    total_loss = 0.0
    correct = 0
    total = 0
    for xb, yb in loader:
        xb = xb.to(device)
        yb = yb.to(device)
        optimizer.zero_grad()
        logits = model(xb)
        loss = criterion(logits, yb)
        loss.backward()
        optimizer.step()
        total_loss += loss.item() * xb.size(0)
        correct += (logits.argmax(dim=1) == yb).sum().item()
        total += xb.size(0)
    return total_loss / total, correct / total


def evaluate(model: nn.Module, loader: DataLoader, criterion: nn.Module, device: torch.device) -> tuple[float, float, np.ndarray, np.ndarray]:
    model.eval()
    total_loss = 0.0
    correct = 0
    total = 0
    all_preds = []
    all_labels = []
    with torch.no_grad():
        for xb, yb in loader:
            xb = xb.to(device)
            yb = yb.to(device)
            logits = model(xb)
            loss = criterion(logits, yb)
            total_loss += loss.item() * xb.size(0)
            correct += (logits.argmax(dim=1) == yb).sum().item()
            total += xb.size(0)
            all_preds.extend(logits.argmax(dim=1).cpu().numpy())
            all_labels.extend(yb.cpu().numpy())
    return total_loss / total, correct / total, np.array(all_preds), np.array(all_labels)


def export_to_onnx(model: nn.Module, output_path: Path, input_dim: int) -> None:
    output_path.parent.mkdir(parents=True, exist_ok=True)
    dummy = torch.randn(1, input_dim)
    torch.onnx.export(
        model,
        dummy,
        output_path,
        input_names=["input"],
        output_names=["logits"],
        dynamic_axes={"input": {0: "batch"}, "logits": {0: "batch"}},
        opset_version=15,
    )
    print(f"ONNX model exported to {output_path}")


def main() -> None:
    parser = argparse.ArgumentParser(description="Train classifier MLP and export to ONNX")
    parser.add_argument("--dataset", type=Path, required=True, help="Path to processed classifier_dataset.csv")
    parser.add_argument("--output", type=Path, default=Path("models/classifier.onnx"), help="Output ONNX path")
    parser.add_argument("--epochs", type=int, default=100)
    parser.add_argument("--lr", type=float, default=1e-3)
    parser.add_argument("--batch-size", type=int, default=32)
    parser.add_argument("--hidden-dim", type=int, default=64)
    parser.add_argument("--val-split", type=float, default=0.2)
    parser.add_argument("--seed", type=int, default=42)
    args = parser.parse_args()

    torch.manual_seed(args.seed)
    np.random.seed(args.seed)

    x, y = load_dataset(args.dataset)
    x_train, x_val, y_train, y_val = train_test_split(x, y, test_size=args.val_split, random_state=args.seed, stratify=y)

    train_ds = TensorDataset(torch.tensor(x_train), torch.tensor(y_train))
    val_ds = TensorDataset(torch.tensor(x_val), torch.tensor(y_val))
    train_loader = DataLoader(train_ds, batch_size=args.batch_size, shuffle=True)
    val_loader = DataLoader(val_ds, batch_size=args.batch_size)

    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    model = ClassifierMLP(input_dim=x.shape[1], hidden_dim=args.hidden_dim).to(device)
    criterion = nn.CrossEntropyLoss()
    optimizer = optim.AdamW(model.parameters(), lr=args.lr)

    best_acc = 0.0
    best_state = None
    for epoch in range(1, args.epochs + 1):
        train_loss, train_acc = train_epoch(model, train_loader, criterion, optimizer, device)
        val_loss, val_acc, preds, labels = evaluate(model, val_loader, criterion, device)
        if val_acc > best_acc:
            best_acc = val_acc
            best_state = {k: v.detach().cpu().clone() for k, v in model.state_dict().items()}

        if epoch % 10 == 0 or epoch == 1:
            print(f"Epoch {epoch:03d}: train_loss={train_loss:.4f} train_acc={train_acc:.4f} val_loss={val_loss:.4f} val_acc={val_acc:.4f}")

    if best_state is not None:
        model.load_state_dict(best_state)
    print(f"\nBest validation accuracy: {best_acc:.4f}")
    print("\nClassification report:")
    print(classification_report(labels, preds, target_names=CLASSES, zero_division=0))
    print(f"Overall accuracy: {accuracy_score(labels, preds):.4f}")

    export_to_onnx(model, args.output, input_dim=x.shape[1])


if __name__ == "__main__":
    main()
