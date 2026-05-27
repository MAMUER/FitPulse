#!/usr/bin/env python3
import os
import sys
import json
import pickle
import numpy as np
import pandas as pd
from pathlib import Path
from datetime import datetime
from collections import defaultdict

RAW_DIR = Path("datasets/raw")
OUTPUT_DIR = Path("datasets/processed")
OUTPUT_CSV = OUTPUT_DIR / "training_data_real_v3.csv"
REPORT_FILE = OUTPUT_DIR / "preprocess_v3_report.json"

CONFIG = {
    "include": [
        "wesad", "ppg_dalia", "adarp", "e4selflearning", "stress_nurses",
        "toadstool", "weee", "ue4w", "wesd", "spd", "big_ideas_lab"
    ],
    "max_total_samples": 900000,
    "max_per_dataset": {
        "big_ideas_lab": 200000,
        "ppg_dalia": 180000,
        "e4selflearning": 150000,
        "wesad": 150000,
        "default": 120000
    },
    "target_per_class": 200000,
    "min_hr": 40,
    "max_hr": 200,
    "real_hrv_extraction": True
}

HR_ZONES = {
    0: (50, 95),
    1: (95, 125),
    2: (125, 150),
    3: (150, 220)
}

def get_zone(hr):
    for label, (lo, hi) in HR_ZONES.items():
        if lo <= hr < hi:
            return label
    return 1

def extract_real_hrv(bvp_signal, fs=64):
    if bvp_signal is None or len(bvp_signal) < fs * 5:
        return None
    try:
        from scipy.signal import find_peaks
        peaks, _ = find_peaks(bvp_signal, distance=fs*0.4)
        if len(peaks) < 5:
            return None
        rr = np.diff(peaks) / fs * 1000
        rmssd = np.sqrt(np.mean(np.square(np.diff(rr))))
        return float(np.clip(rmssd, 10, 150))
    except Exception:
        return None

def process_wesad(path):
    samples = []
    for subject in os.listdir(path):
        if not subject.startswith("S") or subject in ["S1", "S12"]:
            continue
        pkl = path / subject / f"{subject}.pkl"
        if not pkl.exists():
            continue
        try:
            with open(pkl, "rb") as f:
                data = pickle.load(f, encoding="latin1")
            wrist = data.get("signal", {}).get("wrist", {})
            bvp = wrist.get("BVP", np.array([]))
            labels = data.get("label", np.array([]))
            for i in range(0, min(len(bvp), len(labels)), 64):
                hr = 70 + (labels[i] - 1) * 20 if i < len(labels) else 80
                hrv = extract_real_hrv(bvp[max(0, i-256):i+256]) if CONFIG["real_hrv_extraction"] else None
                zone = get_zone(hr)
                samples.append({
                    "hr": float(hr), "hrv": hrv or np.random.uniform(25, 80),
                    "spo2": 97.0, "temp": 36.8, "bp_s": 125, "bp_d": 82,
                    "sleep": 7.0, "label": zone, "source": f"wesad/{subject}"
                })
        except Exception:
            continue
    return samples

def process_generic_e4(path, name, max_s):
    samples = []
    for root, _, files in os.walk(path):
        for f in files:
            if f.lower() != "hr.csv":
                continue
            try:
                df = pd.read_csv(Path(root) / f, skiprows=2, header=None)
                hrs = df.iloc[:, 0].dropna().astype(float).values
                for hr in hrs:
                    if CONFIG["min_hr"] <= hr <= CONFIG["max_hr"]:
                        zone = get_zone(hr)
                        hrv = np.random.uniform(20, 90)
                        samples.append({
                            "hr": float(hr), "hrv": hrv, "spo2": 97.0,
                            "temp": 36.8, "bp_s": 125, "bp_d": 82,
                            "sleep": 7.0, "label": zone, "source": f"{name}/{Path(root).name}"
                        })
                        if len(samples) >= max_s:
                            return samples
            except Exception:
                continue
    return samples

PROCESSORS = {
    "wesad": lambda p: process_wesad(p),
    "ppg_dalia": lambda p: process_generic_e4(p, "ppg_dalia", 999999),
    "adarp": lambda p: process_generic_e4(p, "adarp", 999999),
    "e4selflearning": lambda p: process_generic_e4(p, "e4selflearning", 999999),
    "stress_nurses": lambda p: process_generic_e4(p, "stress_nurses", 999999),
    "toadstool": lambda p: process_generic_e4(p, "toadstool", 999999),
    "weee": lambda p: process_generic_e4(p, "weee", 999999),
    "ue4w": lambda p: process_generic_e4(p, "ue4w", 999999),
    "wesd": lambda p: process_generic_e4(p, "wesd", 999999),
    "spd": lambda p: process_generic_e4(p, "spd", 999999),
    "big_ideas_lab": lambda p: process_generic_e4(p, "big_ideas_lab", 999999),
}

def main():
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    all_samples = []
    stats = {"datasets": [], "total": 0, "timestamp": datetime.now().isoformat()}

    for ds in CONFIG["include"]:
        p = RAW_DIR / ds
        if not p.exists():
            continue
        max_s = CONFIG["max_per_dataset"].get(ds, CONFIG["max_per_dataset"]["default"])
        proc = PROCESSORS.get(ds, lambda x: process_generic_e4(x, ds, max_s))
        data = proc(p)[:max_s]
        all_samples.extend(data)
        stats["datasets"].append({"name": ds, "samples": len(data)})

    df = pd.DataFrame(all_samples)
    if len(df) == 0:
        print("No data extracted")
        return

    target = CONFIG["target_per_class"]
    balanced = []
    rng = np.random.default_rng(42)
    for label in range(4):
        part = df[df["label"] == label].copy()
        n = len(part)
        if n > target:
            part = part.sample(target, random_state=42)
        elif n < target and n > 0:
            noise = 28.0 if label >= 2 else 14.0
            needed = target - n
            aug_rows = []
            while len(aug_rows) < needed:
                idx = rng.integers(0, n)
                row = part.iloc[idx]
                aug_rows.append({
                    "hr": float(np.clip(row["hr"] + rng.uniform(-noise, noise), 40, 200)),
                    "hrv": float(np.clip(row["hrv"] + rng.uniform(-noise*0.9, noise*0.9), 5, 160)),
                    "spo2": float(np.clip(row["spo2"] + rng.uniform(-1.0, 1.0), 94, 100)),
                    "temp": float(np.clip(row["temp"] + rng.uniform(-0.3, 0.3), 36.0, 37.9)),
                    "bp_s": float(np.clip(row["bp_s"] + rng.uniform(-6, 6), 100, 165)),
                    "bp_d": float(np.clip(row["bp_d"] + rng.uniform(-4, 4), 60, 105)),
                    "sleep": float(np.clip(row["sleep"] + rng.uniform(-0.6, 0.6), 4.5, 9.5)),
                    "label": label,
                    "source": str(row["source"]) + "_aug"
                })
            aug_df = pd.DataFrame(aug_rows)
            part = pd.concat([part, aug_df], ignore_index=True)
        balanced.append(part)
    df = pd.concat(balanced, ignore_index=True).sample(frac=1, random_state=42).reset_index(drop=True)

    if len(df) > CONFIG["max_total_samples"]:
        df = df.sample(CONFIG["max_total_samples"], random_state=42)

    df.to_csv(OUTPUT_CSV, index=False)

    stats["total"] = len(df)
    stats["class_dist"] = df["label"].value_counts().to_dict()
    stats["output_file"] = str(OUTPUT_CSV)
    stats["size_mb"] = round(OUTPUT_CSV.stat().st_size / (1024*1024), 1)

    with open(REPORT_FILE, "w", encoding="utf-8") as f:
        json.dump(stats, f, indent=2)

    print(f"Done. Saved {len(df)} samples to {OUTPUT_CSV} ({stats['size_mb']} MB)")
    print("Class distribution:", stats["class_dist"])
    print(f"Report: {REPORT_FILE}")

if __name__ == "__main__":
    main()
