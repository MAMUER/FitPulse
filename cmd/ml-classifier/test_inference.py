#!/usr/bin/env python3
"""
Quick inference test for the trained classifier.
Run this to validate the model before deleting raw data.
"""

# === Подавление CUDA / TensorFlow шума (должно быть самым первым) ===
import os
os.environ['TF_CPP_MIN_LOG_LEVEL'] = '3'
os.environ['TF_ENABLE_ONEDNN_OPTS'] = '0'
os.environ['CUDA_VISIBLE_DEVICES'] = ''
# === конец ===

import pandas as pd
import joblib
from pathlib import Path
import keras
import numpy as np

# Paths (adjusted for container layout where code is in /app/cmd)
SCRIPT_DIR = Path(__file__).resolve().parent
PROJECT_ROOT = SCRIPT_DIR.parent.parent   # /app
MODELS_DIR = PROJECT_ROOT / "models"
DATA_PATH = PROJECT_ROOT / "datasets" / "processed" / "training_data_real_v3.csv"

def find_latest_model():
    models = sorted(MODELS_DIR.glob("classifier_*.keras"))
    if models:
        return models[-1]
    return MODELS_DIR / "classifier.keras"

def load_latest_model_and_scaler():
    model_path = find_latest_model()
    scaler_path = MODELS_DIR / "scaler.pkl"

    print(f"Loading model: {model_path.name}")
    model = keras.models.load_model(model_path)

    print(f"Loading scaler: {scaler_path.name}")
    scaler = joblib.load(scaler_path)

    return model, scaler

def predict_sample(model, scaler, row):
    features = np.array([[
        row["hr"],
        row["hrv"],
        row["spo2"],
        row["temp"],
        row["bp_s"],
        row["bp_d"],
        row["sleep"]
    ]])

    features_scaled = scaler.transform(features)
    features_scaled = np.nan_to_num(features_scaled, nan=0.0)

    probs = model.predict(features_scaled, verbose=0)[0]
    pred_class = int(np.argmax(probs))
    confidence = float(probs[pred_class])

    return pred_class, confidence, probs

def main():
    print("=== Classifier Inference Test ===\n")

    model, scaler = load_latest_model_and_scaler()

    # Load a small sample of the processed data
    print(f"\nLoading test data from {DATA_PATH.name}...")
    df = pd.read_csv(DATA_PATH)

    # Take a few samples from each class for balanced testing
    test_samples = []
    for label in range(4):
        samples = df[df["label"] == label].sample(n=3, random_state=42)
        test_samples.append(samples)

    test_df = pd.concat(test_samples).reset_index(drop=True)

    print(f"\nTesting on {len(test_df)} samples...\n")

    correct = 0
    for idx, row in test_df.iterrows():
        true_label = int(row["label"])
        pred_label, confidence, probs = predict_sample(model, scaler, row)

        status = "✓" if pred_label == true_label else "✗"
        if pred_label == true_label:
            correct += 1

        print(f"Sample {idx+1:2d} | True: {true_label} | Pred: {pred_label} | "
              f"Conf: {confidence:.3f} | {status}")

    accuracy = correct / len(test_df)
    print(f"\n=== Test Accuracy on {len(test_df)} samples: {accuracy:.1%} ===")

    print("\nModel is working. You can safely delete the raw data now if you want.")

if __name__ == "__main__":
    main()
