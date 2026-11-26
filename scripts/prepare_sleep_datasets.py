#!/usr/bin/env python3
"""Готовит датасет сна и SpO2 из открытых PhysioNet-датасетов.

Источники (распакованы в datasets/raw):
  - SHHS PSG          — SpO2 (SaO2), PR, гипнограмма → sleep_hours
  - SleepAccel        — Apple Watch HR + размеченные стадии сна → sleep_hours
  - Apnea-ECG (_r)    — SpO2 (только записи с суффиксом r)
  - PMData (Fitbit)   — minutesAsleep → sleep_hours, HR

Результат: datasets/processed/sleep_dataset.csv с колонками
heart_rate, hrv, spo2, temperature, systolic_pressure, diastolic_pressure,
sleep_hours, label.

Запуск:
    python scripts/prepare_sleep_datasets.py
"""

from __future__ import annotations

import json
import os
from pathlib import Path

import numpy as np
import pandas as pd

PROJECT_ROOT = Path(__file__).resolve().parent.parent
RAW_ROOT = PROJECT_ROOT / "datasets" / "raw"
OUTPUT = PROJECT_ROOT / "datasets" / "processed" / "sleep_dataset.csv"

FEATURE_COLUMNS = [
    "heart_rate",
    "hrv",
    "spo2",
    "temperature",
    "systolic_pressure",
    "diastolic_pressure",
    "sleep_hours",
]

# Стадии сна: 0 = wake, 1..4 = N1/N2/N3/REM, 5 = movement, 6 = unscored.
SLEEP_STAGES = {1, 2, 3, 4}


def empty_row() -> dict:
    return {c: np.nan for c in FEATURE_COLUMNS}


def _finite_or_none(value) -> float | None:
    try:
        v = float(value)
    except (TypeError, ValueError):
        return None
    return v if np.isfinite(v) else None


def _median_in_range(values, low: float, high: float) -> float | None:
    arr = np.asarray(values, dtype=float)
    arr = arr[np.isfinite(arr)]
    arr = arr[(arr > low) & (arr < high)]
    return float(np.median(arr)) if arr.size else None


# ----------------------------------------------------------------------
# Разметка (та же таблица, что в prepare_raw_datasets.py)
# ----------------------------------------------------------------------
def _is_illness(m) -> bool:
    return m[3] >= 38.0 or m[2] <= 93.0


def _is_recovery(m) -> bool:
    return m[0] < 60 and m[1] > 70 and m[4] >= 7


def _is_overtraining(m) -> bool:
    return m[1] < 25 or m[4] < 5


def _is_power_hiit(m) -> bool:
    return m[0] >= 160 and m[1] >= 45


def _is_endurance_threshold(m) -> bool:
    return m[0] >= 140 and m[1] >= 35


def _is_endurance_basic(m) -> bool:
    return m[0] >= 110


LABEL_RULES = (
    ("illness", _is_illness),
    ("recovery", _is_recovery),
    ("overtraining", _is_overtraining),
    ("power_hiit", _is_power_hiit),
    ("endurance_threshold", _is_endurance_threshold),
    ("endurance_basic", _is_endurance_basic),
)


def rule_based_label(row) -> str:
    values = (
        float(row.get("heart_rate", 0) or 0),
        float(row.get("hrv", 0) or 0),
        float(row.get("spo2", 100) or 100),
        float(row.get("temperature", 37) or 37),
        float(row.get("sleep_hours", 7) or 7),
    )
    for label, predicate in LABEL_RULES:
        if predicate(values):
            return label
    return "unknown"


# ----------------------------------------------------------------------
# SHHS
# ----------------------------------------------------------------------
def _shhs_row(sig, fields, ann) -> dict:
    row = empty_row()
    names = [f.lower() for f in fields["sig_name"]]

    spo2_idx = next((i for i, n in enumerate(names) if "sao2" in n or "spo2" in n), None)
    if spo2_idx is not None:
        value = _median_in_range(sig[:, spo2_idx], 50, 100)
        if value is not None:
            row["spo2"] = value

    hr_idx = next((i for i, n in enumerate(names) if n in ("pr", "hr", "heart_rate")), None)
    if hr_idx is not None:
        value = _median_in_range(sig[:, hr_idx], 20, 250)
        if value is not None:
            row["heart_rate"] = value

    fs = float(fields["fs"])
    symbols = list(ann.symbol)
    if fs > 0 and symbols:
        sleep_seconds = sum(30.0 for sym in symbols if str(sym) in {"1", "2", "3", "4"})
        row["sleep_hours"] = min(24.0, sleep_seconds / 3600.0)
    return row


def process_shhs() -> list[dict]:
    import wfdb

    dataset = RAW_ROOT / "sleep-heart-health-study-psg-database-1.0.0"
    if not dataset.exists():
        print("SHHS: missing", flush=True)
        return []

    records = []
    for hea in sorted(dataset.glob("*.hea")):
        try:
            sig, fields = wfdb.rdsamp(str(hea.with_suffix("")))
            ann = wfdb.rdann(str(hea.with_suffix("")), "hypn")
        except Exception as exc:
            print(f"SHHS {hea.stem}: {exc}", flush=True)
            continue
        records.append(_shhs_row(sig, fields, ann))
    return records


# ----------------------------------------------------------------------
# SleepAccel (Apple Watch)
# ----------------------------------------------------------------------
def _sleepaccel_row(label_file: Path, hr_file: Path) -> dict:
    row = empty_row()

    try:
        arr = np.loadtxt(label_file, comments="#")
        if arr.ndim == 2 and arr.shape[1] >= 2:
            seconds, labels = arr[:, 0], arr[:, 1].astype(int)
            total = float(seconds.max() - seconds.min()) + 30.0
            sleep = float(((labels >= 1) & (labels <= 4)).sum() * 30.0)
            if total > 0:
                row["sleep_hours"] = min(24.0, sleep / 3600.0)
    except Exception:
        pass

    if hr_file.exists():
        try:
            hr = np.loadtxt(hr_file, delimiter=",", comments="#")
            if hr.ndim == 2 and hr.shape[1] >= 2:
                value = _median_in_range(hr[:, 1], 20, 250)
                if value is not None:
                    row["heart_rate"] = value
        except Exception:
            pass
    return row


def process_sleepaccel() -> list[dict]:
    dataset = RAW_ROOT / "motion-and-heart-rate-from-a-wrist-worn-wearable-and-labeled-sleep-from-polysomnography-1.0.0"
    if not dataset.exists():
        print("SleepAccel: missing", flush=True)
        return []

    hr_dir = dataset / "heart_rate"
    records = []
    for label_file in sorted((dataset / "labels").glob("*_labeled_sleep.txt")):
        subject = label_file.name.split("_")[0]
        records.append(_sleepaccel_row(label_file, hr_dir / f"{subject}_heartrate.txt"))
    return records


# ----------------------------------------------------------------------
# Apnea-ECG (только записи с SpO2)
# ----------------------------------------------------------------------
def process_apnea_ecg() -> list[dict]:
    import wfdb

    dataset = RAW_ROOT / "apnea-ecg-database-1.0.0"
    if not dataset.exists():
        print("Apnea-ECG: missing", flush=True)
        return []

    records = []
    for hea in sorted(dataset.glob("*.hea")):
        if "spo2" not in hea.read_text(errors="ignore").lower():
            continue
        try:
            sig, fields = wfdb.rdsamp(str(hea.with_suffix("")))
        except Exception:
            continue

        names = [f.lower() for f in fields["sig_name"]]
        idx = next((i for i, n in enumerate(names) if "spo2" in n), None)
        if idx is None:
            continue
        value = _median_in_range(sig[:, idx], 50, 100)
        if value is None:
            continue
        row = empty_row()
        row["spo2"] = value
        records.append(row)
    return records


# ----------------------------------------------------------------------
# PMData (Fitbit)
# ----------------------------------------------------------------------
def _pmdata_sleep_hours(sleep_file: Path) -> float | None:
    try:
        sessions = json.loads(sleep_file.read_text(encoding="utf-8"))
    except Exception:
        return None
    minutes = [float(s.get("minutesAsleep", 0)) for s in sessions if s.get("minutesAsleep")]
    if not minutes:
        return None
    return min(24.0, float(np.mean(minutes)) / 60.0)


def _pmdata_heart_rate(hr_file: Path) -> float | None:
    try:
        samples = json.loads(hr_file.read_text(encoding="utf-8"))
    except Exception:
        return None
    bpm = [
        float(s["value"]["bpm"])
        for s in samples
        if isinstance(s.get("value"), dict) and s["value"].get("bpm")
    ]
    return _median_in_range(bpm, 20, 250)


def process_pmdata() -> list[dict]:
    dataset = RAW_ROOT / "pmdata"
    if not dataset.exists():
        print("PMData: missing", flush=True)
        return []

    records = []
    for subject in sorted(dataset.iterdir()):
        fitbit = subject / "fitbit"
        sleep_file = fitbit / "sleep.json"
        if not sleep_file.exists():
            continue
        row = empty_row()
        hours = _pmdata_sleep_hours(sleep_file)
        if hours is not None:
            row["sleep_hours"] = hours
        hr = _pmdata_heart_rate(fitbit / "heart_rate.json")
        if hr is not None:
            row["heart_rate"] = hr
        records.append(row)
    return records



# ----------------------------------------------------------------------
# UCDDB (EDF .rec + _stage.txt)
# ----------------------------------------------------------------------
def _ucddb_row(rec_path: Path, stage_path: Path) -> dict | None:
    import mne
    import shutil
    import tempfile

    tmp = tempfile.NamedTemporaryFile(suffix=".edf", delete=False)
    tmp_path = tmp.name
    tmp.close()
    try:
        shutil.copy(rec_path, tmp_path)
        raw = mne.io.read_raw_edf(tmp_path, preload=True, verbose="ERROR")
    except Exception:
        return None
    finally:
        try:
            os.remove(tmp_path)
        except OSError:
            pass

    row = empty_row()
    names = [n.lower() for n in raw.ch_names]
    spo2_idx = next((i for i, n in enumerate(names) if "spo2" in n), None)
    pulse_idx = next((i for i, n in enumerate(names) if n in ("pulse", "pr", "hr")), None)

    if spo2_idx is not None:
        value = _median_in_range(raw.get_data()[spo2_idx], 50, 100)
        if value is not None:
            row["spo2"] = value
    if pulse_idx is not None:
        value = _median_in_range(raw.get_data()[pulse_idx], 20, 250)
        if value is not None:
            row["heart_rate"] = value

    try:
        stages = [int(float(x)) for x in stage_path.read_text().split()]
        sleep = sum(30.0 for s in stages if s in SLEEP_STAGES)
        row["sleep_hours"] = min(24.0, sleep / 3600.0)
    except Exception:
        pass
    return row


def process_ucddb() -> list[dict]:
    dataset = RAW_ROOT / "ucddb-1.0.0"
    if not dataset.exists():
        print("UCDDB: missing", flush=True)
        return []

    records = []
    for rec in sorted(dataset.glob("*.rec")):
        stage = rec.with_name(rec.stem + "_stage.txt")
        if not stage.exists():
            continue
        row = _ucddb_row(rec, stage)
        if row is not None:
            records.append(row)
    return records



# ----------------------------------------------------------------------
# Sleep-EDF PSG (core temperature; гипнограммы в скачанной копии битые)
# ----------------------------------------------------------------------
def _sleep_stage_seconds(annotations) -> float:
    """Считает секунды сна по EDF+ аннотациям Sleep-EDF."""
    total = 0.0
    for _, texts in annotations:
        for text in texts:
            low = text.lower()
            if "sleep stage" not in low:
                continue
            stage = low.split("sleep stage", 1)[1].strip()
            if stage.startswith(("1", "2", "3", "4", "r")):
                total += 30.0
    return total


def process_sleep_edf() -> list[dict]:
    import mne
    from edf_reader import read_edf_annotations

    dataset = RAW_ROOT / "sleep-edfx"
    if not dataset.exists():
        print("Sleep-EDF: missing", flush=True)
        return []

    hypnograms: dict[str, Path] = {}
    for path in sorted(dataset.glob("SC*-Hypnogram.edf")):
        hypnograms[path.name[:6]] = path

    records = []
    for psg in sorted(dataset.glob("SC*-PSG.edf")):
        subject = psg.name[:6]

        row = empty_row()
        try:
            raw = mne.io.read_raw_edf(psg, preload=True, verbose="ERROR")
            names = [n.lower() for n in raw.ch_names]
            idx = next((i for i, n in enumerate(names) if "temp" in n), None)
            if idx is not None:
                value = _median_in_range(raw.get_data()[idx], 30, 40)
                if value is not None:
                    row["temperature"] = value
        except Exception:
            pass

        hyp = hypnograms.get(subject)
        if hyp is not None:
            try:
                seconds = _sleep_stage_seconds(read_edf_annotations(hyp))
            except Exception:
                seconds = 0.0  # битый файл (S3 NoSuchKey) — пропускаем
            if seconds > 0:
                row["sleep_hours"] = min(24.0, seconds / 3600.0)

        if not (row["temperature"] == row["temperature"] or row["sleep_hours"] == row["sleep_hours"]):
            continue
        records.append(row)

    return records


def main() -> None:
    records: list[dict] = []
    for processor in (process_shhs, process_sleepaccel, process_apnea_ecg, process_pmdata, process_ucddb):
        records.extend(processor())

    # Sleep-EDF PSG содержит core temperature (~37 °C), а E4 — кожу (~33 °C).
    # Смешивать их в одном признаке нельзя, поэтому пишем отдельный файл.
    edf_rows = process_sleep_edf()
    if edf_rows:
        edf_path = OUTPUT.parent / "sleep_edf_dataset.csv"
        pd.DataFrame(edf_rows).to_csv(edf_path, index=False)
        print(f"Sleep-EDF (core temp): {len(edf_rows)} rows -> {edf_path}", flush=True)

    if not records:
        raise SystemExit("Нет данных для сна/SpO2")

    df = pd.DataFrame(records)
    df["label"] = df.apply(rule_based_label, axis=1)

    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    df.to_csv(OUTPUT, index=False)
    print(f"\nSaved {len(df)} rows to {OUTPUT}", flush=True)
    print(df["label"].value_counts().to_string(), flush=True)
    print(df[FEATURE_COLUMNS].describe().to_string(), flush=True)


if __name__ == "__main__":
    main()