# cmd/ml-classifier/train.py
"""
Training script for ML Classifier - Keras 3 compatible
"""

# === Подавление CUDA / XLA / TensorFlow шума (должно быть САМЫМ ПЕРВЫМ) ===
import os

os.environ["TF_CPP_MIN_LOG_LEVEL"] = "3"
os.environ["TF_ENABLE_ONEDNN_OPTS"] = "0"
os.environ["CUDA_VISIBLE_DEVICES"] = ""
os.environ["KERAS_BACKEND"] = "tensorflow"

import tensorflow as tf

tf.get_logger().setLevel("ERROR")
# === конец подавления ===

import json
from datetime import datetime
from pathlib import Path

import joblib
import keras
import matplotlib.pyplot as plt

# === MLflow integration (LOCAL ONLY) ===
import mlflow
import mlflow.keras
import numpy as np
import pandas as pd
from keras import callbacks, layers, models
from mlflow.models.signature import infer_signature
from sklearn.metrics import classification_report
from sklearn.model_selection import train_test_split
from sklearn.preprocessing import StandardScaler
from sklearn.utils.class_weight import compute_class_weight

# Локальный трекинг — файл в той же директории
mlflow.set_tracking_uri("sqlite:///mlflow.db")
mlflow.set_experiment("fitpulse-classifier")
# === конец MLflow setup ===

# Robust paths based on script location
SCRIPT_DIR = Path(__file__).resolve().parent
PROJECT_ROOT = SCRIPT_DIR.parent.parent
DATA_PATH = PROJECT_ROOT / "datasets" / "processed" / "training_data_real_v3.csv"
MODELS_DIR = PROJECT_ROOT / "models"

TRAINING_CLASSES = {
    0: {"name": "recovery", "name_ru": "Восстановление", "hr_range": "50-65% HRmax"},
    1: {
        "name": "endurance_e1e2",
        "name_ru": "Базовая выносливость (E1-E2)",
        "hr_range": "65-80% HRmax",
    },
    2: {
        "name": "threshold_e3",
        "name_ru": "Пороговая выносливость (E3)",
        "hr_range": "80-90% HRmax",
    },
    3: {
        "name": "strength_hiit",
        "name_ru": "Силовая/HIIT",
        "hr_range": "90-100% HRmax",
    },
}


def load_real_data():
    data_path = str(DATA_PATH)

    if not DATA_PATH.exists():
        raise FileNotFoundError(f"Данные не найдены: {data_path}")

    print(f"Загрузка данных: {data_path}")
    df = pd.read_csv(data_path)

    required_cols = ["hr", "hrv", "spo2", "temp", "bp_s", "bp_d", "sleep", "label"]
    missing_cols = [col for col in required_cols if col not in df.columns]
    if missing_cols:
        raise ValueError(f"Отсутствуют колонки: {missing_cols}")

    for col in required_cols[:-1]:
        df[col] = df[col].replace([np.inf, -np.inf], np.nan)
        df[col] = df[col].fillna(df[col].median())

    print("\nРаспределение классов:")
    print(df["label"].value_counts())

    X = df[required_cols[:-1]].values
    y = df["label"].values.astype(int)

    print(f"\nЗагружено {len(df)} сэмплов")
    print(f"   Источников данных: {df['source'].nunique()}")

    return X, y


def create_classifier_model(input_shape=7, num_classes=4):
    model = models.Sequential(
        [
            layers.Input(shape=(input_shape,)),
            layers.GaussianNoise(0.05),
            layers.Dense(256, activation="relu", kernel_regularizer=keras.regularizers.l2(0.003)),
            layers.BatchNormalization(),
            layers.Dropout(0.4),
            layers.Dense(128, activation="relu", kernel_regularizer=keras.regularizers.l2(0.003)),
            layers.BatchNormalization(),
            layers.Dropout(0.4),
            layers.Dense(64, activation="relu", kernel_regularizer=keras.regularizers.l2(0.003)),
            layers.BatchNormalization(),
            layers.Dropout(0.3),
            layers.Dense(32, activation="relu", kernel_regularizer=keras.regularizers.l2(0.003)),
            layers.BatchNormalization(),
            layers.Dropout(0.3),
            layers.Dense(num_classes, activation="softmax"),
        ]
    )

    model.compile(
        optimizer=keras.optimizers.Adam(learning_rate=0.0003),
        loss="sparse_categorical_crossentropy",
        metrics=["accuracy"],
    )

    return model


def train_model():
    """Main training function with MLflow tracking"""
    print("=" * 70)
    print("ОБУЧЕНИЕ КЛАССИФИКАТОРА v3 (с MLflow)")
    print("=" * 70)

    # === MLflow: начало трекинга ===
    with mlflow.start_run(run_name=f"v3_{datetime.now().strftime('%Y%m%d_%H%M')}") as run:
        run_id = run.info.run_id
        print(f"\n🔬 MLflow Run ID: {run_id}")

        # Логируем параметры
        mlflow.log_params(
            {
                "input_shape": 7,
                "num_classes": 4,
                "optimizer": "Adam",
                "learning_rate": 0.0003,
                "l2_reg": 0.003,
                "dropout_rate": 0.4,
                "batch_size": 256,
                "max_epochs": 10,
                "patience": 5,
            }
        )
        # === конец параметров ===

        print("\n[1/5] Загрузка данных...")
        X, y = load_real_data()

        # Логируем метаданные данных
        mlflow.log_param("total_samples", len(X))
        mlflow.log_param("sources_count", pd.read_csv(DATA_PATH)["source"].nunique())
        mlflow.log_param(
            "class_distribution",
            {TRAINING_CLASSES[i]["name"]: int(np.sum(y == i)) for i in range(4)},
        )

        print("\n[2/5] Разделение данных...")
        X_train, X_test, y_train, y_test = train_test_split(
            X, y, test_size=0.2, random_state=42, stratify=y
        )
        print(f"   Train: {len(X_train)}, Test: {len(X_test)}")

        print("\n[3/5] Скалирование...")
        scaler = StandardScaler()
        X_train_scaled = scaler.fit_transform(X_train)
        X_test_scaled = scaler.transform(X_test)
        X_train_scaled = np.nan_to_num(X_train_scaled, nan=0.0)
        X_test_scaled = np.nan_to_num(X_test_scaled, nan=0.0)

        MODELS_DIR.mkdir(parents=True, exist_ok=True)
        scaler_path = MODELS_DIR / "scaler.pkl"
        joblib.dump(scaler, scaler_path)

        # Логируем scaler как артефакт
        mlflow.log_artifact(str(scaler_path), "preprocessing")

        print("\n[4/5] Создание модели...")
        model = create_classifier_model(input_shape=X_train_scaled.shape[1])

        # Логируем архитектуру модели
        model_summary = []
        model.summary(print_fn=lambda x: model_summary.append(x))
        mlflow.log_text("\n".join(model_summary), "model_architecture.txt")

        print("\n⚖️ Расчет весов классов...")
        class_weights = compute_class_weight("balanced", classes=np.unique(y_train), y=y_train)
        class_weight_dict = dict(enumerate(class_weights))
        print(f"Веса классов: {class_weight_dict}")
        mlflow.log_params({f"class_weight_{i}": float(w) for i, w in class_weight_dict.items()})

        early_stop = callbacks.EarlyStopping(
            monitor="val_loss", patience=5, restore_best_weights=True, verbose=1
        )
        reduce_lr = callbacks.ReduceLROnPlateau(
            monitor="val_loss", factor=0.5, patience=3, min_lr=1e-6, verbose=1
        )
        checkpoint = callbacks.ModelCheckpoint(
            str(MODELS_DIR / "classifier.keras"),
            monitor="val_accuracy",
            save_best_only=True,
            verbose=1,
        )

        print("\n[5/5] Обучение...")
        history = model.fit(
            X_train_scaled,
            y_train,
            validation_data=(X_test_scaled, y_test),
            epochs=10,
            batch_size=256,
            class_weight=class_weight_dict,
            callbacks=[early_stop, reduce_lr, checkpoint],
            verbose=1,
        )

        # === Логируем метрики обучения ===
        for epoch, (acc, val_acc, train_loss, val_loss) in enumerate(
            zip(
                history.history["accuracy"],
                history.history["val_accuracy"],
                history.history["loss"],
                history.history["val_loss"],
            )
        ):
            mlflow.log_metrics(
                {
                    "train_accuracy": acc,
                    "val_accuracy": val_acc,
                    "train_loss": train_loss,
                    "val_loss": val_loss,
                },
                step=epoch,
            )
        # === конец метрик ===

        # === Оценка на тесте ===
        print("\n" + "=" * 70)
        print("РЕЗУЛЬТАТЫ")
        print("=" * 70)

        y_pred = np.argmax(model.predict(X_test_scaled, verbose=0), axis=1)
        test_accuracy = float(np.mean(y_pred == y_test))

        print("\nClassification Report:")
        report = classification_report(
            y_test,
            y_pred,
            target_names=[TRAINING_CLASSES[i]["name_ru"] for i in range(4)],
            output_dict=True,
        )
        print(
            classification_report(
                y_test,
                y_pred,
                target_names=[TRAINING_CLASSES[i]["name_ru"] for i in range(4)],
            )
        )

        # Логируем метрики качества
        for class_name, metrics in report.items():
            if class_name in TRAINING_CLASSES.values():
                mlflow.log_metrics(
                    {
                        f"{class_name}_precision": metrics["precision"],
                        f"{class_name}_recall": metrics["recall"],
                        f"{class_name}_f1": metrics["f1-score"],
                    }
                )

        mlflow.log_metric("test_accuracy", test_accuracy)
        mlflow.log_metric("test_samples", len(X_test))
        # === конец оценки ===

        # === Сохранение модели в MLflow ===
        # Infer signature для serving
        signature = infer_signature(
            X_train_scaled[:1],  # пример входа
            model.predict(X_train_scaled[:1], verbose=0),  # пример выхода
        )

        # Логируем модель
        mlflow.keras.log_model(
            model,
            artifact_path="model",
            signature=signature,
            input_example=X_train_scaled[:1].tolist(),
        )

        # Сохраняем локально тоже
        model_path = MODELS_DIR / "classifier.keras"
        model.save(model_path)
        mlflow.log_artifact(str(model_path), "models")

        # Timestamped version
        ts = datetime.now().strftime("%Y%m%d_%H%M")
        ts_model_path = MODELS_DIR / f"classifier_{ts}.keras"
        model.save(ts_model_path)
        mlflow.log_artifact(str(ts_model_path), "models/timestamped")
        # === конец сохранения ===

        # === История обучения ===
        training_history = {
            "accuracy": [float(a) for a in history.history["accuracy"]],
            "val_accuracy": [float(a) for a in history.history["val_accuracy"]],
            "loss": [float(loss_val) for loss_val in history.history["loss"]],
            "val_loss": [float(loss_val) for loss_val in history.history["val_loss"]],
            "timestamp": datetime.now().isoformat(),
            "run_id": run_id,
            "classes": TRAINING_CLASSES,
            "class_weights": {k: float(v) for k, v in class_weight_dict.items()},
            "metrics": {
                "test_accuracy": test_accuracy,
                "train_samples": len(X_train),
                "test_samples": len(X_test),
            },
        }

        history_path = MODELS_DIR / "training_history.json"
        with open(history_path, "w", encoding="utf-8") as f:
            json.dump(training_history, f, indent=2, ensure_ascii=False)
        mlflow.log_artifact(str(history_path), "metadata")

        # Графики
        plt.figure(figsize=(14, 5))

        plt.subplot(1, 2, 1)
        plt.plot(history.history["accuracy"], label="Train Acc")
        plt.plot(history.history["val_accuracy"], label="Val Acc")
        plt.xlabel("Epoch")
        plt.ylabel("Accuracy")
        plt.legend()
        plt.grid(True, alpha=0.3)

        plt.subplot(1, 2, 2)
        plt.plot(history.history["loss"], label="Train Loss")
        plt.plot(history.history["val_loss"], label="Val Loss")
        plt.xlabel("Epoch")
        plt.ylabel("Loss")
        plt.legend()
        plt.grid(True, alpha=0.3)

        plt.tight_layout()
        plot_path = MODELS_DIR / "training_history.png"
        plt.savefig(plot_path, dpi=150, bbox_inches="tight")
        mlflow.log_artifact(str(plot_path), "plots")
        plt.close()
        # === конец графиков ===

        print(f"\n✅ Model logged to MLflow: {mlflow.get_artifact_uri('model')}")
        print("\n" + "=" * 70)
        print("ОБУЧЕНИЕ ЗАВЕРШЕНО!")
        print(f"MLflow UI: {os.environ.get('MLFLOW_TRACKING_URI', 'http://localhost:5000')}")
        print("=" * 70)

        return model, scaler


if __name__ == "__main__":
    train_model()
