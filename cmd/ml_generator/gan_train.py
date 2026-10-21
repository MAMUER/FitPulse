"""
Скрипт обучения условной ГАН для генерации персонализированных планов тренировок.

Использование:
    python cmd/ml_generator/gan_train.py \
        --dataset datasets/processed/training_plans.csv \
        --epochs 100 \
        --batch-size 32 \
        --output models/gan_generator.pt
"""

import argparse
import json
import os
import numpy as np
import torch
from pathlib import Path

from gan_model import (
    ConditionalGAN, TrainingPlanDataset,
    PLAN_DIM, NUM_CLASSES, CONDITION_DIM,
    build_condition, normalize_plan, denormalize_plan,
)


def load_dataset(path):
    """Загрузка датасета планов тренировок."""
    if not os.path.exists(path):
        raise FileNotFoundError(f"Датасет не найден: {path}. Создайте файл {path} с колонками: план (19 значений), класс, возраст, пол, вес, рост, цель.")

    data = np.loadtxt(path, delimiter=",", skiprows=1)
    plans = data[:, :PLAN_DIM].astype(np.float32)
    class_indices = data[:, PLAN_DIM].astype(int)
    user_features = data[:, PLAN_DIM + 1:PLAN_DIM + 6].astype(np.float32)

    # Проверка на отрицательные значения в числовых полях
    if np.any(plans < 0):
        raise ValueError("Обнаружены отрицательные значения в параметрах плана. Все значения должны быть неотрицательными.")
    if np.any(user_features < 0):
        raise ValueError("Обнаружены отрицательные значения в пользовательских признаках.")

    conditions = np.array([
        build_condition(cls, feat) for cls, feat in zip(class_indices, user_features)
    ], dtype=np.float32)

    return plans, conditions


def train(args):
    plans, conditions = load_dataset(args.dataset)

    # Нормализация планов
    plan_min = plans.min(axis=0)
    plan_max = plans.max(axis=0)
    plans_norm = normalize_plan(plans, plan_min, plan_max)

    dataset = TrainingPlanDataset(plans_norm, conditions)
    dataloader = DataLoader(dataset, batch_size=args.batch_size, shuffle=True)

    device = torch.device("cuda" if torch.cuda.is_available() and not args.cpu else "cpu")
    gan = ConditionalGAN(device=device, lr=args.lr)

    os.makedirs(args.output.parent, exist_ok=True)

    history = {"d_loss": [], "g_loss": []}
    for epoch in range(args.epochs):
        epoch_d_loss = 0.0
        epoch_g_loss = 0.0
        num_batches = 0

        for batch_plans, batch_conditions in dataloader:
            metrics = gan.train_step(batch_plans, batch_conditions)
            epoch_d_loss += metrics["d_loss"]
            epoch_g_loss += metrics["g_loss"]
            num_batches += 1

        epoch_d_loss /= num_batches
        epoch_g_loss /= num_batches
        history["d_loss"].append(epoch_d_loss)
        history["g_loss"].append(epoch_g_loss)

        if (epoch + 1) % 10 == 0 or epoch == 0:
            print(f"Epoch {epoch+1}/{args.epochs}: D_loss={epoch_d_loss:.4f}, G_loss={epoch_g_loss:.4f}")

    # Сохраняем модель
    gan.save(args.output)

    # Сохраняем параметры нормализации
    norm_params = {
        "plan_min": plan_min.tolist(),
        "plan_max": plan_max.tolist(),
    }
    norm_path = args.output.parent / "gan_norm_params.json"
    with open(norm_path, "w") as f:
        json.dump(norm_params, f, indent=2)

    # Сохраняем историю обучения
    history_path = args.output.parent / "gan_history.json"
    with open(history_path, "w") as f:
        json.dump(history, f, indent=2)

    print(f"Модель сохранена: {args.output}")
    print(f"Параметры нормализации: {norm_path}")
    print(f"История обучения: {history_path}")


def generate_plan(args):
    """Генерация одного плана для заданного класса и признаков пользователя."""
    device = torch.device("cuda" if torch.cuda.is_available() and not args.cpu else "cpu")
    gan = ConditionalGAN(device=device)
    gan.load(args.model)

    # Загружаем параметры нормализации
    norm_path = Path(args.model).parent / "gan_norm_params.json"
    with open(norm_path, "r") as f:
        norm_params = json.load(f)
    plan_min = np.array(norm_params["plan_min"])
    plan_max = np.array(norm_params["plan_max"])

    # Построение условия
    user_features = np.array([
        args.age, args.gender, args.weight, args.height, args.goal
    ], dtype=np.float32)
    condition = build_condition(args.class_index, user_features)

    plans = gan.generate(condition, num_samples=args.num_samples)
    plans_denorm = denormalize_plan(plans, plan_min, plan_max)

    result = []
    for i, plan in enumerate(plans_denorm):
        result.append({
            "sample": i,
            "plan": plan.tolist(),
            "class_index": args.class_index,
        })

    output_path = Path(args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    with open(output_path, "w") as f:
        json.dump(result, f, indent=2)

    print(f"Сгенерировано {args.num_samples} планов: {output_path}")
    return result


parser = argparse.ArgumentParser(description="Условная ГАН для персонализированных планов тренировок")
subparsers = parser.add_subparsers(dest="command")

train_parser = subparsers.add_parser("train", help="Обучить ГАН")
train_parser.add_argument("--dataset", type=Path, required=True, help="Путь к CSV с планами")
train_parser.add_argument("--epochs", type=int, default=100, help="Число эпох")
train_parser.add_argument("--batch-size", type=int, default=32, help="Размер батча")
train_parser.add_argument("--lr", type=float, default=2e-4, help="Learning rate")
train_parser.add_argument("--cpu", action="store_true", help="Принудительно использовать CPU")
train_parser.add_argument("--output", type=Path, default=Path("models/gan_generator.pt"), help="Путь для сохранения модели")

gen_parser = subparsers.add_parser("generate", help="Сгенерировать план")
gen_parser.add_argument("--model", type=Path, required=True, help="Путь к обученной модели")
gen_parser.add_argument("--class-index", type=int, required=True, help="Класс состояния (0-6)")
gen_parser.add_argument("--age", type=float, default=30, help="Возраст")
gen_parser.add_argument("--gender", type=float, default=1, help="Пол (0/1)")
gen_parser.add_argument("--weight", type=float, default=75.0, help="Вес")
gen_parser.add_argument("--height", type=float, default=180.0, help="Рост")
gen_parser.add_argument("--goal", type=float, default=1, help="Цель (кодировка)")
gen_parser.add_argument("--num-samples", type=int, default=1, help="Число вариантов плана")
gen_parser.add_argument("--cpu", action="store_true", help="Принудительно использовать CPU")
gen_parser.add_argument("--output", type=Path, default=Path("models/generated_plan.json"), help="Путь для сохранения результата")


def parse_args():
    return parser.parse_args()


if __name__ == "__main__":
    args = parse_args()
    if args.command == "train":
        train(args)
    elif args.command == "generate":
        generate_plan(args)
    else:
        parser.print_help()
